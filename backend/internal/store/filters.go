package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

// Filter actions.
const (
	FilterMarkRead = "mark_read"
	FilterSkip     = "skip"
)

// Filter is a keyword rule applied to new entries of one feed, or of every
// feed when FeedID is nil. Matching lives in package filter.
type Filter struct {
	ID       int64    `json:"id"`
	FeedID   *int64   `json:"feed_id"`
	Keywords []string `json:"keywords"`
	// MatchContent also searches the article text, not just the title.
	MatchContent bool `json:"match_content"`
	// Invert applies the action when none of the keywords appear.
	Invert    bool      `json:"invert"`
	Action    string    `json:"action"`
	CreatedAt time.Time `json:"created_at"`
}

// Keywords are stored one per line; they never contain line breaks.
const filterColumns = `id, feed_id, keywords, match_content, invert, action, created_at`

func scanFilter(row scanner) (*Filter, error) {
	var (
		f         Filter
		feedID    sql.NullInt64
		keywords  string
		createdAt int64
	)
	if err := row.Scan(&f.ID, &feedID, &keywords, &f.MatchContent, &f.Invert, &f.Action, &createdAt); err != nil {
		return nil, err
	}
	if feedID.Valid {
		f.FeedID = &feedID.Int64
	}
	f.Keywords = strings.Split(keywords, "\n")
	f.CreatedAt = fromUnix(createdAt)
	return &f, nil
}

func (s *Store) queryFilters(ctx context.Context, query string, args ...any) ([]*Filter, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []*Filter{}
	for rows.Next() {
		f, err := scanFilter(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// ListFilters returns every filter, rules for all feeds first.
func (s *Store) ListFilters(ctx context.Context) ([]*Filter, error) {
	return s.queryFilters(ctx,
		`SELECT `+filterColumns+` FROM filters ORDER BY feed_id IS NOT NULL, id`)
}

// FiltersForFeed returns the filters that apply to a feed: its own and the
// ones for all feeds.
func (s *Store) FiltersForFeed(ctx context.Context, feedID int64) ([]*Filter, error) {
	return s.queryFilters(ctx,
		`SELECT `+filterColumns+` FROM filters WHERE feed_id IS NULL OR feed_id = ? ORDER BY id`, feedID)
}

func (s *Store) GetFilter(ctx context.Context, id int64) (*Filter, error) {
	f, err := scanFilter(s.db.QueryRowContext(ctx,
		`SELECT `+filterColumns+` FROM filters WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return f, err
}

func (s *Store) CreateFilter(ctx context.Context, f *Filter) error {
	f.CreatedAt = time.Now().UTC().Truncate(time.Second)
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO filters (feed_id, keywords, match_content, invert, action, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		f.FeedID, strings.Join(f.Keywords, "\n"), boolInt(f.MatchContent), boolInt(f.Invert), f.Action, unix(f.CreatedAt))
	if err != nil {
		return err
	}
	f.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) UpdateFilter(ctx context.Context, f *Filter) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE filters SET feed_id = ?, keywords = ?, match_content = ?, invert = ?, action = ? WHERE id = ?`,
		f.FeedID, strings.Join(f.Keywords, "\n"), boolInt(f.MatchContent), boolInt(f.Invert), f.Action, f.ID)
	if err != nil {
		return err
	}
	return requireRow(res)
}

func (s *Store) DeleteFilter(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM filters WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return requireRow(res)
}
