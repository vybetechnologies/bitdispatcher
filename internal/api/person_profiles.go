package api

import (
	"net/http"
	"slices"
	"sort"
	"time"

	"github.com/biodream-llc/bitdispatcher/internal/auth"
	"github.com/biodream-llc/bitdispatcher/internal/cad"
)

type personAssociate struct {
	Record   string `json:"record"`
	Name     string `json:"name"`
	DOB      string `json:"dob,omitempty"`
	Type     string `json:"type"`
	Notes    string `json:"notes,omitempty"`
	Incoming bool   `json:"incoming,omitempty"`
}

// One hop only: cycles never expand a response recursively. Each profile open
// rechecks the target using the same own-unit and person visibility rules.
func (s *Server) personAssociates(st *cad.State, id, agency string, a cad.Actor, now time.Time) []personAssociate {
	out := []personAssociate{}
	seen := map[string]bool{}
	add := func(target, kind, notes string, incoming bool) {
		key := target + ":" + kind
		if seen[key] || target == id {
			return
		}
		p := s.mdtRegisteredOwner(st, target, agency, a, now)
		if p == nil {
			return
		}
		seen[key] = true
		out = append(out, personAssociate{Record: target, Name: p.Name, DOB: p.DOB, Type: kind, Notes: notes, Incoming: incoming})
	}
	for _, link := range st.LASRecords[id].Relationships {
		add(link.PersonID, link.Type, link.Notes, false)
	}
	for source, r := range st.LASRecords {
		if r.Kind != "person" {
			continue
		}
		for _, link := range r.Relationships {
			if link.PersonID == id {
				add(source, cad.ReverseRelationship(link.Type), link.Notes, true)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name == out[j].Name {
			return out[i].Record+out[i].Type < out[j].Record+out[j].Type
		}
		return out[i].Name < out[j].Name
	})
	return out
}
func (s *Server) personProfileRoutes() {
	s.handle("POST /api/mdt/records/person", auth.Viewer, func(w http.ResponseWriter, r *http.Request, se *auth.Session, a cad.Actor) {
		unit, ok := s.mdtUnit(w, r, se)
		if !ok {
			return
		}
		var q struct {
			Record  string `json:"record"`
			Purpose string `json:"purpose"`
		}
		if !readJSON(w, r, &q) {
			return
		}
		if q.Record == "" || len(q.Record) > 120 || !slices.Contains([]string{"assigned-call", "patrol", "follow-up"}, q.Purpose) {
			fail(w, 422, "choose a person record and operational purpose")
			return
		}
		var hit *mdtLookupHit
		now := s.Eng.Now()
		s.Eng.Read(func(st *cad.State) {
			own := st.Units[unit]
			if own == nil {
				return
			}
			profile := s.mdtRegisteredOwner(st, q.Record, own.Agency, a, now)
			if profile == nil {
				return
			}
			saved := st.LASRecords[q.Record]
			hit = &mdtLookupHit{Record: profile.Record, Name: profile.Name, DOB: profile.DOB, Person: profile.Person, Flags: profile.Flags, Seen: saved.Updated, Source: lasLookupSource(saved), Safety: saved.Safety || len(profile.Flags) > 0, Associates: s.personAssociates(st, q.Record, own.Agency, a, now)}
		})
		s.Eng.Record(a, cad.EvSearched, map[string]any{"kind": "mdt-local-person-profile", "unit": unit, "record": q.Record, "purpose": q.Purpose, "found": hit != nil})
		if hit == nil {
			fail(w, 404, "person record is not available")
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, 200, hit)
	})
}
