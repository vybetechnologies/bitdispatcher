package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/biodream-llc/bitdispatcher/internal/cad"
)

const enformionEndpoint = "https://devapi.enformion.com/PersonSearch"

type enformionName struct{ FirstName, MiddleName, LastName, Suffix string }
type enformionPerson struct {
	TahoeID        string
	Name           enformionName
	FullName, Dob  string
	Age            int
	IsOptedOut     bool
	Akas           []enformionName
	Addresses      []struct{ FullAddress string }
	PhoneNumbers   []struct{ PhoneNumber, PhoneType string }
	EmailAddresses []struct{ EmailAddress string }
}

// Credentials stay in the hosting service's secret environment.
func enformionConfigured() bool {
	return os.Getenv("ENFORMION_AP_NAME") != "" && os.Getenv("ENFORMION_AP_PASSWORD") != "" && os.Getenv("ENFORMION_STORAGE_AUTHORIZED") == "true"
}

func enformionRequest(name, dob string) (map[string]any, error) {
	var first, middle, last string
	if parts := strings.SplitN(name, ",", 2); len(parts) == 2 {
		last = strings.TrimSpace(parts[0])
		words := strings.Fields(parts[1])
		if len(words) > 0 {
			first = words[0]
			middle = strings.Join(words[1:], " ")
		}
	} else {
		words := strings.Fields(name)
		if len(words) >= 2 {
			first = words[0]
			last = words[len(words)-1]
			middle = strings.Join(words[1:len(words)-1], " ")
		}
	}
	if first == "" || last == "" {
		return nil, fmt.Errorf("Enformion needs a first and last name; partial names remain local searches")
	}
	b := map[string]any{"FirstName": first, "MiddleName": middle, "LastName": last, "Page": 1, "ResultsPerPage": 10, "Includes": []string{"Addresses", "PhoneNumbers", "EmailAddresses"}}
	if dob != "" {
		d, err := time.Parse("2006-01-02", dob)
		if err != nil {
			return nil, fmt.Errorf("invalid birth date")
		}
		b["Dob"] = d.Format("1/2/2006")
	}
	return b, nil
}

func decodeEnformion(data []byte) ([]enformionPerson, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, fmt.Errorf("Enformion returned an invalid response")
	}
	for key, value := range obj {
		obj[strings.ToLower(key)] = value
	}
	if string(obj["iserror"]) == "true" {
		return nil, fmt.Errorf("Enformion reported a search error; local records are unchanged")
	}
	if raw, ok := obj["persons"]; ok {
		var people []enformionPerson
		if err := json.Unmarshal(raw, &people); err != nil {
			return nil, fmt.Errorf("Enformion returned invalid person results")
		}
		return people, nil
	}
	if _, ok := obj["tahoeid"]; ok {
		var p enformionPerson
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("Enformion returned invalid person details")
		}
		return []enformionPerson{p}, nil
	}
	return nil, fmt.Errorf("Enformion returned an unrecognized response; local records are unchanged")
}

func (s *Server) fetchEnformion(ctx context.Context, name, dob string) ([]enformionPerson, error) {
	body, err := enformionRequest(name, dob)
	if err != nil {
		return nil, err
	}
	encoded, _ := json.Marshal(body)
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", enformionEndpoint, bytes.NewReader(encoded))
	if err != nil {
		return nil, fmt.Errorf("could not prepare Enformion search")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("galaxy-ap-name", os.Getenv("ENFORMION_AP_NAME"))
	req.Header.Set("galaxy-ap-password", os.Getenv("ENFORMION_AP_PASSWORD"))
	req.Header.Set("galaxy-search-type", "Person")
	client := s.enformionHTTP
	if client == nil {
		client = &http.Client{Timeout: 12 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Enformion is unavailable; retry the search later")
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		// Provider details may contain personal data or credentials; expose only
		// fixed diagnostics for known entitlement failures.
		data, _ := io.ReadAll(io.LimitReader(response.Body, 8192))
		if bytes.Contains(bytes.ToLower(data), []byte("does not permit")) {
			return nil, fmt.Errorf("Enformion Person Search is not enabled for this API access profile. Contact Enformion to enable access; LAS records remain available.")
		}
		return nil, fmt.Errorf("Enformion search unavailable (HTTP %d); check API entitlement or quota", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 2*1024*1024+1))
	if err != nil || len(data) > 2*1024*1024 {
		return nil, fmt.Errorf("Enformion response could not be read")
	}
	return decodeEnformion(data)
}

func enformionRecord(v enformionPerson, agency string, now time.Time) (cad.LASRecord, bool) {
	if v.IsOptedOut || v.TahoeID == "" || len(v.TahoeID) > 100 {
		return cad.LASRecord{}, false
	}
	p := cad.Person{Role: "subject", First: v.Name.FirstName, Middle: v.Name.MiddleName, Last: v.Name.LastName, Suffix: v.Name.Suffix, Name: v.FullName, LocKind: "unknown"}
	for _, layout := range []string{"1/2/2006", "01/02/2006", "2006-01-02"} {
		if d, err := time.Parse(layout, v.Dob); err == nil {
			p.DOB = d.Format("2006-01-02")
			break
		}
	}
	if v.Age > 0 && v.Age < 130 {
		p.ApproxAge = strconv.Itoa(v.Age)
	}
	notes := []string{"EnformionGO possible match; imported " + now.UTC().Format(time.RFC3339) + ". Provider identity is not verified by LAS."}
	if p.DOB == "" && v.Dob != "" {
		notes = append(notes, "Provider-reported birth date: "+v.Dob)
	}
	for _, n := range v.Akas {
		alias := strings.TrimSpace(strings.Join([]string{n.FirstName, n.MiddleName, n.LastName, n.Suffix}, " "))
		if alias != "" {
			p.Aliases = append(p.Aliases, alias)
		}
	}
	for i, a := range v.Addresses {
		if i >= 10 {
			break
		}
		if a.FullAddress != "" {
			if p.Address == "" {
				p.Address = a.FullAddress
			}
			notes = append(notes, "Reported address: "+a.FullAddress)
		}
	}
	for i, ph := range v.PhoneNumbers {
		if i >= 10 {
			break
		}
		if len(lookupID(ph.PhoneNumber)) < 3 {
			continue
		}
		kind := "other"
		if strings.EqualFold(ph.PhoneType, "Wireless") {
			kind = "mobile"
		}
		p.Phones = append(p.Phones, cad.PersonPhone{Number: ph.PhoneNumber, Type: kind, Source: "records"})
	}
	for i, email := range v.EmailAddresses {
		if i >= 10 {
			break
		}
		if email.EmailAddress != "" {
			notes = append(notes, "Reported email: "+email.EmailAddress)
		}
	}
	p.Notes = strings.Join(notes, "\n")
	if lookupMinor(p.DOB, p.ApproxAge, now) {
		return cad.LASRecord{}, false
	}
	if err := cad.NormalizeLASPerson(&p); err != nil || p.Name == "" {
		return cad.LASRecord{}, false
	}
	return cad.LASRecord{Kind: "person", Agency: agency, ExternalID: v.TahoeID, Source: "EnformionGO", Person: &p, Name: p.Name, DOB: p.DOB, Aliases: p.Aliases}, true
}

// Serialize misses and recheck locally to avoid duplicate concurrent purchases.
// Imports are create-only; restricted/archived provider IDs are never exposed.
func (s *Server) enformionFallback(ctx context.Context, a cad.Actor, name, dob, agency string) ([]cad.LASRecord, string) {
	if !s.LAS || !enformionConfigured() {
		return nil, ""
	}
	s.enformionMu.Lock()
	defer s.enformionMu.Unlock()
	now := s.Eng.Now()
	existing := []cad.LASRecord{}
	known := map[string]bool{}
	s.Eng.Read(func(st *cad.State) {
		for _, r := range st.LASRecords {
			if r.Source == "EnformionGO" && r.Agency == agency {
				known[r.ExternalID] = true
			}
			if r.Archived || r.Restricted || r.Kind != "person" || (r.Agency != "" && !strings.EqualFold(r.Agency, agency)) {
				continue
			}
			p := cad.LASRecordPerson(r)
			if lookupMinor(p.DOB, p.ApproxAge, now) || !lookupNameMatches(lookupName(name), p) || (dob != "" && dob != p.DOB) {
				continue
			}
			existing = append(existing, r)
		}
	})
	if len(existing) > 0 {
		return existing, "Local records found; Enformion was not queried."
	}
	people, err := s.fetchEnformion(ctx, name, dob)
	if err != nil {
		return nil, err.Error()
	}
	if len(people) > 10 {
		return nil, "Enformion returned too many candidates; refine the name and birth date."
	}
	records := []cad.LASRecord{}
	for _, v := range people {
		if known[v.TahoeID] {
			continue
		}
		r, ok := enformionRecord(v, agency, now)
		if ok {
			records = append(records, r)
			known[v.TahoeID] = true
		}
	}
	if len(records) == 0 {
		return nil, "Enformion returned no eligible new person matches."
	}
	saved, _, err := s.Eng.ImportEnformionPeople(a, records)
	if err != nil {
		return nil, "Enformion results could not be saved; local records are unchanged."
	}
	s.Eng.Record(a, cad.EvSearched, map[string]any{"kind": "enformion-person", "results": len(saved), "cached": true})
	return saved, "Enformion matches saved to LAS records for future searches. Possible matches require identity confirmation."
}
