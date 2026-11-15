package api

import (
	"net/http"
	"slices"

	"github.com/biodream-llc/bitdispatcher/internal/auth"
	"github.com/biodream-llc/bitdispatcher/internal/cad"
)

type workspaceModule struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Path  string `json:"path"`
	Duty  string `json:"-"`
}

func (s *Server) modulesFor(se *auth.Session) []workspaceModule {
	result := []workspaceModule{}
	user, ok := s.Auth.ModuleAccess(se.Username)
	if !ok {
		return result
	}
	has := func(duties ...string) bool {
		return slices.ContainsFunc(duties, func(d string) bool { return slices.Contains(user.Duties, d) })
	}
	choose := func(preferred ...string) string {
		if slices.Contains(preferred, se.Duty) && has(se.Duty) {
			return se.Duty
		}
		for _, d := range preferred {
			if has(d) {
				return d
			}
		}
		return ""
	}
	add := func(id, label, path, duty string) {
		if duty != "" {
			result = append(result, workspaceModule{id, label, path, duty})
		}
	}
	add("cad", "CAD", "/console", choose(auth.DutySupervisor, auth.DutyDispatcher, auth.DutyPolice, auth.DutyCallTaker, auth.DutyAdmin, auth.DutyViewer))
	add("taker", "Call intake", "/taker", choose(auth.DutyCallTaker, auth.DutyDispatcher, auth.DutySupervisor, auth.DutyAdmin))
	if user.MDTUnit != "" && has(auth.DutyField, auth.DutyDispatcher, auth.DutyPolice, auth.DutySupervisor, auth.DutyAdmin) {
		known := false
		s.Eng.Read(func(st *cad.State) { _, known = st.Units[user.MDTUnit] })
		if known {
			duty := choose(auth.DutyField)
			if duty == "" {
				duty = choose(auth.DutyDispatcher, auth.DutyPolice, auth.DutySupervisor, auth.DutyAdmin)
			}
			add("mdt", "MDT", "/mdt", duty)
		}
	}
	if has(auth.DutyRecords, auth.DutyField, auth.DutySupervisor, auth.DutyAdmin, auth.DutyDispatcher, auth.DutyCallTaker, auth.DutyPolice, auth.DutyViewer) {
		duty := choose(auth.DutyRecords, auth.DutyField, auth.DutySupervisor, auth.DutyAdmin, auth.DutyDispatcher, auth.DutyCallTaker, auth.DutyPolice, auth.DutyViewer)
		if has(se.Duty) && (se.Role == auth.Field || auth.AtLeast(se.Role, auth.Dispatcher)) {
			duty = se.Duty
		}
		path := "/rms"
		if duty == auth.DutyRecords {
			path = "/records"
		}
		add("rms", "RMS / Records", path, duty)
	}
	adminDuty := ""
	if has(auth.DutyAdmin) {
		adminDuty = auth.DutyAdmin
	} else if has(auth.DutySupervisor) {
		adminDuty = auth.DutySupervisor
	}
	add("admin", "Administration", "/admin", adminDuty)
	return result
}
func (s *Server) moduleRoutes() {
	s.handle("GET /api/auth/modules", auth.Viewer, func(w http.ResponseWriter, r *http.Request, se *auth.Session, a cad.Actor) {
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, 200, map[string]any{"modules": s.modulesFor(se)})
	})
	s.handle("POST /api/auth/module", auth.Viewer, func(w http.ResponseWriter, r *http.Request, se *auth.Session, a cad.Actor) {
		var body struct {
			Module string `json:"module"`
		}
		if !readJSON(w, r, &body) {
			return
		}
		var target *workspaceModule
		for _, m := range s.modulesFor(se) {
			if m.ID == body.Module {
				copy := m
				target = &copy
				break
			}
		}
		if target == nil {
			fail(w, 403, "your account does not have access to this module")
			return
		}
		next, err := s.Auth.SwitchDuty(se.ID, target.Duty)
		if err != nil {
			fail(w, 403, err.Error())
			return
		}
		changed := next.Role != se.Role || next.Duty != se.Duty || next.Console != se.Console
		if changed {
			s.Eng.Record(a, "session.module.changed", map[string]string{"module": target.ID, "fromDuty": se.Duty, "toDuty": next.Duty, "position": next.Console})
		}
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, 200, map[string]any{"path": target.Path, "changed": changed, "session": next})
	})
}
