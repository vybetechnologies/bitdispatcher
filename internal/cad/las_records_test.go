package cad

import "testing"

func TestLASEmployeeTagDoesNotReplaceSafetyFlags(t *testing.T) {
	e, _, path := testEngine(t)
	a := Actor{User: "fixture-admin", Role: "admin"}
	records, _, err := e.SaveLASRecords(a, []LASRecord{{Kind: "person", Source: "Test", Person: &Person{Role: "subject", First: "Test", Last: "Employee", AgencyEmployee: true}}}, false)
	if err != nil {
		t.Fatal(err)
	}
	r := records[0]
	if !r.Person.AgencyEmployee || r.Safety || r.Person.Safety || len(r.FlagIDs) != 0 {
		t.Fatal("employee affiliation became a safety caution")
	}
	r.Person.Safety = true
	r.Person.SafetyNote = &SafetyConcern{Title: "Existing safety caution", Reason: "Synthetic test", Instruction: "Contact dispatch", Source: "Test", Scope: "permanent"}
	records, _, err = e.SaveLASRecords(a, []LASRecord{r}, false)
	if err != nil {
		t.Fatal(err)
	}
	r = records[0]
	if len(r.FlagIDs) == 0 || !r.Person.AgencyEmployee {
		t.Fatal("core safety flag did not coexist with employee tag")
	}
	r.Person.AgencyEmployee = false
	records, _, err = e.SaveLASRecords(a, []LASRecord{r}, false)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	restored.Read(func(st *State) {
		saved := st.LASRecords[records[0].ID]
		if saved.Person.AgencyEmployee || !saved.Person.Safety || len(saved.FlagIDs) == 0 {
			t.Fatal("removing employee tag changed safety flags or replay")
		}
	})
}

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
