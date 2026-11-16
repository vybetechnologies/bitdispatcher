package cad

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/biodream-llc/bitdispatcher/internal/ledger"
)

const EvLASRecords = "las.records.saved"

type LASRecord struct {
	ID          string    `json:"id"`
	Version     int       `json:"version"`
	Kind        string    `json:"kind"`
	Agency      string    `json:"agency,omitempty"`
	ExternalID  string    `json:"externalId,omitempty"`
	Name        string    `json:"name,omitempty"`
	Aliases     []string  `json:"aliases,omitempty"`
	DOB         string    `json:"dob,omitempty"`
	Sex         string    `json:"sex,omitempty"`
	Description string    `json:"description,omitempty"`
	Plate       string    `json:"plate,omitempty"`
	State       string    `json:"state,omitempty"`
	VIN         string    `json:"vin,omitempty"`
	Year        string    `json:"year,omitempty"`
	Make        string    `json:"make,omitempty"`
	Model       string    `json:"model,omitempty"`
	Color       string    `json:"color,omitempty"`
	Body        string    `json:"body,omitempty"`
	Source      string    `json:"source"`
	Safety      bool      `json:"safetyConcern"`
	Restricted  bool      `json:"restricted"`
	Archived    bool      `json:"archived"`
	Updated     time.Time `json:"updated"`
	By          string    `json:"by"`
}

func LASRecordKey(r LASRecord) string {
	prefix := strings.ToUpper(strings.TrimSpace(r.Agency)) + ":" + r.Kind + ":"
	if r.ExternalID != "" {
		return prefix + "external:" + strings.ToUpper(r.Source) + ":" + strings.ToUpper(r.ExternalID)
	}
	if r.Kind == "vehicle" {
		if r.VIN != "" {
			return prefix + "vin:" + strings.ToUpper(r.VIN)
		}
		return prefix + "plate:" + strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(r.Plate, " ", ""), "-", "")) + ":" + r.State
	}
	if r.DOB != "" {
		return prefix + "person:" + strings.ToUpper(strings.Join(strings.Fields(r.Name), " ")) + ":" + r.DOB
	}
	r.ID = ""
	r.Version = 0
	r.By = ""
	r.Updated = time.Time{}
	r.Archived = false
	b, _ := json.Marshal(r)
	return prefix + string(b)
}

// SaveLASRecords validates the whole batch before writing one attributed ledger
// event. Existing records require their exact version; imports never overwrite.
func (e *Engine) SaveLASRecords(a Actor, records []LASRecord, importing bool) ([]LASRecord, int, error) {
	if a.Role != "admin" {
		return nil, 0, fmt.Errorf("an administrator manages LAS records")
	}
	if len(records) == 0 || len(records) > 500 {
		return nil, 0, fmt.Errorf("save between 1 and 500 records")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	keys := map[string]string{}
	for id, r := range e.st.LASRecords {
		keys[LASRecordKey(r)] = id
	}
	next := []LASRecord{}
	skipped := 0
	for _, r := range records {
		if r.ID != "" {
			old, ok := e.st.LASRecords[r.ID]
			if !ok || old.Version != r.Version {
				return nil, 0, fmt.Errorf("record changed; reload before saving")
			}
			if importing {
				return nil, 0, fmt.Errorf("imports create records only; edit existing records separately")
			}
			delete(keys, LASRecordKey(old))
			r.Version++
		} else {
			r.Version = 1
		}
		key := LASRecordKey(r)
		if _, ok := keys[key]; ok {
			if importing {
				skipped++
				continue
			}
			return nil, 0, fmt.Errorf("a matching record already exists; edit that record")
		}
		if r.ID == "" {
			r.ID = fmt.Sprintf("LASR-%d-%d", e.st.Seq+1, len(next)+1)
		}
		r.Updated = e.now()
		r.By = a.User
		next = append(next, r)
		keys[key] = r.ID
	}
	if len(next) == 0 {
		return next, skipped, nil
	}
	_, err := e.write(a, EvLASRecords, refs{}, next)
	return next, skipped, err
}
func (s *State) applyLASRecords(ev ledger.Event) error {
	var records []LASRecord
	if err := json.Unmarshal(ev.Data, &records); err != nil {
		return err
	}
	if s.LASRecords == nil {
		s.LASRecords = map[string]LASRecord{}
	}
	for _, r := range records {
		s.LASRecords[r.ID] = r
	}
	return nil
}
