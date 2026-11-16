package api

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/biodream-llc/bitdispatcher/internal/auth"
	"github.com/biodream-llc/bitdispatcher/internal/cad"
	"github.com/biodream-llc/bitdispatcher/internal/ledger"
)

func TestMDTLookupPermissionsAndLocalRecords(t *testing.T) {
	ts, s, pw := newServer(t)
	s.LAS = true
	s.mdtLookupRoutes()
	if err := s.Eng.SaveUnit(cad.SystemActor, cad.UnitDef{ID: "SEC1", Type: "patrol", Agency: "LAS"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Auth.SetMDTUnit("admin", "SEC1"); err != nil {
		t.Fatal(err)
	}
	officerPassword := auth.GeneratePassword()
	if err := s.Auth.Save("officer", "Officer", auth.Field, officerPassword, false); err != nil {
		t.Fatal(err)
	}
	// A unique assignment belongs to one account: move the fixture unit from admin to the field user.
	s.Auth.SetMDTUnit("admin", "")
	if err := s.Auth.SetMDTUnit("officer", "SEC1"); err != nil {
		t.Fatal(err)
	}
	number, err := s.Eng.CreateIncident(cad.SystemActor, cad.NewIncident{Type: "OTHER"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Eng.SavePerson(cad.SystemActor, number, cad.Person{Name: "UNIQUE PERSON", First: "UNIQUE", Last: "PERSON", Role: "subject", DOB: "1982-02-03", Aliases: []string{"OTHERALIAS"}, SSN: "123456789", DL: "PRIVATE-DL", Address: "PRIVATE HOME", Notes: "PRIVATE NOTES", Safety: true, SafetyNote: &cad.SafetyConcern{Title: "Documented concern", Reason: "Synthetic test record", Source: "Test fixture", Instruction: "Contact dispatch"}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Eng.SaveVehicle(cad.SystemActor, number, cad.Vehicle{Plate: "ZYX987", State: "ND", VIN: "1HGCM82633A004352", Role: "involved", Make: "HOND", Model: "ACCORD", Owner: "PRIVATE OWNER", Notes: "PRIVATE VEHICLE NOTES"})
	if err != nil {
		t.Fatal(err)
	}
	field := login(t, ts, "officer", officerPassword)
	q := map[string]any{"kind": "person", "name": "PERSON UNIQUE", "dob": "1982-02-03", "purpose": "patrol"}
	if code, _ := field.do("POST", "/api/mdt/records/search", q, false); code != 403 {
		t.Fatal("CSRF bypass", code)
	}
	code, body := field.do("POST", "/api/mdt/records/search", q, true)
	if code != 200 {
		t.Fatal(code, string(body))
	}
	var res struct{ Items []mdtLookupHit }
	json.Unmarshal(body, &res)
	if len(res.Items) != 1 || !res.Items[0].Safety || !strings.Contains(res.Items[0].Name, "UNIQUE") {
		t.Fatal("person search", string(body))
	}
	for _, secret := range []string{"returns"} {
		if strings.Contains(string(body), secret) {
			t.Fatal("private detail leaked", secret)
		}
	}
	if code, _ := field.do("POST", "/api/mdt/records/search?unit=OTHER", q, true); code != 403 {
		t.Fatal("other unit bypass", code)
	}
	code, body = field.do("POST", "/api/mdt/records/search", map[string]any{"kind": "vehicle", "plate": "zyx-98", "state": "nd", "vin": "1HGC", "purpose": "assigned-call"}, true)
	if code != 200 {
		t.Fatal(code, string(body))
	}
	json.Unmarshal(body, &res)
	if len(res.Items) != 1 || res.Items[0].Plate != "ZYX987" {
		t.Fatal("vehicle match", string(body))
	}
	if strings.Contains(string(body), "PRIVATE OWNER") || strings.Contains(string(body), "PRIVATE VEHICLE NOTES") {
		t.Fatal("vehicle private data leaked")
	}
	viewer := login(t, ts, "v1", pw["v1"])
	if code, _ := viewer.do("POST", "/api/mdt/records/search", q, true); code != 403 {
		t.Fatal("viewer can search", code)
	}
	admin := login(t, ts, "admin", pw["admin"])
	if code, _ := admin.do("POST", "/api/mdt/records/search", q, true); code != 403 {
		t.Fatal("unassigned administrator can search", code)
	}
	for _, bad := range []map[string]any{{"kind": "person", "name": "", "purpose": "patrol"}, {"kind": "person", "name": "UNIQUE", "dob": "bad", "purpose": "patrol"}, {"kind": "vehicle", "plate": "Z", "purpose": "patrol"}, {"kind": "person", "name": "UNIQUE", "purpose": "invalid"}} {
		if code, _ := field.do("POST", "/api/mdt/records/search", bad, true); code != 422 {
			t.Fatal("invalid search accepted", code)
		}
	}
	data, _ := os.ReadFile(s.Eng.LedgerPath())
	events := 0
	for _, line := range strings.Split(string(data), "\n") {
		var event ledger.Event
		if json.Unmarshal([]byte(line), &event) == nil && event.Type == cad.EvSearched {
			events++
			if strings.Contains(string(event.Data), "UNIQUE") || strings.Contains(string(event.Data), "ZYX") || !strings.Contains(string(event.Data), "queryHash") {
				t.Fatal("audit exposed raw terms or omitted hash")
			}
		}
	}
	if events != 2 {
		t.Fatal("searches not audited", events)
	}
}

func TestMDTLookupExcludesRestrictedAndRemovedRecords(t *testing.T) {
	ts, s, pw := newServer(t)
	s.LAS = true
	s.mdtLookupRoutes()
	s.Eng.SaveUnit(cad.SystemActor, cad.UnitDef{ID: "SEC1", Type: "patrol", Agency: "LAS"})
	s.Auth.SetMDTUnit("admin", "SEC1")
	for _, kind := range []string{"normal", "foreign", "juvenile", "medical", "patient", "minor", "removed"} {
		typ := "OTHER"
		if kind == "medical" {
			typ = "CHEST"
		}
		number, err := s.Eng.CreateIncident(cad.SystemActor, cad.NewIncident{Type: typ})
		if err != nil {
			t.Fatal(err)
		}
		role := "subject"
		if kind == "patient" {
			role = "patient"
		}
		dob := "1982-01-01"
		if kind == "minor" {
			dob = "2020-01-01"
		}
		id, err := s.Eng.SavePerson(cad.SystemActor, number, cad.Person{Name: "RESTRICTED TEST", Role: role, DOB: dob})
		if err != nil {
			t.Fatal(err)
		}
		s.Eng.Read(func(st *cad.State) {
			in := st.Incidents[number]
			switch kind {
			case "foreign":
				in.Agency = "FOREIGN"
			case "juvenile":
				in.Modifiers = []string{"JUVENILE"}
			case "removed":
				for i := range in.Persons {
					if in.Persons[i].ID == id {
						in.Persons[i].Removed = true
					}
				}
			}
		})
	}
	admin := login(t, ts, "admin", pw["admin"])
	code, body := admin.do("POST", "/api/mdt/records/search", map[string]any{"kind": "person", "name": "RESTRICTED", "purpose": "patrol"}, true)
	if code != 200 {
		t.Fatal(code, string(body))
	}
	var res struct{ Items []mdtLookupHit }
	json.Unmarshal(body, &res)
	if len(res.Items) != 1 {
		t.Fatal("restricted records not excluded", string(body))
	}
}
