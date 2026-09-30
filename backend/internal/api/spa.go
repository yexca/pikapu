package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"io/fs"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"

	"pikapu/internal/fetcher"
)

var inlineScriptRe = regexp.MustCompile(`(?s)<script([^>]*)>(.*?)</script>`)

// contentSecurityPolicy allows only the app's own scripts plus the inline
// scripts in index.html (by hash). Article images and media may come from
// anywhere; frames only from the sanitizer's embed allowlist.
func contentSecurityPolicy(index []byte) string {
	scripts := []string{"'self'"}
	for _, m := range inlineScriptRe.FindAllSubmatch(index, -1) {
		if bytes.Contains(m[1], []byte("src=")) {
			continue
		}
		sum := sha256.Sum256(m[2])
		scripts = append(scripts, "'sha256-"+base64.StdEncoding.EncodeToString(sum[:])+"'")
	}
	return strings.Join([]string{
		"default-src 'self'",
		"script-src " + strings.Join(scripts, " "),
		// UI libraries inject style elements at runtime.
		"style-src 'self' 'unsafe-inline'",
		"img-src * data: blob:",
		"media-src * data: blob:",
		"frame-src " + strings.Join(fetcher.EmbedOrigins, " "),
		"connect-src 'self'",
		"font-src 'self' data:",
		"object-src 'none'",
		"base-uri 'self'",
		"form-action 'self'",
		"frame-ancestors 'self'",
	}, "; ")
}

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
	csp := contentSecurityPolicy(index)

	serveIndex := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Security-Policy", csp)
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
