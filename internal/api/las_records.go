package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/biodream-llc/bitdispatcher/internal/auth"
	"github.com/biodream-llc/bitdispatcher/internal/cad"
)

func validateLASRecord(r *cad.LASRecord, importing bool) error {
	r.Kind = strings.ToLower(strings.TrimSpace(r.Kind))
	if err := cad.ValidateRecordCautions(r.Cautions); err != nil {
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
	if r.Person != nil {
		if r.Kind != "person" {
			return fmt.Errorf("only person records have person details")
		}
		if err := cad.NormalizeLASPerson(r.Person); err != nil {
			return err
		}
		r.Name, r.Aliases, r.DOB, r.Sex, r.Safety = r.Person.Name, r.Person.Aliases, r.Person.DOB, r.Person.Sex, r.Person.Safety
	}
	r.Kind = strings.ToLower(strings.TrimSpace(r.Kind))
	if r.Kind != "person" && r.Kind != "vehicle" {
		return fmt.Errorf("kind must be person or vehicle")
	}
	for _, p := range []*string{&r.ID, &r.Agency, &r.ExternalID, &r.Name, &r.DOB, &r.Sex, &r.Description, &r.Plate, &r.State, &r.VIN, &r.Year, &r.Make, &r.Model, &r.Color, &r.Body, &r.Source} {
		*p = strings.TrimSpace(*p)
		if len(*p) > 600 || strings.ContainsFunc(*p, func(c rune) bool { return unicode.IsControl(c) && c != '\n' && c != '\t' }) {
			return fmt.Errorf("fields must be at most 600 characters without control characters")
		}
	}
	if len(r.Name) > 100 || len(r.Plate) > 20 || len(r.Source) > 120 || len(r.ExternalID) > 100 || len(r.Agency) > 32 || len(r.Aliases) > 20 {
		return fmt.Errorf("name, plate, source, agency, external ID or aliases are too long")
	}
	for i, v := range r.Aliases {
		v = strings.TrimSpace(v)
		if len(v) > 100 || strings.ContainsFunc(v, unicode.IsControl) {
			return fmt.Errorf("aliases must be at most 100 characters")
		}
		r.Aliases[i] = v
	}
	r.State = strings.ToUpper(r.State)
	r.VIN = strings.ToUpper(r.VIN)
	r.Plate = strings.ToUpper(r.Plate)
	if r.Source == "" {
		return fmt.Errorf("source is required")
	}
	if importing && (r.ID != "" || r.Version != 0 || r.Archived) {
		return fmt.Errorf("imports create active records; do not include id, version or archived")
	}
	if r.Kind == "person" {
		if len([]rune(lookupName(r.Name))) < 2 && (r.Person == nil || (r.Person.Name == "" && r.Person.Clothing == "" && r.Person.Notes == "")) {
			return fmt.Errorf("person name needs at least two characters")
		}
		if r.DOB != "" {
			d, e := time.Parse("2006-01-02", r.DOB)
			if e != nil || d.After(time.Now()) || d.Year() < 1900 {
				return fmt.Errorf("birth date must be a valid YYYY-MM-DD date")
			}
		}
		if r.Plate != "" || r.VIN != "" || r.State != "" || r.Year != "" || r.Make != "" || r.Model != "" || r.Color != "" || r.Body != "" {
			return fmt.Errorf("person records cannot contain vehicle fields")
		}
	} else {
		if len(lookupID(r.Plate)) < 2 && r.VIN == "" {
			return fmt.Errorf("vehicle requires a plate or full VIN")
		}
		if r.VIN != "" && (len(r.VIN) != 17 || lookupID(r.VIN) != r.VIN || strings.ContainsAny(r.VIN, "IOQ")) {
			return fmt.Errorf("VIN must be 17 valid characters")
		}
		if r.State != "" && (len(r.State) != 2 || r.State[0] < 'A' || r.State[0] > 'Z' || r.State[1] < 'A' || r.State[1] > 'Z') {
			return fmt.Errorf("plate state uses two letters")
		}
		if r.Name != "" || r.DOB != "" || r.Sex != "" || len(r.Aliases) > 0 {
			return fmt.Errorf("vehicle records cannot contain person fields")
		}
	}
	if r.Year != "" {
		y, e := strconv.Atoi(r.Year)
		if e != nil || len(r.Year) != 4 || y < 1900 || y > time.Now().Year()+2 {
			return fmt.Errorf("vehicle year is invalid")
		}
	}
	return nil
}
func parseLASImport(format, content string) ([]cad.LASRecord, error) {
	content = strings.TrimPrefix(content, "\ufeff")
	records := []cad.LASRecord{}
	switch format {
	case "json":
		dec := json.NewDecoder(strings.NewReader(content))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&records); err != nil {
			return nil, fmt.Errorf("JSON must be an array of record objects using the template fields: %v", err)
		}
		var extra any
		if dec.Decode(&extra) != io.EOF {
			return nil, fmt.Errorf("JSON contains trailing content")
		}
	case "csv":
		rd := csv.NewReader(strings.NewReader(content))
		header, err := rd.Read()
		if err != nil {
			return nil, fmt.Errorf("CSV header is missing")
		}
		allowed := map[string]bool{}
		personFields := "first,middle,last,suffix,locKind,building,floor,unit,phone,phones,role,roles,address,race,ethnicity,height,weight,hair,eyes,clothing,smt,build,ssn,dl,dlState,approxAge,idType,idNumber,idState,notes,agencyEmployee,safety,safetyNote,photo,cautions,warrants,registeredSexOffender,offenderProfileUrl"
		for _, h := range strings.Split("kind,agency,externalId,name,aliases,dob,sex,description,plate,state,vin,year,make,model,color,body,source,safetyConcern,restricted,ownership,ownerName,"+personFields, ",") {
			allowed[h] = true
		}
		seen := map[string]bool{}
		for i, h := range header {
			h = strings.TrimSpace(h)
			if !allowed[h] || seen[h] {
				return nil, fmt.Errorf("unknown or duplicate CSV column %q", h)
			}
			header[i] = h
			seen[h] = true
		}
		if !seen["kind"] || !seen["source"] {
			return nil, fmt.Errorf("CSV requires kind and source columns")
		}
		for row := 2; ; row++ {
			values, err := rd.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("CSV row %d: %v", row, err)
			}
			obj := map[string]any{}
			for i, h := range header {
				v := strings.TrimSpace(values[i])
				if v == "" {
					continue
				}
				if h == "phones" || h == "safetyNote" || h == "cautions" || h == "warrants" {
					var nested any
					if err := json.Unmarshal([]byte(v), &nested); err != nil {
						return nil, fmt.Errorf("CSV row %d: %s must be JSON", row, h)
					}
					obj[h] = nested
				} else if h == "aliases" || h == "roles" {
					obj[h] = strings.Split(v, "|")
				} else if h == "safetyConcern" || h == "restricted" || h == "safety" || h == "agencyEmployee" || h == "registeredSexOffender" {
					b, e := strconv.ParseBool(v)
					if e != nil {
						return nil, fmt.Errorf("CSV row %d: %s must be true or false", row, h)
					}
					obj[h] = b
				} else {
					obj[h] = v
				}
			}
			b, _ := json.Marshal(obj)
			var r cad.LASRecord
			if err := json.Unmarshal(b, &r); err != nil {
				return nil, err
			}
			if strings.EqualFold(r.Kind, "person") {
				full := false
				for _, k := range strings.Split(personFields, ",") {
					if obj[k] != nil {
						full = true
					}
				}
				if full {
					var p cad.Person
					if err := json.Unmarshal(b, &p); err != nil {
						return nil, fmt.Errorf("CSV row %d: %v", row, err)
					}
					p.Safety = p.Safety || r.Safety
					r.Person = &p
					r.Cautions = nil
				}
			}
			records = append(records, r)
			if len(records) > 500 {
				return nil, fmt.Errorf("import at most 500 records per file")
			}
		}
	default:
		return nil, fmt.Errorf("format must be csv or json")
	}
	if len(records) == 0 || len(records) > 500 {
		return nil, fmt.Errorf("import between 1 and 500 records")
	}
	for i := range records {
		if err := validateLASRecord(&records[i], true); err != nil {
			return nil, fmt.Errorf("record %d: %v", i+1, err)
		}
	}
	return records, nil
}
func (s *Server) lasRecordRoutes() {
	s.handle("POST /api/persons/records/search", auth.Dispatcher, func(w http.ResponseWriter, r *http.Request, se *auth.Session, a cad.Actor) {
		var b struct {
			Query string `json:"query"`
		}
		if !readJSON(w, r, &b) {
			return
		}
		q := lookupName(b.Query)
		if len(q) < 2 || len(q) > 150 {
			fail(w, 422, "enter at least two characters of a name or record ID")
			return
		}
		items := []cad.LASRecord{}
		s.Eng.Read(func(st *cad.State) {
			sc := st.AccessScope(se.Username, se.Console, se.Duty)
			for _, v := range st.LASRecords {
				if v.Kind != "person" || v.Archived || (se.Role != auth.Admin && sc != nil && v.Agency != "" && !sc[v.Agency]) {
					continue
				}
				p := cad.LASRecordPerson(v)
				if !strings.Contains(lookupName(v.ID), q) && !lookupNameMatches(q, p) {
					continue
				}
				v.Person = &p
				for _, f := range st.LASPersonFlags(v.ID, p, s.Eng.Now()) {
					v.Flags = append(v.Flags, st.FlagFor(f, cad.DutyOf(a), a.Role))
				}
				for _, ref := range st.LASPersonRefs(v.ID) {
					if in := st.Incidents[ref.Incident]; in != nil && (se.Role == auth.Admin || st.IncidentIn(sc, in)) {
						v.Incidents = append(v.Incidents, ref)
					}
				}
				items = append(items, v)
			}
		})
		providerMessage := ""
		if len(items) == 0 && !strings.HasPrefix(q, "LASR") && !strings.HasPrefix(q, "LASP") {
			agency := ""
			s.Eng.Read(func(st *cad.State) {
				sc := st.AccessScope(se.Username, se.Console, se.Duty)
				if se.Role != auth.Admin && sc != nil {
					for id := range sc {
						agency = id
						break
					}
				}
			})
			records, message := s.enformionFallback(r.Context(), a, b.Query, "", agency)
			items = append(items, records...)
			providerMessage = message
		}
		sort.Slice(items, func(i, j int) bool { return items[i].Updated.After(items[j].Updated) })
		if len(items) > 25 {
			items = items[:25]
		}
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, 200, map[string]any{"items": items, "providerMessage": providerMessage})
	})
	s.handle("GET /api/admin/las-records", auth.Admin, func(w http.ResponseWriter, r *http.Request, se *auth.Session, a cad.Actor) {
		items := []cad.LASRecord{}
		q := lookupName(r.URL.Query().Get("q"))
		kind := r.URL.Query().Get("kind")
		s.Eng.Read(func(st *cad.State) {
			for _, v := range st.LASRecords {
				if v.Kind == "person" {
					v.Incidents = st.LASPersonRefs(v.ID)
					p := cad.LASRecordPerson(v)
					v.Person = &p
					for _, f := range st.LASPersonFlags(v.ID, p, s.Eng.Now()) {
						v.Flags = append(v.Flags, st.FlagFor(f, cad.DutyOf(a), a.Role))
					}
				}
				if kind != "" && v.Kind != kind {
					continue
				}
				text := lookupName(strings.Join([]string{v.ID, v.Name, strings.Join(v.Aliases, " "), v.Plate, v.VIN, v.ExternalID}, " "))
				if q != "" && !strings.Contains(text, q) {
					continue
				}
				items = append(items, v)
			}
		})
		sort.Slice(items, func(i, j int) bool {
			if items[i].Updated.Equal(items[j].Updated) {
				return items[i].ID > items[j].ID
			}
			return items[i].Updated.After(items[j].Updated)
		})
		total := len(items)
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		if offset < 0 {
			offset = 0
		}
		if offset > total {
			offset = total
		}
		end := min(total, offset+100)
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, 200, map[string]any{"items": items[offset:end], "total": total, "offset": offset})
	})
	s.handle("POST /api/admin/las-records", auth.Admin, func(w http.ResponseWriter, r *http.Request, se *auth.Session, a cad.Actor) {
		var v cad.LASRecord
		if !readJSON(w, r, &v) {
			return
		}
		if v.ID != "" || v.Version != 0 {
			fail(w, 422, "new records cannot supply id or version")
			return
		}
		if err := validateLASRecord(&v, false); err != nil {
			fail(w, 422, err.Error())
			return
		}
		items, _, err := s.Eng.SaveLASRecords(a, []cad.LASRecord{v}, false)
		if err != nil {
			fail(w, 409, err.Error())
			return
		}
		writeJSON(w, 201, items[0])
	})
	s.handle("PUT /api/admin/las-records/{id}", auth.Admin, func(w http.ResponseWriter, r *http.Request, se *auth.Session, a cad.Actor) {
		var v cad.LASRecord
		if !readJSON(w, r, &v) {
			return
		}
		if v.ID != "" && v.ID != r.PathValue("id") {
			fail(w, 422, "record ID does not match")
			return
		}
		v.ID = r.PathValue("id")
		if err := validateLASRecord(&v, false); err != nil {
			fail(w, 422, err.Error())
			return
		}
		items, _, err := s.Eng.SaveLASRecords(a, []cad.LASRecord{v}, false)
		if err != nil {
			fail(w, 409, err.Error())
			return
		}
		writeJSON(w, 200, items[0])
	})
	s.handle("POST /api/admin/las-records/import", auth.Admin, func(w http.ResponseWriter, r *http.Request, se *auth.Session, a cad.Actor) {
		var b struct {
			Format  string `json:"format"`
			Content string `json:"content"`
			Commit  bool   `json:"commit"`
		}
		if !readJSON(w, r, &b) {
			return
		}
		if len(b.Content) > 750000 {
			fail(w, 422, "import file must be smaller than 750 KB")
			return
		}
		records, err := parseLASImport(b.Format, b.Content)
		if err != nil {
			fail(w, 422, err.Error())
			return
		}
		if b.Commit {
			saved, skipped, err := s.Eng.SaveLASRecords(a, records, true)
			if err != nil {
				fail(w, 409, err.Error())
				return
			}
			writeJSON(w, 200, map[string]any{"created": len(saved), "skipped": skipped})
			return
		}
		keys := map[string]bool{}
		s.Eng.Read(func(st *cad.State) {
			for _, v := range st.LASRecords {
				keys[cad.LASRecordKey(v)] = true
			}
		})
		create, skipped := 0, 0
		preview := []cad.LASRecord{}
		for _, v := range records {
			k := cad.LASRecordKey(v)
			if keys[k] {
				skipped++
				continue
			}
			keys[k] = true
			create++
			if len(preview) < 10 {
				preview = append(preview, v)
			}
		}
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, 200, map[string]any{"create": create, "skipped": skipped, "total": len(records), "preview": preview})
	})
}
