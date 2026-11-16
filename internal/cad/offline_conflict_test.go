package cad

import (
	"testing"
	"time"
)

func TestOfflinePreconditionsAtCommit(t *testing.T) {
	e, c, _ := testEngine(t)
	var base time.Time
	e.Read(func(st *State) { base = st.Units["E1"].StatusAt })
	c.t = c.t.Add(time.Minute)
	if err := e.SetStatus(dispatcher, "E1", OutOfService, "", "dispatcher update"); err != nil {
		t.Fatal(err)
	}
	a := Actor{User: "fixture", ClientCommand: "fixture-offline-command", ClientUnit: "E1", ClientStatusAt: base}
	// Simulate dispatch winning the race after API preflight but before append.
	e.mu.Lock()
	_, err := e.write(a, EvUnitStatus, refs{Unit: "E1"}, statusData{Status: Available})
	e.mu.Unlock()
	if err == nil {
		t.Fatal("stale offline state was committed")
	}
	e.Read(func(st *State) {
		if st.Units["E1"].Status != OutOfService {
			t.Fatal("dispatch state overwritten")
		}
	})
}

func TestOfflineReportReceiptSurvivesLostAcknowledgement(t *testing.T) {
	e, _, path := testEngine(t)
	a := Actor{User: "fixture", ClientCommand: "fixture-report-create", ClientHash: "fixture-payload"}
	number, err := e.SaveReport(a, RMSReport{Narrative: "Synthetic offline draft"})
	if err != nil {
		t.Fatal(err)
	}
	other, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	result, ok := other.OfflineCommandResult(a.User, a.ClientCommand)
	if !ok || result.Number != number || result.Updated.IsZero() {
		t.Fatalf("report identity/version missing from durable receipt: %+v", result)
	}
}
