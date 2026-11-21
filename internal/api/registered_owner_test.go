package api

import (
	"encoding/json"
	"github.com/biodream-llc/bitdispatcher/internal/cad"
	"testing"
)

func TestRegisteredOwnerMDTAndOfflineRespectPersonVisibility(t *testing.T) {
	ts, s, pw := newServer(t)
	s.LAS = true
	s.mdtLookupRoutes()
	s.desktopRoutes()
	s.Eng.SaveUnit(cad.SystemActor, cad.UnitDef{ID: "SEC1", Type: "patrol", Agency: "LAS"})
	s.Auth.SetMDTUnit("admin", "SEC1")
	a := cad.Actor{User: "admin", Role: "admin"}
	people, _, err := s.Eng.SaveLASRecords(a, []cad.LASRecord{{Kind: "person", Source: "Fixture", Person: &cad.Person{Name: "Test Owner", Role: "subject", DOB: "1980-01-01", RegisteredSexOffender: true}}}, false)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = s.Eng.SaveLASRecords(a, []cad.LASRecord{{Kind: "vehicle", Source: "Fixture", Plate: "TEST-OWNER", RegisteredOwnerID: people[0].ID}}, false)
	if err != nil {
		t.Fatal(err)
	}
	admin := login(t, ts, "admin", pw["admin"])
	lookup := func() mdtLookupHit {
		t.Helper()
		code, body := admin.do("POST", "/api/mdt/records/search", map[string]string{"kind": "vehicle", "plate": "TEST-OWNER", "purpose": "patrol"}, true)
		var result struct{ Items []mdtLookupHit }
		json.Unmarshal(body, &result)
		if code != 200 || len(result.Items) != 1 {
			t.Fatal(code, string(body))
		}
		return result.Items[0]
	}
	hit := lookup()
	if hit.RegisteredOwner == nil || hit.RegisteredOwner.Record != people[0].ID || !hit.RegisteredOwner.Person.RegisteredSexOffender || hit.Person != nil {
		t.Fatal("linked owner missing or misclassified as vehicle subject")
	}
	code, body := admin.do("GET", "/api/mdt/offline/records", nil, true)
	var snapshot struct{ Records []mdtLookupHit }
	json.Unmarshal(body, &snapshot)
	if code != 200 {
		t.Fatal(code, string(body))
	}
	found := false
	for _, record := range snapshot.Records {
		if record.Plate != "" {
			found = record.RegisteredOwner != nil
		}
	}
	if !found {
		t.Fatal("offline owner link missing")
	}
	// The ID resolves fresh information, never a stale stored copy of the person.
	owner := people[0]
	owner.Person.Name = ""
	owner.Person.First = "Updated"
	owner.Person.Last = "Owner"
	people, _, err = s.Eng.SaveLASRecords(a, []cad.LASRecord{owner}, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := lookup().RegisteredOwner; got == nil || got.Name != people[0].Name {
		t.Fatal("owner correction did not resolve")
	}
	for _, mode := range []string{"restricted", "archived", "minor", "agency"} {
		owner = people[0]
		owner.Person.DOB = "1980-01-01"
		owner.Restricted = false
		owner.Archived = false
		owner.Agency = ""
		switch mode {
		case "restricted":
			owner.Restricted = true
		case "archived":
			owner.Archived = true
		case "minor":
			owner.Person.DOB = "2020-01-01"
		case "agency":
			owner.Agency = "OTHER"
		}
		people, _, err = s.Eng.SaveLASRecords(a, []cad.LASRecord{owner}, false)
		if err != nil {
			t.Fatal(err)
		}
		if lookup().RegisteredOwner != nil {
			t.Fatalf("exposed %s owner", mode)
		}
	}
}
