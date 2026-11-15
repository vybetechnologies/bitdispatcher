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
