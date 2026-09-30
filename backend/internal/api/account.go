package api

import (
	"errors"
	"net/http"
	"time"

	"pikapu/internal/auth"
	"pikapu/internal/store"
)

// ---- auth (public) ----

func (h *handler) authStatus(w http.ResponseWriter, r *http.Request) {
	type status struct {
		Mode          string `json:"mode"`
		SetupRequired bool   `json:"setup_required"`
		Authenticated bool   `json:"authenticated"`
		Username      string `json:"username,omitempty"`
	}
	out := status{Mode: "production"}
	if h.auth.dev {
		out.Mode = "development"
		out.Authenticated = true
	}
	acct, err := h.store.GetAccount(r.Context())
	switch {
	case errors.Is(err, store.ErrNotFound):
		out.SetupRequired = !h.auth.dev
		writeJSON(w, http.StatusOK, out)
		return
	case err != nil:
		h.fail(w, err)
		return
	}
	if !h.auth.dev {
		ses, err := h.auth.session(w, r)
		if err != nil {
			h.fail(w, err)
			return
		}
		out.Authenticated = ses != nil
	}
	if out.Authenticated {
		out.Username = acct.Username
	}
	writeJSON(w, http.StatusOK, out)
}

// validCredentials checks a new username and password, writing a 400 on
// failure. An empty password is accepted when optional is set.
func validCredentials(w http.ResponseWriter, username, password string, optional bool) (string, bool) {
	name, ok := auth.NormalizeUsername(username)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_username", "username must be 1-64 characters")
		return "", false
	}
	if (password != "" || !optional) && !auth.ValidPassword(password) {
		writeError(w, http.StatusBadRequest, "invalid_new_password", "password must be 8-128 characters")
		return "", false
	}
	return name, true
}

func (h *handler) setup(w http.ResponseWriter, r *http.Request) {
	if h.auth.throttled(w, r) {
		return
	}
	var body struct {
		Token    string `json:"token"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if exists, err := h.store.HasAccount(r.Context()); err != nil {
		h.fail(w, err)
		return
	} else if exists {
		writeError(w, http.StatusConflict, "already_set_up", "the admin account already exists")
		return
	}
	ip := h.auth.clientIP(r)
	if !h.auth.checkSetupToken(body.Token) {
		h.auth.limiter.Fail(ip)
		writeError(w, http.StatusForbidden, "invalid_setup_token", "incorrect setup token")
		return
	}
	name, ok := validCredentials(w, body.Username, body.Password, false)
	if !ok {
		return
	}
	hash, err := auth.HashPassword(body.Password)
	if err != nil {
		h.fail(w, err)
		return
	}
	if err := h.store.CreateAccount(r.Context(), name, hash); errors.Is(err, store.ErrConflict) {
		writeError(w, http.StatusConflict, "already_set_up", "the admin account already exists")
		return
	} else if err != nil {
		h.fail(w, err)
		return
	}
	h.auth.clearSetupToken()
	h.auth.limiter.Succeed(ip)
	h.log.Info("admin account created", "username", name)
	if err := h.auth.startSession(w, r); err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"username": name})
}

func (h *handler) login(w http.ResponseWriter, r *http.Request) {
	if h.auth.throttled(w, r) {
		return
	}
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	acct, err := h.store.GetAccount(r.Context())
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusConflict, "setup_required", "set up the admin account first")
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	ip := h.auth.clientIP(r)
	// Always verify the password so a wrong username takes as long.
	passwordOK := auth.VerifyPassword(acct.PasswordHash, body.Password)
	if !passwordOK || !auth.SameUsername(body.Username, acct.Username) {
		h.auth.limiter.Fail(ip)
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "incorrect username or password")
		return
	}
	h.auth.limiter.Succeed(ip)
	if err := h.auth.startSession(w, r); err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"username": acct.Username})
}

func (h *handler) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieName(r)); err == nil && c.Value != "" {
		ses, err := h.store.SessionByToken(r.Context(), hashToken(c.Value), time.Now())
		if err == nil {
			err = h.store.DeleteSession(r.Context(), ses.ID)
		}
		if err != nil && !errors.Is(err, store.ErrNotFound) {
			h.fail(w, err)
			return
		}
	}
	h.auth.clearCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}

// ---- account (signed in) ----

func (h *handler) getAccount(w http.ResponseWriter, r *http.Request) {
	acct, err := h.store.GetAccount(r.Context())
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, acct)
}

func (h *handler) updateAccount(w http.ResponseWriter, r *http.Request) {
	if h.auth.throttled(w, r) {
		return
	}
	var body struct {
		CurrentPassword string `json:"current_password"`
		Username        string `json:"username"`
		NewPassword     string `json:"new_password"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	acct, err := h.store.GetAccount(r.Context())
	if err != nil {
		h.fail(w, err)
		return
	}
	ip := h.auth.clientIP(r)
	// 403 rather than 401: the session is fine, only the confirmation failed.
	if !auth.VerifyPassword(acct.PasswordHash, body.CurrentPassword) {
		h.auth.limiter.Fail(ip)
		writeError(w, http.StatusForbidden, "invalid_password", "incorrect password")
		return
	}
	h.auth.limiter.Succeed(ip)
	name, ok := validCredentials(w, body.Username, body.NewPassword, true)
	if !ok {
		return
	}
	hash := ""
	if body.NewPassword != "" {
		if hash, err = auth.HashPassword(body.NewPassword); err != nil {
			h.fail(w, err)
			return
		}
	}
	var keep int64
	if ses := currentSession(r); ses != nil {
		keep = ses.ID
	}
	if err := h.store.UpdateAccount(r.Context(), name, hash, keep); err != nil {
		h.fail(w, err)
		return
	}
	if hash != "" {
		h.log.Info("admin password changed; other sessions signed out")
	}
	h.getAccount(w, r)
}

func (h *handler) listSessions(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.ListSessions(r.Context(), time.Now())
	if err != nil {
		h.fail(w, err)
		return
	}
	type item struct {
		*store.Session
		Current bool `json:"current"`
	}
	current := currentSession(r)
	out := make([]item, len(list))
	for i, ses := range list {
		out[i] = item{Session: ses, Current: current != nil && ses.ID == current.ID}
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handler) deleteSession(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	if err := h.store.DeleteSession(r.Context(), id); err != nil {
		h.fail(w, err)
		return
	}
	if ses := currentSession(r); ses != nil && ses.ID == id {
		h.auth.clearCookie(w, r)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) deleteOtherSessions(w http.ResponseWriter, r *http.Request) {
	var keep int64
	if ses := currentSession(r); ses != nil {
		keep = ses.ID
	}
	n, err := h.store.DeleteOtherSessions(r.Context(), keep)
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"revoked": n})
}
