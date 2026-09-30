package recommend

import (
	"testing"
	"time"

	"pikapu/internal/store"
)

var now = time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

func entry(id, feedID int64, age time.Duration) *store.Entry {
	return &store.Entry{ID: id, FeedID: feedID, PublishedAt: now.Add(-age)}
}

func ids(picks []Pick) []int64 {
	out := make([]int64, len(picks))
	for i, p := range picks {
		out[i] = p.ID
	}
	return out
}

func TestRankPrefersRareFeeds(t *testing.T) {
	entries := []*store.Entry{
		entry(1, 1, time.Hour), // busy feed
		entry(2, 2, time.Hour), // quiet feed
	}
	stats := map[int64]store.FeedStat{1: {Recent: 200}, 2: {Recent: 1}}

	picks := Rank(entries, stats, now, 2)
	if got := ids(picks); got[0] != 2 {
		t.Fatalf("order = %v, want the quiet feed first", got)
	}
	if picks[0].Reason != ReasonRare {
		t.Errorf("reason = %q, want %q", picks[0].Reason, ReasonRare)
	}
}

func TestRankPrefersFeedsTheReaderEngagesWith(t *testing.T) {
	entries := []*store.Entry{
		entry(1, 1, time.Hour),
		entry(2, 2, 2*time.Hour),
	}
	stats := map[int64]store.FeedStat{
		1: {Recent: 20},
		2: {Recent: 20, Affinity: 12},
	}

	picks := Rank(entries, stats, now, 2)
	if got := ids(picks); got[0] != 2 {
		t.Fatalf("order = %v, want the favorite feed first", got)
	}
	if picks[0].Reason != ReasonFavorite {
		t.Errorf("reason = %q, want %q", picks[0].Reason, ReasonFavorite)
	}
}

func TestRankPrefersFreshEntries(t *testing.T) {
	entries := []*store.Entry{
		entry(1, 1, time.Hour),
		entry(2, 1, 4*24*time.Hour),
	}
	stats := map[int64]store.FeedStat{1: {Recent: 30}}

	picks := Rank(entries, stats, now, 2)
	if got := ids(picks); got[0] != 1 {
		t.Fatalf("order = %v, want the newer entry first", got)
	}
	if picks[0].Reason != ReasonFresh || picks[1].Reason != "" {
		t.Errorf("reasons = %q, %q", picks[0].Reason, picks[1].Reason)
	}
}

func TestRankSpreadsPicksAcrossFeeds(t *testing.T) {
	var entries []*store.Entry
	for i := range 5 {
		entries = append(entries, entry(int64(i+1), 1, time.Duration(i)*time.Minute))
	}
	// Older and from a feed of the same volume, but the only one of its feed.
	entries = append(entries, entry(99, 2, 12*time.Hour))
	stats := map[int64]store.FeedStat{1: {Recent: 30}, 2: {Recent: 30}}

	got := ids(Rank(entries, stats, now, 3))
	if got[0] != 1 || got[1] != 99 {
		t.Fatalf("order = %v, want [1 99 ...]", got)
	}
}

func TestRankLimitAndEmpty(t *testing.T) {
	if got := Rank(nil, nil, now, 5); got == nil || len(got) != 0 {
		t.Fatalf("empty input = %#v, want an empty non-nil slice", got)
	}
	entries := []*store.Entry{entry(1, 1, 0), entry(2, 1, 0), entry(3, 1, 0)}
	if got := Rank(entries, nil, now, 2); len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
}
