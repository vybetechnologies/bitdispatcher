package api

import (
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"

	"github.com/biodream-llc/bitdispatcher/internal/cad"
)

func TestRecordDetailsCSVAndMDT(t *testing.T) {
	var input strings.Builder
	writer := csv.NewWriter(&input)
	writer.Write([]string{"kind", "name", "plate", "source", "cautions", "warrants", "registeredSexOffender", "offenderProfileUrl", "ownership", "ownerName"})
	writer.Write([]string{"person", "Registry Fixture", "", "Fixture", `[{"type":"Weapons","level":"danger"}]`, `[{"number":"W1","agency":"Test court","offense":"Fixture","status":"active"}]`, "true", "https://registry.example/1", "", ""})
	writer.Write([]string{"vehicle", "", "TEST99", "Fixture", `[{"type":"Stolen plate","level":"danger"}]`, "", "", "", "employee", "Test Employee"})
	writer.Flush()
	rows, err := parseLASImport("csv", input.String())
	if err != nil {
		t.Fatal(err)
	}
	ts, s, pw := newServer(t)
	s.LAS = true
	s.lasRecordRoutes()
	s.mdtLookupRoutes()
	s.desktopRoutes()
	s.Eng.SaveUnit(cad.SystemActor, cad.UnitDef{ID: "SEC1", Type: "patrol", Agency: "LAS"})
	s.Auth.SetMDTUnit("admin", "SEC1")
	admin := login(t, ts, "admin", pw["admin"])
	for _, r := range rows {
		code, body := admin.do("POST", "/api/admin/las-records", r, true)
		if code != 201 {
			t.Fatal(code, string(body))
		}
	}
	for _, q := range []map[string]string{{"kind": "person", "name": "Registry Fixture", "purpose": "patrol"}, {"kind": "vehicle", "plate": "TEST99", "purpose": "patrol"}} {
		code, body := admin.do("POST", "/api/mdt/records/search", q, true)
		var result struct{ Items []mdtLookupHit }
		json.Unmarshal(body, &result)
		if code != 200 || len(result.Items) != 1 {
			t.Fatal(code, string(body))
		}
		hit := result.Items[0]
		if q["kind"] == "person" {
			if hit.Person == nil || !hit.Person.RegisteredSexOffender || len(hit.Person.Warrants) != 1 || len(hit.Person.Cautions) != 1 {
				t.Fatal("person details missing", string(body))
			}
		} else if hit.Ownership != "employee" || hit.OwnerName != "Test Employee" || len(hit.Cautions) != 1 {
			t.Fatal("vehicle details missing", string(body))
		}
	}
	code, body := admin.do("GET", "/api/mdt/offline/records", nil, true)
	var snapshot struct{ Records []mdtLookupHit }
	json.Unmarshal(body, &snapshot)
	if code != 200 || len(snapshot.Records) != 2 {
		t.Fatal("offline record snapshot missing", code, string(body))
	}
	for _, hit := range snapshot.Records {
		if hit.Person != nil {
			if !hit.Person.RegisteredSexOffender || len(hit.Person.Warrants) != 1 {
				t.Fatal("offline person details lost")
			}
		} else if hit.Ownership != "employee" || len(hit.Cautions) != 1 {
			t.Fatal("offline vehicle details lost")
		}
	}
}
