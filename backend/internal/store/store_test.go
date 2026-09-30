package store

import (
	"context"
	"fmt"
	"path/filepath"
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

	n, err := s.SaveEntries(ctx, f.ID, items(3, now), time.Time{})
	if err != nil || n != 3 {
		t.Fatalf("first save: n=%d err=%v", n, err)
	}

	changed := items(3, now)
	changed[0].Title = "Updated"
	n, err = s.SaveEntries(ctx, f.ID, changed, time.Time{})
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

	n, err := s.SaveEntries(ctx, f.ID, items(2, old), time.Now().AddDate(0, 0, -30))
	if err != nil || n != 0 {
		t.Fatalf("old items should be skipped: n=%d err=%v", n, err)
	}
}

func TestListEntriesPagination(t *testing.T) {
	ctx := context.Background()
	s := openTest(t)
	f := newFeed(t, s, "https://feed-a.example.com/feed", nil)
	start := time.Now().Add(-48 * time.Hour).UTC()
	if _, err := s.SaveEntries(ctx, f.ID, items(25, start), time.Time{}); err != nil {
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
	s.SaveEntries(ctx, a.ID, items(3, now), time.Time{})
	s.SaveEntries(ctx, b.ID, items(2, now), time.Time{})

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
	}, time.Time{})

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
	s.SaveEntries(ctx, f.ID, items(3, old), time.Time{})
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
