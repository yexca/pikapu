package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"

	"pikapu/internal/fetcher"
	"pikapu/internal/service"
	"pikapu/internal/store"
)

type testServer struct {
	t   *testing.T
	srv *httptest.Server
	log *bytes.Buffer
}

func newTestServer(t *testing.T, opts Options) *testServer {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	web := fstest.MapFS{"index.html": {Data: []byte("<!doctype html><div id=root></div>")}}
	h, err := New(ctx, st, service.New(ctx, st, fetcher.New(), log), opts, web, log)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &testServer{t: t, srv: srv, log: &buf}
}

type result struct {
	status int
	header http.Header
	body   map[string]any
	raw    string
}

func (ts *testServer) call(method, path, cookie string, body any, headers ...string) result {
	ts.t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, ts.srv.URL+path, rd)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		ts.t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	out := result{status: res.StatusCode, header: res.Header, raw: string(raw)}
	_ = json.Unmarshal(raw, &out.body)
	return out
}

func sessionCookie(t *testing.T, r result) string {
	t.Helper()
	for _, c := range (&http.Response{Header: r.header}).Cookies() {
		if c.Name == cookieHTTP && c.Value != "" {
			if !c.HttpOnly {
				t.Error("session cookie is not HttpOnly")
			}
			return c.Name + "=" + c.Value
		}
	}
	t.Fatalf("no session cookie in %v", r.header)
	return ""
}

func expect(t *testing.T, name string, r result, status int, code string) {
	t.Helper()
	if r.status != status || (code != "" && r.body["code"] != code) {
		t.Fatalf("%s: got %d %s, want %d %s", name, r.status, r.raw, status, code)
	}
}

func TestAccountFlow(t *testing.T) {
	ts := newTestServer(t, Options{})

	status := ts.call("GET", "/api/auth/status", "", nil)
	if status.body["setup_required"] != true || status.body["authenticated"] != false || status.body["mode"] != "production" {
		t.Fatalf("fresh status: %s", status.raw)
	}
	expect(t, "API before setup", ts.call("GET", "/api/feeds", "", nil), 401, "unauthorized")
	expect(t, "login before setup", ts.call("POST", "/api/auth/login", "",
		map[string]string{"username": "admin", "password": "synthetic-password"}), 409, "setup_required")

	m := regexp.MustCompile(`setup_token=(\S+)`).FindStringSubmatch(ts.log.String())
	if m == nil {
		t.Fatalf("setup token not logged:\n%s", ts.log)
	}
	token := m[1]
	creds := map[string]string{"token": "wrong", "username": "Reader", "password": "synthetic-password"}
	expect(t, "wrong setup token", ts.call("POST", "/api/auth/setup", "", creds), 403, "invalid_setup_token")
	creds["token"] = token
	creds["password"] = "short"
	expect(t, "weak password", ts.call("POST", "/api/auth/setup", "", creds), 400, "invalid_new_password")
	creds["password"] = "synthetic-password"
	setup := ts.call("POST", "/api/auth/setup", "", creds)
	expect(t, "setup", setup, 201, "")
	cookieA := sessionCookie(t, setup)
	expect(t, "second setup", ts.call("POST", "/api/auth/setup", "", creds), 409, "already_set_up")

	status = ts.call("GET", "/api/auth/status", cookieA, nil)
	if status.body["authenticated"] != true || status.body["username"] != "Reader" {
		t.Fatalf("signed-in status: %s", status.raw)
	}
	expect(t, "API after setup", ts.call("GET", "/api/feeds", cookieA, nil), 200, "")

	expect(t, "wrong password", ts.call("POST", "/api/auth/login", "",
		map[string]string{"username": "reader", "password": "wrong-password"}), 401, "invalid_credentials")
	expect(t, "wrong username", ts.call("POST", "/api/auth/login", "",
		map[string]string{"username": "someone", "password": "synthetic-password"}), 401, "invalid_credentials")
	login := ts.call("POST", "/api/auth/login", "",
		map[string]string{"username": "reader", "password": "synthetic-password"})
	expect(t, "login", login, 200, "")
	cookieB := sessionCookie(t, login)

	var sessions []map[string]any
	list := ts.call("GET", "/api/account/sessions", cookieA, nil)
	_ = json.Unmarshal([]byte(list.raw), &sessions)
	if len(sessions) != 2 || sessions[0]["current"] == sessions[1]["current"] {
		t.Fatalf("sessions: %s", list.raw)
	}

	expect(t, "cross-origin write", ts.call("POST", "/api/categories", cookieA,
		map[string]string{"name": "Example"}, "Sec-Fetch-Site", "cross-site"), 403, "cross_origin")
	expect(t, "same-origin write", ts.call("POST", "/api/categories", cookieA,
		map[string]string{"name": "Example"}, "Sec-Fetch-Site", "same-origin"), 201, "")

	expect(t, "wrong current password", ts.call("PUT", "/api/account", cookieA,
		map[string]string{"current_password": "wrong-password", "username": "reader"}), 403, "invalid_password")
	update := ts.call("PUT", "/api/account", cookieA, map[string]string{
		"current_password": "synthetic-password", "username": "reader", "new_password": "another-password",
	})
	expect(t, "change password", update, 200, "")
	if update.body["username"] != "reader" {
		t.Fatalf("update: %s", update.raw)
	}
	expect(t, "other session after password change", ts.call("GET", "/api/feeds", cookieB, nil), 401, "unauthorized")
	expect(t, "current session after password change", ts.call("GET", "/api/feeds", cookieA, nil), 200, "")

	expect(t, "logout", ts.call("POST", "/api/auth/logout", cookieA, nil), 204, "")
	expect(t, "session after logout", ts.call("GET", "/api/feeds", cookieA, nil), 401, "unauthorized")
}

func TestLoginRateLimit(t *testing.T) {
	ts := newTestServer(t, Options{})
	m := regexp.MustCompile(`setup_token=(\S+)`).FindStringSubmatch(ts.log.String())
	expect(t, "setup", ts.call("POST", "/api/auth/setup", "", map[string]string{
		"token": m[1], "username": "admin", "password": "synthetic-password",
	}), 201, "")

	wrong := map[string]string{"username": "admin", "password": "wrong-password"}
	for i := 0; i < 5; i++ {
		expect(t, "wrong password", ts.call("POST", "/api/auth/login", "", wrong), 401, "invalid_credentials")
	}
	blocked := ts.call("POST", "/api/auth/login", "",
		map[string]string{"username": "admin", "password": "synthetic-password"})
	expect(t, "sixth attempt", blocked, 429, "too_many_attempts")
	if blocked.header.Get("Retry-After") == "" {
		t.Error("429 without Retry-After")
	}
}

func TestDevelopmentMode(t *testing.T) {
	ts := newTestServer(t, Options{Development: true})
	status := ts.call("GET", "/api/auth/status", "", nil)
	if status.body["mode"] != "development" || status.body["authenticated"] != true || status.body["setup_required"] != false {
		t.Fatalf("status: %s", status.raw)
	}
	expect(t, "API without sign-in", ts.call("GET", "/api/feeds", "", nil), 200, "")
	if strings.Contains(ts.log.String(), "setup_token") {
		t.Error("development mode logged a setup token")
	}
}

func TestClientIP(t *testing.T) {
	a := &authn{proxies: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}}
	cases := []struct {
		remote, xff, want string
	}{
		{"192.0.2.1:5000", "", "192.0.2.1"},
		{"192.0.2.1:5000", "198.51.100.9", "192.0.2.1"}, // untrusted peer: header ignored
		{"10.0.0.2:5000", "198.51.100.9", "198.51.100.9"},
		{"10.0.0.2:5000", "203.0.113.5, 198.51.100.9, 10.0.0.3", "198.51.100.9"},
		{"10.0.0.2:5000", "garbage", "10.0.0.2"},
		{"[::ffff:192.0.2.1]:5000", "", "192.0.2.1"},
	}
	for _, c := range cases {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = c.remote
		if c.xff != "" {
			r.Header.Set("X-Forwarded-For", c.xff)
		}
		if got := a.clientIP(r); got != c.want {
			t.Errorf("remote %s, XFF %q: got %s, want %s", c.remote, c.xff, got, c.want)
		}
	}
}
