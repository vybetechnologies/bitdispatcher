package api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/biodream-llc/bitdispatcher/internal/cad"
)

func TestLASRichPersonRecordConnectsIncidentAndMDTWithCoreFlags(t *testing.T) {
	ts, s, pw := newServer(t)
	s.LAS = true
	s.Eng.EnableLASPeople()
	s.lasRecordRoutes()
	s.mdtLookupRoutes()
	s.Eng.SaveUnit(cad.SystemActor, cad.UnitDef{ID: "SEC1", Type: "patrol", Agency: "LAS"})
	s.Auth.SetMDTUnit("admin", "SEC1")
	admin := login(t, ts, "admin", pw["admin"])
	p := cad.Person{First: "Shared", Middle: "Test", Last: "Record", Role: "subject", DOB: "1980-02-03", SSN: "123456789", DL: "FULL-LICENCE", DLState: "ND", IDType: "passport", IDNumber: "FULL-OTHER-ID", IDState: "US", Address: "Test address", Phones: []cad.PersonPhone{{Number: "7015550100", Type: "mobile", Source: "records", Verified: true}}, Race: "W", Ethnicity: "N", Height: "510", Weight: "180", Build: "medium", Hair: "BRO", Eyes: "BLU", Clothing: "Test coat", SMT: "Test tattoo", ApproxAge: "46", Notes: "Full fixture notes", Safety: true, SafetyNote: &cad.SafetyConcern{Title: "Core safety fixture", Reason: "Synthetic test concern", Source: "Test fixture", Instruction: "Contact dispatch", Scope: "permanent"}}
	code, body := admin.do("POST", "/api/admin/las-records", cad.LASRecord{Kind: "person", Source: "Fixture", Person: &p}, true)
	if code != 201 {
		t.Fatal(code, string(body))
	}
	var saved cad.LASRecord
	json.Unmarshal(body, &saved)
	if saved.Person == nil || saved.Person.SSN != "123-45-6789" || saved.Person.DL != "FULL-LICENCE" || saved.Person.SafetyNote.Flag == "" {
		t.Fatal("full fields or core flag missing", string(body))
	}
	var numbers []string
	for _, role := range []string{"caller", "witness"} {
		n, err := s.Eng.CreateIncident(cad.SystemActor, cad.NewIncident{Type: "OTHER"})
		if err != nil {
			t.Fatal(err)
		}
		numbers = append(numbers, n)
		person := *saved.Person
		person.RecordID = saved.ID
		person.Role = role
		person.Roles = []string{role}
		person.Safety = false
		person.SafetyNote = nil
		code, body = admin.do("POST", "/api/incidents/"+n+"/persons", person, true)
		if code != 200 {
			t.Fatal(code, string(body))
		}
	}
	code, body = admin.do("GET", "/api/admin/las-records", nil, true)
	var list struct{ Items []cad.LASRecord }
	json.Unmarshal(body, &list)
	if code != 200 || len(list.Items) != 1 || len(list.Items[0].Incidents) != 2 || len(list.Items[0].Flags) != 1 {
		t.Fatal("record/link/flag disconnected", string(body))
	}
	if !strings.Contains(string(body), "123-45-6789") || strings.Contains(string(body), "***") {
		t.Fatal("unexpected redaction", string(body))
	}
	code, body = admin.do("POST", "/api/persons/records/search", map[string]string{"query": "Shared Record"}, true)
	if code != 200 || !strings.Contains(string(body), saved.ID) || !strings.Contains(string(body), "FULL-LICENCE") {
		t.Fatal("call taker selector", code, string(body))
	}
	code, body = admin.do("POST", "/api/mdt/records/search", map[string]string{"kind": "person", "name": "Shared Record", "purpose": "patrol"}, true)
	var result struct{ Items []mdtLookupHit }
	json.Unmarshal(body, &result)
	if code != 200 || len(result.Items) != 1 || result.Items[0].Person == nil || result.Items[0].Person.SSN != "123-45-6789" || len(result.Items[0].Flags) != 1 || len(result.Items[0].Incidents) != 2 {
		t.Fatal("MDT full profile/link/flag", code, string(body))
	}
	// Correcting identity details keeps already approved core flags associated.
	record := list.Items[0]
	record.Person.Last = "Corrected"
	record.Person.Name = ""
	code, body = admin.do("PUT", "/api/admin/las-records/"+record.ID, record, true)
	if code != 200 {
		t.Fatal(code, string(body))
	}
	s.Eng.Read(func(st *cad.State) {
		in := st.Incidents[numbers[0]]
		if len(st.IncidentFlags(in, s.Eng.Now())) != 1 {
			t.Fatal("core incident warning lost")
		}
		out := s.incidentView(in)
		if out.Persons[0].SSN != "123-45-6789" {
			t.Fatal("incident fields redacted")
		}
	})
}
func TestLASRichPersonCSVAndSafetyValidation(t *testing.T) {
	rows, err := parseLASImport("csv", "kind,first,last,dob,ssn,dl,dlState,address,phone,notes,source\nperson,CSV,Fixture,1980-01-01,123456789,FULL-DL,ND,Test address,7015550100,Test notes,Fixture\n")
	if err != nil || len(rows) != 1 || rows[0].Person == nil || rows[0].Person.SSN != "123-45-6789" {
		t.Fatal("expanded CSV", rows, err)
	}
	_, err = parseLASImport("json", `[{"kind":"person","source":"Fixture","person":{"first":"Test","last":"Person","safety":true}}]`)
	if err == nil {
		t.Fatal("bypassed core safety concern requirements")
	}
}
