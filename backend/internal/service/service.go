package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"

	"pikapu/internal/fetcher"
	"pikapu/internal/filter"
	"pikapu/internal/opml"
	"pikapu/internal/store"
)

const (
	refreshConcurrency = 6
	iconMaxAge         = 30 * 24 * time.Hour
	iconRetryAfter     = 3 * 24 * time.Hour
	cleanupEvery       = 12 * time.Hour
)

var ErrFeedExists = errors.New("feed already exists")

// Service coordinates fetching feeds and persisting the results, and runs the
// background refresh schedule.
type Service struct {
	ctx     context.Context
	store   *store.Store
	fetcher *fetcher.Fetcher
	log     *slog.Logger

	refreshing atomic.Bool
	icons      singleflight.Group

	mu          sync.Mutex
	lastCleanup time.Time
}

// New creates a service; ctx bounds all background work.
func New(ctx context.Context, st *store.Store, f *fetcher.Fetcher, log *slog.Logger) *Service {
	return &Service{ctx: ctx, store: st, fetcher: f, log: log}
}

// Refreshing reports whether a bulk refresh is in progress.
func (s *Service) Refreshing() bool { return s.refreshing.Load() }

// AddFeed discovers the feed behind rawURL, stores it with its current items
// and returns the stored record.
func (s *Service) AddFeed(ctx context.Context, rawURL string, categoryID *int64) (*store.Feed, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	res, err := s.fetcher.Discover(ctx, rawURL)
	if err != nil {
		return nil, fetcher.Classify(err)
	}
	if _, err := s.store.FeedByURL(ctx, res.URL); err == nil {
		return nil, ErrFeedExists
	}

	meta := fetcher.Meta(res.Feed, res.URL)
	feed := &store.Feed{
		CategoryID:  categoryID,
		Title:       meta.Title,
		FeedURL:     res.URL,
		SiteURL:     meta.SiteURL,
		Description: meta.Description,
	}
	if feed.Title == "" {
		feed.Title = hostOf(res.URL)
	}
	if err := s.store.CreateFeed(ctx, feed); err != nil {
		if errors.Is(err, store.ErrConflict) {
			return nil, ErrFeedExists
		}
		return nil, err
	}
	if err := s.save(context.WithoutCancel(ctx), feed, res); err != nil {
		return nil, err
	}
	return s.store.GetFeed(ctx, feed.ID)
}

// RefreshFeed fetches one feed now and records the outcome.
func (s *Service) RefreshFeed(ctx context.Context, feed *store.Feed) error {
	fetchCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	bg := context.WithoutCancel(ctx)

	res, err := s.fetcher.FetchFeed(fetchCtx, feed.FeedURL, feed.ETag, feed.LastModified)
	if err != nil {
		fe := fetcher.Classify(err)
		s.log.Warn("refresh failed", "feed", feed.FeedURL, "code", fe.Code, "err", fe.Error())
		if dbErr := s.store.FetchFailure(bg, feed.ID, fe.Code, fe.Error()); dbErr != nil {
			return dbErr
		}
		return fe
	}
	if res.NotModified {
		return s.store.FetchNotModified(bg, feed.ID)
	}
	return s.save(bg, feed, res)
}

func (s *Service) save(ctx context.Context, feed *store.Feed, res *fetcher.FeedResult) error {
	var skipBefore time.Time
	if feed.LastFetchedAt != nil {
		if settings, err := s.store.GetSettings(ctx); err == nil && settings.RetentionDays > 0 {
			skipBefore = time.Now().AddDate(0, 0, -settings.RetentionDays)
		}
	}
	filters, err := s.store.FiltersForFeed(ctx, feed.ID)
	if err != nil {
		return fmt.Errorf("load filters: %w", err)
	}
	var triage store.Triage
	if set := filter.Compile(filters); !set.Empty() {
		triage = set.Triage
	}
	items := fetcher.ConvertItems(res.Feed, feed.FeedURL)
	added, err := s.store.SaveEntries(ctx, feed.ID, items, skipBefore, triage)
	if err != nil {
		return fmt.Errorf("save entries: %w", err)
	}
	if added > 0 {
		s.log.Info("new entries", "feed", feed.Title, "count", added)
	}
	meta := fetcher.Meta(res.Feed, feed.FeedURL)
	return s.store.FetchSuccess(ctx, feed.ID, res.ETag, res.LastModified, meta.SiteURL, meta.Description)
}

// ApplyFilter runs one filter over the unread, unstarred entries in its scope
// and returns how many were marked read or removed.
func (s *Service) ApplyFilter(ctx context.Context, f *store.Filter) (int64, error) {
	return s.store.TriageUnread(ctx, f.FeedID, filter.Compile([]*store.Filter{f}).Triage)
}

// RefreshAll starts refreshing every feed in the background. It returns false
// if a bulk refresh is already running.
func (s *Service) RefreshAll() bool { return s.startRefresh(false) }

// RefreshDue starts refreshing feeds that are due (including never-fetched
// ones) in the background.
func (s *Service) RefreshDue() bool { return s.startRefresh(true) }

func (s *Service) startRefresh(onlyDue bool) bool {
	if !s.refreshing.CompareAndSwap(false, true) {
		return false
	}
	go func() {
		defer s.refreshing.Store(false)
		s.refreshFeeds(onlyDue)
	}()
	return true
}

func (s *Service) refreshFeeds(onlyDue bool) {
	feeds, err := s.store.ListFeeds(s.ctx)
	if err != nil {
		s.log.Error("list feeds", "err", err)
		return
	}
	if onlyDue {
		settings, err := s.store.GetSettings(s.ctx)
		if err != nil {
			s.log.Error("load settings", "err", err)
			return
		}
		feeds = dueFeeds(feeds, time.Duration(max(settings.RefreshIntervalMinutes, 5))*time.Minute)
	}
	var g errgroup.Group
	g.SetLimit(refreshConcurrency)
	for _, f := range feeds {
		if s.ctx.Err() != nil {
			break
		}
		g.Go(func() error {
			_ = s.RefreshFeed(s.ctx, f)
			return nil
		})
	}
	_ = g.Wait()
}

func dueFeeds(feeds []*store.Feed, interval time.Duration) []*store.Feed {
	var due []*store.Feed
	for _, f := range feeds {
		if f.LastFetchedAt == nil {
			due = append(due, f)
			continue
		}
		// Back off exponentially on failing feeds, up to 16x the interval.
		wait := interval * time.Duration(min(1<<min(f.ErrorCount, 4), 16))
		if time.Since(*f.LastFetchedAt) >= wait-30*time.Second {
			due = append(due, f)
		}
	}
	return due
}

func (s *Service) cleanup() {
	s.mu.Lock()
	if time.Since(s.lastCleanup) < cleanupEvery {
		s.mu.Unlock()
		return
	}
	s.lastCleanup = time.Now()
	s.mu.Unlock()

	settings, err := s.store.GetSettings(s.ctx)
	if err != nil || settings.RetentionDays <= 0 {
		return
	}
	n, err := s.store.DeleteOldEntries(s.ctx, time.Now().AddDate(0, 0, -settings.RetentionDays))
	if err != nil {
		s.log.Error("cleanup", "err", err)
		return
	}
	if n > 0 {
		s.log.Info("removed old entries", "count", n)
	}
}

// Run drives the refresh schedule until the service context ends.
func (s *Service) Run() {
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-timer.C:
		}
		if s.refreshing.CompareAndSwap(false, true) {
			s.refreshFeeds(true)
			s.refreshing.Store(false)
		}
		s.cleanup()
		timer.Reset(time.Minute)
	}
}

// FeedIcon returns the cached icon for a feed, looking it up on first use and
// refreshing stale ones in the background.
func (s *Service) FeedIcon(ctx context.Context, id int64) ([]byte, string, error) {
	feed, err := s.store.GetFeed(ctx, id)
	if err != nil {
		return nil, "", err
	}
	if feed.IconCheckedAt == nil {
		s.lookupIcon(feed)
	}
	data, mime, err := s.store.GetFeedIcon(ctx, id)
	if err != nil {
		return nil, "", err
	}
	if feed.IconCheckedAt != nil {
		age := time.Since(*feed.IconCheckedAt)
		if (len(data) > 0 && age > iconMaxAge) || (len(data) == 0 && age > iconRetryAfter) {
			go s.lookupIcon(feed)
		}
	}
	if len(data) == 0 {
		return nil, "", store.ErrNotFound
	}
	return data, mime, nil
}

func (s *Service) lookupIcon(feed *store.Feed) {
	_, _, _ = s.icons.Do(strconv.FormatInt(feed.ID, 10), func() (any, error) {
		ctx, cancel := context.WithTimeout(s.ctx, 15*time.Second)
		defer cancel()
		data, mime, err := s.fetcher.FetchIcon(ctx, feed.SiteURL, feed.FeedURL)
		if err != nil {
			if feed.IconCheckedAt == nil {
				return nil, s.store.SetFeedIcon(s.ctx, feed.ID, nil, "")
			}
			// Keep whatever icon we already had.
			return nil, s.store.MarkIconChecked(s.ctx, feed.ID)
		}
		return nil, s.store.SetFeedIcon(s.ctx, feed.ID, data, mime)
	})
}

// ImportResult summarizes an OPML import.
type ImportResult struct {
	Added   int `json:"added"`
	Skipped int `json:"skipped"`
}

// ImportOPML adds every subscription in the file that is not already present.
// New feeds are fetched in the background right away.
func (s *Service) ImportOPML(ctx context.Context, r io.Reader) (*ImportResult, error) {
	subs, err := opml.Parse(r)
	if err != nil {
		return nil, err
	}
	res := &ImportResult{}
	for _, sub := range subs {
		feedURL, err := fetcher.NormalizeURL(sub.FeedURL)
		if err != nil {
			res.Skipped++
			continue
		}
		if _, err := s.store.FeedByURL(ctx, feedURL); err == nil {
			res.Skipped++
			continue
		}
		var categoryID *int64
		if sub.Category != "" {
			c, err := s.store.EnsureCategory(ctx, sub.Category)
			if err != nil {
				return nil, err
			}
			categoryID = &c.ID
		}
		title := sub.Title
		if title == "" {
			title = hostOf(feedURL)
		}
		err = s.store.CreateFeed(ctx, &store.Feed{CategoryID: categoryID, Title: title, FeedURL: feedURL, SiteURL: sub.SiteURL})
		if errors.Is(err, store.ErrConflict) {
			res.Skipped++
			continue
		}
		if err != nil {
			return nil, err
		}
		res.Added++
	}
	if res.Added > 0 {
		s.RefreshDue()
	}
	return res, nil
}

// ExportOPML writes all subscriptions grouped by category.
func (s *Service) ExportOPML(ctx context.Context, w io.Writer) error {
	categories, err := s.store.ListCategories(ctx)
	if err != nil {
		return err
	}
	feeds, err := s.store.ListFeeds(ctx)
	if err != nil {
		return err
	}
	byCategory := map[int64][]opml.Subscription{}
	var loose []opml.Subscription
	for _, f := range feeds {
		sub := opml.Subscription{Title: f.Title, FeedURL: f.FeedURL, SiteURL: f.SiteURL}
		if f.CategoryID == nil {
			loose = append(loose, sub)
		} else {
			byCategory[*f.CategoryID] = append(byCategory[*f.CategoryID], sub)
		}
	}
	groups := []opml.Group{{Subscriptions: loose}}
	for _, c := range categories {
		if subs := byCategory[c.ID]; len(subs) > 0 {
			groups = append(groups, opml.Group{Name: c.Name, Subscriptions: subs})
		}
	}
	return opml.Write(w, "Pikapu subscriptions", groups)
}

func hostOf(raw string) string {
	if u, err := url.Parse(raw); err == nil && u.Host != "" {
		return u.Host
	}
	return raw
}
