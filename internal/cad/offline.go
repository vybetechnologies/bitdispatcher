package cad

import (
	"encoding/json"
	"github.com/biodream-llc/bitdispatcher/internal/ledger"
	"time"
)

type OfflineCommandResult struct {
	Hash, Type, Number, ID, Key string
	At, Updated                 time.Time
}

func (s *State) rememberOfflineCommand(ev ledger.Event) {
	if s.clientCommands == nil {
		s.clientCommands = map[string]OfflineCommandResult{}
	}
	if _, exists := s.clientCommands[ev.Actor+"|"+ev.ClientCommand]; exists {
		return
	}
	var ids struct {
		Number, ID, Key   string
		Updated, StatusAt time.Time
	}
	_ = json.Unmarshal(ev.Data, &ids)
	at := ev.At
	if !ids.StatusAt.IsZero() {
		at = ids.StatusAt
	}
	s.clientCommands[ev.Actor+"|"+ev.ClientCommand] = OfflineCommandResult{At: at, Updated: ids.Updated, Hash: ev.ClientHash, Type: ev.Type, Number: ids.Number, ID: ids.ID, Key: ids.Key}
}
func (e *Engine) OfflineCommandResult(user, id string) (OfflineCommandResult, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	result, ok := e.st.clientCommands[user+"|"+id]
	return result, ok
}
