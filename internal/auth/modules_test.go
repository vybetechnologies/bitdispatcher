package auth

import "testing"

func TestDutySwitchPreservesIdentityAndRestrictsGrants(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	pw := GeneratePassword()
	s.Save("owner", "Owner", Admin, pw, false)
	s.SetDuties("owner", []string{DutyAdmin, DutyDispatcher, DutyField})
	s.SetMDTUnit("owner", "SEC1")
	se, _, err := s.SignIn(Request{Username: "owner", Password: pw, Duty: DutyAdmin, Console: "DESK1"})
	if err != nil {
		t.Fatal(err)
	}
	field, err := s.SwitchDuty(se.ID, DutyField)
	if err != nil {
		t.Fatal(err)
	}
	if field.Role != Field || field.Console != "SEC1" || field.Username != se.Username || field.ID != se.ID {
		t.Fatal("incorrect field transition", field)
	}
	admin, err := s.SwitchDuty(se.ID, DutyAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if admin.Role != Admin || admin.Console != "DESK1" {
		t.Fatal("desk not restored", admin)
	}
	if _, err = s.SwitchDuty(se.ID, DutyFire); err == nil {
		t.Fatal("unassigned duty granted")
	}
	s.Lock(se.ID)
	if _, err = s.SwitchDuty(se.ID, DutyField); err == nil {
		t.Fatal("locked switch accepted")
	}
}
func TestDutySwitchRequiresVerifiedMFAForElevatedWork(t *testing.T) {
	s, _ := Open(t.TempDir())
	pw := GeneratePassword()
	s.Save("owner", "Owner", Admin, pw, false)
	s.SetDuties("owner", []string{DutyAdmin, DutyField})
	s.SetMDTUnit("owner", "SEC1")
	se, _, err := s.SignIn(Request{Username: "owner", Password: pw, Duty: DutyField, Console: "DESK1"})
	if err != nil {
		t.Fatal(err)
	}
	s.SetPolicy("admins", 0, "LAS")
	if _, err = s.SwitchDuty(se.ID, DutyAdmin); err == nil {
		t.Fatal("unverified session elevated under MFA policy")
	}
	s.mu.Lock()
	s.sessions[se.ID].MFAVerified = true
	s.mu.Unlock()
	if _, err = s.SwitchDuty(se.ID, DutyAdmin); err != nil {
		t.Fatal(err)
	}
}
