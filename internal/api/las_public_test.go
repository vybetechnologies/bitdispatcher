package api

import (
	"encoding/json"
	"github.com/biodream-llc/bitdispatcher/internal/cad"
	"github.com/biodream-llc/bitdispatcher/internal/ng911"
	"strings"
	"testing"
	"time"
)

func TestLASAutomaticFeedPrivacyHistoryDelayAndWithholding(t *testing.T) {
	_, s, _ := newServer(t)
	s.LAS = true
	if err := s.Eng.SaveCallType(cad.SystemActor, cad.LASCallTypes()[0]); err != nil {
		t.Fatal(err)
	}
	id, err := s.Eng.CreateIncident(cad.SystemActor, cad.NewIncident{Type: "PATROL", Location: &ng911.Location{Address: "123 Main St APT 77, Private Client", Lat: ptr(46.87), Lon: ptr(-96.8)}, Narrative: "SECRET NARRATIVE"})
	if err != nil {
		t.Fatal(err)
	}
	if len(s.lasPublicDoc().Calls) != 0 {
		t.Fatal("delay ignored")
	}
	s.Eng.Read(func(st *cad.State) { st.Incidents[id].Created = time.Now().Add(-20 * time.Minute) })
	doc := s.lasPublicDoc()
	if len(doc.Calls) != 1 || doc.Calls[0].Block != "100 block Main St" || doc.Calls[0].Status != "Active" {
		t.Fatal("automatic call missing", doc)
	}
	encoded, _ := json.Marshal(doc)
	for _, secret := range []string{"123 Main", "APT", "Private Client", "SECRET NARRATIVE", "latitude", "longitude", "createdBy", "units", "caller", "narrative"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatal("private field leaked", secret)
		}
	}
	var seq uint64
	s.Eng.Read(func(st *cad.State) { seq = st.Seq })
	if err = s.Eng.SaveLASPublicCall(cad.Actor{User: "owner", Role: "admin"}, cad.LASPublicCall{Incident: id, Withheld: true}, seq); err != nil {
		t.Fatal(err)
	}
	if len(s.lasPublicDoc().Calls) != 0 {
		t.Fatal("withheld call published")
	}
	s.Eng.Read(func(st *cad.State) { delete(st.LASPublicCalls, id); st.Incidents[id].Status = cad.Closed })
	if s.lasPublicDoc().Calls[0].Status != "Closed" {
		t.Fatal("history not updated")
	}
	s.Eng.Read(func(st *cad.State) { st.Incidents[id].Type = "WELF" })
	if len(s.lasPublicDoc().Calls) != 0 {
		t.Fatal("sensitive type published")
	}
	s.Eng.Read(func(st *cad.State) {
		st.Incidents[id].Type = "PATROL"
		st.Incidents[id].Created = time.Now().AddDate(0, 0, -91)
	})
	if len(s.lasPublicDoc().Calls) != 0 {
		t.Fatal("retention ignored")
	}
}
func TestLASBlockWithholdsUnstructuredLocations(t *testing.T) {
	for _, a := range []string{"Private Client HQ", "123 Main St / SECRET", "Mason's home", "123 Main St #12 / confidential", "Main St & 2nd Ave"} {
		if lasPublicBlock(a) != "" {
			t.Fatal("unsafe location published", a)
		}
	}
}
