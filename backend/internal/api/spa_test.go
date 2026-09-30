package api

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestSPAHandler(t *testing.T) {
	web := fstest.MapFS{
		"index.html":           {Data: []byte("<!doctype html><title>index</title>")},
		"manifest.webmanifest": {Data: []byte(`{"name":"Example"}`)},
		"assets/app-abc123.js": {Data: []byte("console.log(1)")},
	}
	h := spaHandler(web)

	tests := []struct {
		path         string
		body         string
		contentType  string
		cacheControl string
	}{
		{"/", "<!doctype html><title>index</title>", "text/html; charset=utf-8", "no-cache"},
		{"/feeds/3", "<!doctype html><title>index</title>", "text/html; charset=utf-8", "no-cache"},
		{"/manifest.webmanifest", `{"name":"Example"}`, "application/manifest+json", ""},
		{"/assets/app-abc123.js", "console.log(1)", "text/javascript; charset=utf-8", "public, max-age=31536000, immutable"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			if got := rec.Body.String(); got != tt.body {
				t.Errorf("body = %q, want %q", got, tt.body)
			}
			if got := rec.Header().Get("Content-Type"); got != tt.contentType {
				t.Errorf("Content-Type = %q, want %q", got, tt.contentType)
			}
			if got := rec.Header().Get("Cache-Control"); got != tt.cacheControl {
				t.Errorf("Cache-Control = %q, want %q", got, tt.cacheControl)
			}
		})
	}
}

func TestSPAContentSecurityPolicy(t *testing.T) {
	inline := "document.documentElement.classList.add('light')"
	web := fstest.MapFS{"index.html": {Data: []byte(
		"<script>" + inline + "</script><script type=\"module\" src=\"/assets/app.js\"></script>")}}
	rec := httptest.NewRecorder()
	spaHandler(web).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	csp := rec.Header().Get("Content-Security-Policy")
	sum := sha256.Sum256([]byte(inline))
	want := "script-src 'self' 'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "';"
	if !strings.Contains(csp, want) {
		t.Errorf("CSP %q does not contain %q", csp, want)
	}
	if !strings.Contains(csp, "frame-src https://www.youtube.com") || !strings.Contains(csp, "object-src 'none'") {
		t.Errorf("CSP %q is missing frame or object rules", csp)
	}
}
