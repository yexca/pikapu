package api

import (
	"net/http"
	"net/http/httptest"
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
