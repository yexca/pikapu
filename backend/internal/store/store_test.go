package store

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func openTest(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func newFeed(t *testing.T, s *Store, url string, categoryID *int64) *Feed {
	t.Helper()
	f := &Feed{Title: url, FeedURL: url, CategoryID: categoryID}
	if err := s.CreateFeed(context.Background(), f); err != nil {
		t.Fatal(err)
	}
	return f
}

func items(n int, start time.Time) []EntryInput {
	out := make([]EntryInput, n)
	for i := range out {
		out[i] = EntryInput{
			GUID:        fmt.Sprintf("g%d", i),
			Title:       fmt.Sprintf("Entry %d", i),
			Content:     "<p>body</p>",
			PublishedAt: start.Add(time.Duration(i) * time.Hour),
		}
	}
	return out
}

func TestSaveEntriesUpsert(t *testing.T) {
	ctx := context.Background()
	s := openTest(t)
	f := newFeed(t, s, "https://feed-a.example.com/feed", nil)
	now := time.Now().UTC().Truncate(time.Second)

	n, err := s.SaveEntries(ctx, f.ID, items(3, now), time.Time{}, nil)
	if err != nil || n != 3 {
		t.Fatalf("first save: n=%d err=%v", n, err)
	}

	changed := items(3, now)
	changed[0].Title = "Updated"
	n, err = s.SaveEntries(ctx, f.ID, changed, time.Time{}, nil)
	if err != nil || n != 0 {
		t.Fatalf("second save: n=%d err=%v", n, err)
	}
	list, _, err := s.ListEntries(ctx, EntryFilter{FeedID: f.ID}, nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 || list[2].Title != "Updated" {
		t.Fatalf("unexpected entries: %+v", list)
	}
}

func TestSaveEntriesSkipsOldUnseen(t *testing.T) {
	ctx := context.Background()
	s := openTest(t)
	f := newFeed(t, s, "https://feed-a.example.com/feed", nil)
	old := time.Now().AddDate(0, 0, -100)

	n, err := s.SaveEntries(ctx, f.ID, items(2, old), time.Now().AddDate(0, 0, -30), nil)
	if err != nil || n != 0 {
		t.Fatalf("old items should be skipped: n=%d err=%v", n, err)
	}
}

func TestListEntriesPagination(t *testing.T) {
	ctx := context.Background()
	s := openTest(t)
	f := newFeed(t, s, "https://feed-a.example.com/feed", nil)
	start := time.Now().Add(-48 * time.Hour).UTC()
	if _, err := s.SaveEntries(ctx, f.ID, items(25, start), time.Time{}, nil); err != nil {
		t.Fatal(err)
	}

	seen := map[int64]bool{}
	var cursor *Cursor
	pages := 0
	for {
		list, next, err := s.ListEntries(ctx, EntryFilter{}, cursor, 10)
		if err != nil {
			t.Fatal(err)
		}
		pages++
		for i, e := range list {
			if seen[e.ID] {
				t.Fatalf("duplicate entry %d", e.ID)
			}
			seen[e.ID] = true
			if i > 0 && e.PublishedAt.After(list[i-1].PublishedAt) {
				t.Fatal("entries not sorted newest first")
			}
		}
		if next == nil {
			break
		}
		parsed, err := ParseCursor(next.String())
		if err != nil {
			t.Fatal(err)
		}
		cursor = parsed
	}
	if len(seen) != 25 || pages != 3 {
		t.Fatalf("got %d entries over %d pages", len(seen), pages)
	}
}

func TestReadStateAndCounters(t *testing.T) {
	ctx := context.Background()
	s := openTest(t)
	cat, err := s.CreateCategory(ctx, "Tech")
	if err != nil {
		t.Fatal(err)
	}
	a := newFeed(t, s, "https://feed-a.example.com/feed", &cat.ID)
	b := newFeed(t, s, "https://feed-b.example.com/feed", nil)
	now := time.Now().UTC()
	s.SaveEntries(ctx, a.ID, items(3, now), time.Time{}, nil)
	s.SaveEntries(ctx, b.ID, items(2, now), time.Time{}, nil)

	list, _, _ := s.ListEntries(ctx, EntryFilter{FeedID: a.ID}, nil, 10)
	yes := true
	if err := s.UpdateEntryState(ctx, list[0].ID, &yes, &yes); err != nil {
		t.Fatal(err)
	}

	c, err := s.Counters(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if c.Unread != 4 || c.Starred != 1 || c.Feeds[a.ID] != 2 || c.Feeds[b.ID] != 2 {
		t.Fatalf("counters: %+v", c)
	}

	n, err := s.MarkAllRead(ctx, EntryFilter{CategoryID: cat.ID})
	if err != nil || n != 2 {
		t.Fatalf("mark category read: n=%d err=%v", n, err)
	}
	c, _ = s.Counters(ctx)
	if c.Unread != 2 || c.Feeds[a.ID] != 0 {
		t.Fatalf("counters after mark all: %+v", c)
	}

	unread, _, _ := s.ListEntries(ctx, EntryFilter{Unread: true}, nil, 10)
	if len(unread) != 2 {
		t.Fatalf("unread filter: got %d", len(unread))
	}
	starred, _, _ := s.ListEntries(ctx, EntryFilter{Starred: true}, nil, 10)
	if len(starred) != 1 {
		t.Fatalf("starred filter: got %d", len(starred))
	}
}

func TestSearchEscapesWildcards(t *testing.T) {
	ctx := context.Background()
	s := openTest(t)
	f := newFeed(t, s, "https://feed-a.example.com/feed", nil)
	now := time.Now().UTC()
	s.SaveEntries(ctx, f.ID, []EntryInput{
		{GUID: "1", Title: "100% 纯中文标题", PublishedAt: now},
		{GUID: "2", Title: "Another post", PublishedAt: now},
	}, time.Time{}, nil)

	for q, want := range map[string]int{"中文": 1, "100%": 1, "%": 1, "post": 1, "nothing": 0} {
		list, _, err := s.ListEntries(ctx, EntryFilter{Query: q}, nil, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != want {
			t.Errorf("query %q: got %d, want %d", q, len(list), want)
		}
	}
}

func TestDeleteOldEntriesKeepsStarredAndUnread(t *testing.T) {
	ctx := context.Background()
	s := openTest(t)
	f := newFeed(t, s, "https://feed-a.example.com/feed", nil)
	old := time.Now().AddDate(0, 0, -200).UTC()
	s.SaveEntries(ctx, f.ID, items(3, old), time.Time{}, nil)
	list, _, _ := s.ListEntries(ctx, EntryFilter{}, nil, 10)

	yes := true
	s.UpdateEntryState(ctx, list[0].ID, &yes, nil)  // read -> deleted
	s.UpdateEntryState(ctx, list[1].ID, &yes, &yes) // read + starred -> kept
	// list[2] stays unread -> kept

	n, err := s.DeleteOldEntries(ctx, time.Now().AddDate(0, 0, -90))
	if err != nil || n != 1 {
		t.Fatalf("deleted %d, err %v", n, err)
	}
}

func TestDeleteCategoryUncategorizesFeeds(t *testing.T) {
	ctx := context.Background()
	s := openTest(t)
	cat, _ := s.CreateCategory(ctx, "News")
	f := newFeed(t, s, "https://feed-a.example.com/feed", &cat.ID)
	if _, err := s.CreateCategory(ctx, "news"); err != ErrConflict {
		t.Fatalf("case-insensitive duplicate should conflict, got %v", err)
	}
	if err := s.DeleteCategory(ctx, cat.ID); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetFeed(ctx, f.ID)
	if err != nil || got.CategoryID != nil {
		t.Fatalf("feed should be uncategorized: %+v %v", got, err)
	}
}

func TestSaveEntriesTriagesOnlyNewItems(t *testing.T) {
	ctx := context.Background()
	s := openTest(t)
	f := newFeed(t, s, "https://feed-a.example.com/feed", nil)
	now := time.Now().UTC()

	var seen []string
	triage := func(it EntryInput) Verdict {
		seen = append(seen, it.GUID)
		switch it.GUID {
		case "g0":
			return KeepRead
		case "g1":
			return Drop
		}
		return Keep
	}
	n, err := s.SaveEntries(ctx, f.ID, items(3, now), time.Time{}, triage)
	if err != nil || n != 2 {
		t.Fatalf("save: n=%d err=%v", n, err)
	}
	list, _, _ := s.ListEntries(ctx, EntryFilter{FeedID: f.ID}, nil, 10)
	if len(list) != 2 || list[0].Title != "Entry 2" || list[0].IsRead || list[1].Title != "Entry 0" || !list[1].IsRead {
		t.Fatalf("unexpected entries: %+v %+v", list[0], list[1])
	}

	// A known entry is never triaged again, so marking it unread sticks; a
	// dropped item is offered to triage again on the next fetch.
	no := false
	s.UpdateEntryState(ctx, list[1].ID, &no, nil)
	seen = nil
	s.SaveEntries(ctx, f.ID, items(3, now), time.Time{}, triage)
	if !slices.Equal(seen, []string{"g1"}) {
		t.Fatalf("triaged %v, want only the dropped item", seen)
	}
	if e, _ := s.GetEntry(ctx, list[1].ID); e.IsRead {
		t.Fatal("a known entry's read state must not change")
	}
}

func TestFiltersForFeedAndCascade(t *testing.T) {
	ctx := context.Background()
	s := openTest(t)
	a := newFeed(t, s, "https://feed-a.example.com/feed", nil)
	b := newFeed(t, s, "https://feed-b.example.com/feed", nil)

	global := &Filter{Keywords: []string{"Sponsored", "广告"}, Action: FilterSkip}
	onA := &Filter{FeedID: &a.ID, Keywords: []string{"digest"}, MatchContent: true, Invert: true, Action: FilterMarkRead}
	for _, f := range []*Filter{onA, global} {
		if err := s.CreateFilter(ctx, f); err != nil {
			t.Fatal(err)
		}
	}

	got, err := s.GetFilter(ctx, onA.ID)
	if err != nil || *got.FeedID != a.ID || !slices.Equal(got.Keywords, []string{"digest"}) ||
		!got.MatchContent || !got.Invert || got.Action != FilterMarkRead {
		t.Fatalf("round trip: %+v %v", got, err)
	}
	all, _ := s.ListFilters(ctx)
	if len(all) != 2 || all[0].ID != global.ID {
		t.Fatalf("filters for all feeds should be listed first: %+v", all)
	}
	if forB, _ := s.FiltersForFeed(ctx, b.ID); len(forB) != 1 || forB[0].ID != global.ID {
		t.Fatalf("feed b should only get the global filter: %+v", forB)
	}
	if forA, _ := s.FiltersForFeed(ctx, a.ID); len(forA) != 2 {
		t.Fatalf("feed a should get both filters: %+v", forA)
	}

	global.Keywords = []string{"giveaway"}
	if err := s.UpdateFilter(ctx, global); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.GetFilter(ctx, global.ID); !slices.Equal(got.Keywords, []string{"giveaway"}) {
		t.Fatalf("update: %+v", got)
	}

	if err := s.DeleteFeed(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetFilter(ctx, onA.ID); err != ErrNotFound {
		t.Fatalf("a feed's filters should be deleted with it, got %v", err)
	}
	if err := s.DeleteFilter(ctx, global.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteFilter(ctx, global.ID); err != ErrNotFound {
		t.Fatalf("deleting twice: %v", err)
	}
}

func TestTriageUnreadSkipsReadAndStarred(t *testing.T) {
	ctx := context.Background()
	s := openTest(t)
	a := newFeed(t, s, "https://feed-a.example.com/feed", nil)
	b := newFeed(t, s, "https://feed-b.example.com/feed", nil)
	now := time.Now().UTC()
	s.SaveEntries(ctx, a.ID, items(4, now), time.Time{}, nil)
	s.SaveEntries(ctx, b.ID, items(2, now), time.Time{}, nil)

	list, _, _ := s.ListEntries(ctx, EntryFilter{FeedID: a.ID}, nil, 10) // Entry 3..0
	yes := true
	s.UpdateEntryState(ctx, list[0].ID, &yes, nil) // Entry 3: read
	s.UpdateEntryState(ctx, list[1].ID, nil, &yes) // Entry 2: starred

	triage := func(it EntryInput) Verdict {
		if it.Title == "Entry 1" {
			return Drop
		}
		return KeepRead
	}
	n, err := s.TriageUnread(ctx, &a.ID, triage)
	if err != nil || n != 2 {
		t.Fatalf("n=%d err=%v, want Entry 1 dropped and Entry 0 read", n, err)
	}
	after, _, _ := s.ListEntries(ctx, EntryFilter{FeedID: a.ID}, nil, 10)
	if len(after) != 3 || after[1].IsRead || !after[2].IsRead || after[2].Title != "Entry 0" {
		t.Fatalf("unexpected entries after triage: %+v", after)
	}
	if c, _ := s.Counters(ctx); c.Feeds[b.ID] != 2 {
		t.Fatalf("other feeds must be untouched: %+v", c.Feeds)
	}
}

func TestAffinityCountsOnlyIndividualEngagement(t *testing.T) {
	ctx := context.Background()
	s := openTest(t)
	a := newFeed(t, s, "https://feed-a.example.com/feed", nil)
	b := newFeed(t, s, "https://feed-b.example.com/feed", nil)
	now := time.Now().UTC()
	s.SaveEntries(ctx, a.ID, items(3, now.Add(-10*time.Hour)), time.Time{}, nil)
	s.SaveEntries(ctx, b.ID, items(3, now.Add(-10*time.Hour)), time.Time{}, nil)

	list, _, _ := s.ListEntries(ctx, EntryFilter{FeedID: a.ID}, nil, 10)
	yes, no := true, false
	if err := s.UpdateEntryState(ctx, list[0].ID, &yes, nil); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateEntryState(ctx, list[1].ID, nil, &yes); err != nil {
		t.Fatal(err)
	}
	// Repeating a state or undoing it adds nothing.
	if err := s.UpdateEntryState(ctx, list[0].ID, &yes, nil); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateEntryState(ctx, list[1].ID, nil, &no); err != nil {
		t.Fatal(err)
	}
	if _, err := s.MarkAllRead(ctx, EntryFilter{FeedID: b.ID}); err != nil {
		t.Fatal(err)
	}

	stats, err := s.FeedStats(ctx, now.Add(-24*time.Hour), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if got := stats[a.ID].Affinity; got < 3.99 || got > 4 {
		t.Errorf("feed a affinity = %v, want ~4", got)
	}
	if got := stats[b.ID].Affinity; got != 0 {
		t.Errorf("feed b affinity = %v, want 0 after mark all read", got)
	}
	if stats[a.ID].Recent != 3 {
		t.Errorf("feed a recent = %d, want 3", stats[a.ID].Recent)
	}

	if err := s.UpdateEntryState(ctx, 9999, &yes, nil); err != ErrNotFound {
		t.Errorf("unknown entry: err = %v, want ErrNotFound", err)
	}
}

func TestDecayAffinityHalves(t *testing.T) {
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if got := decayAffinity(8, at, at.Add(AffinityHalfLife)); got < 3.999 || got > 4.001 {
		t.Fatalf("after one half-life: %v, want 4", got)
	}
	if got := decayAffinity(8, at, at.Add(-time.Hour)); got != 8 {
		t.Fatalf("clock skew: %v, want 8", got)
	}
}

func TestRecentUnreadWindowAndScope(t *testing.T) {
	ctx := context.Background()
	s := openTest(t)
	cat, _ := s.CreateCategory(ctx, "News")
	a := newFeed(t, s, "https://feed-a.example.com/feed", &cat.ID)
	b := newFeed(t, s, "https://feed-b.example.com/feed", nil)
	now := time.Now().UTC()
	// Three recent entries and three older than the window in feed a.
	s.SaveEntries(ctx, a.ID, items(3, now.Add(-5*time.Hour)), time.Time{}, nil)
	old := items(3, now.Add(-30*24*time.Hour))
	for i := range old {
		old[i].GUID = fmt.Sprintf("old%d", i)
	}
	s.SaveEntries(ctx, a.ID, old, time.Time{}, nil)
	s.SaveEntries(ctx, b.ID, items(2, now.Add(-5*time.Hour)), time.Time{}, nil)

	list, _, _ := s.ListEntries(ctx, EntryFilter{FeedID: a.ID}, nil, 1)
	yes := true
	s.UpdateEntryState(ctx, list[0].ID, &yes, nil)

	since := now.Add(-7 * 24 * time.Hour)
	got, err := s.RecentUnread(ctx, EntryFilter{CategoryID: cat.ID}, since, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("category a: got %d entries, want 2 recent unread", len(got))
	}
	all, _ := s.RecentUnread(ctx, EntryFilter{}, since, 100)
	if len(all) != 4 {
		t.Fatalf("all feeds: got %d entries, want 4", len(all))
	}
}
