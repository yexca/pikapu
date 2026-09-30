package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// Account is the single admin account. PasswordHash is an encoded hash
// produced by the auth package; the store never sees plain passwords.
type Account struct {
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Session is a signed-in browser. Only a hash of its token is stored.
type Session struct {
	ID         int64     `json:"id"`
	UserAgent  string    `json:"user_agent"`
	IP         string    `json:"ip"`
	CreatedAt  time.Time `json:"created_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

func (s *Store) GetAccount(ctx context.Context) (*Account, error) {
	var a Account
	var created, updated int64
	err := s.db.QueryRowContext(ctx,
		`SELECT username, password_hash, created_at, updated_at FROM account WHERE id = 1`,
	).Scan(&a.Username, &a.PasswordHash, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	a.CreatedAt, a.UpdatedAt = fromUnix(created), fromUnix(updated)
	return &a, nil
}

func (s *Store) HasAccount(ctx context.Context) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM account`).Scan(&n)
	return n > 0, err
}

// CreateAccount stores the admin account. It returns ErrConflict when one
// already exists, so concurrent setup attempts cannot both succeed.
func (s *Store) CreateAccount(ctx context.Context, username, passwordHash string) error {
	now := unix(time.Now())
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO account (id, username, password_hash, created_at, updated_at)
		 VALUES (1, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING`,
		username, passwordHash, now, now)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrConflict
	}
	return nil
}

// UpdateAccount changes the username and, when passwordHash is not empty,
// the password. A password change also deletes every session except
// keepSessionID (0 keeps none), in the same transaction.
func (s *Store) UpdateAccount(ctx context.Context, username, passwordHash string, keepSessionID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := unix(time.Now())
	var res sql.Result
	if passwordHash == "" {
		res, err = tx.ExecContext(ctx,
			`UPDATE account SET username = ?, updated_at = ? WHERE id = 1`, username, now)
	} else {
		res, err = tx.ExecContext(ctx,
			`UPDATE account SET username = ?, password_hash = ?, updated_at = ? WHERE id = 1`,
			username, passwordHash, now)
	}
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	if passwordHash != "" {
		if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE id != ?`, keepSessionID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) CreateSession(ctx context.Context, tokenHash []byte, userAgent, ip string, expires time.Time) (*Session, error) {
	now := time.Now().UTC().Truncate(time.Second)
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (token_hash, user_agent, ip, created_at, last_seen_at, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		tokenHash, userAgent, ip, unix(now), unix(now), unix(expires))
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &Session{
		ID: id, UserAgent: userAgent, IP: ip,
		CreatedAt: now, LastSeenAt: now, ExpiresAt: fromUnix(unix(expires)),
	}, nil
}

const sessionColumns = `id, user_agent, ip, created_at, last_seen_at, expires_at`

func scanSession(row interface{ Scan(...any) error }) (*Session, error) {
	var ses Session
	var created, seen, expires int64
	if err := row.Scan(&ses.ID, &ses.UserAgent, &ses.IP, &created, &seen, &expires); err != nil {
		return nil, err
	}
	ses.CreatedAt, ses.LastSeenAt, ses.ExpiresAt = fromUnix(created), fromUnix(seen), fromUnix(expires)
	return &ses, nil
}

// SessionByToken returns the unexpired session with this token hash.
func (s *Store) SessionByToken(ctx context.Context, tokenHash []byte, now time.Time) (*Session, error) {
	ses, err := scanSession(s.db.QueryRowContext(ctx,
		`SELECT `+sessionColumns+` FROM sessions WHERE token_hash = ? AND expires_at > ?`,
		tokenHash, unix(now)))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return ses, err
}

// TouchSession records activity and extends the session's expiry.
func (s *Store) TouchSession(ctx context.Context, id int64, ip string, seen, expires time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE sessions SET ip = ?, last_seen_at = ?, expires_at = ? WHERE id = ?`,
		ip, unix(seen), unix(expires), id)
	return err
}

// ListSessions returns unexpired sessions, most recently used first.
func (s *Store) ListSessions(ctx context.Context, now time.Time) ([]*Session, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+sessionColumns+` FROM sessions WHERE expires_at > ?
		 ORDER BY last_seen_at DESC, id DESC`, unix(now))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Session{}
	for rows.Next() {
		ses, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ses)
	}
	return out, rows.Err()
}

func (s *Store) DeleteSession(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteOtherSessions deletes every session except keepID and returns how
// many were removed. keepID 0 deletes them all.
func (s *Store) DeleteOtherSessions(ctx context.Context, keepID int64) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id != ?`, keepID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *Store) DeleteExpiredSessions(ctx context.Context, now time.Time) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= ?`, unix(now))
	return err
}
