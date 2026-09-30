package api

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"pikapu/internal/store"
)

const (
	sessionCookie = "pikapu_session"
	sessionTTL    = 30 * 24 * time.Hour
)

// auth implements optional single-password protection using an HMAC-signed
// session cookie. With no password configured every request is allowed.
type auth struct {
	password string
	secret   []byte
}

func newAuth(ctx context.Context, st *store.Store, password string) (*auth, error) {
	a := &auth{password: password}
	if password == "" {
		return a, nil
	}
	stored, err := st.GetSetting(ctx, "session_secret")
	if err == nil {
		if a.secret, err = hex.DecodeString(stored); err == nil && len(a.secret) >= 32 {
			return a, nil
		}
	} else if !errors.Is(err, store.ErrNotFound) {
		return nil, err
	}
	a.secret = make([]byte, 32)
	if _, err := rand.Read(a.secret); err != nil {
		return nil, err
	}
	return a, st.SetSetting(ctx, "session_secret", hex.EncodeToString(a.secret))
}

func (a *auth) enabled() bool { return a.password != "" }

// sign binds the token to the current password so changing it logs everyone out.
func (a *auth) sign(payload string) string {
	pw := sha256.Sum256([]byte(a.password))
	mac := hmac.New(sha256.New, a.secret)
	mac.Write(pw[:])
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (a *auth) checkPassword(candidate string) bool {
	x := sha256.Sum256([]byte(candidate))
	y := sha256.Sum256([]byte(a.password))
	return subtle.ConstantTimeCompare(x[:], y[:]) == 1
}

func (a *auth) authenticated(r *http.Request) bool {
	if !a.enabled() {
		return true
	}
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return false
	}
	payload, sig, ok := strings.Cut(c.Value, ".")
	if !ok {
		return false
	}
	exp, err := strconv.ParseInt(payload, 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return false
	}
	return hmac.Equal([]byte(sig), []byte(a.sign(payload)))
}

func (a *auth) setSession(w http.ResponseWriter, r *http.Request) {
	exp := time.Now().Add(sessionTTL)
	payload := strconv.FormatInt(exp.Unix(), 10)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    payload + "." + a.sign(payload),
		Path:     "/",
		Expires:  exp,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   isHTTPS(r),
	})
}

func (a *auth) clearSession(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   isHTTPS(r),
	})
}

func (a *auth) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.authenticated(r) {
			writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isHTTPS(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}
