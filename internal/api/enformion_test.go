package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/biodream-llc/bitdispatcher/internal/cad"
)

type enformionTransport func(*http.Request) (*http.Response, error)

func (f enformionTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestEnformionLocalFirstAndCreateOnly(t *testing.T) {
	_, s, _ := newServer(t)
	s.LAS = true
	t.Setenv("ENFORMION_AP_NAME", "fixture-name")
	t.Setenv("ENFORMION_AP_PASSWORD", "fixture-password")
	t.Setenv("ENFORMION_STORAGE_AUTHORIZED", "true")
	calls := 0
	s.enformionHTTP = &http.Client{Transport: enformionTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.String() != enformionEndpoint || r.Header.Get("galaxy-ap-password") != "fixture-password" || r.Header.Get("galaxy-search-type") != "Person" {
			t.Fatal("incorrect provider request")
		}
		var b map[string]any
		json.NewDecoder(r.Body).Decode(&b)
		if b["FirstName"] != "Alex" || b["LastName"] != "Sample" {
			t.Fatal(b)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"TahoeId":"fixture-1","Name":{"FirstName":"Alex","LastName":"Sample"},"Dob":"2/3/1982","Age":44,"Addresses":[{"FullAddress":"100 Synthetic St"}],"PhoneNumbers":[{"PhoneNumber":"5550000000","PhoneType":"Wireless"}]}`))}, nil
	})}
	actor := cad.Actor{User: "officer", Role: "field"}
	records, message := s.enformionFallback(context.Background(), actor, "Alex Sample", "1982-02-03", "LAS")
	if len(records) != 1 || records[0].Person.Address != "100 Synthetic St" || records[0].By != "officer" || records[0].Source != "EnformionGO" || records[0].ExternalID != "fixture-1" {
		t.Fatal(records, message)
	}
	id := records[0].ID
	records, message = s.enformionFallback(context.Background(), actor, "Sample, Alex", "1982-02-03", "LAS")
	if len(records) != 1 || records[0].ID != id || calls != 1 {
		t.Fatal("did not reuse local record", calls, message)
	}
	// Normal field users still cannot edit arbitrary records.
	if _, _, err := s.Eng.SaveLASRecords(actor, records, false); err == nil {
		t.Fatal("field user gained administrative access")
	}
	bad := records[0]
	bad.ID = ""
	bad.Person.Safety = true
	if _, _, err := s.Eng.ImportEnformionPeople(actor, []cad.LASRecord{bad}); err == nil {
		t.Fatal("provider created safety flag")
	}
}
func TestEnformionFailureAndPartialBirthDate(t *testing.T) {
	now := time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)
	r, ok := enformionRecord(enformionPerson{TahoeID: "fixture", FullName: "Alex Sample", Dob: "2/1982", Age: 44}, "LAS", now)
	if !ok || r.DOB != "" || !strings.Contains(r.Person.Notes, "2/1982") {
		t.Fatal("invented full DOB", r)
	}
	if _, err := enformionRequest("Alex", ""); err == nil {
		t.Fatal("partial name charged")
	}
	if _, err := decodeEnformion([]byte(`{"Error":"fixture-password"}`)); err == nil || strings.Contains(err.Error(), "fixture-password") {
		t.Fatal("provider error interpreted as no-match or disclosed")
	}
	people, err := decodeEnformion([]byte(`{"Persons":[]}`))
	if err != nil || len(people) != 0 {
		t.Fatal(err)
	}
	if _, ok := enformionRecord(enformionPerson{TahoeID: "optedout", FullName: "Alex Sample", IsOptedOut: true}, "LAS", now); ok {
		t.Fatal("opted out person imported")
	}
}
