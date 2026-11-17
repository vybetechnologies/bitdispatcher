package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/biodream-llc/bitdispatcher/internal/auth"
	"github.com/biodream-llc/bitdispatcher/internal/cad"
)

type offlineCommandKey struct{}
type offlineCommandMeta struct {
	Unit, Incident, Report  string
	StatusAt, ReportUpdated time.Time
	ID, Hash                string
	At                      time.Time
}
type desktopCommand struct {
	Context                   string
	ID                        string
	Owner                     string
	Unit                      string
	Method                    string
	Path                      string
	Binary, FileName, Caption string
	Body                      json.RawMessage
	Incident                  string
	BaseStatusAt              string
	At                        time.Time
}

var desktopCommandID = regexp.MustCompile("^[a-zA-Z0-9-]{16,80}$")

func desktopWriteAllowed(method, path string) bool {
	if method == "DELETE" && regexp.MustCompile("^/rms/reports/[A-Za-z0-9_-]+$").MatchString(path) {
		return true
	}
	if strings.ContainsAny(path, "?#%") || strings.Contains(path, "..") {
		return false
	}
	if method == "PUT" && path == "/mdt/crew" {
		return true
	}
	if method != "POST" {
		return false
	}
	switch path {
	case "/mdt/incidents", "/mdt/close", "/mdt/status", "/mdt/markout", "/mdt/note", "/mdt/position", "/mdt/emergency", "/mdt/message", "/mdt/attach", "/rms/reports":
		return true
	}
	return regexp.MustCompile("^/mdt/message/[A-Za-z0-9-]+/read$").MatchString(path) || regexp.MustCompile("^/comms/[A-Za-z0-9_-]+/(open|ack)$").MatchString(path) || regexp.MustCompile("^/incidents/[A-Za-z0-9_-]+/files$").MatchString(path)
}

func (s *Server) desktopRoutes() {
	s.desktopDownloadRoutes()
	s.handle("GET /api/mdt/offline/map-package", auth.Viewer, s.desktopMap)
	s.handle("GET /api/mdt/offline/records", auth.Viewer, s.desktopRecords)
	s.handle("POST /api/mdt/offline/sync", auth.Viewer, func(w http.ResponseWriter, r *http.Request, se *auth.Session, a cad.Actor) {
		unit, ok := s.mdtUnit(w, r, se)
		if !ok {
			return
		}
		var cmd desktopCommand
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 36<<20)).Decode(&cmd) != nil {
			fail(w, 400, "invalid offline command")
			return
		}
		if !desktopCommandID.MatchString(cmd.ID) || cmd.Owner != se.Username || cmd.Unit != unit || !desktopWriteAllowed(cmd.Method, cmd.Path) || cmd.At.IsZero() || cmd.At.After(time.Now().Add(5*time.Minute)) {
			fail(w, 422, "invalid offline command or account/unit binding")
			return
		}
		s.offlineMu.Lock()
		expectedContext := "live"
		if se.Training {
			board := se.Exercise
			if board == "" {
				board = se.Hash
			}
			expectedContext = "training:" + board
		}
		if cmd.Context != expectedContext {
			s.offlineMu.Unlock()
			fail(w, 409, "Saved work belongs to a different live/training context or an older app; retain it for review and update the Windows MDT.")
			return
		}
		defer s.offlineMu.Unlock()
		hash := fmt.Sprintf("%x", sha256.Sum256([]byte(cmd.Method+"|"+cmd.Path+"|"+cmd.Unit+"|"+cmd.Incident+"|"+cmd.BaseStatusAt+"|"+cmd.At.UTC().Format(time.RFC3339Nano)+"|"+string(cmd.Body)+"|"+cmd.Binary+"|"+cmd.FileName+"|"+cmd.Caption)))
		if prior, found := s.Eng.OfflineCommandResult(a.User, cmd.ID); found {
			if prior.Hash != hash {
				fail(w, 409, "this offline command ID was already used with different content")
				return
			}
			result := map[string]any{"applied": true, "duplicate": true, "id": prior.ID, "key": prior.Key, "appliedUpdated": prior.Updated}
			if prior.Number != "" {
				result["number"] = prior.Number
				s.Eng.Read(func(st *cad.State) {
					if rp := st.Reports[prior.Number]; rp != nil {
						result["report"] = rp
						result["checks"] = []any{}
					}
				})
			}
			writeJSON(w, 200, result)
			return
		}
		a.ClientCommand, a.ClientHash, a.ClientAt = cmd.ID, hash, cmd.At
		meta := offlineCommandMeta{ID: cmd.ID, Hash: hash, At: cmd.At}
		var own *cad.Unit
		var oldIncidentAllowed bool
		s.Eng.Read(func(st *cad.State) {
			if u := st.Units[unit]; u != nil {
				cp := *u
				own = &cp
			}
			if in := st.Incidents[cmd.Incident]; in != nil && (in.Agency == "" || own != nil && strings.EqualFold(in.Agency, own.Agency)) {
				for _, as := range in.Units {
					if as.Unit == unit {
						oldIncidentAllowed = true
					}
				}
			}
		})
		if own == nil {
			fail(w, 403, "assigned unit is unavailable")
			return
		}
		if strings.HasPrefix(cmd.Path, "/incidents/") && (!oldIncidentAllowed || cmd.Path != "/incidents/"+cmd.Incident+"/files") {
			fail(w, 409, "this file is not bound to the original assigned call")
			return
		}
		if cmd.Path == "/mdt/note" {
			var b struct{ Text string }
			if json.Unmarshal(cmd.Body, &b) != nil || !oldIncidentAllowed {
				fail(w, 409, "the original call cannot accept this offline note; review the saved entry")
				return
			}
			if err := s.Eng.AddNote(a, cmd.Incident, unit+": "+b.Text, false); err != nil {
				writeErr(w, err)
				return
			}
			w.WriteHeader(204)
			return
		}
		if cmd.Path == "/mdt/position" {
			var p cad.Position
			if json.Unmarshal(cmd.Body, &p) != nil || p.Lat < -90 || p.Lat > 90 || p.Lon < -180 || p.Lon > 180 || p.Lat == 0 && p.Lon == 0 {
				fail(w, 422, "invalid saved GPS position")
				return
			}
			p.At, p.Source = cmd.At, "mdt-offline"
			if time.Since(cmd.At) < time.Minute && (own.Position == nil || cmd.At.After(own.Position.At)) {
				if err := s.Eng.UnitPosition(a, unit, p); err != nil {
					writeErr(w, err)
					return
				}
			}
			// Historical locations belong in the audit trail, never as a fresh live AVL.
			if err := s.Eng.Record(a, cad.EvMDTOfflinePosition, map[string]any{"unit": unit, "position": p}); err != nil {
				writeErr(w, err)
				return
			}
			w.WriteHeader(204)
			return
		}
		if cmd.Path == "/mdt/incidents" || cmd.Path == "/mdt/close" || cmd.Path == "/mdt/status" || cmd.Path == "/mdt/markout" || cmd.Path == "/mdt/attach" || cmd.Path == "/mdt/crew" {
			base := cmd.BaseStatusAt
			if strings.HasPrefix(base, "local:") {
				previous, found := s.Eng.OfflineCommandResult(a.User, strings.TrimPrefix(base, "local:"))
				if found {
					base = previous.At.UTC().Format(time.RFC3339Nano)
				}
			}
			captured, err := time.Parse(time.RFC3339Nano, base)
			meta.Unit, meta.Incident, meta.StatusAt = unit, cmd.Incident, captured
			if err != nil || !captured.Equal(own.StatusAt) || cmd.Incident != own.Incident {
				fail(w, 409, "dispatch changed this unit or call while offline; review the saved action before retrying")
				return
			}
		}
		if cmd.Path == "/mdt/emergency" {
			if time.Since(cmd.At) > 5*time.Minute {
				fail(w, 409, "this saved emergency is over five minutes old; contact dispatch and review it before sending")
				return
			}
			key, err := s.Eng.RaiseEmergency(a, cad.Emergency{Unit: unit, Alias: unit, ChannelName: "Windows MDT offline sync", Raised: cmd.At})
			if err != nil {
				writeErr(w, err)
				return
			}
			writeJSON(w, 200, map[string]string{"key": key})
			return
		}
		if cmd.Path == "/mdt/message" {
			var b struct {
				To     []string
				Text   string
				Urgent bool
			}
			if json.Unmarshal(cmd.Body, &b) != nil {
				fail(w, 422, "invalid saved message")
				return
			}
			if cmd.Incident != "" && !oldIncidentAllowed {
				fail(w, 409, "the original call is no longer accessible")
				return
			}
			if len(b.To) == 0 {
				b.To = []string{cad.ToDispatch}
			}
			id, err := s.Eng.SendMessage(a, unit, b.To, cmd.Incident, b.Text, b.Urgent)
			if err != nil {
				writeErr(w, err)
				return
			}
			writeJSON(w, 200, map[string]string{"id": id})
			return
		}
		if cmd.Path == "/rms/reports" {
			var rp cad.RMSReport
			if json.Unmarshal(cmd.Body, &rp) != nil {
				fail(w, 422, "invalid report")
				return
			}
			if rp.Number != "" {
				meta.Report, meta.ReportUpdated = rp.Number, rp.Updated
				changed := false
				s.Eng.Read(func(st *cad.State) {
					if current := st.Reports[rp.Number]; current == nil || !current.Updated.Equal(rp.Updated) {
						changed = true
					}
				})
				if changed {
					fail(w, 409, "this report changed on the server; keep the local draft and review the differences")
					return
				}
			}
		}
		// Reuse each operation's normal authorization and validation. Metadata comes
		// from a private context value; clients cannot inject it into arbitrary routes.
		target := "/api" + cmd.Path
		rawBody := []byte(cmd.Body)
		if cmd.Binary != "" {
			if !strings.HasPrefix(cmd.Path, "/incidents/") || !strings.HasSuffix(cmd.Path, "/files") {
				fail(w, 422, "binary data is only accepted for incident attachments")
				return
			}
			var err error
			rawBody, err = base64.StdEncoding.DecodeString(cmd.Binary)
			if err != nil || len(rawBody) > maxFile {
				fail(w, 422, "invalid saved file or file over 25 MB")
				return
			}
			target += "?name=" + url.QueryEscape(cmd.FileName) + "&caption=" + url.QueryEscape(cmd.Caption)
		}
		request := httptest.NewRequest(cmd.Method, target, bytes.NewReader(rawBody))
		request.Header.Set("X-BitDispatcher", "1")
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Cookie", r.Header.Get("Cookie"))
		request = request.WithContext(context.WithValue(r.Context(), offlineCommandKey{}, meta))
		response := httptest.NewRecorder()
		s.mux.ServeHTTP(response, request)
		if response.Code >= 200 && response.Code < 300 {
			if _, found := s.Eng.OfflineCommandResult(a.User, cmd.ID); !found {
				// Successful no-op operations also receive a durable deduplication receipt.
				if err := s.Eng.Record(a, cad.EvMDTOfflineApplied, map[string]any{"path": cmd.Path, "unit": unit, "statusAt": own.StatusAt}); err != nil {
					writeErr(w, err)
					return
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(response.Code)
		_, _ = w.Write(response.Body.Bytes())
	})
}
func (s *Server) desktopRecords(w http.ResponseWriter, r *http.Request, se *auth.Session, a cad.Actor) {
	unit, ok := s.mdtUnit(w, r, se)
	if !ok {
		return
	}
	records := []mdtLookupHit{}
	now := s.Eng.Now()
	s.Eng.Read(func(st *cad.State) {
		own := st.Units[unit]
		if own == nil {
			return
		}
		for _, record := range st.LASRecords {
			if record.Archived || record.Restricted || (record.Agency != "" && !strings.EqualFold(record.Agency, own.Agency)) {
				continue
			}
			p := cad.LASRecordPerson(record)
			if record.Kind == "person" && lookupMinor(p.DOB, p.ApproxAge, now) {
				continue
			}
			flags := []*cad.Flag{}
			for _, f := range st.LASPersonFlags(record.ID, p, now) {
				flags = append(flags, st.FlagFor(f, cad.DutyOf(a), a.Role))
			}
			hit := mdtLookupHit{Record: record.ID, Source: record.Source, Seen: record.Updated, Name: record.Name, DOB: record.DOB, Sex: record.Sex, Description: record.Description, Plate: record.Plate, State: record.State, VIN: record.VIN, Year: record.Year, Make: record.Make, Model: record.Model, Color: record.Color, Body: record.Body, Safety: record.Safety || len(flags) > 0, Flags: flags}
			if record.Kind == "person" {
				hit.Person = &p
			}
			records = append(records, hit)
		}
	})
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, map[string]any{"unit": unit, "owner": se.Username, "records": records, "at": now})
}
