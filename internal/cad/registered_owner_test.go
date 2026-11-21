package cad

import "testing"

func TestRegisteredOwnerOptionalStableLinkAndValidation(t *testing.T) {
	e, _, _ := testEngine(t)
	a := Actor{User: "fixture-admin", Role: "admin"}
	people, _, err := e.SaveLASRecords(a, []LASRecord{{Kind: "person", Name: "Test Owner", Source: "Fixture"}, {Kind: "person", Name: "Scoped Owner", Agency: "OTHER", Source: "Fixture"}}, false)
	if err != nil {
		t.Fatal(err)
	}
	vehicles, _, err := e.SaveLASRecords(a, []LASRecord{{Kind: "vehicle", Plate: "TEST1", Source: "Fixture", Ownership: "employee", Cautions: []RecordCaution{{Type: "Weapons", Level: "danger"}}}}, false)
	if err != nil {
		t.Fatal("owner should be optional", err)
	}
	v := vehicles[0]
	v.RegisteredOwnerID = people[0].ID
	saved, _, err := e.SaveLASRecords(a, []LASRecord{v}, false)
	if err != nil || saved[0].RegisteredOwnerID != people[0].ID {
		t.Fatal("link failed", err)
	}
	v = saved[0]
	v.RegisteredOwnerID = "missing"
	if _, _, err := e.SaveLASRecords(a, []LASRecord{v}, false); err == nil {
		t.Fatal("allowed missing person")
	}
	v.RegisteredOwnerID = v.ID
	if _, _, err := e.SaveLASRecords(a, []LASRecord{v}, false); err == nil {
		t.Fatal("allowed vehicle as person")
	}
	v.RegisteredOwnerID = people[1].ID
	if _, _, err := e.SaveLASRecords(a, []LASRecord{v}, false); err == nil {
		t.Fatal("allowed cross-agency person link")
	}
	v.RegisteredOwnerID = ""
	saved, _, err = e.SaveLASRecords(a, []LASRecord{v}, false)
	if err != nil || saved[0].RegisteredOwnerID != "" || saved[0].Ownership != "employee" || len(saved[0].Cautions) != 1 {
		t.Fatal("unlink altered vehicle flags", err)
	}
}
