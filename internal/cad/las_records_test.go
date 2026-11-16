package cad

import "testing"

func TestLASRecordsReplayAndRoles(t *testing.T) {
	path := t.TempDir() + "/ledger.jsonl"
	e, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	owner := Actor{User: "owner", Role: "admin"}
	if _, _, err = e.SaveLASRecords(Actor{User: "officer", Role: "field"}, []LASRecord{{Kind: "person", Name: "Fixture", Source: "Test"}}, false); err == nil {
		t.Fatal("field wrote records")
	}
	saved, _, err := e.SaveLASRecords(owner, []LASRecord{{Kind: "person", Name: "Fixture", Source: "Test"}}, false)
	if err != nil {
		t.Fatal(err)
	}
	e.Close()
	e, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	e.Read(func(st *State) {
		r, ok := st.LASRecords[saved[0].ID]
		if !ok || r.Name != "Fixture" || r.Version != 1 || r.By != "owner" {
			t.Fatal("record lost on restart", r)
		}
	})
}

func TestLASExistingIncidentPeopleConnectOnRestartWithoutNameMerging(t *testing.T) {
	path := t.TempDir() + "/ledger.jsonl"
	e, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	numbers := []string{}
	for range 2 {
		n, err := e.CreateIncident(SystemActor, NewIncident{Type: "OTHER"})
		if err != nil {
			t.Fatal(err)
		}
		numbers = append(numbers, n)
		if _, err = e.SavePerson(SystemActor, n, Person{Name: "Separate Person", Role: "witness", DOB: "1980-01-01", SSN: "123456789"}); err != nil {
			t.Fatal(err)
		}
	}
	var seq uint64
	e.Read(func(st *State) { seq = st.Seq })
	e.EnableLASPeople()
	ids := []string{}
	e.Read(func(st *State) {
		if st.Seq != seq || len(st.LASRecords) != 2 {
			t.Fatal("connection rewrote history or merged people")
		}
		for _, n := range numbers {
			p := st.Incidents[n].Persons[0]
			ids = append(ids, p.RecordID)
			if st.LASRecords[p.RecordID].Person.SSN != "123-45-6789" {
				t.Fatal("lost existing fields")
			}
		}
	})
	e.Close()
	e, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	e.EnableLASPeople()
	e.Read(func(st *State) {
		for i, n := range numbers {
			if st.Incidents[n].Persons[0].RecordID != ids[i] {
				t.Fatal("connection changed on restart")
			}
		}
	})
}
