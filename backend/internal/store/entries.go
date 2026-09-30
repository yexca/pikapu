package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
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

// Verdict says how an unseen item should be stored.
type Verdict int

const (
	Keep     Verdict = iota // store as unread
	KeepRead                // store already marked as read
	Drop                    // do not store
)

// Triage decides the Verdict for an item; a nil Triage keeps everything.
type Triage func(EntryInput) Verdict

// SaveEntries inserts unseen items and refreshes the text of known ones.
// Unseen items published before skipBefore are dropped so that entries removed
// by retention cleanup do not come back; the zero time disables the check.
// triage runs only for unseen items, so known entries keep their state.
func (s *Store) SaveEntries(ctx context.Context, feedID int64, items []EntryInput, skipBefore time.Time, triage Triage) (int, error) {
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
		`INSERT INTO entries (feed_id, guid, url, title, author, summary, content, image_url, published_at, created_at, is_read)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
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
			verdict := Keep
			if triage != nil {
				verdict = triage(it)
			}
			if verdict == Drop {
				continue
			}
			if _, err := insert.ExecContext(ctx, feedID, it.GUID, it.URL, it.Title, it.Author,
				it.Summary, it.Content, it.ImageURL, unix(it.PublishedAt), now, boolInt(verdict == KeepRead)); err != nil {
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

// Interest added to a feed's affinity when one of its entries is read or
// starred individually. Bulk actions (mark all as read, filters) do not count.
const (
	readAffinity = 1.0
	starAffinity = 3.0

	// AffinityHalfLife is how long past interest in a feed takes to count half.
	AffinityHalfLife = 30 * 24 * time.Hour
)

// decayAffinity returns v, recorded at at, as it counts at now.
func decayAffinity(v float64, at, now time.Time) float64 {
	age := now.Sub(at)
	if v == 0 || age <= 0 {
		return v
	}
	return v * math.Exp2(-age.Hours()/AffinityHalfLife.Hours())
}

// UpdateEntryState sets the read and/or starred flags; nil leaves a flag as is.
// Marking an entry read or starred also raises its feed's affinity.
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

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var (
		feedID              int64
		wasRead, wasStarred bool
	)
	err = tx.QueryRowContext(ctx, `SELECT feed_id, is_read, is_starred FROM entries WHERE id = ?`, id).
		Scan(&feedID, &wasRead, &wasStarred)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	args = append(args, id)
	if _, err := tx.ExecContext(ctx,
		`UPDATE entries SET `+strings.Join(sets, ", ")+` WHERE id = ?`, args...); err != nil {
		return err
	}

	bump := 0.0
	if read != nil && *read && !wasRead {
		bump += readAffinity
	}
	if starred != nil && *starred && !wasStarred {
		bump += starAffinity
	}
	if bump > 0 {
		var (
			affinity float64
			at       int64
		)
		if err := tx.QueryRowContext(ctx, `SELECT affinity, affinity_at FROM feeds WHERE id = ?`, feedID).
			Scan(&affinity, &at); err != nil {
			return err
		}
		now := time.Now()
		affinity = decayAffinity(affinity, fromUnix(at), now) + bump
		if _, err := tx.ExecContext(ctx, `UPDATE feeds SET affinity = ?, affinity_at = ? WHERE id = ?`,
			affinity, unix(now), feedID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// RecentUnread returns up to limit unread entries matching f that were
// published at or after since, newest first.
func (s *Store) RecentUnread(ctx context.Context, f EntryFilter, since time.Time, limit int) ([]*Entry, error) {
	f.Unread = true
	where, args := f.where()
	args = append(args, unix(since), limit)
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+entryListColumns+` FROM entries WHERE `+where+
			` AND published_at >= ? ORDER BY published_at DESC, id DESC LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []*Entry{}
	for rows.Next() {
		e, err := scanEntry(rows, false)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// FeedStat is what recommendations know about a feed.
type FeedStat struct {
	// Affinity is the decayed interest in the feed at the time of the query.
	Affinity float64
	// Recent counts the feed's entries published since the query's window start.
	Recent int
}

// FeedStats returns a FeedStat for every feed, counting entries published at
// or after since and decaying affinity to now.
func (s *Store) FeedStats(ctx context.Context, since, now time.Time) (map[int64]FeedStat, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT f.id, f.affinity, f.affinity_at,
			(SELECT COUNT(*) FROM entries e WHERE e.feed_id = f.id AND e.published_at >= ?)
		 FROM feeds f`, unix(since))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[int64]FeedStat{}
	for rows.Next() {
		var (
			id       int64
			affinity float64
			at       int64
			recent   int
		)
		if err := rows.Scan(&id, &affinity, &at, &recent); err != nil {
			return nil, err
		}
		out[id] = FeedStat{Affinity: decayAffinity(affinity, fromUnix(at), now), Recent: recent}
	}
	return out, rows.Err()
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

// TriageUnread runs triage over the unread, unstarred entries of one feed (or
// of all feeds when feedID is nil), marking KeepRead ones as read and deleting
// Drop ones. It returns how many entries changed.
func (s *Store) TriageUnread(ctx context.Context, feedID *int64, triage Triage) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	query := `SELECT id, title, content FROM entries WHERE is_read = 0 AND is_starred = 0`
	var args []any
	if feedID != nil {
		query += ` AND feed_id = ?`
		args = append(args, *feedID)
	}
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	var read, drop []int64
	for rows.Next() {
		var (
			id int64
			it EntryInput
		)
		if err := rows.Scan(&id, &it.Title, &it.Content); err != nil {
			rows.Close()
			return 0, err
		}
		switch triage(it) {
		case KeepRead:
			read = append(read, id)
		case Drop:
			drop = append(drop, id)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	for _, id := range read {
		if _, err := tx.ExecContext(ctx, `UPDATE entries SET is_read = 1 WHERE id = ?`, id); err != nil {
			return 0, err
		}
	}
	for _, id := range drop {
		if _, err := tx.ExecContext(ctx, `DELETE FROM entries WHERE id = ?`, id); err != nil {
			return 0, err
		}
	}
	return int64(len(read) + len(drop)), tx.Commit()
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
