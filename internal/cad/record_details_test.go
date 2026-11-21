package cad

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"testing"
)

func TestRecordDetailsValidation(t *testing.T) {
	var b bytes.Buffer
	_ = png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	photo := "data:image/png;base64," + base64.StdEncoding.EncodeToString(b.Bytes())
	p := Person{Photo: photo, RegisteredSexOffender: true, OffenderProfileURL: "https://registry.example/official/1", Cautions: []RecordCaution{{Type: "Weapons", Level: "danger"}}, Warrants: []PersonWarrant{{Number: "W1", Agency: "Test court", Offense: "Fixture", Status: "active", Issued: "2026-01-01"}}}
	if err := ValidatePersonRecordDetails(&p); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []Person{{Photo: "data:image/svg+xml;base64,abc"}, {Photo: "data:image/png;base64,abc"}, {OffenderProfileURL: "javascript:alert(1)"}, {OffenderProfileURL: "https://user:pass@registry.example/"}, {Warrants: []PersonWarrant{{Number: "W", Agency: "A", Offense: "Test", Status: "active", Issued: "2026-02-01", Expires: "2026-01-01"}}}, {Cautions: []RecordCaution{{Type: "Weapons", Level: "unknown"}}}} {
		if ValidatePersonRecordDetails(&bad) == nil {
			t.Fatalf("accepted invalid record: %+v", bad)
		}
	}
}
func TestPersonAndVehicleRecordDetailsPersist(t *testing.T) {
	e, _, _ := testEngine(t)
	a := Actor{User: "fixture-admin", Role: "admin"}
	rows, _, err := e.SaveLASRecords(a, []LASRecord{{Kind: "person", Source: "Fixture", Person: &Person{Role: "subject", Name: "Test Person", AgencyEmployee: true, RegisteredSexOffender: true, OffenderProfileURL: "https://registry.example/1", Cautions: []RecordCaution{{Type: "Weapons", Level: "danger"}}, Warrants: []PersonWarrant{{Number: "W1", Agency: "Test court", Offense: "Fixture", Status: "active"}}}}, {Kind: "vehicle", Source: "Fixture", Plate: "TEST1", Ownership: "employee", OwnerName: "Test Employee", Cautions: []RecordCaution{{Type: "Stolen plate", Level: "danger"}}}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !rows[0].Person.RegisteredSexOffender || len(rows[0].Person.Warrants) != 1 || !rows[0].Person.AgencyEmployee || rows[0].Person.Safety {
		t.Fatal("person details lost or changed core safety")
	}
	if rows[1].Ownership != "employee" || len(rows[1].Cautions) != 1 || rows[1].Safety {
		t.Fatal("vehicle details lost or ownership became safety concern")
	}
	rows[0].Person.Warrants[0].Status = "served"
	saved, _, err := e.SaveLASRecords(a, []LASRecord{rows[0]}, false)
	if err != nil || saved[0].Person.Warrants[0].Status != "served" {
		t.Fatal("warrant status update failed", err)
	}
}

func TestDocumentedWarrantCanPreserveMissingFields(t *testing.T) {
	p := Person{Warrants: []PersonWarrant{{Agency: "Test county", Source: "Synthetic source.pdf", AsOf: "2026-03-31", SourcePage: "2", Bond: "No bond", Status: "active"}}}
	if err := ValidatePersonRecordDetails(&p); err != nil {
		t.Fatal("source listing without invented warrant number or charge rejected", err)
	}
	p.Warrants[0].Source = ""
	if ValidatePersonRecordDetails(&p) == nil {
		t.Fatal("undocumented empty warrant accepted")
	}
}
