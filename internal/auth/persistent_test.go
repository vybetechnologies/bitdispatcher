package auth

import (
	"testing"
	"time"
)

func TestLASPersistentSessionsAcrossTimeRestartAndLogout(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.EnablePersistentSessions(); err != nil {
		t.Fatal(err)
	}
	pw := GeneratePassword()
	for _, role := range []string{Admin, Dispatcher, Supervisor, Field, Viewer} {
		if err = s.Save("user-"+role, role, role, pw, false); err != nil {
			t.Fatal(err)
		}
	}
	s.SetPolicy("", time.Minute, "LAS")
	tokens := []string{}
	for _, u := range s.List() {
		se, err := s.Login(u.Username, pw, "WEB", "")
		if err != nil {
			t.Fatal(err)
		}
		tokens = append(tokens, se.ID)
	}
	future := time.Now().AddDate(20, 0, 0)
	s.now = func() time.Time { return future }
	for _, id := range tokens {
		if se, ok := s.Session(id); !ok || se.Locked {
			t.Fatal("session expired or automatically locked", id)
		}
	}
	s.mu.Lock()
	s.sessions[tokens[0]].MFAVerified = true
	s.unlock()
	restored, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err = restored.EnablePersistentSessions(); err != nil {
		t.Fatal(err)
	}
	restored.now = func() time.Time { return future }
	if len(restored.Active()) != len(tokens) {
		t.Fatal("active sessions lost")
	}
	for _, id := range tokens {
		if _, ok := restored.Session(id); !ok {
			t.Fatal("session lost on restart")
		}
	}
	if !restored.sessions[tokens[0]].MFAVerified {
		t.Fatal("MFA state lost")
	}
	restored.Logout(tokens[0])
	again, _ := Open(dir)
	if err = again.EnablePersistentSessions(); err != nil {
		t.Fatal(err)
	}
	if _, ok := again.Session(tokens[0]); ok {
		t.Fatal("manual signout did not survive restart")
	}
	// Explicit account disabling also revokes its persistent sessions.
	se, _ := again.Session(tokens[1])
	if err = again.Save(se.Username, se.Name, se.Role, "", true); err != nil {
		t.Fatal(err)
	}
	final, _ := Open(dir)
	if err = final.EnablePersistentSessions(); err != nil {
		t.Fatal(err)
	}
	if _, ok := final.Session(tokens[1]); ok {
		t.Fatal("disabled account restored")
	}
}
