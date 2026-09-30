package api

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"

	"pikapu/internal/auth"
	"pikapu/internal/store"
)

const (
	// Sessions expire after this long without use.
	sessionTTL = 30 * 24 * time.Hour
	// Activity is written back at most this often per session.
	touchInterval = time.Hour
	// HTTPS responses use the __Host- prefix, which browsers only accept
	// for Secure, host-only cookies with Path=/.
	cookieHTTP  = "pikapu_session"
	cookieHTTPS = "__Host-pikapu_session"
	maxUALength = 256
)

// Options configure authentication for the API.
type Options struct {
	// Development disables sign-in for local work.
	Development bool
	// TrustedProxies may set X-Forwarded-For.
	TrustedProxies []netip.Prefix
}

// authn guards the API with the admin account's server-side sessions.
type authn struct {
	st      *store.Store
	dev     bool
	proxies []netip.Prefix
	limiter *auth.Limiter
	now     func() time.Time

	mu         sync.Mutex
	setupToken string
}

type sessionKey struct{}

func newAuthn(st *store.Store, opts Options) *authn {
	return &authn{
		st:      st,
		dev:     opts.Development,
		proxies: opts.TrustedProxies,
		limiter: auth.NewLimiter(),
		now:     time.Now,
	}
}

// checkSetupToken reports whether candidate is the current setup token.
func (a *authn) checkSetupToken(candidate string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.setupToken == "" {
		return false
	}
	x := sha256.Sum256([]byte(strings.TrimSpace(candidate)))
	y := sha256.Sum256([]byte(a.setupToken))
	return subtle.ConstantTimeCompare(x[:], y[:]) == 1
}

func (a *authn) clearSetupToken() {
	a.mu.Lock()
	a.setupToken = ""
	a.mu.Unlock()
}

func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

func cookieName(r *http.Request) string {
	if isHTTPS(r) {
		return cookieHTTPS
	}
	return cookieHTTP
}

// session returns the request's valid session, or nil. It extends the
// session (and its cookie) when it has not been seen for a while.
func (a *authn) session(w http.ResponseWriter, r *http.Request) (*store.Session, error) {
	c, err := r.Cookie(cookieName(r))
	if err != nil || c.Value == "" {
		return nil, nil
	}
	now := a.now()
	ses, err := a.st.SessionByToken(r.Context(), hashToken(c.Value), now)
	if errors.Is(err, store.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if now.Sub(ses.LastSeenAt) >= touchInterval {
		exp := now.Add(sessionTTL)
		ip := a.clientIP(r)
		if err := a.st.TouchSession(r.Context(), ses.ID, ip, now, exp); err != nil {
			return nil, err
		}
		ses.IP, ses.LastSeenAt, ses.ExpiresAt = ip, now.UTC().Truncate(time.Second), exp
		a.setCookie(w, r, c.Value, exp)
	}
	return ses, nil
}

// startSession signs the browser in with a new session.
func (a *authn) startSession(w http.ResponseWriter, r *http.Request) error {
	now := a.now()
	token := auth.RandomToken(32)
	exp := now.Add(sessionTTL)
	ua := r.UserAgent()
	if len(ua) > maxUALength {
		ua = strings.ToValidUTF8(ua[:maxUALength], "")
	}
	if _, err := a.st.CreateSession(r.Context(), hashToken(token), ua, a.clientIP(r), exp); err != nil {
		return err
	}
	a.setCookie(w, r, token, exp)
	// Opportunistic cleanup; a failure only leaves expired rows behind.
	_ = a.st.DeleteExpiredSessions(r.Context(), now)
	return nil
}

func (a *authn) setCookie(w http.ResponseWriter, r *http.Request, token string, exp time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName(r),
		Value:    token,
		Path:     "/",
		Expires:  exp,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   isHTTPS(r),
	})
}

func (a *authn) clearCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName(r),
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   isHTTPS(r),
	})
}

func (a *authn) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.dev {
			next.ServeHTTP(w, r)
			return
		}
		ses, err := a.session(w, r)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", "internal server error")
			return
		}
		if ses == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), sessionKey{}, ses)))
	})
}

// currentSession is the session that authenticated the request; nil in
// development mode.
func currentSession(r *http.Request) *store.Session {
	ses, _ := r.Context().Value(sessionKey{}).(*store.Session)
	return ses
}

// throttled answers 429 when the client must wait before another
// credential check.
func (a *authn) throttled(w http.ResponseWriter, r *http.Request) bool {
	wait := a.limiter.Allow(a.clientIP(r))
	if wait <= 0 {
		return false
	}
	w.Header().Set("Retry-After", strconv.Itoa(int(wait/time.Second)))
	writeError(w, http.StatusTooManyRequests, "too_many_attempts", "too many attempts, try again later")
	return true
}

// clientIP is the connecting address, or the nearest untrusted address in
// X-Forwarded-For when the connection comes from a trusted proxy.
func (a *authn) clientIP(r *http.Request) string {
	ap, err := netip.ParseAddrPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	ip := ap.Addr().Unmap()
	if !a.trusted(ip) {
		return ip.String()
	}
	hops := strings.Split(strings.Join(r.Header.Values("X-Forwarded-For"), ","), ",")
	for i := len(hops) - 1; i >= 0; i-- {
		hop, err := netip.ParseAddr(strings.TrimSpace(hops[i]))
		if err != nil {
			break
		}
		ip = hop.Unmap()
		if !a.trusted(ip) {
			break
		}
	}
	return ip.String()
}

func (a *authn) trusted(ip netip.Addr) bool {
	for _, p := range a.proxies {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}

func isHTTPS(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}
