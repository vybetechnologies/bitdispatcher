package auth

import (
	"path/filepath"
	"testing"
	"time"
)

// The browser submits a valid code, receives the duty prompt, then submits the same
// code with the selected duty. Intermediate prompts must not consume that code.
func TestMFACodeSurvivesDutySelection(t *testing.T) {
	for _, enrolling := range []bool{false, true} {
		t.Run(map[bool]string{false: "enrolled", true: "enrollment"}[enrolling], func(t *testing.T) {
			s, err := Open(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			pw := GeneratePassword()
			if err = s.Save("owner", "Owner", Admin, pw, false); err != nil {
				t.Fatal(err)
			}
			now := time.Date(2026, 11, 15, 6, 0, 0, 0, time.UTC)
			s.now = func() time.Time { return now }
			s.SetPolicy("admins", 0, "LAS")
			secret := newSecret()
			if enrolling {
				s.users["owner"].PendingTOTP = secret
			} else {
				s.users["owner"].TOTP = secret
			}
			code := TOTP(secret, now.Unix()/30)
			r := Request{Username: "owner", Password: pw, Code: code}
			se, need, err := s.SignIn(r)
			if err != nil || se != nil || need == nil || len(need.Duties) == 0 {
				t.Fatal("expected duty prompt after valid MFA")
			}
			if s.users["owner"].TOTPStep != 0 {
				t.Fatal("intermediate duty prompt consumed code")
			}
			r.Duty = DutyAdmin
			se, need, err = s.SignIn(r)
			if err != nil || se == nil || need != nil {
				t.Fatal("valid code rejected when selecting duty")
			}
			if _, _, err = s.SignIn(r); err != ErrBadCode {
				t.Fatal("completed sign-in allowed replay")
			}
			// Replay protection must survive a process restart, not only the current session.
			reopened, err := Open(filepath.Dir(s.path))
			if err != nil {
				t.Fatal(err)
			}
			reopened.now = func() time.Time { return now }
			reopened.SetPolicy("admins", 0, "LAS")
			if _, _, err = reopened.SignIn(r); err != ErrBadCode {
				t.Fatal("restart lost replay protection")
			}
		})
	}
}

func TestAdministratorCanAlsoWorkFieldDuty(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	pw := GeneratePassword()
	if err = s.Save("chief", "Chief", Admin, pw, false); err != nil {
		t.Fatal(err)
	}
	if err = s.SetDuties("chief", []string{DutyAdmin, DutyField, DutySupervisor, DutyRecords}); err != nil {
		t.Fatal(err)
	}
	if s.UserRole("chief") != Admin {
		t.Fatal("MDT duty removed administrator access")
	}
	se, _, err := s.SignIn(Request{Username: "chief", Password: pw, Duty: DutyField, Console: "SEC1"})
	if err != nil || se == nil || se.Role != Field {
		t.Fatal("field session did not receive field-only rights")
	}
	se, _, err = s.SignIn(Request{Username: "chief", Password: pw, Duty: DutyAdmin})
	if err != nil || se == nil || se.Role != Admin {
		t.Fatal("administrator duty unavailable")
	}
}
