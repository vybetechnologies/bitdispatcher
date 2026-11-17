package api

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/biodream-llc/bitdispatcher/internal/auth"
	"github.com/biodream-llc/bitdispatcher/internal/cad"
)

type fbiNotice struct {
	UID            string   `json:"uid"`
	Title          string   `json:"title"`
	URL            string   `json:"url"`
	Aliases        []string `json:"aliases"`
	DOB            []string `json:"dates_of_birth_used"`
	Subjects       []string `json:"subjects"`
	Classification string   `json:"person_classification"`
	Sex            string   `json:"sex"`
	Race           string   `json:"race"`
	Status         string   `json:"status"`
	Caution        string   `json:"caution"`
	Warning        string   `json:"warning_message"`
	Description    string   `json:"description"`
}
type fbiSearchResult struct {
	Items   []fbiNotice `json:"items"`
	Checked time.Time   `json:"checked"`
	More    bool        `json:"more"`
	Message string      `json:"message,omitempty"`
}

var fbiMarkup = regexp.MustCompile(`<[^>]*>`)

func fbiText(s string) string {
	return strings.Join(strings.Fields(html.UnescapeString(fbiMarkup.ReplaceAllString(s, " "))), " ")
}
func officialFBIPage(value string) bool {
	u, err := url.Parse(value)
	return err == nil && u.Scheme == "https" && (u.Host == "www.fbi.gov" || u.Host == "fbi.gov") && u.User == nil
}
func fbiDOBMatches(query string, values []string) bool {
	if query == "" {
		return true
	}
	known := false
	for _, v := range values {
		for _, layout := range []string{"January 2, 2006", "January 02, 2006", "2006-01-02", "01/02/2006", "Jan 2, 2006"} {
			if d, err := time.Parse(layout, v); err == nil {
				known = true
				if d.Format("2006-01-02") == query {
					return true
				}
			}
		}
	}
	return !known
}
func (s *Server) fbiWantedRoutes() {
	s.handle("POST /api/mdt/records/fbi", auth.Viewer, func(w http.ResponseWriter, r *http.Request, se *auth.Session, a cad.Actor) {
		w.Header().Set("Cache-Control", "private, no-store")
		unit, ok := s.mdtUnit(w, r, se)
		if !ok {
			return
		}
		var q mdtLookupQuery
		if !readJSON(w, r, &q) {
			return
		}
		name := lookupName(q.Name)
		if len([]rune(name)) < 2 || len(q.Name) > 100 || !slices.Contains([]string{"assigned-call", "patrol", "follow-up"}, q.Purpose) {
			fail(w, 422, "enter a person name and operational purpose")
			return
		}
		if q.DOB != "" {
			if _, err := time.Parse("2006-01-02", q.DOB); err != nil {
				fail(w, 422, "invalid date of birth")
				return
			}
		}
		key := name + "|" + q.DOB
		digest := fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
		audit := func(count int, available bool) {
			s.Eng.Record(a, cad.EvSearched, map[string]any{"kind": "fbi-public-notices", "unit": unit, "purpose": q.Purpose, "queryHash": digest, "results": count, "available": available})
		}
		s.fbiMu.Lock()
		cached, found := s.fbiCache[key]
		s.fbiMu.Unlock()
		if found && time.Since(cached.Checked) < 2*time.Minute {
			audit(len(cached.Items), true)
			writeJSON(w, 200, cached)
			return
		}
		result, err := s.searchFBI(r.Context(), name, q.DOB)
		if err != nil {
			audit(0, false)
			fail(w, 502, "FBI public notices are unavailable. Local records and Enformion searches are unaffected.")
			return
		}
		s.fbiMu.Lock()
		if s.fbiCache == nil || len(s.fbiCache) >= 128 {
			s.fbiCache = map[string]fbiSearchResult{}
		}
		s.fbiCache[key] = result
		s.fbiMu.Unlock()
		audit(len(result.Items), true)
		w.Header().Set("Cache-Control", "private, no-store")
		writeJSON(w, 200, result)
	})
}
func (s *Server) searchFBI(parent context.Context, name, dob string) (fbiSearchResult, error) {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	client := s.fbiHTTP
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	result := fbiSearchResult{Items: []fbiNotice{}, Checked: time.Now().UTC()}
	seen := map[string]bool{}
	for page := 1; page <= 2; page++ {
		params := url.Values{"title": {name}, "pageSize": {"50"}, "page": {strconv.Itoa(page)}}
		req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.fbi.gov/wanted/v1/list?"+params.Encode(), nil)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "LakesAreaSecurity-MDT/1.0")
		response, err := client.Do(req)
		if err != nil {
			if page == 1 {
				return result, err
			}
			result.More = true
			result.Message = "FBI search is incomplete; refine the name or verify on the official site."
			break
		}
		var data struct {
			Total int         `json:"total"`
			Items []fbiNotice `json:"items"`
		}
		if response.StatusCode != 200 {
			response.Body.Close()
			return result, fmt.Errorf("FBI status %d", response.StatusCode)
		}
		err = json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(&data)
		response.Body.Close()
		if err != nil {
			return result, err
		}
		for _, notice := range data.Items {
			// The upstream title filter can return OR matches. Require all entered name
			// words in a title or single alias before issuing any possible-match caution.
			if !lookupNameMatches(name, cad.Person{Name: notice.Title, Aliases: notice.Aliases}) || !fbiDOBMatches(dob, notice.DOB) || !officialFBIPage(notice.URL) || seen[notice.UID] {
				continue
			}
			status := strings.ToLower(notice.Status)
			if status == "captured" || status == "located" || status == "deceased" {
				continue
			}
			notice.Caution = fbiText(notice.Caution)
			notice.Warning = fbiText(notice.Warning)
			notice.Description = fbiText(notice.Description)
			notice.Title = fbiText(notice.Title)
			result.Items = append(result.Items, notice)
			seen[notice.UID] = true
		}
		result.More = data.Total > page*50
		if !result.More {
			break
		}
	}
	return result, nil
}
