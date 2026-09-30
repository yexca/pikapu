package store

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
)

// Settings are the user-adjustable options persisted in the settings table.
type Settings struct {
	RefreshIntervalMinutes int `json:"refresh_interval_minutes"`
	RetentionDays          int `json:"retention_days"`
}

var DefaultSettings = Settings{
	RefreshIntervalMinutes: 30,
	RetentionDays:          90,
}

func (s *Store) GetSetting(ctx context.Context, key string) (string, error) {
	var v string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return v, err
}

func (s *Store) SetSetting(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO settings (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

func (s *Store) intSetting(ctx context.Context, key string, fallback int) (int, error) {
	v, err := s.GetSetting(ctx, key)
	if errors.Is(err, ErrNotFound) {
		return fallback, nil
	}
	if err != nil {
		return 0, err
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback, nil
	}
	return n, nil
}

func (s *Store) GetSettings(ctx context.Context) (Settings, error) {
	var out Settings
	var err error
	if out.RefreshIntervalMinutes, err = s.intSetting(ctx, "refresh_interval_minutes", DefaultSettings.RefreshIntervalMinutes); err != nil {
		return out, err
	}
	if out.RetentionDays, err = s.intSetting(ctx, "retention_days", DefaultSettings.RetentionDays); err != nil {
		return out, err
	}
	return out, nil
}

func (s *Store) SaveSettings(ctx context.Context, v Settings) error {
	if err := s.SetSetting(ctx, "refresh_interval_minutes", strconv.Itoa(v.RefreshIntervalMinutes)); err != nil {
		return err
	}
	return s.SetSetting(ctx, "retention_days", strconv.Itoa(v.RetentionDays))
}
