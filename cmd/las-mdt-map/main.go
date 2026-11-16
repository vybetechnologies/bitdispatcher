package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/biodream-llc/bitdispatcher/internal/mapdata"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type request struct {
	Path string
	Body json.RawMessage
}
type answer struct {
	Status                      int
	ContentType, Encoding, Data string
}

func main() {
	var st *mapdata.Store
	if len(os.Args) > 1 {
		st, _ = mapdata.Open(os.Args[1])
	}
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 65536), 2<<20)
	encoder := json.NewEncoder(os.Stdout)
	for scanner.Scan() {
		var q request
		if json.Unmarshal(scanner.Bytes(), &q) != nil {
			encoder.Encode(answer{Status: 400})
			continue
		}
		result := handle(st, q)
		encoder.Encode(result)
	}
	if st != nil {
		st.Close()
	}
}
func jsonAnswer(v any) answer {
	b, _ := json.Marshal(v)
	return answer{Status: 200, ContentType: "application/json", Data: base64.StdEncoding.EncodeToString(b)}
}
func handle(st *mapdata.Store, q request) answer {
	u, err := url.Parse(q.Path)
	if err != nil {
		return answer{Status: 400}
	}
	ctx := context.Background()
	path := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(path) == 5 && path[0] == "api" && path[1] == "map" && path[2] == "fonts" {
		b := mapdata.Glyphs(path[3], path[4])
		return answer{Status: 200, ContentType: "application/x-protobuf", Data: base64.StdEncoding.EncodeToString(b)}
	}
	if st == nil {
		return answer{Status: 404}
	}
	if len(path) == 6 && path[2] == "tiles" {
		z, e1 := strconv.Atoi(path[3])
		x, e2 := strconv.Atoi(path[4])
		y, e3 := strconv.Atoi(path[5])
		if e1 != nil || e2 != nil || e3 != nil || z < 0 || z > 20 || x < 0 || y < 0 {
			return answer{Status: 400}
		}
		b, err := st.Tile(ctx, z, x, y)
		if err != nil {
			return answer{Status: 500}
		}
		if len(b) == 0 {
			return answer{Status: 204}
		}
		return answer{Status: 200, ContentType: "application/vnd.mapbox-vector-tile", Encoding: "gzip", Data: base64.StdEncoding.EncodeToString(b)}
	}
	switch u.Path {
	case "/api/map/routes":
		var b struct {
			To    struct{ Lat, Lon float64 }
			Roads struct {
				Closures []struct {
					Line                [][2]float64
					Center              *[2]float64
					Radius              float64
					From, Until, Lifted time.Time
				}
			}
			From []struct {
				ID       string
				Lat, Lon float64
			}
		}
		if json.Unmarshal(q.Body, &b) != nil || len(b.From) > 60 {
			return answer{Status: 400}
		}
		router, err := st.Router()
		if err != nil {
			return answer{Status: 500}
		}
		factors := make([]float32, router.Segments())
		now := time.Now()
		for _, closure := range b.Roads.Closures {
			if !closure.Lifted.IsZero() || closure.From.After(now) || !closure.Until.IsZero() && closure.Until.Before(now) {
				continue
			}
			var segments []uint32
			if closure.Center != nil {
				segments = router.InCircle(closure.Center[0], closure.Center[1], closure.Radius)
			} else {
				segments = router.AlongLine(closure.Line, 18)
			}
			for _, segment := range segments {
				factors[segment] = mapdata.Closed
			}
		}
		router.SetFactors(factors)
		tree := router.To(b.To.Lat, b.To.Lon, 7200)
		result := []map[string]any{}
		for _, from := range b.From {
			route, ok := tree.From(from.Lat, from.Lon, true)
			points := [][2]float64{}
			for _, p := range route.Path {
				lat, lon := p.Deg()
				points = append(points, [2]float64{lon, lat})
			}
			result = append(result, map[string]any{"id": from.ID, "ok": ok, "seconds": route.Seconds, "metres": route.Metres, "via": route.Via, "path": points})
		}
		return jsonAnswer(result)
	case "/api/map/search":
		result, err := st.Search(ctx, u.Query().Get("q"), 0, 0, 20)
		if err != nil {
			return answer{Status: 500}
		}
		return jsonAnswer(result)
	case "/api/map/reverse":
		lat, _ := strconv.ParseFloat(u.Query().Get("lat"), 64)
		lon, _ := strconv.ParseFloat(u.Query().Get("lon"), 64)
		result, err := st.Reverse(ctx, lat, lon)
		if err != nil {
			return answer{Status: 404}
		}
		return jsonAnswer(result)
	case "/api/map/zones":
		zones, err := st.Zones()
		if err != nil {
			return answer{Status: 500}
		}
		return jsonAnswer(mapdata.ZonesGeoJSON(zones))
	}
	return answer{Status: 404, Data: base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("unsupported offline map path %q", u.Path)))}
}
