package api

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/biodream-llc/bitdispatcher/internal/auth"
	"github.com/biodream-llc/bitdispatcher/internal/cad"
)

type mdtLookupQuery struct {
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	DOB     string `json:"dob"`
	Plate   string `json:"plate"`
	State   string `json:"state"`
	VIN     string `json:"vin"`
	Purpose string `json:"purpose"`
}
type mdtLookupHit struct {
	Flags       []*cad.Flag     `json:"flags,omitempty"`
	Person      *cad.Person     `json:"person,omitempty"`
	Incidents   []cad.PersonRef `json:"incidents,omitempty"`
	Source      string          `json:"source,omitempty"`
	Record      string          `json:"record"`
	Incident    string          `json:"incident"`
	Seen        time.Time       `json:"seen"`
	Name        string          `json:"name,omitempty"`
	DOB         string          `json:"dob,omitempty"`
	Sex         string          `json:"sex,omitempty"`
	Description string          `json:"description,omitempty"`
	Safety      bool            `json:"safetyConcern,omitempty"`
	Plate       string          `json:"plate,omitempty"`
	State       string          `json:"state,omitempty"`
	VIN         string          `json:"vin,omitempty"`
	Year        string          `json:"year,omitempty"`
	Make        string          `json:"make,omitempty"`
	Model       string          `json:"model,omitempty"`
	Color       string          `json:"color,omitempty"`
	Body        string          `json:"body,omitempty"`
}

func lookupID(value string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, strings.ToUpper(value))
}
func lookupName(value string) string {
	return strings.Join(strings.FieldsFunc(strings.ToUpper(value), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) }), " ")
}
func lookupNameMatches(query string, p cad.Person) bool {
	names := append([]string{p.Name, strings.Join([]string{p.First, p.Middle, p.Last, p.Suffix}, " ")}, p.Aliases...)
	for _, name := range names {
		normalized := lookupName(name)
		matched := true
		for _, word := range strings.Fields(query) {
			if !strings.Contains(normalized, word) {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}
func lookupMinor(dob, approximateAge string, now time.Time) bool {
	born, err := time.Parse("2006-01-02", dob)
	if err == nil {
		return born.AddDate(18, 0, 0).After(now)
	}
	for _, value := range []string{approximateAge, dob} {
		value = strings.TrimSpace(value)
		if value != "" && len(value) <= 2 {
			if age, e := strconv.Atoi(value); e == nil && age >= 0 && age < 18 {
				return true
			}
		}
	}
	return false
}

func (s *Server) mdtLookupRoutes() {
	s.handle("POST /api/mdt/records/search", auth.Viewer, func(w http.ResponseWriter, r *http.Request, se *auth.Session, a cad.Actor) {
		unit, ok := s.mdtUnit(w, r, se)
		if !ok {
			return
		}
		var q mdtLookupQuery
		if !readJSON(w, r, &q) {
			return
		}
		q.Kind = strings.ToLower(strings.TrimSpace(q.Kind))
		q.Name = strings.TrimSpace(q.Name)
		q.DOB = strings.TrimSpace(q.DOB)
		q.State = strings.ToUpper(strings.TrimSpace(q.State))
		q.Plate = strings.TrimSpace(q.Plate)
		q.VIN = strings.TrimSpace(q.VIN)
		if !slices.Contains([]string{"assigned-call", "patrol", "follow-up"}, q.Purpose) {
			fail(w, 422, "choose an operational purpose for this search")
			return
		}
		if len(q.Name) > 100 || len(q.Plate) > 20 || len(q.VIN) > 17 || len(q.State) > 2 || len(q.DOB) > 10 {
			fail(w, 422, "a search field is too long")
			return
		}
		name, plate, vin := lookupName(q.Name), lookupID(q.Plate), lookupID(q.VIN)
		switch q.Kind {
		case "person":
			if len([]rune(name)) < 2 {
				fail(w, 422, "enter at least two characters of a person's name")
				return
			}
			if q.DOB != "" {
				if _, err := time.Parse("2006-01-02", q.DOB); err != nil {
					fail(w, 422, "date of birth uses YYYY-MM-DD")
					return
				}
			}
			q.Plate, q.State, q.VIN = "", "", ""
		case "vehicle":
			if len(plate) < 2 && len(vin) < 4 {
				fail(w, 422, "enter at least two plate characters or four VIN characters")
				return
			}
			if q.Plate != "" && len(plate) < 2 || q.VIN != "" && len(vin) < 4 {
				fail(w, 422, "plate needs two characters and VIN needs four when provided")
				return
			}
			q.Name, q.DOB = "", ""
		default:
			fail(w, 422, "choose persons or vehicles")
			return
		}
		hits := []mdtLookupHit{}
		now := s.Eng.Now()
		agency := ""
		s.Eng.Read(func(st *cad.State) {
			own := st.Units[unit]
			if own == nil {
				return
			}
			agency = own.Agency
			for _, in := range st.Incidents {
				if in.Agency != "" && !strings.EqualFold(in.Agency, own.Agency) {
					continue
				}
				// The mobile lookup deliberately excludes juvenile and medical records, even for owner sessions.
				if slices.ContainsFunc(in.Modifiers, func(v string) bool { return strings.EqualFold(v, "JUVENILE") }) {
					continue
				}
				if ct := st.CallTypes[in.Type]; ct != nil && ct.Discipline == "ems" || slices.Contains([]string{"BRTH", "CARD", "CHEST", "FALL", "OD", "TRANSFER", "SICK", "STROKE"}, in.Type) {
					continue
				}
				if q.Kind == "person" {
					for _, p := range in.Persons {
						if p.RecordID != "" {
							if _, ok := st.LASRecords[p.RecordID]; ok {
								continue
							}
						}
						if p.Removed || strings.EqualFold(p.Role, "patient") || slices.ContainsFunc(p.Roles, func(role string) bool { return strings.EqualFold(role, "patient") }) || lookupMinor(p.DOB, p.ApproxAge, now) || !lookupNameMatches(name, p) || q.DOB != "" && p.DOB != q.DOB {
							continue
						}
						display := p.Name
						if display == "" {
							display = strings.TrimSpace(strings.Join([]string{p.First, p.Middle, p.Last, p.Suffix}, " "))
						}
						description := []string{}
						for _, part := range []struct{ label, value string }{{"Height", p.Height}, {"Weight", p.Weight}, {"Hair", p.Hair}, {"Eyes", p.Eyes}} {
							if part.value != "" {
								description = append(description, part.label+": "+part.value)
							}
						}
						p.Returns = nil
						hits = append(hits, mdtLookupHit{Person: &p, Record: p.ID, Incident: in.Number, Seen: in.Created, Name: display, DOB: p.DOB, Sex: p.Sex, Description: strings.Join(description, " · "), Safety: p.Safety})
					}
				} else {
					for _, v := range in.Vehicles {
						if v.Removed || plate != "" && !strings.Contains(lookupID(v.Plate), plate) || vin != "" && !strings.Contains(lookupID(v.VIN), vin) || q.State != "" && !strings.EqualFold(v.State, q.State) {
							continue
						}
						hits = append(hits, mdtLookupHit{Record: v.ID, Incident: in.Number, Seen: in.Created, Plate: v.Plate, State: v.State, VIN: v.VIN, Year: v.Year, Make: v.Make, Model: v.Model, Color: v.Color, Body: v.Body})
					}
				}
			}
			for _, r := range st.LASRecords {
				if r.Archived || r.Restricted || r.Kind != q.Kind || (r.Agency != "" && !strings.EqualFold(r.Agency, own.Agency)) {
					continue
				}
				if q.Kind == "person" {
					p := cad.LASRecordPerson(r)
					flags := []*cad.Flag{}
					for _, f := range st.LASPersonFlags(r.ID, p, now) {
						flags = append(flags, st.FlagFor(f, cad.DutyOf(a), a.Role))
					}
					if lookupMinor(r.DOB, p.ApproxAge, now) || !lookupNameMatches(name, p) || (q.DOB != "" && q.DOB != r.DOB) {
						continue
					}
					refs := []cad.PersonRef{}
					for _, ref := range st.LASPersonRefs(r.ID) {
						if in := st.Incidents[ref.Incident]; in != nil && (in.Agency == "" || strings.EqualFold(in.Agency, own.Agency)) {
							refs = append(refs, ref)
						}
					}
					hits = append(hits, mdtLookupHit{Flags: flags, Person: &p, Incidents: refs, Source: lasLookupSource(r), Record: r.ID, Seen: r.Updated, Name: r.Name, DOB: r.DOB, Sex: r.Sex, Description: r.Description, Safety: r.Safety || len(flags) > 0})
				} else {
					if plate != "" && !strings.Contains(lookupID(r.Plate), plate) || vin != "" && !strings.Contains(lookupID(r.VIN), vin) || q.State != "" && !strings.EqualFold(q.State, r.State) {
						continue
					}
					hits = append(hits, mdtLookupHit{Source: lasLookupSource(r), Record: r.ID, Seen: r.Updated, Plate: r.Plate, State: r.State, VIN: r.VIN, Year: r.Year, Make: r.Make, Model: r.Model, Color: r.Color, Body: r.Body, Description: r.Description, Safety: r.Safety})
				}
			}
		})
		providerMessage := ""
		if q.Kind == "person" && len(hits) == 0 {
			records, message := s.enformionFallback(r.Context(), a, q.Name, q.DOB, agency)
			providerMessage = message
			for _, record := range records {
				p := cad.LASRecordPerson(record)
				flags := []*cad.Flag{}
				s.Eng.Read(func(st *cad.State) {
					for _, f := range st.LASPersonFlags(record.ID, p, now) {
						flags = append(flags, st.FlagFor(f, cad.DutyOf(a), a.Role))
					}
				})
				hits = append(hits, mdtLookupHit{Person: &p, Flags: flags, Source: record.Source, Record: record.ID, Seen: record.Updated, Name: record.Name, DOB: record.DOB, Safety: record.Safety || len(flags) > 0})
			}
		}
		sort.Slice(hits, func(i, j int) bool {
			if hits[i].Seen.Equal(hits[j].Seen) {
				if hits[i].Incident == hits[j].Incident {
					return hits[i].Record < hits[j].Record
				}
				return hits[i].Incident > hits[j].Incident
			}
			return hits[i].Seen.After(hits[j].Seen)
		})
		more := len(hits) > 50
		if more {
			hits = hits[:50]
		}
		fields := []string{}
		for field, value := range map[string]string{"name": q.Name, "dob": q.DOB, "plate": q.Plate, "state": q.State, "vin": q.VIN} {
			if value != "" {
				fields = append(fields, field)
			}
		}
		slices.Sort(fields)
		encoded, _ := json.Marshal(q)
		hash := sha256.Sum256(encoded)
		s.Eng.Record(a, cad.EvSearched, map[string]any{"kind": "mdt-local-" + q.Kind, "unit": unit, "purpose": q.Purpose, "fields": fields, "queryHash": fmt.Sprintf("%x", hash), "results": len(hits), "more": more})
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, 200, map[string]any{"kind": q.Kind, "items": hits, "more": more, "source": "LAS CAD records", "providerMessage": providerMessage, "searchedAt": now.UTC()})
	})
}

func lasLookupSource(r cad.LASRecord) string {
	if r.Source == "EnformionGO" {
		return r.Source
	}
	return "LAS directory"
}
