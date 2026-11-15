package api

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/biodream-llc/bitdispatcher/internal/auth"
	"github.com/biodream-llc/bitdispatcher/internal/cad"
)

// No free-text call type, narrative, unit or person is copied into this feed.
var lasPublicCategories = map[string]string{
	"PATROL": "Patrol / property check", "SITE": "Security response", "ALARM": "Alarm response",
	"ACCESS": "Access assistance", "LOCKUP": "Access assistance", "UNLOCK": "Access assistance", "LOCKOUT": "Access assistance",
	"SUSP": "Security response", "DIST": "Security response", "TRESPASS": "Security response", "DAMAGE": "Security response", "THEFT": "Security response",
	"ESCORT": "Client assistance", "CLIENT": "Client assistance", "EVENT": "Event security",
}

// Named places, intersections and annotated addresses are withheld.
var lasStreet = regexp.MustCompile(`(?i)^\s*([0-9]{1,6})\s+([A-Za-z0-9][A-Za-z0-9 .'-]{0,65}?\s(?:STREET|ST|AVENUE|AVE|ROAD|RD|DRIVE|DR|LANE|LN|BOULEVARD|BLVD|COURT|CT|CIRCLE|CIR|WAY|PARKWAY|PKWY|PLACE|PL|TERRACE|TER|TRAIL|TRL)(?:\s+[NSEW]{1,2})?)\s*(?:(?:APT|UNIT|STE|SUITE|#)\s*[A-Za-z0-9-]+)?\s*$`)

func lasPublicBlock(address string) string {
	first := strings.TrimSpace(strings.SplitN(address, ",", 2)[0])
	m := lasStreet.FindStringSubmatch(first)
	if m == nil {
		return ""
	}
	number, err := strconv.Atoi(m[1])
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%d block %s", number/100*100, strings.TrimSpace(m[2]))
}

type LASPublicEntry struct {
	Reference string    `json:"reference"`
	Category  string    `json:"category"`
	Area      string    `json:"area"`
	Block     string    `json:"block,omitempty"`
	Received  time.Time `json:"received"`
	Status    string    `json:"status"`
}
type LASPublicDoc struct {
	Agency    string           `json:"agency"`
	Generated time.Time        `json:"generated"`
	Calls     []LASPublicEntry `json:"calls"`
}

func (s *Server) lasPublicDoc() LASPublicDoc {
	now := s.Eng.Now().UTC()
	doc := LASPublicDoc{Agency: "Lakes Area Security", Generated: now, Calls: []LASPublicEntry{}}
	s.Eng.Read(func(st *cad.State) {
		for number, in := range st.Incidents {
			category, eligible := lasPublicCategories[in.Type]
			if !eligible || st.LASPublicCalls[number].Withheld || in.Created.After(now.Add(-5*time.Minute)) || in.Created.Before(now.AddDate(0, 0, -90)) {
				continue
			}
			status := "Active"
			if in.Status == cad.Closed {
				status = "Closed"
			}
			hash := sha256.Sum256([]byte("LAS-public-call:" + number))
			item := LASPublicEntry{Reference: fmt.Sprintf("LAS-%x", hash[:6]), Category: category, Area: "Area withheld", Received: in.Created.UTC().Truncate(15 * time.Minute), Status: status}
			if loc := in.Location; loc != nil {
				item.Block = lasPublicBlock(loc.Address)
				if loc.Lat != nil && loc.Lon != nil && *loc.Lat >= 46.75 && *loc.Lat <= 47.05 && *loc.Lon >= -97.0 && *loc.Lon <= -96.6 {
					item.Area = "Fargo metro area"
				}
			}
			doc.Calls = append(doc.Calls, item)
		}
	})
	sort.Slice(doc.Calls, func(i, j int) bool {
		if doc.Calls[i].Received.Equal(doc.Calls[j].Received) {
			return doc.Calls[i].Reference < doc.Calls[j].Reference
		}
		return doc.Calls[i].Received.After(doc.Calls[j].Received)
	})
	if len(doc.Calls) > 1000 {
		doc.Calls = doc.Calls[:1000]
	}
	return doc
}
func (s *Server) lasPublicRoutes() {
	s.mux.HandleFunc("GET /api/public/las-calls", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Access-Control-Allow-Origin", "https://lakesareasecurity.com")
		writeJSON(w, 200, s.lasPublicDoc())
	})
	s.handle("GET /api/admin/las-public-calls", auth.Admin, func(w http.ResponseWriter, r *http.Request, se *auth.Session, a cad.Actor) {
		type item struct {
			Incident string `json:"incident"`
			Type     string `json:"type"`
			Status   string `json:"status"`
			Eligible bool   `json:"eligible"`
			Withheld bool   `json:"withheld"`
		}
		items := []item{}
		var seq uint64
		s.Eng.Read(func(st *cad.State) {
			seq = st.Seq
			for id, in := range st.Incidents {
				_, eligible := lasPublicCategories[in.Type]
				items = append(items, item{id, in.Type, in.Status, eligible, st.LASPublicCalls[id].Withheld})
			}
		})
		sort.Slice(items, func(i, j int) bool { return items[i].Incident > items[j].Incident })
		writeJSON(w, 200, map[string]any{"seq": seq, "items": items})
	})
	s.handle("PUT /api/admin/las-public-calls/{incident}", auth.Admin, func(w http.ResponseWriter, r *http.Request, se *auth.Session, a cad.Actor) {
		var b struct {
			Seq      uint64 `json:"seq"`
			Withheld bool   `json:"withheld"`
		}
		if !readJSON(w, r, &b) {
			return
		}
		if err := s.Eng.SaveLASPublicCall(a, cad.LASPublicCall{Incident: r.PathValue("incident"), Withheld: b.Withheld}, b.Seq); err != nil {
			writeErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
