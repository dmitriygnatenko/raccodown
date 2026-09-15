// Package sqlite is the SQLite driver adapter: it implements every operation the repositories in
// internal/repository ask for, in SQLite's own dialect ("?" placeholders). The queries live next to
// this file, one file per table group (note.go, user.go, session.go).
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"modernc.org/sqlite"

	storageError "raccodown/internal/storage/error"
)

// Storage wraps the connection pool the queries run against. It satisfies each repository's own
// Storage interface; internal/app is where a single one of these is handed to all of them.
type Storage struct {
	*sql.DB
}

// wrapUnique normalizes a UNIQUE constraint violation into storageError.UniqueViolationError, so a
// repository can recognize it without knowing anything about SQLite. Any other error (including nil)
// passes through.
//
// Matching is done on the error message rather than sqlite.Error.Code(): a plain UNIQUE index and a
// composite PRIMARY KEY collision report different extended result codes, but both — and every other
// row-uniqueness violation — share the same "UNIQUE constraint failed" message text, which is the
// stable thing to match on.
func wrapUnique(err error) error {
	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) && strings.Contains(sqliteErr.Error(), "UNIQUE constraint failed") {
		return fmt.Errorf("%w: %w", storageError.UniqueViolationError, err)
	}

	return err
}

// affected reports whether a statement touched any row, which is how the storage layer answers
// "found?" for updates and deletes.
func affected(res sql.Result) (bool, error) {
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	return n > 0, nil
}

// insertReturningID runs an INSERT and returns the new row's id via LastInsertId — used by the
// users table. notes is autoincrement too, but CreateNote can't use this helper: it needs to run
// inside the same transaction as the tag inserts, so it reads LastInsertId off the tx-scoped Result
// directly instead (see note.go).
func (s *Storage) insertReturningID(ctx context.Context, query string, args ...any) (uint64, error) {
	res, err := s.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()

	return uint64(id), err
}

// EnsureDatabase creates the parent directory of the database file if it doesn't exist yet — the
// driver creates the file itself on first connection.
func EnsureDatabase(cfg Config) error {
	if dir := filepath.Dir(cfg.Path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating directory %q for the SQLite file: %w", dir, err)
		}
	}

	return nil
}

// Open opens the connection pool for the database file and wraps it in Storage.
func Open(cfg Config) (*Storage, error) {
	raw, err := sql.Open("sqlite", cfg.dsn())
	if err != nil {
		return nil, fmt.Errorf("opening the database: %w", err)
	}

	raw.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	// SQLite doesn't tolerate concurrent writers — multiple simultaneous connections just produce
	// "database is locked" under load.
	raw.SetMaxOpenConns(1)

	if err := raw.Ping(); err != nil {
		return nil, fmt.Errorf("checking the database connection: %w", err)
	}

	return &Storage{DB: raw}, nil
}

// Close closes the underlying connection pool.
func (s *Storage) Close() error {
	return s.DB.Close()
}
