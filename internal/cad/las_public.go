package cad

import "time"

const EvLASPublicCall = "las.public-call.visibility"

// LASPublicCall is an optional confidentiality exclusion. Eligible calls otherwise publish automatically.
type LASPublicCall struct {
	Incident string    `json:"incident"`
	Withheld bool      `json:"withheld"`
	By       string    `json:"by"`
	At       time.Time `json:"at"`
}

func (e *Engine) SaveLASPublicCall(a Actor, visibility LASPublicCall, expected uint64) error {
	if a.Role != "admin" {
		return problem("role", "an administrator changes public visibility")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.st.Seq != expected {
		return problem("seq", "the system changed: reload before saving")
	}
	if e.st.Incidents[visibility.Incident] == nil {
		return problem("incident", "no such incident")
	}
	visibility.By, visibility.At = a.User, e.Now()
	_, err := e.write(a, EvLASPublicCall, refs{Incident: visibility.Incident}, visibility)
	return err
}
