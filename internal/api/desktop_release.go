package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type windowsRelease struct {
	Available bool
	Version   string
	URL       string
	SHA256    string
	Size      int64
	Message   string
}

var windowsReleases struct {
	sync.Mutex
	At      time.Time
	Release windowsRelease
}

func windowsReleaseInfo() windowsRelease {
	windowsReleases.Lock()
	defer windowsReleases.Unlock()
	if time.Since(windowsReleases.At) < 15*time.Minute {
		return windowsReleases.Release
	}
	client := http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", "https://api.github.com/repos/vybetechnologies/bitdispatcher/releases?per_page=20", nil)
	req.Header.Set("User-Agent", "LAS-MDT-release-channel")
	req.Header.Set("Accept", "application/vnd.github+json")
	response, err := client.Do(req)
	if err != nil {
		return windowsRelease{Message: "Windows downloads are temporarily unavailable"}
	}
	defer response.Body.Close()
	var releases []struct {
		TagName           string
		Draft, Prerelease bool
		Assets            []struct {
			Name, BrowserDownloadURL, Digest string
			Size                             int64
		}
	}
	// Explicit API field names are defined in the published GitHub release schema.
	var raw []json.RawMessage
	if response.StatusCode != 200 || json.NewDecoder(http.MaxBytesReader(nil, response.Body, 2<<20)).Decode(&raw) != nil {
		return windowsRelease{Message: "Windows downloads are temporarily unavailable"}
	}
	for _, entry := range raw {
		var v struct {
			TagName    string `json:"tag_name"`
			Draft      bool
			Prerelease bool
			Assets     []struct {
				Name               string
				BrowserDownloadURL string `json:"browser_download_url"`
				Digest             string
				Size               int64
			}
		}
		if json.Unmarshal(entry, &v) == nil {
			releases = append(releases, struct {
				TagName           string
				Draft, Prerelease bool
				Assets            []struct {
					Name, BrowserDownloadURL, Digest string
					Size                             int64
				}
			}{v.TagName, v.Draft, v.Prerelease, nil})
			for _, a := range v.Assets {
				releases[len(releases)-1].Assets = append(releases[len(releases)-1].Assets, struct {
					Name, BrowserDownloadURL, Digest string
					Size                             int64
				}{a.Name, a.BrowserDownloadURL, a.Digest, a.Size})
			}
		}
	}
	result := windowsRelease{Message: "Windows MDT is being prepared"}
	valid := regexp.MustCompile("^mdt-v([0-9]+)[.]([0-9]+)[.]([0-9]+)$")
	best := int64(-1)
	for _, r := range releases {
		parts := valid.FindStringSubmatch(r.TagName)
		if parts == nil || r.Draft || r.Prerelease {
			continue
		}
		major, _ := strconv.ParseInt(parts[1], 10, 32)
		minor, _ := strconv.ParseInt(parts[2], 10, 32)
		patch, _ := strconv.ParseInt(parts[3], 10, 32)
		version := major*100000000 + minor*10000 + patch
		if version <= best {
			continue
		}
		for _, a := range r.Assets {
			if a.Name != "LAS-MDT-Setup-x64.exe" || !strings.HasPrefix(a.BrowserDownloadURL, "https://github.com/vybetechnologies/bitdispatcher/releases/download/"+r.TagName+"/") || !regexp.MustCompile("^sha256:[a-f0-9]{64}$").MatchString(a.Digest) {
				continue
			}
			result = windowsRelease{Available: true, Version: strings.TrimPrefix(r.TagName, "mdt-v"), URL: a.BrowserDownloadURL, SHA256: strings.TrimPrefix(a.Digest, "sha256:"), Size: a.Size}
			best = version
		}
	}
	windowsReleases.At = time.Now()
	windowsReleases.Release = result
	return result
}
func (s *Server) desktopDownloadRoutes() {
	s.mux.HandleFunc("GET /api/mdt/windows-release", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=300")
		writeJSON(w, 200, windowsReleaseInfo())
	})
	s.mux.HandleFunc("GET /api/mdt/windows-download", func(w http.ResponseWriter, r *http.Request) {
		release := windowsReleaseInfo()
		if !release.Available {
			fail(w, 503, release.Message)
			return
		}
		http.Redirect(w, r, release.URL, http.StatusFound)
	})
}
