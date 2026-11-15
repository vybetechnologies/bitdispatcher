package cad

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/biodream-llc/bitdispatcher/internal/ledger"
)

func TestLASConfigurationPreservesOwnerSettingsAndReplays(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.jsonl")
	e, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	a := Actor{User: "owner", Role: "admin"}
	s := DefaultSettings()
	s.MapCenter = [2]float64{46.8, -96.8}
	s.IncidentPrefix = "KEEP"
	s.Security.MFA = "everyone"
	s.Next.Cameras.On = true
	if err = e.SaveSettings(a, s); err != nil {
		t.Fatal(err)
	}
	var seq uint64
	e.Read(func(st *State) { seq = st.Seq })
	if err = e.ConfigureLAS(a, seq); err != nil {
		t.Fatal(err)
	}
	e.Read(func(st *State) {
		if st.Settings.MapCenter != s.MapCenter || st.Settings.IncidentPrefix != "KEEP" || st.Settings.Security.MFA != "everyone" || !st.Settings.Next.Cameras.On {
			t.Error("owner settings changed")
		}
		if len(st.CallTypes) != len(LASCallTypes()) || st.CallTypes["SF"] != nil || st.CallTypes["PATROL"] == nil {
			t.Error("wrong LAS catalog")
		}
		if len(st.Units) != 0 || len(st.Stations) != 0 {
			t.Error("invented operational resources")
		}
	})
	e.Close()
	if r := ledger.Verify(path); !r.OK {
		t.Fatal(r.Problem)
	}
	e, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	e.Read(func(st *State) {
		if st.Settings.Agency != "Lakes Area Security" || st.CallTypes["PATROL"] == nil {
			t.Error("LAS config did not replay")
		}
	})
}

func TestLASConfigurationRefusesUnsafeBulkChanges(t *testing.T) {
	for _, what := range []string{"stale", "custom", "operational", "role"} {
		t.Run(what, func(t *testing.T) {
			e, err := Open(filepath.Join(t.TempDir(), "ledger.jsonl"))
			if err != nil {
				t.Fatal(err)
			}
			defer e.Close()
			a := Actor{User: "owner", Role: "admin"}
			if what == "custom" {
				err = e.SaveCallType(a, CallType{Code: "CUSTOM", Name: "Custom policy", Priority: 3, Discipline: "other"})
			}
			if what == "operational" {
				_, err = e.CreateIncident(a, NewIncident{Type: "OTHER"})
			}
			if err != nil {
				t.Fatal(err)
			}
			var seq uint64
			var before Settings
			e.Read(func(st *State) { seq = st.Seq; before = st.Settings })
			expect := seq
			if what == "stale" {
				expect--
			}
			if what == "role" {
				a.Role = "dispatcher"
			}
			if err = e.ConfigureLAS(a, expect); err == nil {
				t.Fatal("unsafe bulk change accepted")
			}
			e.Read(func(st *State) {
				if st.Seq != seq || !reflect.DeepEqual(st.Settings, before) {
					t.Error("refusal changed state")
				}
			})
		})
	}
}

func TestLASFreshDefaultsDoNotMigrateExistingLedger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.jsonl")
	e, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	e.Close()
	e, err = OpenWithDefaults(path, LASSettings(), LASCallTypes())
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	e.Read(func(st *State) {
		if st.Settings.Agency != "" || st.CallTypes["SF"] == nil {
			t.Error("startup migrated existing ledger")
		}
	})
	path = filepath.Join(t.TempDir(), "ledger.jsonl")
	fresh, err := OpenWithDefaults(path, LASSettings(), LASCallTypes())
	if err != nil {
		t.Fatal(err)
	}
	defer fresh.Close()
	fresh.Read(func(st *State) {
		if st.Settings.Shift.TimeZone != "America/Chicago" || fresh.st.CallTypes["SF"] != nil {
			t.Error("wrong LAS defaults")
		}
	})
}

func TestLASPreservesCompatibleUnitsAndRejectsIncompatibleTypes(t *testing.T) {
	for _, typ := range []string{"patrol", "engine"} {
		t.Run(typ, func(t *testing.T) {
			e, err := Open(filepath.Join(t.TempDir(), "ledger.jsonl"))
			if err != nil {
				t.Fatal(err)
			}
			defer e.Close()
			a := Actor{User: "owner", Role: "admin"}
			if err = e.SaveUnit(a, UnitDef{ID: "SEC1", Name: "Existing unit", Type: typ}); err != nil {
				t.Fatal(err)
			}
			var before Unit
			var seq uint64
			e.Read(func(st *State) { before = *st.Units["SEC1"]; seq = st.Seq })
			err = e.ConfigureLAS(a, seq)
			if typ == "patrol" && err != nil {
				t.Fatal(err)
			}
			if typ == "engine" && err == nil {
				t.Fatal("incompatible unit accepted")
			}
			e.Read(func(st *State) {
				if !reflect.DeepEqual(before, *st.Units["SEC1"]) {
					t.Error("existing unit changed")
				}
				if typ == "engine" && st.Seq != seq {
					t.Error("refusal changed ledger")
				}
			})
		})
	}
}
