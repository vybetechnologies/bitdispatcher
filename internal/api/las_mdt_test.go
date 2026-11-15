package api

import (
	"github.com/biodream-llc/bitdispatcher/internal/cad"
	"testing"
)

func TestLASMDTUsesAssignedUnitEvenForAdministrators(t *testing.T) {
	ts, s, pw := newServer(t)
	s.LAS = true
	s.lasRoutes()
	for _, id := range []string{"E1", "E2"} {
		if err := s.Eng.SaveUnit(cad.SystemActor, cad.UnitDef{ID: id, Type: "engine"}); err != nil {
			t.Fatal(err)
		}
	}
	admin := login(t, ts, "admin", pw["admin"])
	if code, _ := admin.do("GET", "/api/mdt?unit=E1", nil, true); code != 403 {
		t.Fatal("unassigned admin obtained MDT", code)
	}
	if code, _ := admin.do("PUT", "/api/admin/users/admin/mdt-unit", map[string]string{"unit": "E1"}, true); code != 204 {
		t.Fatal(code)
	}
	if code, _ := admin.do("GET", "/api/mdt", nil, true); code != 200 {
		t.Fatal("own unit inaccessible", code)
	}
	for _, path := range []string{"/api/mdt?unit=E2", "/api/mdt?unit=e2"} {
		if code, _ := admin.do("GET", path, nil, true); code != 403 {
			t.Fatal("other unit readable", code)
		}
	}
	for _, path := range []string{"/api/mdt/status?unit=E2", "/api/mdt/note?unit=E2", "/api/mdt/position?unit=E2", "/api/mdt/emergency?unit=E2", "/api/mdt/message?unit=E2", "/api/mdt/attach?unit=E2"} {
		if code, _ := admin.do("POST", path, map[string]any{}, true); code != 403 {
			t.Fatal("other unit operation accepted", path, code)
		}
	}
	if code, _ := admin.do("PUT", "/api/admin/users/d1/mdt-unit", map[string]string{"unit": "E1"}, true); code != 422 {
		t.Fatal("duplicate assignment accepted", code)
	}
	if code, _ := admin.do("PUT", "/api/admin/users/admin/mdt-unit", map[string]string{"unit": ""}, true); code != 204 {
		t.Fatal(code)
	}
	if code, _ := admin.do("GET", "/api/mdt", nil, true); code != 403 {
		t.Fatal("revocation not immediate", code)
	}
	if code, _ := admin.do("GET", "/api/admin/users", nil, true); code != 200 {
		t.Fatal("MDT restriction removed admin rights", code)
	}
}
