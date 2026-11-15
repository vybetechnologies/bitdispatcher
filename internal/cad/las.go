package cad

import (
	"reflect"
	"sort"
)

// LASSettings is the North Dakota private-security starting configuration. Internal police
// discipline/capability keys are retained for compatibility, not as a claim of police authority.
func LASSettings() Settings {
	s := DefaultSettings()
	s.Agency = "Lakes Area Security"
	s.IncidentPrefix = "LAS"
	s.MapCenter = [2]float64{46.8772, -96.7898}
	s.Shift.TimeZone = "America/Chicago"
	s.SignInNotice = "AUTHORIZED USE ONLY\nLakes Area Security Corporation\nPrivate security operations. Access and activity are recorded."
	s.ORI, s.NCICMode, s.NCICAddr = "", "off", ""
	s.Next.LocLinkText = "Lakes Area Security: to share your location with our staff, open this link and press Share: {link}"
	s.Dispositions = []string{"Handled", "Report taken", "No issue found", "Unable to locate", "Client notified", "Referred to public responders", "Cancelled", "False alarm", "Duplicate", "Follow-up required"}
	s.Qualifications = []string{"security-officer", "site-supervisor", "first-aid"}
	s.CapNeeds = []CapNeed{}
	s.UnitTypes = []UnitType{
		{Code: "patrol", Name: "Security patrol", Discipline: "police", Capabilities: []string{"police", "security", "patrol"}},
		{Code: "site", Name: "Site security", Discipline: "police", Capabilities: []string{"police", "security", "site"}},
		{Code: "security-supervisor", Name: "Security supervisor", Discipline: "police", Capabilities: []string{"police", "security", "supervisor"}},
	}
	s.MarkOuts = []MarkOutType{
		{Code: "MEAL", Name: "Meal break", Minutes: 30, Interruptible: true},
		{Code: "FUEL", Name: "Fuel", Minutes: 15, Interruptible: true},
		{Code: "RPT", Name: "Report writing", Minutes: 45, Interruptible: true},
		{Code: "FU", Name: "Client follow-up", Minutes: 30, Interruptible: true},
		{Code: "SEC", Name: "Property / business check", Minutes: 15, Interruptible: true},
		{Code: "PTL", Name: "Directed patrol", Minutes: 30, Interruptible: true},
		{Code: "ADMIN", Name: "Administrative", Minutes: 30, Interruptible: true},
		{Code: "TRN", Name: "Training", Minutes: 120},
		{Code: "MAINT", Name: "Vehicle maintenance", Minutes: 60},
	}
	return s
}

// LASCallTypes are reviewable starting run cards, not a promise of staffing or response capacity.
func LASCallTypes() []CallType {
	type entry struct {
		code, name, group string
		priority, units   int
	}
	entries := []entry{
		{"PATROL", "Scheduled patrol / property check", "Patrol", 4, 1},
		{"SITE", "Site security assignment", "Site security", 4, 1},
		{"ALARM", "Alarm response / verification", "Response", 2, 1},
		{"ACCESS", "Access control / unauthorized entry", "Access", 3, 1},
		{"LOCKUP", "Property lockup", "Access", 4, 1},
		{"UNLOCK", "Property unlock", "Access", 4, 1},
		{"LOCKOUT", "Authorized lockout assistance", "Access", 3, 1},
		{"SUSP", "Suspicious activity", "Response", 3, 1},
		{"DIST", "Disturbance / conflict", "Response", 2, 2},
		{"TRESPASS", "Trespass / property-rule concern", "Response", 3, 1},
		{"DAMAGE", "Property damage / vandalism", "Reports", 3, 1},
		{"THEFT", "Theft / loss report", "Reports", 3, 1},
		{"ESCORT", "Safety escort", "Client assistance", 3, 1},
		{"WELF", "Requested welfare / safety check", "Client assistance", 2, 1},
		{"CLIENT", "Client / tenant assistance", "Client assistance", 3, 1},
		{"EVENT", "Event security / standby", "Events", 4, 0},
		{"FOLLOWUP", "Incident follow-up", "Reports", 4, 1},
		{"EMERG", "Emergency reported — contact public responders", "Referral", 1, 0},
		{"OTHER", "Other / administrative", "Administrative", 4, 0},
	}
	out := make([]CallType, 0, len(entries))
	for _, x := range entries {
		c := CallType{Code: x.code, Name: x.name, Group: x.group, Priority: x.priority, Discipline: "police"}
		if x.units > 0 {
			c.RunCard = []Requirement{{Capability: "security", Count: x.units}}
		}
		out = append(out, c)
	}
	return out
}

// ConfigureLAS changes an unused pilot through ordinary, hash-chained configuration events.
// It never rewrites history. Active systems and custom catalogs require individual reviewed changes.
// A disk failure can leave a partial application; retries recognize both starting catalogs.
func (e *Engine) ConfigureLAS(a Actor, expected uint64) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if a.Role != "admin" {
		return problem("role", "an administrator configures Lakes Area Security")
	}
	if e.st.Seq != expected {
		return problem("seq", "the system changed: reload the preview before applying")
	}
	if len(e.st.Incidents) > 0 || len(e.st.Reports) > 0 || len(e.st.Calls) > 0 || len(e.st.Plans) > 0 {
		return problem("profile", "this system has operational records, units or response plans: review individual configuration changes instead")
	}
	// Configured units are preserved when their existing types remain valid under LAS.
	for _, u := range e.st.Units {
		compatible := false
		for _, typ := range LASSettings().UnitTypes {
			if u.Type == typ.Code {
				compatible = true
			}
		}
		if !compatible {
			return problem("units", "unit %s has type %s: review its type before applying LAS defaults", u.ID, u.Type)
		}
	}
	known := map[string][]CallType{}
	for _, c := range append(DefaultCallTypes(), LASCallTypes()...) {
		known[c.Code] = append(known[c.Code], c)
	}
	for code, c := range e.st.CallTypes {
		ok := false
		for _, k := range known[code] {
			if reflect.DeepEqual(*c, k) {
				ok = true
			}
		}
		if !ok {
			return problem("calltypes", "custom call type %s requires individual review; no configuration changed", code)
		}
	}
	s := e.st.Settings
	las := LASSettings()
	s.Agency, s.UnitTypes, s.MarkOuts, s.Dispositions = las.Agency, las.UnitTypes, las.MarkOuts, las.Dispositions
	s.Qualifications, s.CapNeeds = las.Qualifications, las.CapNeeds
	s.ORI, s.NCICMode, s.NCICAddr = "", "off", ""
	s.Shift.TimeZone = las.Shift.TimeZone
	s.Next.LocLinkText = las.Next.LocLinkText
	// Keep existing map, incident numbering, security/MFA, GIS, cameras and other owner settings.
	if s.SignInNotice == "" {
		s.SignInNotice = las.SignInNotice
	}
	if _, err := e.write(a, EvSettings, refs{}, s); err != nil {
		return err
	}
	keep := map[string]bool{}
	for _, c := range LASCallTypes() {
		keep[c.Code] = true
	}
	var remove []string
	for code := range e.st.CallTypes {
		if !keep[code] {
			remove = append(remove, code)
		}
	}
	sort.Strings(remove)
	for _, code := range remove {
		if _, err := e.write(a, EvCallTypeRemoved, refs{}, idData{code}); err != nil {
			return err
		}
	}
	for _, c := range LASCallTypes() {
		if _, err := e.write(a, EvCallTypeSaved, refs{}, c); err != nil {
			return err
		}
	}
	return nil
}
