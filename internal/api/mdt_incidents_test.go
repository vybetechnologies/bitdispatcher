package api

import (
	"encoding/json"
	"github.com/biodream-llc/bitdispatcher/internal/cad"
	"testing"
)

func TestMDTCreateAndCloseUseOwnAssignedUnit(t *testing.T) {
	ts, s, pw := newServer(t)
	s.LAS = true
	for _, id := range []string{"MDT-A", "MDT-B"} {
		if err := s.Eng.SaveUnit(cad.SystemActor, cad.UnitDef{ID: id, Type: "engine"}); err != nil {
			t.Fatal(err)
		}
		if err := s.Eng.SetStatus(cad.SystemActor, id, cad.Available, "", ""); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Auth.SetMDTUnit("admin", "MDT-A"); err != nil {
		t.Fatal(err)
	}
	c := login(t, ts, "admin", pw["admin"])
	q := map[string]any{"type": "SF", "location": map[string]string{"address": "Synthetic test location"}, "narrative": "Fixture"}
	if code, _ := c.do("POST", "/api/mdt/incidents?unit=MDT-B", q, true); code != 403 {
		t.Fatal("other unit accepted", code)
	}
	code, body := c.do("POST", "/api/mdt/incidents", q, true)
	if code != 200 {
		t.Fatal(code, string(body))
	}
	var created struct{ Number string }
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatal(err)
	}
	var disposition string
	s.Eng.Read(func(st *cad.State) {
		disposition = st.Settings.Dispositions[0]
		if st.Units["MDT-A"].Incident != created.Number || st.Units["MDT-B"].Incident != "" {
			t.Fatal("wrong assignment")
		}
	})
	if code, body = c.do("POST", "/api/mdt/close", map[string]string{"incident": created.Number, "disposition": disposition}, true); code != 204 {
		t.Fatal(code, string(body))
	}
}
