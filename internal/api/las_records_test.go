package api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/biodream-llc/bitdispatcher/internal/auth"
	"github.com/biodream-llc/bitdispatcher/internal/cad"
)

func TestLASRecordAdminImportAndMDTSearch(t *testing.T) {
	ts, s, pw := newServer(t)
	s.LAS = true
	s.lasRecordRoutes()
	s.mdtLookupRoutes()
	s.Eng.SaveUnit(cad.SystemActor, cad.UnitDef{ID: "SEC1", Type: "patrol", Agency: "LAS"})
	s.Auth.SetMDTUnit("admin", "SEC1")
	admin := login(t, ts, "admin", pw["admin"])
	viewer := login(t, ts, "v1", pw["v1"])
	if code, _ := viewer.do("GET", "/api/admin/las-records", nil, true); code != 403 {
		t.Fatal("nonadmin directory", code)
	}
	person := cad.LASRecord{Kind: "person", Name: "Fixture Adult", DOB: "1980-01-02", Aliases: []string{"FixtureAlias"}, Source: "Test fixture", Description: "Blue coat"}
	if code, _ := admin.do("POST", "/api/admin/las-records", person, false); code != 403 {
		t.Fatal("CSRF bypass", code)
	}
	code, body := admin.do("POST", "/api/admin/las-records", person, true)
	if code != 201 {
		t.Fatal(code, string(body))
	}
	var saved cad.LASRecord
	json.Unmarshal(body, &saved)
	if saved.ID == "" || saved.Version != 1 || saved.By != "admin" {
		t.Fatal(string(body))
	}
	content := "kind,plate,state,source,externalId\nvehicle,XYZ987,ND,Fixture,veh-1\nvehicle,XYZ987,ND,Fixture,veh-1\n"
	request := map[string]any{"format": "csv", "content": content, "commit": false}
	code, body = admin.do("POST", "/api/admin/las-records/import", request, true)
	if code != 200 || !strings.Contains(string(body), `"create":1`) || !strings.Contains(string(body), `"skipped":1`) {
		t.Fatal(code, string(body))
	}
	var count int
	s.Eng.Read(func(st *cad.State) { count = len(st.LASRecords) })
	if count != 1 {
		t.Fatal("preview wrote records")
	}
	request["commit"] = true
	code, body = admin.do("POST", "/api/admin/las-records/import", request, true)
	if code != 200 || !strings.Contains(string(body), `"created":1`) {
		t.Fatal(code, string(body))
	}
	code, body = admin.do("POST", "/api/admin/las-records/import", request, true)
	if code != 200 || !strings.Contains(string(body), `"created":0`) {
		t.Fatal("import not idempotent", code, string(body))
	}
	request["content"] = "kind,plate,state,source\nvehicle,NEW555,ND,Fixture\nvehicle,,,Fixture\n"
	if code, _ := admin.do("POST", "/api/admin/las-records/import", request, true); code != 422 {
		t.Fatal("bad import accepted", code)
	}
	s.Eng.Read(func(st *cad.State) {
		if len(st.LASRecords) != 2 {
			t.Fatal("partial import wrote records")
		}
	})
	search := map[string]any{"kind": "person", "name": "FixtureAlias", "purpose": "patrol"}
	code, body = admin.do("POST", "/api/mdt/records/search", search, true)
	if code != 200 || !strings.Contains(string(body), `"source":"LAS directory"`) || !strings.Contains(string(body), "Blue coat") {
		t.Fatal(code, string(body))
	}
	// A stale edit cannot overwrite a newer change; archives stop officer discovery.
	stale := saved
	saved.Archived = true
	code, body = admin.do("PUT", "/api/admin/las-records/"+saved.ID, saved, true)
	if code != 200 {
		t.Fatal(code, string(body))
	}
	if code, _ := admin.do("PUT", "/api/admin/las-records/"+saved.ID, stale, true); code != 409 {
		t.Fatal("stale overwrite", code)
	}
	code, body = admin.do("POST", "/api/mdt/records/search", search, true)
	if code != 200 || !strings.Contains(string(body), `"items":[]`) {
		t.Fatal("archived visible", string(body))
	}
	// Restriction, agency and underage exclusions apply even to an owner's MDT.
	for _, r := range []cad.LASRecord{{Kind: "person", Name: "Hidden Fixture", DOB: "1980-01-01", Source: "Restricted", Restricted: true}, {Kind: "person", Name: "Hidden Fixture", DOB: "2018-01-01", Source: "Minor"}, {Kind: "person", Name: "Hidden Fixture", DOB: "1982-01-01", Source: "Other agency", Agency: "OTHER"}} {
		if code, _ := admin.do("POST", "/api/admin/las-records", r, true); code != 201 {
			t.Fatal(code)
		}
	}
	search["name"] = "Hidden Fixture"
	code, body = admin.do("POST", "/api/mdt/records/search", search, true)
	if code != 200 || !strings.Contains(string(body), `"items":[]`) {
		t.Fatal("restricted record leaked", string(body))
	}
	// Switch to the existing Field duty: officers can search, but cannot manage records.
	s.Auth.SetDuties("admin", []string{auth.DutyAdmin, auth.DutyField})
	s.moduleRoutes()
	if code, body = admin.do("POST", "/api/auth/module", map[string]string{"module": "mdt"}, true); code != 200 {
		t.Fatal(code, string(body))
	}
	if code, _ := admin.do("POST", "/api/admin/las-records", person, true); code != 403 {
		t.Fatal("field created directory record", code)
	}
	code, body = admin.do("POST", "/api/mdt/records/search", map[string]string{"kind": "vehicle", "plate": "XYZ98", "purpose": "patrol"}, true)
	if code != 200 || !strings.Contains(string(body), "XYZ987") {
		t.Fatal("field vehicle lookup", code, string(body))
	}
}
func TestLASImportRejectsUnknownFieldsAndInvalidIdentifiers(t *testing.T) {
	for _, v := range []struct{ format, content string }{{"json", `[{"kind":"person","name":"Test","source":"Fixture","ssn":"private"}]`}, {"csv", "kind,name,source,extra\nperson,Test,Fixture,data\n"}, {"json", `[{"kind":"vehicle","source":"Fixture","vin":"INVALID"}]`}, {"json", `[{"kind":"person","name":"Test","source":"Fixture","dob":"2099-01-01"}]`}} {
		if _, err := parseLASImport(v.format, v.content); err == nil {
			t.Fatal("invalid import accepted", v.content)
		}
	}
}
