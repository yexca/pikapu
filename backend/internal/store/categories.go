package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

type Category struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Position int    `json:"position"`
}

func (s *Store) ListCategories(ctx context.Context) ([]Category, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, position FROM categories ORDER BY position, name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Category{}
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Position); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) GetCategory(ctx context.Context, id int64) (*Category, error) {
	var c Category
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, position FROM categories WHERE id = ?`, id).
		Scan(&c.ID, &c.Name, &c.Position)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &c, err
}

func (s *Store) CategoryByName(ctx context.Context, name string) (*Category, error) {
	var c Category
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, position FROM categories WHERE name = ?`, name).
		Scan(&c.ID, &c.Name, &c.Position)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &c, err
}

func (s *Store) CreateCategory(ctx context.Context, name string) (*Category, error) {
	var pos int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(position), -1) + 1 FROM categories`).Scan(&pos); err != nil {
		return nil, err
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO categories (name, position, created_at) VALUES (?, ?, ?)`,
		name, pos, unix(time.Now()))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Category{ID: id, Name: name, Position: pos}, nil
}

// EnsureCategory returns the category with the given name, creating it if needed.
func (s *Store) EnsureCategory(ctx context.Context, name string) (*Category, error) {
	if c, err := s.CategoryByName(ctx, name); err == nil {
		return c, nil
	} else if !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	return s.CreateCategory(ctx, name)
}

func (s *Store) RenameCategory(ctx context.Context, id int64, name string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE categories SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	}
	return requireRow(res)
}

// ReorderCategories assigns positions following the order of ids.
func (s *Store) ReorderCategories(ctx context.Context, ids []int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for i, id := range ids {
		if _, err := tx.ExecContext(ctx, `UPDATE categories SET position = ? WHERE id = ?`, i, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) DeleteCategory(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM categories WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return requireRow(res)
}

func requireRow(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}
