package auth

import (
	"errors"
	"strings"
)

// MDTUnitFor is looked up on every request so a revoked assignment takes effect immediately.
func (s *Store) MDTUnitFor(username string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u := s.users[strings.ToLower(username)]; u != nil && !u.Disabled {
		return u.MDTUnit
	}
	return ""
}

// SetMDTUnit binds a named account to one terminal unit without changing duties or credentials.
func (s *Store) SetMDTUnit(username, unit string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	username = strings.ToLower(strings.TrimSpace(username))
	unit = strings.ToUpper(strings.TrimSpace(unit))
	u := s.users[username]
	if u == nil {
		return errors.New("no such user")
	}
	for name, other := range s.users {
		if unit != "" && name != username && other.MDTUnit == unit {
			return errors.New("that MDT unit is already assigned to another account")
		}
	}
	before := u.MDTUnit
	u.MDTUnit = unit
	if err := s.save(); err != nil {
		u.MDTUnit = before
		return err
	}
	for _, se := range s.sessions {
		if se.Username == username {
			se.MDTUnit = unit
			if se.Role == Field {
				se.Console = unit
			}
		}
	}
	return nil
}
