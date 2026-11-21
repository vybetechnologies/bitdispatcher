package cad

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/biodream-llc/bitdispatcher/internal/ledger"
)

const EvLASRecords = "las.records.saved"

type LASRecord struct {
	Cautions     []RecordCaution `json:"cautions,omitempty"`
	Ownership    string          `json:"ownership,omitempty"`
	OwnerName    string          `json:"ownerName,omitempty"`
	FlagIDs      []string        `json:"flagIds,omitempty"`
	Flags        []*Flag         `json:"flags,omitempty"`
	Incidents    []PersonRef     `json:"incidents,omitempty"`
	Person       *Person         `json:"person,omitempty"`
	FromIncident string          `json:"fromIncident,omitempty"`
	ID           string          `json:"id"`
	Version      int             `json:"version"`
	Kind         string          `json:"kind"`
	Agency       string          `json:"agency,omitempty"`
	ExternalID   string          `json:"externalId,omitempty"`
	Name         string          `json:"name,omitempty"`
	Aliases      []string        `json:"aliases,omitempty"`
	DOB          string          `json:"dob,omitempty"`
	Sex          string          `json:"sex,omitempty"`
	Description  string          `json:"description,omitempty"`
	Plate        string          `json:"plate,omitempty"`
	State        string          `json:"state,omitempty"`
	VIN          string          `json:"vin,omitempty"`
	Year         string          `json:"year,omitempty"`
	Make         string          `json:"make,omitempty"`
	Model        string          `json:"model,omitempty"`
	Color        string          `json:"color,omitempty"`
	Body         string          `json:"body,omitempty"`
	Source       string          `json:"source"`
	Safety       bool            `json:"safetyConcern"`
	Restricted   bool            `json:"restricted"`
	Archived     bool            `json:"archived"`
	Updated      time.Time       `json:"updated"`
	By           string          `json:"by"`
}

func LASRecordKey(r LASRecord) string {
	prefix := strings.ToUpper(strings.TrimSpace(r.Agency)) + ":" + r.Kind + ":"
	if r.FromIncident != "" {
		return prefix + "incident:" + r.ID
	}
	if r.ExternalID != "" {
		return prefix + "external:" + strings.ToUpper(r.Source) + ":" + strings.ToUpper(r.ExternalID)
	}
	if r.Kind == "vehicle" {
		if r.VIN != "" {
			return prefix + "vin:" + strings.ToUpper(r.VIN)
		}
		return prefix + "plate:" + strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(r.Plate, " ", ""), "-", "")) + ":" + r.State
	}
	if r.DOB != "" {
		return prefix + "person:" + strings.ToUpper(strings.Join(strings.Fields(r.Name), " ")) + ":" + r.DOB
	}
	r.ID = ""
	r.Version = 0
	r.By = ""
	r.Updated = time.Time{}
	r.Archived = false
	r.FlagIDs = nil
	r.Flags = nil
	r.Incidents = nil
	if r.Person != nil {
		p := *r.Person
		p.ID = ""
		p.RecordID = ""
		p.Added = time.Time{}
		p.Updated = time.Time{}
		p.AddedBy = ""
		p.Returns = nil
		if p.SafetyNote != nil {
			n := *p.SafetyNote
			n.Flag = ""
			p.SafetyNote = &n
		}
		r.Person = &p
	}
	b, _ := json.Marshal(r)
	return prefix + string(b)
}

// SaveLASRecords validates the whole batch before writing one attributed ledger
// event. Existing records require their exact version; imports never overwrite.
func (e *Engine) SaveLASRecords(a Actor, records []LASRecord, importing bool) ([]LASRecord, int, error) {
	if a.Role != "admin" {
		return nil, 0, fmt.Errorf("an administrator manages LAS records")
	}
	return e.saveLASRecords(a, records, importing)
}

// ImportEnformionPeople is a create-only ingestion command, not an administrator
// privilege escalation. The authenticated search actor remains on the ledger.
func (e *Engine) ImportEnformionPeople(a Actor, records []LASRecord) ([]LASRecord, int, error) {
	for _, r := range records {
		if r.ID != "" || r.Kind != "person" || r.Source != "EnformionGO" || r.ExternalID == "" || r.Person == nil || r.Safety || r.Person.Safety || r.Person.SafetyNote != nil || len(r.FlagIDs) != 0 {
			return nil, 0, fmt.Errorf("invalid Enformion person import")
		}
	}
	return e.saveLASRecords(a, records, true)
}

func (e *Engine) saveLASRecords(a Actor, records []LASRecord, importing bool) ([]LASRecord, int, error) {
	if len(records) == 0 || len(records) > 500 {
		return nil, 0, fmt.Errorf("save between 1 and 500 records")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	keys := map[string]string{}
	for id, r := range e.st.LASRecords {
		keys[LASRecordKey(r)] = id
	}
	next := []LASRecord{}
	skipped := 0
	for _, r := range records {
		r.Incidents = nil
		r.Flags = nil
		if err := e.prepareLASPerson(&r); err != nil {
			return nil, 0, err
		}
		if r.ID != "" {
			old, ok := e.st.LASRecords[r.ID]
			if !ok || old.Version != r.Version {
				return nil, 0, fmt.Errorf("record changed; reload before saving")
			}
			if importing {
				return nil, 0, fmt.Errorf("imports create records only; edit existing records separately")
			}
			delete(keys, LASRecordKey(old))
			r.FromIncident = old.FromIncident
			r.FlagIDs = old.FlagIDs
			r.Version++
		} else {
			r.FromIncident = ""
			r.FlagIDs = nil
			r.Version = 1
		}
		key := LASRecordKey(r)
		if _, ok := keys[key]; ok {
			if importing {
				skipped++
				continue
			}
			return nil, 0, fmt.Errorf("a matching record already exists; edit that record")
		}
		if r.ID == "" {
			r.ID = fmt.Sprintf("LASR-%d-%d", e.st.Seq+1, len(next)+1)
		}
		r.Updated = e.now()
		r.By = a.User
		next = append(next, r)
		keys[key] = r.ID
	}
	if len(next) == 0 {
		return next, skipped, nil
	}
	for i := range next {
		if next[i].Person != nil {
			e.proposePersonFlag(a, "", next[i].Person)
			next[i].FlagIDs = e.st.lasFlagIDs(next[i], *next[i].Person)
		}
	}
	_, err := e.write(a, EvLASRecords, refs{}, next)
	return next, skipped, err
}

// NormalizeLASRecord uses the exact incident-person normalizer and safety rules.
func (e *Engine) NormalizeLASRecord(r *LASRecord) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.prepareLASPerson(r)
}
func (e *Engine) prepareLASPerson(r *LASRecord) error {
	if err := ValidateRecordCautions(r.Cautions); err != nil {
		return err
	}
	if r.Ownership != "" && r.Ownership != "agency" && r.Ownership != "employee" {
		return fmt.Errorf("invalid vehicle ownership")
	}
	if len(r.OwnerName) > 120 {
		return fmt.Errorf("owner name is too long")
	}
	if r.Kind == "person" && (r.Ownership != "" || r.OwnerName != "" || len(r.Cautions) > 0) {
		return fmt.Errorf("use person details for person cautions")
	}
	if r.Kind != "person" {
		if r.Person != nil {
			return fmt.Errorf("vehicle records cannot contain person fields")
		}
		return nil
	}
	if r.Person == nil {
		return nil
	} // LAS9 flat records remain valid.
	p := *r.Person
	if p.SafetyNote != nil {
		n := *p.SafetyNote
		p.SafetyNote = &n
		old := e.st.LASRecords[r.ID]
		if old.Person == nil || old.Person.SafetyNote == nil || old.Person.SafetyNote.Flag != n.Flag {
			p.SafetyNote.Flag = ""
		}
	}
	p.ID = ""
	p.RecordID = r.ID
	p.Returns = nil
	p.Removed = false
	if p.Role == "" {
		p.Role = "subject"
	}
	if err := e.normPerson(&p); err != nil {
		return err
	}
	r.Person = &p
	r.Name = p.Name
	r.Aliases = p.Aliases
	r.DOB = p.DOB
	r.Sex = p.Sex
	r.Safety = p.Safety
	return nil
}
func NormalizeLASPerson(p *Person) error { return (&Engine{}).normPerson(p) }
func LASRecordPerson(r LASRecord) Person {
	if r.Person != nil {
		p := *r.Person
		p.RecordID = r.ID
		p.Returns = nil
		return p
	}
	p := Person{RecordID: r.ID, Role: "subject", Name: r.Name, Aliases: r.Aliases, DOB: r.DOB, Sex: r.Sex, Safety: r.Safety, Notes: r.Description}
	_ = NormalizeLASPerson(&p)
	return p
}
func lasIncidentPersonID(number, id string) string { return "LASP-" + number + "-" + id }

// Existing incident people are a deterministic projection of the existing ledger,
// so no synthetic incident or name-based merge is needed to connect old records.
func (e *Engine) EnableLASPeople() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.st.lasPeople = true
	for _, in := range e.st.Incidents {
		for i := range in.Persons {
			p := &in.Persons[i]
			if p.Removed {
				continue
			}
			if p.RecordID == "" {
				p.RecordID = lasIncidentPersonID(in.Number, p.ID)
			}
			at := p.Updated
			if at.IsZero() {
				at = p.Added
			}
			if at.IsZero() {
				at = in.Created
			}
			e.st.syncLASPerson(in, *p, at, firstNonEmpty(p.AddedBy, in.CreatedBy))
		}
	}
}
func (s *State) LASPersonRefs(id string) []PersonRef {
	out := []PersonRef{}
	for _, in := range s.Incidents {
		for _, p := range in.Persons {
			rid := p.RecordID
			if rid == "" {
				rid = lasIncidentPersonID(in.Number, p.ID)
			}
			if !p.Removed && rid == id {
				out = append(out, PersonRef{Incident: in.Number, Person: p.ID, Role: p.Role, At: p.Added})
			}
		}
	}
	return out
}
func (s *State) syncLASPerson(in *Incident, p Person, at time.Time, by string) {
	if p.Removed {
		return
	}
	if s.LASRecords == nil {
		s.LASRecords = map[string]LASRecord{}
	}
	id := p.RecordID
	if id == "" {
		id = lasIncidentPersonID(in.Number, p.ID)
	}
	r, exists := s.LASRecords[id]
	if exists && !at.After(r.Updated) {
		return
	}
	if !exists {
		r = LASRecord{ID: id, Kind: "person", Agency: in.Agency, Source: "Incident " + in.Number, FromIncident: in.Number}
	}
	p.RecordID = id
	p.Returns = nil
	r.FlagIDs = s.lasFlagIDs(r, p)
	// A different call's roles and incident-only safety note must not replace
	// the shared record's established safety context. Core flags remain linked.
	if exists && r.Person != nil && r.FromIncident != in.Number {
		p.Safety = r.Person.Safety
		p.SafetyNote = r.Person.SafetyNote
		p.AgencyEmployee = r.Person.AgencyEmployee
		p.RegisteredSexOffender = r.Person.RegisteredSexOffender
		p.OffenderProfileURL = r.Person.OffenderProfileURL
		p.Photo = r.Person.Photo
		p.Cautions = r.Person.Cautions
		p.Warrants = r.Person.Warrants
	}
	r.Person = &p
	r.Name = p.Name
	r.DOB = p.DOB
	r.Sex = p.Sex
	r.Aliases = p.Aliases
	r.Safety = p.Safety
	r.Updated = at
	r.By = by
	r.Version++
	if p.Role == "patient" {
		r.Restricted = true
	}
	s.LASRecords[id] = r
}
func (s *State) lasFlagIDs(r LASRecord, p Person) []string {
	ids := append([]string(nil), r.FlagIDs...)
	for id, f := range s.Flags {
		if f.Scope.Person == PersonKey(p) {
			found := false
			for _, v := range ids {
				if v == id {
					found = true
				}
			}
			if !found {
				ids = append(ids, id)
			}
		}
	}
	return ids
}
func (s *State) LASPersonFlags(id string, p Person, now time.Time) []*Flag {
	out := s.FlagsForPerson(PersonKey(p), now)
	for _, fid := range s.LASRecords[id].FlagIDs {
		f := s.Flags[fid]
		if f == nil || !f.ActiveAt(now) {
			continue
		}
		found := false
		for _, v := range out {
			if v.ID == fid {
				found = true
			}
		}
		if !found {
			out = append(out, f)
		}
	}
	sortFlags(out)
	return out
}
func (s *State) applyLASRecords(ev ledger.Event) error {
	var records []LASRecord
	if err := json.Unmarshal(ev.Data, &records); err != nil {
		return err
	}
	if s.LASRecords == nil {
		s.LASRecords = map[string]LASRecord{}
	}
	for _, r := range records {
		s.LASRecords[r.ID] = r
	}
	return nil
}
