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
