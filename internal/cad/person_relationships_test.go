package cad

import "testing"

func TestPersonRelationshipsValidationAndReplay(t *testing.T) {
	e, _, path := testEngine(t)
	a := Actor{User: "fixture-admin", Role: "admin"}
	rows, _, err := e.SaveLASRecords(a, []LASRecord{{Kind: "person", Name: "First Person", Source: "Fixture"}, {Kind: "person", Name: "Second Person", Source: "Fixture"}}, false)
	if err != nil {
		t.Fatal(err)
	}
	r := rows[0]
	for _, link := range []PersonRelationship{{PersonID: r.ID, Type: "friend"}, {PersonID: "missing", Type: "friend"}, {PersonID: rows[1].ID, Type: "unknown"}} {
		r.Relationships = []PersonRelationship{link}
		if _, _, err := e.SaveLASRecords(a, []LASRecord{r}, false); err == nil {
			t.Fatal("accepted invalid relationship", link)
		}
	}
	r.Relationships = []PersonRelationship{{PersonID: rows[1].ID, Type: "parent"}}
	saved, _, err := e.SaveLASRecords(a, []LASRecord{r}, false)
	if err != nil {
		t.Fatal(err)
	}
	if saved[0].Relationships[0].Name != rows[1].Name {
		t.Fatal("canonical name missing")
	}
	e.Close()
	restored, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	restored.Read(func(st *State) {
		if st.LASRecords[r.ID].Relationships[0].PersonID != rows[1].ID {
			t.Fatal("link lost after restart")
		}
	})
}
