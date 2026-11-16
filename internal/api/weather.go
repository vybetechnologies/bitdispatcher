package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/biodream-llc/bitdispatcher/internal/auth"
	"github.com/biodream-llc/bitdispatcher/internal/cad"
)

// Fixed public NWS scope: Cass ND and Clay MN county and forecast zones. No
// officer location or CAD data leaves the server. County alerts often have no
// polygon; use their official affected-zone boundaries, never an invented box.
const weatherScope = "Cass County, ND / Clay County, MN"
const weatherPath = "/alerts/active?zone=NDC017,MNC027,NDZ039,MNZ003"

var weatherZone = regexp.MustCompile(`^/zones/(county|forecast)/(NDC017|MNC027|NDZ039|MNZ003)$`)

type weatherProperties struct {
	ID            string     `json:"id"`
	Event         string     `json:"event"`
	Headline      string     `json:"headline"`
	Description   string     `json:"description"`
	Instruction   string     `json:"instruction"`
	Severity      string     `json:"severity"`
	Certainty     string     `json:"certainty"`
	Urgency       string     `json:"urgency"`
	AreaDesc      string     `json:"areaDesc"`
	Status        string     `json:"status"`
	MessageType   string     `json:"messageType"`
	Sent          time.Time  `json:"sent"`
	Effective     time.Time  `json:"effective"`
	Onset         *time.Time `json:"onset"`
	Ends          *time.Time `json:"ends"`
	Expires       time.Time  `json:"expires"`
	AffectedZones []string   `json:"affectedZones"`
}
type weatherFeature struct {
	Type       string            `json:"type"`
	ID         string            `json:"id"`
	Geometry   json.RawMessage   `json:"geometry"`
	Properties weatherProperties `json:"properties"`
}
type weatherFeed struct {
	Type     string           `json:"type"`
	Features []weatherFeature `json:"features"`
	Checked  time.Time        `json:"checked"`
	Scope    string           `json:"scope"`
	Stale    bool             `json:"stale"`
	Error    string           `json:"error,omitempty"`
}
type weatherCache struct {
	mu    sync.Mutex
	retry time.Time
	feed  weatherFeed
	zones map[string]json.RawMessage
	// Injected only in tests. Production requests use the fixed NWS origin.
	client *http.Client
}

func (c *weatherCache) fetch(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.weather.gov"+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "LakesAreaSecurityCAD/1.0 (+https://lakesareasecurity.com)")
	req.Header.Set("Accept", "application/geo+json")
	client := c.client
	if client == nil {
		client = &http.Client{Timeout: 12 * time.Second, CheckRedirect: func(r *http.Request, via []*http.Request) error {
			if r.URL.Scheme != "https" || r.URL.Host != "api.weather.gov" || len(via) > 3 {
				return fmt.Errorf("unexpected NWS redirect")
			}
			return nil
		}}
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("NWS HTTP %d", res.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(res.Body, 4<<20)).Decode(out)
}
func weatherActive(p weatherProperties, now time.Time) bool {
	return p.Status == "Actual" && p.MessageType != "Cancel" && !p.Expires.IsZero() && p.Expires.After(now) && (p.Ends == nil || p.Ends.After(now)) && !p.Effective.After(now)
}
func (c *weatherCache) get(now time.Time) weatherFeed {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !now.Before(c.retry) {
		c.retry = now.Add(time.Minute)
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		var raw struct {
			Features []weatherFeature `json:"features"`
		}
		if err := c.fetch(ctx, weatherPath, &raw); err != nil {
			c.feed.Stale = true
			c.feed.Error = "NWS unavailable; weather coverage is not current"
		} else {
			feed := weatherFeed{Type: "FeatureCollection", Features: []weatherFeature{}, Checked: now, Scope: weatherScope}
			if c.zones == nil {
				c.zones = map[string]json.RawMessage{}
			}
			for _, f := range raw.Features {
				if !weatherActive(f.Properties, now) {
					continue
				}
				f.Type = "Feature"
				if f.ID == "" {
					f.ID = f.Properties.ID
				}
				if len(f.Geometry) == 0 || string(f.Geometry) == "null" {
					geometries := []json.RawMessage{}
					for _, z := range f.Properties.AffectedZones {
						const origin = "https://api.weather.gov"
						if len(z) <= len(origin) || z[:len(origin)] != origin || !weatherZone.MatchString(z[len(origin):]) {
							continue
						}
						g, ok := c.zones[z]
						if !ok {
							var zone struct {
								Geometry json.RawMessage `json:"geometry"`
							}
							if c.fetch(ctx, z[len(origin):], &zone) == nil && len(zone.Geometry) > 0 && string(zone.Geometry) != "null" {
								g = zone.Geometry
								c.zones[z] = g
							}
						}
						if len(g) > 0 {
							geometries = append(geometries, g)
						}
					}
					if len(geometries) > 0 {
						f.Geometry, _ = json.Marshal(map[string]any{"type": "GeometryCollection", "geometries": geometries})
					}
				}
				feed.Features = append(feed.Features, f)
			}
			c.feed = feed
		}
	}
	result := c.feed
	result.Type = "FeatureCollection"
	result.Scope = weatherScope
	result.Features = []weatherFeature{}
	if !result.Checked.IsZero() && now.Sub(result.Checked) < 30*time.Minute {
		for _, f := range c.feed.Features {
			if weatherActive(f.Properties, now) {
				result.Features = append(result.Features, f)
			}
		}
	}
	return result
}
func (s *Server) weatherRoutes() {
	s.handle("GET /api/weather/alerts", auth.Viewer, func(w http.ResponseWriter, r *http.Request, se *auth.Session, a cad.Actor) {
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, 200, s.weather.get(time.Now()))
	})
}
