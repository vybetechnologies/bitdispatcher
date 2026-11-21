package cad

import (
	"fmt"
	"strings"
)

// Type describes the linked person's relationship to the source person.
type PersonRelationship struct {
	Name     string `json:"name,omitempty"`
	PersonID string `json:"personId"`
	Type     string `json:"type"`
	Notes    string `json:"notes,omitempty"`
}

func ReverseRelationship(kind string) string {
	switch kind {
	case "parent":
		return "child"
	case "child":
		return "parent"
	case "employer":
		return "employee"
	case "employee":
		return "employer"
	case "guardian":
		return "dependent"
	case "dependent":
		return "guardian"
	}
	return kind
}
func (e *Engine) validatePersonRelationships(r LASRecord) error {
	if len(r.Relationships) > 100 {
		return fmt.Errorf("at most 100 person relationships")
	}
	seen := map[string]bool{}
	for _, link := range r.Relationships {
		if r.Kind != "person" {
			return fmt.Errorf("relationships belong to person records")
		}
		if !strings.Contains("|spouse|partner|parent|child|sibling|relative|friend|associate|roommate|coworker|employer|employee|guardian|dependent|other|", "|"+link.Type+"|") || link.Type == "" || len(link.Notes) > 600 {
			return fmt.Errorf("invalid relationship type or notes")
		}
		target, ok := e.st.LASRecords[link.PersonID]
		if !ok || target.Kind != "person" || link.PersonID == r.ID {
			return fmt.Errorf("relationship must link to a different existing person record")
		}
		if target.Agency != "" && !strings.EqualFold(target.Agency, r.Agency) {
			return fmt.Errorf("linked people must share agency scope")
		}
		key := link.PersonID + ":" + link.Type
		if seen[key] {
			return fmt.Errorf("duplicate person relationship")
		}
		seen[key] = true
	}
	return nil
}
