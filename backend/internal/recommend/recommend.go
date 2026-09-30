// Package recommend ranks unread entries for the hub layout's "For you"
// picks. Ranking is a transparent heuristic over three signals: how fresh an
// entry is, how often the reader engages with its feed (store affinity), and
// how rarely the feed publishes. A per-feed penalty keeps one busy feed from
// filling every slot.
package recommend

import (
	"math"
	"time"

	"pikapu/internal/store"
)

// Window is how far back candidates and feed volumes are considered.
const Window = 7 * 24 * time.Hour

// Reason is a stable code explaining why an entry was picked; clients
// localize it.
type Reason string

const (
	ReasonFavorite Reason = "favorite_source"
	ReasonRare     Reason = "rare_source"
	ReasonFresh    Reason = "fresh"
)

const (
	// freshHalfLife is the age at which an entry's freshness halves.
	freshHalfLife = 48 * time.Hour
	// affinityScale is the affinity at which the feed boost reaches half of
	// affinityBoost; the boost saturates for feeds read very often.
	affinityScale = 5.0
	affinityBoost = 1.5
	// sameFeedPenalty multiplies an entry's score once per entry already
	// picked from its feed.
	sameFeedPenalty = 0.5
	imageBoost      = 1.1

	favoriteAffinity = 5.0
	rareRecent       = 3
	freshAge         = 3 * time.Hour
)

// Pick is a recommended entry and, when one stands out, why it was chosen.
type Pick struct {
	*store.Entry
	Reason Reason `json:"reason,omitempty"`
}

// Rank orders unread entries by recommendation score and returns at most
// limit of them. stats holds the Window-based FeedStat of each feed.
func Rank(entries []*store.Entry, stats map[int64]store.FeedStat, now time.Time, limit int) []Pick {
	base := make([]float64, len(entries))
	for i, e := range entries {
		base[i] = score(e, stats[e.FeedID], now)
	}

	out := []Pick{}
	picked := make([]bool, len(entries))
	perFeed := map[int64]int{}
	for len(out) < limit {
		best, bestScore := -1, 0.0
		for i, e := range entries {
			if picked[i] {
				continue
			}
			s := base[i] * math.Pow(sameFeedPenalty, float64(perFeed[e.FeedID]))
			// Strictly greater keeps the newer entry on ties, as input is newest first.
			if best < 0 || s > bestScore {
				best, bestScore = i, s
			}
		}
		if best < 0 {
			break
		}
		e := entries[best]
		picked[best] = true
		perFeed[e.FeedID]++
		out = append(out, Pick{Entry: e, Reason: reason(e, stats[e.FeedID], now)})
	}
	return out
}

func score(e *store.Entry, st store.FeedStat, now time.Time) float64 {
	age := max(now.Sub(e.PublishedAt), 0)
	fresh := math.Exp2(-age.Hours() / freshHalfLife.Hours())

	affinity := 1 + affinityBoost*st.Affinity/(st.Affinity+affinityScale)

	// A feed that posts once a week stands out more than one that posts
	// dozens of times a day.
	perDay := float64(st.Recent) / (Window.Hours() / 24)
	rarity := 0.4 + 0.6/math.Sqrt(1+perDay)

	s := fresh * affinity * rarity
	if e.ImageURL != "" {
		s *= imageBoost
	}
	return s
}

func reason(e *store.Entry, st store.FeedStat, now time.Time) Reason {
	switch {
	case st.Affinity >= favoriteAffinity:
		return ReasonFavorite
	case st.Recent <= rareRecent:
		return ReasonRare
	case now.Sub(e.PublishedAt) < freshAge:
		return ReasonFresh
	}
	return ""
}
