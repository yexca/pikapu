package api

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"pikapu/internal/auth"
	"pikapu/internal/buildinfo"
	"pikapu/internal/service"
	"pikapu/internal/store"
)

type handler struct {
	store *store.Store
	svc   *service.Service
	auth  *authn
	log   *slog.Logger
}

// New builds the HTTP handler serving the JSON API under /api and the
// single-page frontend everywhere else.
func New(ctx context.Context, st *store.Store, svc *service.Service, opts Options, web fs.FS, log *slog.Logger) (http.Handler, error) {
	a := newAuthn(st, opts)
	if !opts.Development {
		exists, err := st.HasAccount(ctx)
		if err != nil {
			return nil, err
		}
		if !exists {
			// Only someone who can read the server log can claim the instance.
			a.setupToken = auth.RandomToken(18)
			log.Warn("no admin account yet: open Pikapu and enter this setup token to create it",
				"setup_token", a.setupToken)
		}
	}
	h := &handler{store: st, svc: svc, auth: a, log: log}

	// Rejects state-changing requests that browsers mark as cross-origin.
	csrf := http.NewCrossOriginProtection()
	csrf.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusForbidden, "cross_origin", "cross-origin request rejected")
	}))

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(securityHeaders)
	r.Use(csrf.Handler)
	r.Use(middleware.Compress(5, "application/json", "text/html", "text/css",
		"text/javascript", "application/javascript", "image/svg+xml", "text/x-opml",
		"application/manifest+json"))

	r.Route("/api", func(r chi.Router) {
		r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": buildinfo.Version})
		})
		r.Get("/auth/status", h.authStatus)
		r.Post("/auth/setup", h.setup)
		r.Post("/auth/login", h.login)
		r.Post("/auth/logout", h.logout)

		r.Group(func(r chi.Router) {
			r.Use(a.middleware)

			r.Get("/account", h.getAccount)
			r.Put("/account", h.updateAccount)
			r.Get("/account/sessions", h.listSessions)
			r.Delete("/account/sessions", h.deleteOtherSessions)
			r.Delete("/account/sessions/{id}", h.deleteSession)

			r.Get("/categories", h.listCategories)
			r.Post("/categories", h.createCategory)
			r.Put("/categories/order", h.reorderCategories)
			r.Put("/categories/{id}", h.updateCategory)
			r.Delete("/categories/{id}", h.deleteCategory)

			r.Get("/feeds", h.listFeeds)
			r.Post("/feeds", h.createFeed)
			r.Post("/feeds/refresh", h.refreshAll)
			r.Put("/feeds/{id}", h.updateFeed)
			r.Delete("/feeds/{id}", h.deleteFeed)
			r.Post("/feeds/{id}/refresh", h.refreshFeed)
			r.Get("/feeds/{id}/icon", h.feedIcon)

			r.Get("/entries", h.listEntries)
			r.Get("/entries/recommended", h.recommendedEntries)
			r.Post("/entries/mark-all-read", h.markAllRead)
			r.Get("/entries/{id}", h.getEntry)
			r.Patch("/entries/{id}", h.updateEntry)

			r.Get("/filters", h.listFilters)
			r.Post("/filters", h.createFilter)
			r.Put("/filters/{id}", h.updateFilter)
			r.Delete("/filters/{id}", h.deleteFilter)
			r.Post("/filters/{id}/apply", h.applyFilter)

			r.Get("/counters", h.counters)
			r.Get("/settings", h.getSettings)
			r.Put("/settings", h.updateSettings)
			r.Get("/opml", h.exportOPML)
			r.Post("/opml", h.importOPML)
		})

		r.NotFound(func(w http.ResponseWriter, r *http.Request) {
			writeError(w, http.StatusNotFound, "not_found", "endpoint not found")
		})
		r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		})
	})

	r.Handle("/*", spaHandler(web))
	return r, nil
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hdr := w.Header()
		hdr.Set("X-Content-Type-Options", "nosniff")
		hdr.Set("Referrer-Policy", "no-referrer")
		hdr.Set("X-Frame-Options", "SAMEORIGIN")
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError sends an English message with a stable code. Clients localize
// by code and may show the message as detail or fallback text.
func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]string{"error": msg, "code": code})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return false
	}
	return true
}

func idParam(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid id")
		return 0, false
	}
	return id, true
}

// fail writes a response for errors not handled more specifically.
func (h *handler) fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "not found")
	case errors.Is(err, store.ErrConflict):
		writeError(w, http.StatusConflict, "conflict", "already exists")
	default:
		h.log.Error("request failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "internal server error")
	}
}
