package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type weatherTransport func(*http.Request) (*http.Response, error)

func (f weatherTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func weatherResponse(v any) *http.Response {
	b, _ := json.Marshal(v)
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(b)))}
}
func TestWeatherCacheZonesExpiryAndOutage(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	props := weatherProperties{ID: "ice", Event: "Ice Storm Warning", Status: "Actual", MessageType: "Alert", Effective: now.Add(-time.Minute), Expires: now.Add(10 * time.Minute), AffectedZones: []string{"https://evil.example/zones/county/NDC017", "https://api.weather.gov/zones/county/NDC017"}}
	bad := props
	bad.ID = "test"
	bad.Status = "Test"
	expired := props
	expired.ID = "expired"
	expired.Expires = now.Add(-time.Second)
	calls := 0
	outage := false
	c := weatherCache{client: &http.Client{Transport: weatherTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if outage {
			return nil, fmt.Errorf("offline")
		}
		if r.URL.Host != "api.weather.gov" || r.Header.Get("User-Agent") == "" {
			t.Fatal("bad upstream request", r.URL)
		}
		switch r.URL.RequestURI() {
		case weatherPath:
			return weatherResponse(map[string]any{"features": []weatherFeature{{ID: "ice", Properties: props, Geometry: json.RawMessage("null")}, {ID: "test", Properties: bad}, {ID: "expired", Properties: expired}}}), nil
		case "/zones/county/NDC017":
			return weatherResponse(map[string]any{"geometry": map[string]any{"type": "Polygon", "coordinates": [][][]float64{{{-97, 46}, {-96, 46}, {-96, 47}, {-97, 46}}}}}), nil
		default:
			t.Fatal("unexpected path", r.URL)
			return nil, fmt.Errorf("unexpected path")
		}
	})}}
	f := c.get(now)
	if f.Stale || len(f.Features) != 1 || !strings.Contains(string(f.Features[0].Geometry), "GeometryCollection") {
		t.Fatalf("missing official boundary or inactive filtering: %+v", f)
	}
	c.get(now.Add(10 * time.Second))
	if calls != 2 {
		t.Fatal("cache not shared", calls)
	}
	outage = true
	f = c.get(now.Add(2 * time.Minute))
	if !f.Stale || f.Error == "" || len(f.Features) != 1 {
		t.Fatal("lost explicit stale indication", f)
	}
	f = c.get(now.Add(11 * time.Minute))
	if len(f.Features) != 0 {
		t.Fatal("expired warning still visible", f)
	}
}
func TestWeatherRouteRequiresSession(t *testing.T) {
	ts, s, pw := newServer(t)
	s.weatherRoutes()
	s.weather.retry = time.Now().Add(time.Hour)
	s.weather.feed = weatherFeed{Type: "FeatureCollection", Checked: time.Now(), Features: []weatherFeature{}, Scope: weatherScope}
	r, err := ts.Client().Get(ts.URL + "/api/weather/alerts")
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.StatusCode != 401 {
		t.Fatal("unauthenticated weather", r.StatusCode)
	}
	c := login(t, ts, "admin", pw["admin"])
	code, body := c.do("GET", "/api/weather/alerts", nil, true)
	if code != 200 || !strings.Contains(string(body), "Cass County") {
		t.Fatal(code, string(body))
	}
}
