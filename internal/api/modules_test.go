package api

import (
	"encoding/json"
	"github.com/biodream-llc/bitdispatcher/internal/auth"
	"github.com/biodream-llc/bitdispatcher/internal/cad"
	"testing"
)

func TestWorkspaceModulesSwitchBetweenGrantedDuties(t *testing.T) {
	ts, s, pw := newServer(t)
	s.LAS = true
	s.moduleRoutes()
	s.Eng.SaveUnit(cad.SystemActor, cad.UnitDef{ID: "SEC1", Type: "patrol"})
	s.Auth.SetMDTUnit("admin", "SEC1")
	s.Auth.SetDuties("admin", []string{auth.DutyAdmin, auth.DutySupervisor, auth.DutyDispatcher, auth.DutyCallTaker, auth.DutyRecords, auth.DutyField})
	admin := login(t, ts, "admin", pw["admin"])
	code, body := admin.do("GET", "/api/auth/modules", nil, true)
	if code != 200 {
		t.Fatal(code)
	}
	var list struct{ Modules []workspaceModule }
	json.Unmarshal(body, &list)
	if len(list.Modules) != 5 {
		t.Fatal("missing module grants", string(body))
	}
	if code, _ := admin.do("POST", "/api/auth/module", map[string]string{"module": "mdt"}, false); code != 403 {
		t.Fatal("CSRF bypass", code)
	}
	code, body = admin.do("POST", "/api/auth/module", map[string]string{"module": "mdt"}, true)
	if code != 200 {
		t.Fatal(code, string(body))
	}
	var result struct {
		Path    string
		Changed bool
		Session auth.Session
	}
	json.Unmarshal(body, &result)
	if !result.Changed || result.Session.Role != auth.Field || result.Session.Console != "SEC1" || result.Path != "/mdt" {
		t.Fatal(string(body))
	}
	if code, _ := admin.do("GET", "/api/admin/users", nil, true); code != 403 {
		t.Fatal("field work retained admin session rights", code)
	}
	code, body = admin.do("POST", "/api/auth/module", map[string]string{"module": "admin"}, true)
	if code != 200 {
		t.Fatal(code, string(body))
	}
	json.Unmarshal(body, &result)
	if result.Session.Role != auth.Admin || result.Path != "/admin" {
		t.Fatal(string(body))
	}
	if code, _ := admin.do("GET", "/api/admin/users", nil, true); code != 200 {
		t.Fatal("admin access not restored", code)
	}
}
func TestWorkspaceModulesRespectSingleModuleAndRevocation(t *testing.T) {
	ts, s, pw := newServer(t)
	s.LAS = true
	s.moduleRoutes()
	s.Auth.SetDuties("v1", []string{auth.DutyRecords})
	viewer := login(t, ts, "v1", pw["v1"])
	code, body := viewer.do("GET", "/api/auth/modules", nil, true)
	if code != 200 {
		t.Fatal(code)
	}
	var list struct{ Modules []workspaceModule }
	json.Unmarshal(body, &list)
	if len(list.Modules) != 1 || list.Modules[0].ID != "rms" {
		t.Fatal("viewer module list", string(body))
	}
	if code, _ := viewer.do("POST", "/api/auth/module", map[string]string{"module": "admin"}, true); code != 403 {
		t.Fatal("viewer admin escalation", code)
	}
	admin := login(t, ts, "admin", pw["admin"])
	s.Auth.SetDuties("admin", []string{auth.DutyAdmin, auth.DutyField})
	s.Eng.SaveUnit(cad.SystemActor, cad.UnitDef{ID: "SEC1", Type: "patrol"})
	s.Auth.SetMDTUnit("admin", "SEC1")
	s.Auth.SetMDTUnit("admin", "")
	if code, _ := admin.do("POST", "/api/auth/module", map[string]string{"module": "mdt"}, true); code != 403 {
		t.Fatal("revoked unit used", code)
	}
	if code, _ := admin.do("POST", "/api/auth/module", map[string]string{"module": "unknown"}, true); code != 403 {
		t.Fatal("unknown module accepted", code)
	}
}
