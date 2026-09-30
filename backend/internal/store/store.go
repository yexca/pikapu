package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("already exists")
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	dsn := path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)" +
		"&_pragma=foreign_keys(1)&_pragma=synchronous(NORMAL)&_txlock=immediate"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4)
	db.SetConnMaxIdleTime(5 * time.Minute)

	s := &Store{db: db}
	if err := s.migrate(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

// migrations are applied in order; PRAGMA user_version records how many ran.
// Released entries are immutable: append a new entry for every schema change.
var migrations = []string{
	`
	CREATE TABLE categories (
		id         INTEGER PRIMARY KEY,
		name       TEXT NOT NULL UNIQUE COLLATE NOCASE,
		position   INTEGER NOT NULL DEFAULT 0,
		created_at INTEGER NOT NULL
	);

	CREATE TABLE feeds (
		id              INTEGER PRIMARY KEY,
		category_id     INTEGER REFERENCES categories(id) ON DELETE SET NULL,
		title           TEXT NOT NULL,
		feed_url        TEXT NOT NULL UNIQUE,
		site_url        TEXT NOT NULL DEFAULT '',
		description     TEXT NOT NULL DEFAULT '',
		etag            TEXT NOT NULL DEFAULT '',
		last_modified   TEXT NOT NULL DEFAULT '',
		last_fetched_at INTEGER,
		last_error      TEXT NOT NULL DEFAULT '',
		error_count     INTEGER NOT NULL DEFAULT 0,
		icon            BLOB,
		icon_mime       TEXT NOT NULL DEFAULT '',
		icon_checked_at INTEGER,
		created_at      INTEGER NOT NULL
	);
	CREATE INDEX idx_feeds_category ON feeds(category_id);

	CREATE TABLE entries (
		id           INTEGER PRIMARY KEY,
		feed_id      INTEGER NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,
		guid         TEXT NOT NULL,
		url          TEXT NOT NULL DEFAULT '',
		title        TEXT NOT NULL DEFAULT '',
		author       TEXT NOT NULL DEFAULT '',
		summary      TEXT NOT NULL DEFAULT '',
		content      TEXT NOT NULL DEFAULT '',
		image_url    TEXT NOT NULL DEFAULT '',
		published_at INTEGER NOT NULL,
		created_at   INTEGER NOT NULL,
		is_read      INTEGER NOT NULL DEFAULT 0,
		is_starred   INTEGER NOT NULL DEFAULT 0,
		UNIQUE (feed_id, guid)
	);
	CREATE INDEX idx_entries_published ON entries(published_at DESC, id DESC);
	CREATE INDEX idx_entries_feed ON entries(feed_id, published_at DESC, id DESC);
	CREATE INDEX idx_entries_unread ON entries(is_read, published_at DESC, id DESC);
	CREATE INDEX idx_entries_starred ON entries(is_starred, published_at DESC, id DESC);
	CREATE INDEX idx_entries_feed_read ON entries(feed_id, is_read);

	CREATE TABLE settings (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	`,
	// 2: machine-readable fetch failure code for localized UI messages.
	`ALTER TABLE feeds ADD COLUMN last_error_code TEXT NOT NULL DEFAULT '';`,
	// 3: keyword filters applied to newly fetched entries.
	`
	CREATE TABLE filters (
		id            INTEGER PRIMARY KEY,
		feed_id       INTEGER REFERENCES feeds(id) ON DELETE CASCADE,
		keywords      TEXT NOT NULL,
		match_content INTEGER NOT NULL DEFAULT 0,
		invert        INTEGER NOT NULL DEFAULT 0,
		action        TEXT NOT NULL,
		created_at    INTEGER NOT NULL
	);
	CREATE INDEX idx_filters_feed ON filters(feed_id);
	`,
}

func (s *Store) migrate(ctx context.Context) error {
	var version int
	if err := s.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	for i := version; i < len(migrations); i++ {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, migrations[i]); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: %w", i+1, err)
		}
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", i+1)); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func unix(t time.Time) int64 { return t.Unix() }

func fromUnix(v int64) time.Time { return time.Unix(v, 0).UTC() }

func nullTime(v sql.NullInt64) *time.Time {
	if !v.Valid {
		return nil
	}
	t := fromUnix(v.Int64)
	return &t
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
