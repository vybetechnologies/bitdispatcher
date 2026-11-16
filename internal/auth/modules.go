package auth

import (
	"errors"
	"slices"
	"strings"
)

// ModuleAccess returns only the signed-in account's current grants, never cached session rights.
func (s *Store) ModuleAccess(username string) (Public, bool) {
	s.mu.Lock()
	defer s.unlock()
	u := s.users[strings.ToLower(username)]
	if u == nil || u.Disabled {
		return Public{}, false
	}
	return Public{Username: u.Username, Role: u.Role, Duties: slices.Clone(u.duties()), MDTUnit: u.MDTUnit, MFA: u.TOTP != ""}, true
}

// SwitchDuty changes work within the same authenticated identity. It cannot grant a new duty.
func (s *Store) SwitchDuty(id, duty string) (*Session, error) {
	s.mu.Lock()
	defer s.unlock()
	se := s.sessions[id]
	if se == nil {
		return nil, errors.New("sign in again")
	}
	if se.Locked {
		return nil, errors.New("unlock the console before switching modules")
	}
	u := s.users[se.Username]
	if u == nil || u.Disabled {
		return nil, errors.New("sign in again")
	}
	duty = strings.ToLower(strings.TrimSpace(duty))
	if !slices.Contains(u.duties(), duty) {
		return nil, errors.New("this duty is not assigned to your account")
	}
	role := SessionRole(duty, u.Role)
	if AtLeast(role, Supervisor) && s.mfaRequired(u) && !se.MFAVerified {
		return nil, errors.New("sign in again with your authenticator before opening this module")
	}
	if duty == DutyField && u.MDTUnit == "" {
		return nil, errors.New("an administrator must assign your MDT unit first")
	}
	if se.WorkConsole == "" {
		se.WorkConsole = "WEB-" + strings.ToUpper(se.Hash)
	}
	se.Duty, se.Role, se.MDTUnit = duty, role, u.MDTUnit
	if duty == DutyField {
		se.Console = u.MDTUnit
	} else {
		se.Console = se.WorkConsole
	}
	se.Active = s.now()
	cp := *se
	return &cp, nil
}
