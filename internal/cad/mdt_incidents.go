package cad

import "strings"

// CreateMDTIncident validates and assigns only the authenticated officer's unit,
// under the same lock, so dispatch cannot reassign it between the two operations.
func (e *Engine) CreateMDTIncident(a Actor, unit string, n NewIncident) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	u := e.st.Units[unit]
	if u == nil {
		return "", problem("unit", "assigned unit is unavailable")
	}
	if !a.ClientStatusAt.IsZero() && (!u.StatusAt.Equal(a.ClientStatusAt) || u.Incident != a.ClientIncident) {
		return "", problem("offlineConflict", "dispatch changed this unit while offline; review the saved call")
	}
	if u.Incident != "" || u.Status != Available {
		return "", problem("unit", "your unit must be available without an assignment to create a call")
	}
	if err := e.st.mayCommand(a, u); err != nil {
		return "", err
	}
	if len(e.st.Staffing(u)) > 0 && u.StaffOverride == "" && (e.st.Settings.Staffing.Dispatch == "block" || e.st.Settings.Staffing.Dispatch == "override") {
		return "", problem("unit", "dispatch must resolve your unit's staffing before creating an assigned call")
	}
	if n.Location == nil || (!n.Location.HasPoint() && strings.TrimSpace(n.Location.Address) == "") {
		return "", problem("location", "enter a location or use device GPS")
	}
	if (n.Location.Lat == nil) != (n.Location.Lon == nil) {
		return "", problem("location", "provide both latitude and longitude")
	}
	if n.Location.HasPoint() && (*n.Location.Lat < -90 || *n.Location.Lat > 90 || *n.Location.Lon < -180 || *n.Location.Lon > 180) {
		return "", problem("location", "GPS coordinates are outside the valid range")
	}
	n.Agency = u.Agency
	n.Call = ""
	n.LocVerified = false
	n.Location.Source = "mdt"
	number, err := e.createIncidentLocked(a, n)
	if err != nil {
		return "", err
	}
	if err = e.dispatchLocked(a, number, []string{unit}, nil, "Officer-initiated incident", "", ""); err != nil {
		return number, err
	}
	return number, nil
}

func (e *Engine) CloseMDTIncident(a Actor, unit, number, disposition, text string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	u := e.st.Units[unit]
	if u == nil || number == "" || u.Incident != number {
		return problem("incident", "you can close only your currently assigned incident")
	}
	if err := e.st.mayCommand(a, u); err != nil {
		return err
	}
	in, err := e.openIncident(number)
	if err != nil {
		return err
	}
	for _, as := range in.Units {
		if as.Unit != unit && as.Cleared.IsZero() {
			return problem("incident", "other units are still assigned; dispatch must close this incident")
		}
	}
	return e.closeIncidentLocked(a, number, disposition, text)
}
