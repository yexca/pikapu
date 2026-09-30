package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Entry struct {
	ID          int64     `json:"id"`
	FeedID      int64     `json:"feed_id"`
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Author      string    `json:"author"`
	Summary     string    `json:"summary"`
	Content     string    `json:"content,omitempty"`
	ImageURL    string    `json:"image_url"`
	PublishedAt time.Time `json:"published_at"`
	IsRead      bool      `json:"is_read"`
	IsStarred   bool      `json:"is_starred"`
}

// EntryInput is a parsed feed item ready to be stored.
type EntryInput struct {
	GUID        string
	URL         string
	Title       string
	Author      string
	Summary     string
	Content     string
	ImageURL    string
	PublishedAt time.Time
}

// SaveEntries inserts unseen items and refreshes the text of known ones.
// Unseen items published before skipBefore are dropped so that entries removed
// by retention cleanup do not come back; the zero time disables the check.
func (s *Store) SaveEntries(ctx context.Context, feedID int64, items []EntryInput, skipBefore time.Time) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	lookup, err := tx.PrepareContext(ctx, `SELECT id FROM entries WHERE feed_id = ? AND guid = ?`)
	if err != nil {
		return 0, err
	}
	defer lookup.Close()

	insert, err := tx.PrepareContext(ctx,
		`INSERT INTO entries (feed_id, guid, url, title, author, summary, content, image_url, published_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, err
	}
	defer insert.Close()

	update, err := tx.PrepareContext(ctx,
		`UPDATE entries SET url = ?1, title = ?2, author = ?3, summary = ?4, content = ?5, image_url = ?6
		 WHERE id = ?7 AND (url != ?1 OR title != ?2 OR author != ?3 OR summary != ?4 OR content != ?5 OR image_url != ?6)`)
	if err != nil {
		return 0, err
	}
	defer update.Close()

	now := unix(time.Now())
	inserted := 0
	seen := make(map[string]bool, len(items))
	for _, it := range items {
		if seen[it.GUID] {
			continue
		}
		seen[it.GUID] = true

		var id int64
		err := lookup.QueryRowContext(ctx, feedID, it.GUID).Scan(&id)
		switch {
		case err == nil:
			if _, err := update.ExecContext(ctx, it.URL, it.Title, it.Author, it.Summary, it.Content, it.ImageURL, id); err != nil {
				return 0, err
			}
		case errors.Is(err, sql.ErrNoRows):
			if !skipBefore.IsZero() && it.PublishedAt.Before(skipBefore) {
				continue
			}
			if _, err := insert.ExecContext(ctx, feedID, it.GUID, it.URL, it.Title, it.Author,
				it.Summary, it.Content, it.ImageURL, unix(it.PublishedAt), now); err != nil {
				return 0, err
			}
			inserted++
		default:
			return 0, err
		}
	}
	return inserted, tx.Commit()
}

type EntryFilter struct {
	FeedID     int64
	CategoryID int64
	Starred    bool
	Unread     bool
	Query      string
}

func (f EntryFilter) where() (string, []any) {
	conds := []string{"1 = 1"}
	var args []any
	if f.FeedID > 0 {
		conds = append(conds, "feed_id = ?")
		args = append(args, f.FeedID)
	}
	if f.CategoryID > 0 {
		conds = append(conds, "feed_id IN (SELECT id FROM feeds WHERE category_id = ?)")
		args = append(args, f.CategoryID)
	}
	if f.Starred {
		conds = append(conds, "is_starred = 1")
	}
	if f.Unread {
		conds = append(conds, "is_read = 0")
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		like := "%" + escapeLike(q) + "%"
		conds = append(conds, `(title LIKE ? ESCAPE '\' OR content LIKE ? ESCAPE '\')`)
		args = append(args, like, like)
	}
	return strings.Join(conds, " AND "), args
}

func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

// Cursor marks a position in the newest-first entry ordering.
type Cursor struct {
	PublishedAt int64
	ID          int64
}

func (c Cursor) String() string { return fmt.Sprintf("%d_%d", c.PublishedAt, c.ID) }

func ParseCursor(s string) (*Cursor, error) {
	a, b, ok := strings.Cut(s, "_")
	if !ok {
		return nil, errors.New("invalid cursor")
	}
	ts, err1 := strconv.ParseInt(a, 10, 64)
	id, err2 := strconv.ParseInt(b, 10, 64)
	if err1 != nil || err2 != nil {
		return nil, errors.New("invalid cursor")
	}
	return &Cursor{PublishedAt: ts, ID: id}, nil
}

const entryListColumns = `id, feed_id, url, title, author, summary, image_url, published_at, is_read, is_starred`

func scanEntry(row scanner, withContent bool) (*Entry, error) {
	var (
		e         Entry
		published int64
	)
	dest := []any{&e.ID, &e.FeedID, &e.URL, &e.Title, &e.Author, &e.Summary, &e.ImageURL, &published, &e.IsRead, &e.IsStarred}
	if withContent {
		dest = append(dest, &e.Content)
	}
	if err := row.Scan(dest...); err != nil {
		return nil, err
	}
	e.PublishedAt = fromUnix(published)
	return &e, nil
}

// ListEntries returns up to limit entries after the cursor, newest first,
// along with the cursor for the following page (nil when exhausted).
func (s *Store) ListEntries(ctx context.Context, f EntryFilter, after *Cursor, limit int) ([]*Entry, *Cursor, error) {
	where, args := f.where()
	if after != nil {
		where += " AND (published_at < ? OR (published_at = ? AND id < ?))"
		args = append(args, after.PublishedAt, after.PublishedAt, after.ID)
	}
	args = append(args, limit+1)

	rows, err := s.db.QueryContext(ctx,
		`SELECT `+entryListColumns+` FROM entries WHERE `+where+
			` ORDER BY published_at DESC, id DESC LIMIT ?`, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	out := []*Entry{}
	for rows.Next() {
		e, err := scanEntry(rows, false)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	var next *Cursor
	if len(out) > limit {
		out = out[:limit]
		last := out[len(out)-1]
		next = &Cursor{PublishedAt: last.PublishedAt.Unix(), ID: last.ID}
	}
	return out, next, nil
}

func (s *Store) GetEntry(ctx context.Context, id int64) (*Entry, error) {
	e, err := scanEntry(s.db.QueryRowContext(ctx,
		`SELECT `+entryListColumns+`, content FROM entries WHERE id = ?`, id), true)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return e, err
}

// UpdateEntryState sets the read and/or starred flags; nil leaves a flag as is.
func (s *Store) UpdateEntryState(ctx context.Context, id int64, read, starred *bool) error {
	sets := []string{}
	args := []any{}
	if read != nil {
		sets = append(sets, "is_read = ?")
		args = append(args, boolInt(*read))
	}
	if starred != nil {
		sets = append(sets, "is_starred = ?")
		args = append(args, boolInt(*starred))
	}
	if len(sets) == 0 {
		return nil
	}
	args = append(args, id)
	res, err := s.db.ExecContext(ctx,
		`UPDATE entries SET `+strings.Join(sets, ", ")+` WHERE id = ?`, args...)
	if err != nil {
		return err
	}
	return requireRow(res)
}

// MarkAllRead marks every unread entry matching the filter as read.
func (s *Store) MarkAllRead(ctx context.Context, f EntryFilter) (int64, error) {
	f.Unread = true
	where, args := f.where()
	res, err := s.db.ExecContext(ctx, `UPDATE entries SET is_read = 1 WHERE `+where, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

type Counters struct {
	Unread  int           `json:"unread"`
	Starred int           `json:"starred"`
	Feeds   map[int64]int `json:"feeds"`
}

func (s *Store) Counters(ctx context.Context) (*Counters, error) {
	c := &Counters{Feeds: map[int64]int{}}
	rows, err := s.db.QueryContext(ctx,
		`SELECT feed_id, COUNT(*) FROM entries WHERE is_read = 0 GROUP BY feed_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		c.Feeds[id] = n
		c.Unread += n
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM entries WHERE is_starred = 1`).Scan(&c.Starred)
	return c, err
}

// DeleteOldEntries removes read, unstarred entries published before cutoff.
func (s *Store) DeleteOldEntries(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM entries WHERE is_read = 1 AND is_starred = 0 AND published_at < ?`, unix(cutoff))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
