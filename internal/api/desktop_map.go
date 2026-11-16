package api

import (
	"crypto/sha256"
	"fmt"
	"github.com/biodream-llc/bitdispatcher/internal/auth"
	"github.com/biodream-llc/bitdispatcher/internal/cad"
	"io"
	"net/http"
	"os"
	"time"
)

func (s *Server) desktopMap(w http.ResponseWriter, r *http.Request, se *auth.Session, a cad.Actor) {
	if _, ok := s.mdtUnit(w, r, se); !ok {
		return
	}
	st := s.mapStore(w)
	if st == nil {
		return
	}
	f, err := os.CreateTemp(s.DataDir, "mdt-map-*.db")
	if err != nil {
		writeErr(w, err)
		return
	}
	path := f.Name()
	f.Close()
	defer os.Remove(path)
	if err = st.ExportSnapshot(r.Context(), path); err != nil {
		writeErr(w, err)
		return
	}
	f, err = os.Open(path)
	if err != nil {
		writeErr(w, err)
		return
	}
	defer f.Close()
	hash := sha256.New()
	if _, err = io.Copy(hash, f); err != nil {
		writeErr(w, err)
		return
	}
	f.Seek(0, io.SeekStart)
	w.Header().Set("Content-Type", "application/vnd.sqlite3")
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-LAS-SHA256", fmt.Sprintf("%x", hash.Sum(nil)))
	w.Header().Set("X-LAS-Map-Built", st.Built.Format(time.RFC3339))
	http.ServeContent(w, r, "las-offline-map.db", st.Built, f)
}
