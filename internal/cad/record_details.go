package cad

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/url"
	"strings"
	"time"
)

type EmployerInfo struct {
	Name       string `json:"name,omitempty"`
	Occupation string `json:"occupation,omitempty"`
	Address    string `json:"address,omitempty"`
	Phone      string `json:"phone,omitempty"`
	Notes      string `json:"notes,omitempty"`
}

// These attributed local entries supplement, rather than replace, core flags.
type RecordCaution struct {
	Type  string `json:"type"`
	Level string `json:"level"`
	Note  string `json:"note,omitempty"`
}
type PersonWarrant struct {
	Type         string `json:"type,omitempty"`
	Court        string `json:"court,omitempty"`
	Bond         string `json:"bond,omitempty"`
	Extradition  string `json:"extradition,omitempty"`
	CaseNumber   string `json:"caseNumber,omitempty"`
	Source       string `json:"source,omitempty"`
	AsOf         string `json:"asOf,omitempty"`
	SourcePage   string `json:"sourcePage,omitempty"`
	Number       string `json:"number"`
	Agency       string `json:"agency"`
	Offense      string `json:"offense"`
	Status       string `json:"status"`
	Issued       string `json:"issued,omitempty"`
	Expires      string `json:"expires,omitempty"`
	Instructions string `json:"instructions,omitempty"`
}

func ValidateRecordCautions(items []RecordCaution) error {
	if len(items) > 30 {
		return fmt.Errorf("at most 30 cautions")
	}
	for _, c := range items {
		if strings.TrimSpace(c.Type) == "" || len(c.Type) > 80 || len(c.Note) > 600 || (c.Level != "info" && c.Level != "caution" && c.Level != "danger") {
			return fmt.Errorf("cautions require a type and valid level; notes are limited to 600 characters")
		}
	}
	return nil
}
func ValidatePersonRecordDetails(p *Person) error {
	if p.Employer != nil {
		for _, v := range []string{p.Employer.Name, p.Employer.Occupation, p.Employer.Address, p.Employer.Phone, p.Employer.Notes} {
			if len(v) > 600 {
				return fmt.Errorf("employer fields must be at most 600 characters")
			}
		}
	}
	if p.OffenderProfileURL != "" {
		u, err := url.Parse(p.OffenderProfileURL)
		if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || len(p.OffenderProfileURL) > 1000 {
			return fmt.Errorf("offender profile must be a valid HTTPS registry URL")
		}
	}
	if err := ValidateRecordCautions(p.Cautions); err != nil {
		return err
	}
	if len(p.Warrants) > 30 {
		return fmt.Errorf("at most 30 warrants")
	}
	for _, w := range p.Warrants {
		if strings.TrimSpace(w.Agency) == "" || (strings.TrimSpace(w.Offense) == "" && strings.TrimSpace(w.Type) == "" && strings.TrimSpace(w.Source) == "") || (strings.TrimSpace(w.Number) == "" && strings.TrimSpace(w.Source) == "") || len(w.Number) > 100 || len(w.Agency) > 120 || len(w.Offense) > 600 || len(w.Instructions) > 600 {
			return fmt.Errorf("warrant requires an issuing agency and offense, type or documented source; a missing number requires a source")
		}
		if w.Status != "active" && w.Status != "recalled" && w.Status != "served" && w.Status != "expired" {
			return fmt.Errorf("invalid warrant status")
		}
		for _, v := range []string{w.Type, w.Court, w.Bond, w.Extradition, w.CaseNumber, w.Source, w.SourcePage} {
			if len(v) > 600 {
				return fmt.Errorf("warrant source and detail fields must be at most 600 characters")
			}
		}
		for _, date := range []string{w.Issued, w.Expires, w.AsOf} {
			if date != "" {
				if _, err := time.Parse("2006-01-02", date); err != nil {
					return fmt.Errorf("invalid warrant date")
				}
			}
		}
		if w.Issued != "" && w.Expires != "" && w.Expires < w.Issued {
			return fmt.Errorf("warrant expiration precedes issue date")
		}
	}
	if p.Photo != "" {
		if len(p.Photo) > 350000 {
			return fmt.Errorf("subject photo is too large")
		}
		parts := strings.SplitN(p.Photo, ",", 2)
		if len(parts) != 2 || (parts[0] != "data:image/jpeg;base64" && parts[0] != "data:image/png;base64") {
			return fmt.Errorf("subject photo must be JPEG or PNG")
		}
		b, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return fmt.Errorf("invalid subject photo")
		}
		cfg, format, err := image.DecodeConfig(bytes.NewReader(b))
		if err != nil || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 1600 || cfg.Height > 1600 || (format == "jpeg" && parts[0] != "data:image/jpeg;base64") || (format == "png" && parts[0] != "data:image/png;base64") {
			return fmt.Errorf("invalid subject photo dimensions or format")
		}
	}
	return nil
}
