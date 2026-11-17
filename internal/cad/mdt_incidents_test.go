package cad

import (
	"github.com/biodream-llc/bitdispatcher/internal/ng911"
	"testing"
)

func TestMDTIncidentLifecycleAndOtherUnits(t *testing.T) {
	e, _, _ := testEngine(t)
	a := Actor{User: "fixture-officer", Role: "field", Duty: DutyField}
	n := NewIncident{Type: "SF", Location: &ng911.Location{Address: "Synthetic location"}, Narrative: "Synthetic officer-initiated call"}
	number, err := e.CreateMDTIncident(a, "E1", n)
	if err != nil {
		t.Fatal(err)
	}
	e.Read(func(st *State) {
		if st.Units["E1"].Incident != number || st.Incidents[number].Status != Active {
			t.Fatal("created call was not assigned")
		}
	})
	if _, err = e.CreateMDTIncident(a, "E1", n); err == nil {
		t.Fatal("busy unit created another call")
	}
	if err = e.CloseMDTIncident(a, "E2", number, "invalid", ""); err == nil {
		t.Fatal("other unit closed call")
	}
	if err = e.Dispatch(dispatcher, number, []string{"E2"}, nil); err != nil {
		t.Fatal(err)
	}
	var disposition string
	e.Read(func(st *State) { disposition = st.Settings.Dispositions[0] })
	if err = e.CloseMDTIncident(a, "E1", number, disposition, ""); err == nil {
		t.Fatal("cleared another active unit")
	}
	if err = e.SetStatus(dispatcher, "E2", Available, "", ""); err != nil {
		t.Fatal(err)
	}
	if err = e.CloseMDTIncident(a, "E1", number, "invalid", ""); err == nil {
		t.Fatal("invalid disposition accepted")
	}
	if err = e.CloseMDTIncident(a, "E1", number, disposition, "Synthetic closing note"); err != nil {
		t.Fatal(err)
	}
	e.Read(func(st *State) {
		if st.Incidents[number].Status != Closed || st.Units["E1"].Status != Available || st.Units["E1"].Incident != "" {
			t.Fatal("close did not clear own unit")
		}
	})
}
func TestDeleteDraftReportOwnershipAndReplay(t *testing.T) {
	e, _, path := testEngine(t)
	owner := Actor{User: "fixture-owner", Role: "field"}
	number, err := e.CreateIncident(dispatcher, NewIncident{Type: "SF"})
	if err != nil {
		t.Fatal(err)
	}
	report, err := e.SaveReport(owner, RMSReport{Incident: number, Narrative: "Synthetic draft"})
	if err != nil {
		t.Fatal(err)
	}
	if err = e.DeleteDraftReport(Actor{User: "other", Role: "field"}, report); err == nil {
		t.Fatal("other officer deleted draft")
	}
	if err = e.DeleteDraftReport(owner, report); err != nil {
		t.Fatal(err)
	}
	replay, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer replay.Close()
	replay.Read(func(st *State) {
		if st.Reports[report] != nil || len(st.Incidents[number].Reports) != 0 {
			t.Fatal("deleted report returned on replay")
		}
	})
	e.mu.Lock()
	e.st.Reports["submitted-test"] = &RMSReport{Number: "submitted-test", Author: owner.User, Status: RptSubmitted}
	e.mu.Unlock()
	if err = e.DeleteDraftReport(Actor{User: "admin", Role: "admin"}, "submitted-test"); err == nil {
		t.Fatal("submitted report deleted")
	}
}
