package auth

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"time"
)

// Persistent sessions are LAS's explicit operational policy. They retain the
// authenticated identity across restarts without storing credentials in a browser.
type savedSession struct {
	Token             string
	Session           Session
	WorkConsole       string
	MFAVerified       bool
	CredentialVersion string
}

func credentialVersion(u *User) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(u.Hash+"|"+u.TOTP)))
}

// EnablePersistentSessions restores only sessions belonging to current, enabled
// accounts with unchanged credentials. Normal duty and unit grants remain authoritative.
func (s *Store) EnablePersistentSessions() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.persistent {
		return nil
	}
	path := filepath.Join(filepath.Dir(s.path), "sessions.json")
	b, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil {
		var sessions []savedSession
		if err = json.Unmarshal(b, &sessions); err != nil {
			return fmt.Errorf("read persistent sessions: %w", err)
		}
		for _, saved := range sessions {
			se := saved.Session
			u := s.users[se.Username]
			if saved.Token == "" || u == nil || u.Disabled || saved.CredentialVersion != credentialVersion(u) {
				continue
			}
			if !slices.Contains(u.duties(), se.Duty) {
				se.Duty = u.defaultDuty()
			}
			se.Last, se.Active = s.now(), s.now()
			se.ID = saved.Token
			se.WorkConsole = saved.WorkConsole
			se.MFAVerified = saved.MFAVerified
			se.Name = u.Name
			se.Role = SessionRole(se.Duty, u.Role)
			se.MDTUnit = u.MDTUnit
			if se.Duty == DutyField {
				se.Console = u.MDTUnit
			}
			s.sessions[se.ID] = &se
		}
	}
	s.persistent = true
	return s.persistSessions()
}

// unlock persists identity/grant changes atomically. Request heartbeat timestamps
// are omitted from the comparison, avoiding a disk write for each poll or SSE read.
func (s *Store) unlock() {
	if s.persistent {
		if err := s.persistSessions(); err != nil {
			slog.Error("could not persist sessions", "error", err)
		}
	}
	s.mu.Unlock()
}
func (s *Store) persistSessions() error {
	items := []savedSession{}
	for token, se := range s.sessions {
		u := s.users[se.Username]
		if u == nil || u.Disabled {
			continue
		}
		cp := *se
		cp.Last = time.Time{}
		cp.Active = time.Time{}
		items = append(items, savedSession{Token: token, Session: cp, WorkConsole: se.WorkConsole, MFAVerified: se.MFAVerified, CredentialVersion: credentialVersion(u)})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Token < items[j].Token })
	b, err := json.Marshal(items)
	if err != nil {
		return err
	}
	if string(b) == s.sessionSnapshot {
		return nil
	}
	path := filepath.Join(filepath.Dir(s.path), "sessions.json")
	if err = os.WriteFile(path+".tmp", b, 0600); err != nil {
		return err
	}
	if err = os.Rename(path+".tmp", path); err != nil {
		return err
	}
	s.sessionSnapshot = string(b)
	return nil
}
