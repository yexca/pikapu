package api

import (
	"bytes"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

// spaHandler serves the built frontend. Unknown paths fall back to
// index.html so client-side routes work on reload.
func spaHandler(web fs.FS) http.Handler {
	index, err := fs.ReadFile(web, "index.html")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("Frontend is not built. Run `npm run build` in frontend/ and copy dist to backend/web/dist.\n"))
		})
	}
	files := http.FileServerFS(web)
	started := time.Now()

	serveIndex := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeContent(w, r, "index.html", started, bytes.NewReader(index))
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name == "" || name == "index.html" {
			serveIndex(w, r)
			return
		}
		info, err := fs.Stat(web, name)
		if err != nil || info.IsDir() {
			serveIndex(w, r)
			return
		}
		if strings.HasPrefix(name, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		// Go's MIME table has no entry for web app manifests.
		if path.Ext(name) == ".webmanifest" {
			w.Header().Set("Content-Type", "application/manifest+json")
		}
		files.ServeHTTP(w, r)
	})
}
