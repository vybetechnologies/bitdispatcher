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
	if err := e.SetStatus(dispatcher, "E1", Available, "", "dispatcher update"); err != nil {
		t.Fatal(err)
	}
	a := Actor{User: "fixture", ClientCommand: "fixture-offline-command", ClientUnit: "E1", ClientStatusAt: base}
	// Simulate dispatch winning the race after API preflight but before append.
	e.mu.Lock()
	_, err := e.write(a, EvUnitStatus, refs{Unit: "E1"}, statusData{Status: OutOfService})
	e.mu.Unlock()
	if err == nil {
		t.Fatal("stale offline state was committed")
	}
	e.Read(func(st *State) {
		if st.Units["E1"].Status != Available {
			t.Fatal("dispatch state overwritten")
		}
	})
}
