package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"pikapu/internal/fetcher"
	"pikapu/internal/filter"
	"pikapu/internal/opml"
	"pikapu/internal/recommend"
	"pikapu/internal/service"
	"pikapu/internal/store"
)

// ---- categories ----

func (h *handler) listCategories(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.ListCategories(r.Context())
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func validName(w http.ResponseWriter, name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 64 {
		writeError(w, http.StatusBadRequest, "invalid_name", "name must be 1-64 characters")
		return "", false
	}
	return name, true
}

func (h *handler) createCategory(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	name, ok := validName(w, body.Name)
	if !ok {
		return
	}
	c, err := h.store.CreateCategory(r.Context(), name)
	if errors.Is(err, store.ErrConflict) {
		writeError(w, http.StatusConflict, "category_exists", "category already exists")
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (h *handler) updateCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	name, ok := validName(w, body.Name)
	if !ok {
		return
	}
	err := h.store.RenameCategory(r.Context(), id, name)
	if errors.Is(err, store.ErrConflict) {
		writeError(w, http.StatusConflict, "category_exists", "category already exists")
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	c, err := h.store.GetCategory(r.Context(), id)
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *handler) reorderCategories(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IDs []int64 `json:"ids"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if err := h.store.ReorderCategories(r.Context(), body.IDs); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) deleteCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	if err := h.store.DeleteCategory(r.Context(), id); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- feeds ----

func (h *handler) listFeeds(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.ListFeeds(r.Context())
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// resolveCategory validates an optional category id, or creates a category
// from a name when one is given instead.
func (h *handler) resolveCategory(w http.ResponseWriter, r *http.Request, id *int64, name string) (*int64, bool) {
	if name = strings.TrimSpace(name); name != "" {
		name, ok := validName(w, name)
		if !ok {
			return nil, false
		}
		c, err := h.store.EnsureCategory(r.Context(), name)
		if err != nil {
			h.fail(w, err)
			return nil, false
		}
		return &c.ID, true
	}
	if id == nil || *id <= 0 {
		return nil, true
	}
	if _, err := h.store.GetCategory(r.Context(), *id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusBadRequest, "category_not_found", "category not found")
		} else {
			h.fail(w, err)
		}
		return nil, false
	}
	return id, true
}

func (h *handler) createFeed(w http.ResponseWriter, r *http.Request) {
	var body struct {
		URL          string `json:"url"`
		CategoryID   *int64 `json:"category_id"`
		CategoryName string `json:"category_name"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if strings.TrimSpace(body.URL) == "" {
		writeError(w, http.StatusBadRequest, "url_required", "feed URL is required")
		return
	}
	categoryID, ok := h.resolveCategory(w, r, body.CategoryID, body.CategoryName)
	if !ok {
		return
	}
	feed, err := h.svc.AddFeed(r.Context(), body.URL, categoryID)
	var fetchErr *fetcher.Error
	switch {
	case errors.Is(err, service.ErrFeedExists):
		writeError(w, http.StatusConflict, "feed_exists", err.Error())
	case errors.As(err, &fetchErr) && fetchErr.Code == fetcher.CodeInvalidURL:
		writeError(w, http.StatusBadRequest, fetchErr.Code, fetchErr.Error())
	case errors.As(err, &fetchErr):
		writeError(w, http.StatusBadGateway, fetchErr.Code, fetchErr.Error())
	case err != nil:
		h.fail(w, err)
	default:
		writeJSON(w, http.StatusCreated, feed)
	}
}

func (h *handler) updateFeed(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	var body struct {
		Title        string `json:"title"`
		FeedURL      string `json:"feed_url"`
		CategoryID   *int64 `json:"category_id"`
		CategoryName string `json:"category_name"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	title := strings.TrimSpace(body.Title)
	if title == "" || utf8.RuneCountInString(title) > 200 {
		writeError(w, http.StatusBadRequest, "invalid_title", "title must be 1-200 characters")
		return
	}
	feedURL, err := fetcher.NormalizeURL(body.FeedURL)
	if err != nil {
		writeError(w, http.StatusBadRequest, fetcher.CodeInvalidURL, err.Error())
		return
	}
	categoryID, ok := h.resolveCategory(w, r, body.CategoryID, body.CategoryName)
	if !ok {
		return
	}
	err = h.store.UpdateFeed(r.Context(), id, title, feedURL, categoryID)
	if errors.Is(err, store.ErrConflict) {
		writeError(w, http.StatusConflict, "feed_exists", "feed already exists")
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	feed, err := h.store.GetFeed(r.Context(), id)
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, feed)
}

func (h *handler) deleteFeed(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	if err := h.store.DeleteFeed(r.Context(), id); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// refreshFeed fetches a single feed synchronously. Fetch failures are
// reported through the returned feed's last_error field.
func (h *handler) refreshFeed(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	feed, err := h.store.GetFeed(r.Context(), id)
	if err != nil {
		h.fail(w, err)
		return
	}
	_ = h.svc.RefreshFeed(r.Context(), feed)
	if feed, err = h.store.GetFeed(r.Context(), id); err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, feed)
}

func (h *handler) refreshAll(w http.ResponseWriter, r *http.Request) {
	started := h.svc.RefreshAll()
	writeJSON(w, http.StatusAccepted, map[string]bool{"started": started})
}

func (h *handler) feedIcon(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	data, mime, err := h.svc.FeedIcon(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		w.Header().Set("Cache-Control", "private, max-age=3600")
		http.NotFound(w, r)
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; sandbox")
	_, _ = w.Write(data)
}

// ---- entries ----

func parseFilter(get func(string) string) store.EntryFilter {
	id := func(key string) int64 {
		v, _ := strconv.ParseInt(get(key), 10, 64)
		return v
	}
	flag := func(key string) bool {
		v := get(key)
		return v == "1" || v == "true"
	}
	return store.EntryFilter{
		FeedID:     id("feed_id"),
		CategoryID: id("category_id"),
		Starred:    flag("starred"),
		Unread:     flag("unread"),
		Query:      get("q"),
	}
}

func (h *handler) listEntries(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := parseFilter(q.Get)
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var after *store.Cursor
	if c := q.Get("cursor"); c != "" {
		var err error
		if after, err = store.ParseCursor(c); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_cursor", "invalid cursor")
			return
		}
	}
	entries, next, err := h.store.ListEntries(r.Context(), filter, after, limit)
	if err != nil {
		h.fail(w, err)
		return
	}
	resp := map[string]any{"entries": entries, "next_cursor": nil}
	if next != nil {
		resp["next_cursor"] = next.String()
	}
	writeJSON(w, http.StatusOK, resp)
}

// maxCandidates bounds how many recent unread entries are scored per request.
const maxCandidates = 1000

func (h *handler) recommendedEntries(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	scope := parseFilter(q.Get)
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	now := time.Now()
	since := now.Add(-recommend.Window)
	candidates, err := h.store.RecentUnread(r.Context(),
		store.EntryFilter{FeedID: scope.FeedID, CategoryID: scope.CategoryID}, since, maxCandidates)
	if err != nil {
		h.fail(w, err)
		return
	}
	stats, err := h.store.FeedStats(r.Context(), since, now)
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": recommend.Rank(candidates, stats, now, limit)})
}

func (h *handler) getEntry(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	e, err := h.store.GetEntry(r.Context(), id)
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (h *handler) updateEntry(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	var body struct {
		IsRead    *bool `json:"is_read"`
		IsStarred *bool `json:"is_starred"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if err := h.store.UpdateEntryState(r.Context(), id, body.IsRead, body.IsStarred); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) markAllRead(w http.ResponseWriter, r *http.Request) {
	var body struct {
		FeedID     int64  `json:"feed_id"`
		CategoryID int64  `json:"category_id"`
		Starred    bool   `json:"starred"`
		Query      string `json:"q"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	n, err := h.store.MarkAllRead(r.Context(), store.EntryFilter{
		FeedID: body.FeedID, CategoryID: body.CategoryID, Starred: body.Starred, Query: body.Query,
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"updated": n})
}

func (h *handler) counters(w http.ResponseWriter, r *http.Request) {
	c, err := h.store.Counters(r.Context())
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"unread":     c.Unread,
		"starred":    c.Starred,
		"feeds":      c.Feeds,
		"refreshing": h.svc.Refreshing(),
	})
}

// ---- filters ----

func (h *handler) listFilters(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.ListFilters(r.Context())
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// decodeFilter reads and validates a filter body. A missing or non-positive
// feed_id means the filter applies to all feeds.
func (h *handler) decodeFilter(w http.ResponseWriter, r *http.Request) (*store.Filter, bool) {
	var body struct {
		FeedID       *int64   `json:"feed_id"`
		Keywords     []string `json:"keywords"`
		MatchContent bool     `json:"match_content"`
		Invert       bool     `json:"invert"`
		Action       string   `json:"action"`
	}
	if !decodeJSON(w, r, &body) {
		return nil, false
	}
	keywords, ok := filter.CleanKeywords(body.Keywords)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_keywords",
			fmt.Sprintf("keywords must be 1-%d entries of up to %d characters", filter.MaxKeywords, filter.MaxKeywordLength))
		return nil, false
	}
	if body.Action != store.FilterMarkRead && body.Action != store.FilterSkip {
		writeError(w, http.StatusBadRequest, "invalid_filter_action", "action must be mark_read or skip")
		return nil, false
	}
	f := &store.Filter{Keywords: keywords, MatchContent: body.MatchContent, Invert: body.Invert, Action: body.Action}
	if body.FeedID != nil && *body.FeedID > 0 {
		if _, err := h.store.GetFeed(r.Context(), *body.FeedID); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusBadRequest, "unknown_feed", "feed does not exist")
			} else {
				h.fail(w, err)
			}
			return nil, false
		}
		f.FeedID = body.FeedID
	}
	return f, true
}

func (h *handler) createFilter(w http.ResponseWriter, r *http.Request) {
	f, ok := h.decodeFilter(w, r)
	if !ok {
		return
	}
	if err := h.store.CreateFilter(r.Context(), f); err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, f)
}

func (h *handler) updateFilter(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	f, ok := h.decodeFilter(w, r)
	if !ok {
		return
	}
	f.ID = id
	if err := h.store.UpdateFilter(r.Context(), f); err != nil {
		h.fail(w, err)
		return
	}
	saved, err := h.store.GetFilter(r.Context(), id)
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (h *handler) deleteFilter(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	if err := h.store.DeleteFilter(r.Context(), id); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// applyFilter runs a saved filter over current unread, unstarred entries.
func (h *handler) applyFilter(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	f, err := h.store.GetFilter(r.Context(), id)
	if err != nil {
		h.fail(w, err)
		return
	}
	n, err := h.svc.ApplyFilter(r.Context(), f)
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"updated": n})
}

// ---- settings ----

func (h *handler) getSettings(w http.ResponseWriter, r *http.Request) {
	s, err := h.store.GetSettings(r.Context())
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func (h *handler) updateSettings(w http.ResponseWriter, r *http.Request) {
	var body store.Settings
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.RefreshIntervalMinutes < 5 || body.RefreshIntervalMinutes > 1440 {
		writeError(w, http.StatusBadRequest, "invalid_refresh_interval", "refresh interval must be between 5 and 1440 minutes")
		return
	}
	if body.RetentionDays < 0 || body.RetentionDays > 3650 {
		writeError(w, http.StatusBadRequest, "invalid_retention", "retention must be between 0 and 3650 days")
		return
	}
	if err := h.store.SaveSettings(r.Context(), body); err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, body)
}

// ---- OPML ----

func (h *handler) exportOPML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/x-opml; charset=utf-8")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="pikapu-%s.opml"`, time.Now().Format("20060102")))
	if err := h.svc.ExportOPML(r.Context(), w); err != nil {
		h.log.Error("export opml", "err", err)
	}
}

func (h *handler) importOPML(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "opml_required", "OPML file is required")
		return
	}
	defer file.Close()
	res, err := h.svc.ImportOPML(r.Context(), file)
	if errors.Is(err, opml.ErrInvalid) {
		writeError(w, http.StatusBadRequest, "opml_invalid", err.Error())
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}
