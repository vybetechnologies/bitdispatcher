package api

import (
	"github.com/biodream-llc/bitdispatcher/internal/auth"
	"github.com/biodream-llc/bitdispatcher/internal/cad"
	"net/http"
)

func (s *Server) mdtCreateIncident(w http.ResponseWriter, r *http.Request, se *auth.Session, a cad.Actor) {
	unit, ok := s.mdtUnit(w, r, se)
	if !ok {
		return
	}
	var n cad.NewIncident
	if !readJSON(w, r, &n) {
		return
	}
	number, err := s.Eng.CreateMDTIncident(a, unit, n)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, 200, map[string]string{"number": number})
}
func (s *Server) mdtCloseIncident(w http.ResponseWriter, r *http.Request, se *auth.Session, a cad.Actor) {
	unit, ok := s.mdtUnit(w, r, se)
	if !ok {
		return
	}
	var b struct {
		Incident    string `json:"incident"`
		Disposition string `json:"disposition"`
		Text        string `json:"text"`
	}
	if !readJSON(w, r, &b) {
		return
	}
	if err := s.Eng.CloseMDTIncident(a, unit, b.Incident, b.Disposition, b.Text); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
