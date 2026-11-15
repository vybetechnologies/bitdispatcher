package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLASProfileAuthorizationAndCSRF(t *testing.T) {
	ts, s, pw := newServer(t)
	s.LAS = true
	s.lasRoutes()
	viewer := login(t, ts, "v1", pw["v1"])
	if code, _ := viewer.do("GET", "/api/admin/las", nil, true); code != 403 {
		t.Fatal("viewer read admin profile")
	}
	admin := login(t, ts, "admin", pw["admin"])
	code, b := admin.do("GET", "/api/admin/las", nil, true)
	if code != 200 {
		t.Fatal(code)
	}
	var v struct {
		Seq uint64 `json:"seq"`
	}
	json.Unmarshal(b, &v)
	if code, _ = admin.do("POST", "/api/admin/las", map[string]any{"seq": v.Seq, "confirm": true}, false); code != 403 {
		t.Fatal("profile write without CSRF accepted")
	}
	if code, _ = admin.do("POST", "/api/admin/las", map[string]any{"seq": v.Seq, "confirm": false}, true); code != 422 {
		t.Fatal("profile write without review accepted")
	}
	if code, _ = admin.do("POST", "/api/admin/las", map[string]any{"seq": v.Seq, "confirm": true}, true); code != 204 {
		t.Fatal("profile apply failed", code)
	}
}

func TestLASBlocksGovernmentRoutesAndLoopback(t *testing.T) {
	_, s, _ := newServer(t)
	s.LAS = true
	for _, p := range []string{"/api/ncic/query", "/api/rms/ibr", "/api/admin/neris", "/api/ng911/eido", "/api/alerts", "/api/incidents/TEST/records/person/run"} {
		r := httptest.NewRequest(http.MethodPost, p, nil)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Errorf("government route reachable: %s (%d)", p, w.Code)
		}
	}
	w := httptest.NewRecorder()
	s.HooksHandler().ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/ng911/eido", nil))
	if w.Code != 403 {
		t.Fatal("loopback bypass")
	}
	for _, p := range []string{"/api/admin/las", "/api/admin/cameras", "/api/admin/gis", "/api/mdt", "/api/rms/reports"} {
		if restrictedLASPath(p) {
			t.Error("private-security route blocked", p)
		}
	}
}

func TestLASChecklistOverrideKeepsDiagnostics(t *testing.T) {
	ts, s, pw := newServer(t)
	s.LAS = true
	admin := login(t, ts, "admin", pw["admin"])
	if code, _ := admin.do("PUT", "/api/admin/setup/units", map[string]any{"done": true}, true); code != 422 {
		t.Fatal("reasonless override accepted", code)
	}
	if code, _ := admin.do("PUT", "/api/admin/setup/units", map[string]any{"done": true, "note": "Owner-directed checklist closure; configuration review remains open"}, true); code != 200 {
		t.Fatal(code)
	}
	v := s.setupView()
	found := false
	for _, x := range v.Steps {
		if x.ID == "units" {
			found = true
			if x.Status != "warn" || x.Tick == nil || x.Tick.By != "admin" {
				t.Fatal("override not attributed as warning", x)
			}
		}
	}
	if !found {
		t.Fatal("units step missing")
	}
	if v.GoLive != nil {
		t.Fatal("override invented a go-live result")
	}
}

func TestLASDutyRestrictionsAndSelfAdminPreservation(t *testing.T) {
	ts, s, pw := newServer(t)
	s.LAS = true
	admin := login(t, ts, "admin", pw["admin"])
	if code, _ := admin.do("PUT", "/api/admin/users/admin/duties", map[string]any{"duties": []string{"admin", "field", "records"}}, true); code != 204 {
		t.Fatal("combined admin and MDT rejected", code)
	}
	if code, _ := admin.do("PUT", "/api/admin/users/admin/duties", map[string]any{"duties": []string{"field"}}, true); code != 422 {
		t.Fatal("self-admin removal accepted", code)
	}
	if code, _ := admin.do("PUT", "/api/admin/users/d1/duties", map[string]any{"duties": []string{" FIRE "}}, true); code != 422 {
		t.Fatal("case bypass of duty restriction", code)
	}
	if code, _ := admin.do("POST", "/api/auth/login", map[string]any{"username": "admin", "password": pw["admin"], "duty": " EMS "}, true); code != 403 {
		t.Fatal("case bypass of login restriction", code)
	}
}
