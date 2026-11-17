package api

import (
	"context"
	"github.com/biodream-llc/bitdispatcher/internal/cad"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestFBIWantedFiltersBroadMatchesAndSanitizesCautions(t *testing.T) {
	s := &Server{fbiHTTP: &http.Client{Transport: enformionTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "api.fbi.gov" || r.URL.Query().Get("title") != "TEST PERSON" || r.URL.Query().Get("dob") != "" {
			t.Fatal("unexpected FBI request")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"total":4,"items":[{"uid":"valid","title":"TEST PERSON","url":"https://www.fbi.gov/wanted/test-person","dates_of_birth_used":["January 2, 1980"],"caution":"<p>Published caution &amp; instruction</p>"},{"uid":"wrong-name","title":"ANOTHER PERSON","url":"https://www.fbi.gov/wanted/another"},{"uid":"wrong-dob","title":"TEST PERSON","url":"https://www.fbi.gov/wanted/test","dates_of_birth_used":["January 3, 1980"]},{"uid":"captured","title":"TEST PERSON","status":"captured","url":"https://www.fbi.gov/wanted/captured"}]}`))}, nil
	})}}
	r, err := s.searchFBI(context.Background(), "TEST PERSON", "1980-01-02")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Items) != 1 || r.Items[0].UID != "valid" || r.Items[0].Caution != "Published caution & instruction" {
		t.Fatalf("incorrect possible matches: %+v", r.Items)
	}
	if officialFBIPage("https://www.fbi.gov.evil.test/x") || officialFBIPage("javascript:alert(1)") {
		t.Fatal("non-FBI notice accepted")
	}
}
func TestFBIUnavailableDoesNotBreakLocalLookup(t *testing.T) {
	ts, s, pw := newServer(t)
	s.LAS = true
	s.mdtLookupRoutes()
	if err := s.Eng.SaveUnit(cad.SystemActor, cad.UnitDef{ID: "FBI-TEST", Type: "patrol"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Auth.SetMDTUnit("admin", "FBI-TEST"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Eng.SaveLASRecords(cad.Actor{User: "admin", Role: "admin"}, []cad.LASRecord{{Kind: "person", Name: "Test Person", Source: "Test"}}, false); err != nil {
		t.Fatal(err)
	}
	s.fbiHTTP = &http.Client{Transport: enformionTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 503, Body: io.NopCloser(strings.NewReader("unavailable"))}, nil
	})}
	client := login(t, ts, "admin", pw["admin"])
	q := map[string]string{"kind": "person", "name": "Test Person", "purpose": "patrol"}
	if code, _ := client.do("POST", "/api/mdt/records/fbi", q, true); code != 502 {
		t.Fatal("FBI failure not explicit", code)
	}
	code, body := client.do("POST", "/api/mdt/records/search", q, true)
	if code != 200 || !strings.Contains(string(body), "Test Person") {
		t.Fatal("local lookup affected by FBI failure", code, string(body))
	}
	s.Eng.Read(func(st *cad.State) {
		if len(st.LASRecords) != 1 {
			t.Fatal("FBI search changed the local record store")
		}
	})
}
