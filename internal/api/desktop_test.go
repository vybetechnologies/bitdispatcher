package api

import (
	"encoding/json"
	"github.com/biodream-llc/bitdispatcher/internal/cad"
	"strings"
	"testing"
	"time"
)

func TestDesktopOfflineOfficerIncidentAndCloseReceipts(t *testing.T) {
	ts, s, pw := newServer(t)
	s.LAS = true
	s.desktopRoutes()
	if err := s.Eng.SaveUnit(cad.SystemActor, cad.UnitDef{ID: "OFFLINE1", Type: "engine"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Eng.SetStatus(cad.SystemActor, "OFFLINE1", cad.Available, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.Auth.SetMDTUnit("admin", "OFFLINE1"); err != nil {
		t.Fatal(err)
	}
	c := login(t, ts, "admin", pw["admin"])
	var base time.Time
	var disposition string
	s.Eng.Read(func(st *cad.State) { base = st.Units["OFFLINE1"].StatusAt; disposition = st.Settings.Dispositions[0] })
	create := desktopCommand{Context: "live", ID: "fixture-offline-create-001", Owner: "admin", Unit: "OFFLINE1", Method: "POST", Path: "/mdt/incidents", Body: json.RawMessage(`{"type":"SF","location":{"address":"Synthetic location"}}`), BaseStatusAt: base.Format(time.RFC3339Nano), At: time.Now()}
	code, body := c.do("POST", "/api/mdt/offline/sync", create, true)
	if code != 200 {
		t.Fatal(code, string(body))
	}
	var result struct{ Number string }
	json.Unmarshal(body, &result)
	receipt, ok := s.Eng.OfflineCommandResult("admin", create.ID)
	if !ok || receipt.Number != result.Number {
		t.Fatal("created call number missing from receipt")
	}
	s.Eng.Read(func(st *cad.State) {
		if !receipt.At.Equal(st.Units["OFFLINE1"].StatusAt) {
			t.Fatal("call receipt has wrong unit timestamp")
		}
	})
	if code, body = c.do("POST", "/api/mdt/offline/sync", create, true); code != 200 || !strings.Contains(string(body), result.Number) {
		t.Fatal("lost creation acknowledgement cannot recover call number", code, string(body))
	}
	raw, _ := json.Marshal(map[string]string{"incident": result.Number, "disposition": disposition})
	closeCmd := desktopCommand{Context: "live", ID: "fixture-offline-close-0001", Owner: "admin", Unit: "OFFLINE1", Method: "POST", Path: "/mdt/close", Incident: result.Number, Body: raw, BaseStatusAt: "local:" + create.ID, At: time.Now()}
	if code, body = c.do("POST", "/api/mdt/offline/sync", closeCmd, true); code != 204 {
		t.Fatal(code, string(body))
	}
}

func TestDesktopOfflineUnitBindingDeduplicationAndConflicts(t *testing.T) {
	ts, s, pw := newServer(t)
	s.LAS = true
	s.desktopRoutes()
	if err := s.Eng.SaveUnit(cad.SystemActor, cad.UnitDef{ID: "LAS1", Type: "patrol", Agency: "LAS"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Auth.SetMDTUnit("admin", "LAS1"); err != nil {
		t.Fatal(err)
	}
	client := login(t, ts, "admin", pw["admin"])
	var at time.Time
	s.Eng.Read(func(st *cad.State) { at = st.Units["LAS1"].StatusAt })
	cmd := desktopCommand{Context: "live", ID: "fixture-command-00000001", Owner: "admin", Unit: "LAS1", Method: "POST", Path: "/mdt/status", Body: json.RawMessage(`{"status":"available"}`), BaseStatusAt: at.Format(time.RFC3339Nano), At: time.Now().Add(-time.Minute)}
	cmd.Context = "training:fixture"
	if code, _ := client.do("POST", "/api/mdt/offline/sync", cmd, true); code != 409 {
		t.Fatal("training work accepted on the live board", code)
	}
	cmd.Context = "live"
	code, body := client.do("POST", "/api/mdt/offline/sync", cmd, true)
	if code != 204 {
		t.Fatal(code, string(body))
	}
	result, ok := s.Eng.OfflineCommandResult("admin", cmd.ID)
	if !ok || result.Hash == "" {
		t.Fatal("no durable command receipt")
	}
	var seq uint64
	s.Eng.Read(func(st *cad.State) { seq = st.Seq })
	code, body = client.do("POST", "/api/mdt/offline/sync", cmd, true)
	if code != 200 || !strings.Contains(string(body), "duplicate") {
		t.Fatal(code, string(body))
	}
	s.Eng.Read(func(st *cad.State) {
		if st.Seq != seq {
			t.Fatal("duplicate wrote another event")
		}
	})
	cmd.Body = json.RawMessage(`{"status":"oos"}`)
	if code, _ = client.do("POST", "/api/mdt/offline/sync", cmd, true); code != 409 {
		t.Fatal("command ID reused with new payload", code)
	}
	cmd.ID = "fixture-command-00000002"
	if code, _ = client.do("POST", "/api/mdt/offline/sync", cmd, true); code != 409 {
		t.Fatal("stale status overwrote dispatch", code)
	}
	cmd.Unit = "OTHER"
	if code, _ = client.do("POST", "/api/mdt/offline/sync", cmd, true); code != 422 {
		t.Fatal("wrong unit accepted", code)
	}
	cmd.Unit = "LAS1"
	cmd.Path = "/admin/settings"
	if code, _ = client.do("POST", "/api/mdt/offline/sync", cmd, true); code != 422 {
		t.Fatal("arbitrary route replay accepted", code)
	}
	// Replaying the ledger reconstructs deduplication after a server restart.
	other, err := cad.Open(s.Eng.LedgerPath())
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	if _, ok := other.OfflineCommandResult("admin", "fixture-command-00000001"); !ok {
		t.Fatal("deduplication lost on restart")
	}
}
