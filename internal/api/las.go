package api

import (
	"net/http"
	"strings"

	"github.com/biodream-llc/bitdispatcher/internal/auth"
	"github.com/biodream-llc/bitdispatcher/internal/cad"
)

// Government-only surfaces are unavailable in the LAS runtime, including direct API requests.
func restrictedLASPath(path string) bool {
	for _, p := range []string{"/api/ng911/", "/api/radio/", "/api/ncic/", "/api/cad/eido/", "/api/rms/ibr", "/api/admin/neris", "/api/admin/nemsis", "/api/admin/pip", "/api/admin/alerting", "/api/alerts", "/api/public/alerts"} {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return strings.HasPrefix(path, "/api/incidents/") && (strings.HasSuffix(path, "/neris") || strings.HasSuffix(path, "/nemsis") || strings.HasSuffix(path, "/run") || strings.HasSuffix(path, "/hit-confirmation"))
}

func (s *Server) lasRoutes() {
	s.handle("GET /api/admin/las", auth.Admin, func(w http.ResponseWriter, r *http.Request, se *auth.Session, a cad.Actor) {
		var seq uint64
		var unused bool
		s.Eng.Read(func(st *cad.State) {
			seq = st.Seq
			unused = len(st.Incidents) == 0 && len(st.Reports) == 0 && len(st.Calls) == 0 && len(st.Plans) == 0
		})
		writeJSON(w, 200, map[string]any{"seq": seq, "unused": unused, "callTypes": cad.LASCallTypes(), "settings": cad.LASSettings()})
	})
	s.handle("POST /api/admin/las", auth.Admin, func(w http.ResponseWriter, r *http.Request, se *auth.Session, a cad.Actor) {
		var b struct {
			Seq     uint64 `json:"seq"`
			Confirm bool   `json:"confirm"`
		}
		if !readJSON(w, r, &b) {
			return
		}
		if !b.Confirm {
			fail(w, 422, "review and confirm the Lakes Area Security configuration")
			return
		}
		if err := s.Eng.ConfigureLAS(a, b.Seq); err != nil {
			writeErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
