package api

import (
	"encoding/json"
	"github.com/biodream-llc/bitdispatcher/internal/cad"
	"testing"
)

func TestPersonProfilesTraverseCyclesAndRespectVisibility(t *testing.T) {
	ts, s, pw := newServer(t)
	s.LAS = true
	s.mdtLookupRoutes()
	s.desktopRoutes()
	s.Eng.SaveUnit(cad.SystemActor, cad.UnitDef{ID: "SEC1", Type: "patrol", Agency: "LAS"})
	s.Auth.SetMDTUnit("admin", "SEC1")
	a := cad.Actor{User: "admin", Role: "admin"}
	rows, _, err := s.Eng.SaveLASRecords(a, []cad.LASRecord{
		{Kind: "person", Source: "Fixture", Person: &cad.Person{Name: "Alpha Person", Role: "subject", DOB: "1980-01-01", Employer: &cad.EmployerInfo{Name: "Fixture Company", Occupation: "Guard", Phone: "5550100"}}},
		{Kind: "person", Source: "Fixture", Person: &cad.Person{Name: "Beta Person", Role: "subject", DOB: "1980-01-01", Cautions: []cad.RecordCaution{{Type: "Weapons", Level: "danger"}}}},
		{Kind: "person", Source: "Fixture", Person: &cad.Person{Name: "Gamma Person", Role: "subject", DOB: "1980-01-01"}},
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	for i, typ := range []string{"parent", "friend", "associate"} {
		rows[i].Relationships = []cad.PersonRelationship{{PersonID: rows[(i+1)%3].ID, Type: typ, Notes: "Fixture relationship"}}
		saved, _, err := s.Eng.SaveLASRecords(a, []cad.LASRecord{rows[i]}, false)
		if err != nil {
			t.Fatal(err)
		}
		rows[i] = saved[0]
	}
	admin := login(t, ts, "admin", pw["admin"])
	profile := func(id string) mdtLookupHit {
		t.Helper()
		code, body := admin.do("POST", "/api/mdt/records/person", map[string]string{"record": id, "purpose": "patrol"}, true)
		var out mdtLookupHit
		json.Unmarshal(body, &out)
		if code != 200 {
			t.Fatal(code, string(body))
		}
		return out
	}
	for _, r := range rows {
		p := profile(r.ID)
		if len(p.Associates) != 2 {
			t.Fatal("cycle did not return bounded direct links", p.Associates)
		}
	}
	alpha := profile(rows[0].ID)
	if alpha.Person.Employer.Name != "Fixture Company" || len(alpha.Person.Cautions) != 0 {
		t.Fatal("employer missing or associate caution inherited")
	}
	beta := profile(rows[1].ID)
	found := false
	for _, link := range beta.Associates {
		if link.Record == rows[0].ID && link.Type == "child" && link.Incoming {
			found = true
		}
	}
	if !found {
		t.Fatal("reverse parent/child relationship missing")
	}
	code, body := admin.do("GET", "/api/mdt/offline/records", nil, true)
	var snapshot struct{ Records []mdtLookupHit }
	json.Unmarshal(body, &snapshot)
	if code != 200 || len(snapshot.Records) != 3 {
		t.Fatal(code, string(body))
	}
	for _, r := range snapshot.Records {
		if len(r.Associates) != 2 {
			t.Fatal("offline graph missing")
		}
	}
	// An excluded node cannot be disclosed by incoming links or direct ID opens.
	hidden := rows[1]
	hidden.Restricted = true
	saved, _, err := s.Eng.SaveLASRecords(a, []cad.LASRecord{hidden}, false)
	if err != nil {
		t.Fatal(err)
	}
	rows[1] = saved[0]
	if len(profile(rows[0].ID).Associates) != 1 {
		t.Fatal("restricted associate disclosed")
	}
	code, _ = admin.do("POST", "/api/mdt/records/person", map[string]string{"record": hidden.ID, "purpose": "patrol"}, true)
	if code != 404 {
		t.Fatal("direct restricted profile allowed", code)
	}
	code, _ = admin.do("POST", "/api/mdt/records/person", map[string]string{"record": rows[0].ID, "purpose": "invalid"}, true)
	if code != 422 {
		t.Fatal("purpose not enforced", code)
	}
}
