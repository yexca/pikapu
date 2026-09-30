package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type Feed struct {
	ID            int64      `json:"id"`
	CategoryID    *int64     `json:"category_id"`
	Title         string     `json:"title"`
	FeedURL       string     `json:"feed_url"`
	SiteURL       string     `json:"site_url"`
	Description   string     `json:"description"`
	LastFetchedAt *time.Time `json:"last_fetched_at"`
	LastError     string     `json:"last_error"`
	LastErrorCode string     `json:"last_error_code"`
	ErrorCount    int        `json:"error_count"`
	CreatedAt     time.Time  `json:"created_at"`

	ETag          string     `json:"-"`
	LastModified  string     `json:"-"`
	IconCheckedAt *time.Time `json:"-"`
}

const feedColumns = `id, category_id, title, feed_url, site_url, description,
	last_fetched_at, last_error, last_error_code, error_count, created_at, etag, last_modified, icon_checked_at`

type scanner interface{ Scan(dest ...any) error }

func scanFeed(row scanner) (*Feed, error) {
	var (
		f           Feed
		categoryID  sql.NullInt64
		lastFetched sql.NullInt64
		iconChecked sql.NullInt64
		createdAt   int64
	)
	err := row.Scan(&f.ID, &categoryID, &f.Title, &f.FeedURL, &f.SiteURL, &f.Description,
		&lastFetched, &f.LastError, &f.LastErrorCode, &f.ErrorCount, &createdAt, &f.ETag, &f.LastModified, &iconChecked)
	if err != nil {
		return nil, err
	}
	if categoryID.Valid {
		f.CategoryID = &categoryID.Int64
	}
	f.LastFetchedAt = nullTime(lastFetched)
	f.IconCheckedAt = nullTime(iconChecked)
	f.CreatedAt = fromUnix(createdAt)
	return &f, nil
}

func (s *Store) ListFeeds(ctx context.Context) ([]*Feed, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+feedColumns+` FROM feeds ORDER BY title COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []*Feed{}
	for rows.Next() {
		f, err := scanFeed(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (s *Store) GetFeed(ctx context.Context, id int64) (*Feed, error) {
	f, err := scanFeed(s.db.QueryRowContext(ctx,
		`SELECT `+feedColumns+` FROM feeds WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return f, err
}

func (s *Store) FeedByURL(ctx context.Context, feedURL string) (*Feed, error) {
	f, err := scanFeed(s.db.QueryRowContext(ctx,
		`SELECT `+feedColumns+` FROM feeds WHERE feed_url = ?`, feedURL))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return f, err
}

func (s *Store) CreateFeed(ctx context.Context, f *Feed) error {
	f.CreatedAt = time.Now().UTC().Truncate(time.Second)
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO feeds (category_id, title, feed_url, site_url, description, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		f.CategoryID, f.Title, f.FeedURL, f.SiteURL, f.Description, unix(f.CreatedAt))
	if err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	}
	f.ID, _ = res.LastInsertId()
	return nil
}

// UpdateFeed changes the user-editable fields of a feed.
func (s *Store) UpdateFeed(ctx context.Context, id int64, title, feedURL string, categoryID *int64) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE feeds SET title = ?, feed_url = ?, category_id = ? WHERE id = ?`,
		title, feedURL, categoryID, id)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	}
	return requireRow(res)
}

func (s *Store) DeleteFeed(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM feeds WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return requireRow(res)
}

// FetchSuccess records a successful fetch. Empty siteURL/description leave
// the stored values untouched.
func (s *Store) FetchSuccess(ctx context.Context, id int64, etag, lastModified, siteURL, description string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE feeds SET
			etag = ?, last_modified = ?,
			site_url = CASE WHEN ? != '' THEN ? ELSE site_url END,
			description = CASE WHEN ? != '' THEN ? ELSE description END,
			last_fetched_at = ?, last_error = '', last_error_code = '', error_count = 0
		 WHERE id = ?`,
		etag, lastModified, siteURL, siteURL, description, description, unix(time.Now()), id)
	return err
}

// FetchNotModified records a 304 response.
func (s *Store) FetchNotModified(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE feeds SET last_fetched_at = ?, last_error = '', last_error_code = '', error_count = 0 WHERE id = ?`,
		unix(time.Now()), id)
	return err
}

// FetchFailure records a failed fetch with its failure code and English detail.
func (s *Store) FetchFailure(ctx context.Context, id int64, code, msg string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE feeds SET last_fetched_at = ?, last_error = ?, last_error_code = ?, error_count = error_count + 1 WHERE id = ?`,
		unix(time.Now()), msg, code, id)
	return err
}

func (s *Store) GetFeedIcon(ctx context.Context, id int64) (data []byte, mime string, err error) {
	err = s.db.QueryRowContext(ctx,
		`SELECT icon, icon_mime FROM feeds WHERE id = ?`, id).Scan(&data, &mime)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", ErrNotFound
	}
	return data, mime, err
}

// SetFeedIcon stores an icon; a nil data slice records a failed lookup so it
// is not retried on every request.
func (s *Store) SetFeedIcon(ctx context.Context, id int64, data []byte, mime string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE feeds SET icon = ?, icon_mime = ?, icon_checked_at = ? WHERE id = ?`,
		data, mime, unix(time.Now()), id)
	return err
}

// MarkIconChecked bumps the check time without replacing an existing icon.
func (s *Store) MarkIconChecked(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE feeds SET icon_checked_at = ? WHERE id = ?`, unix(time.Now()), id)
	return err
}
