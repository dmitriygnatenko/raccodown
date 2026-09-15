package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/require"

	sqlitedriver "modernc.org/sqlite"

	storageError "raccodown/internal/storage/error"
)

// newTestStorage opens a fresh in-memory database and runs migrations against it — every adapter
// test in this package talks to a real SQLite engine rather than a mocked driver, so the actual SQL
// (and the schema it runs against) is what's under test.
func newTestStorage(t *testing.T) *Storage {
	t.Helper()

	db, err := sql.Open("sqlite", "file::memory:?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	// A shared in-memory database is torn down the moment its last connection closes — one
	// connection keeps the schema alive for every query the test issues.
	db.SetMaxOpenConns(1)

	s := &Storage{DB: db}
	require.NoError(t, Migrate(context.Background(), s))

	return s
}

// sqliteUniqueErr returns a real *sqlite.Error for a UNIQUE violation. modernc.org/sqlite doesn't
// export a constructor for its Error type (its fields are private) — running a real statement
// engineered to fail is the only way to get one.
var (
	sqliteErrOnce      sync.Once
	sqliteUniqueErrVal *sqlitedriver.Error
)

func sqliteUniqueErr() *sqlitedriver.Error {
	sqliteErrOnce.Do(func() {
		db, err := sql.Open("sqlite", "file::memory:")
		if err != nil {
			panic(err)
		}
		defer db.Close()

		if _, err := db.Exec(`CREATE TABLE uniq (val INTEGER UNIQUE)`); err != nil {
			panic(err)
		}

		if _, err := db.Exec(`INSERT INTO uniq (val) VALUES (1)`); err != nil {
			panic(err)
		}

		if _, err := db.Exec(`INSERT INTO uniq (val) VALUES (1)`); !errors.As(err, &sqliteUniqueErrVal) {
			panic("expected a *sqlite.Error for the unique violation")
		}
	})

	return sqliteUniqueErrVal
}

var errStub = errors.New(gofakeit.Sentence())

// TestAffected pins the translation from a driver Result to the "found?" answer the storage layer
// gives updates and deletes.
func TestAffected(t *testing.T) {
	t.Parallel()

	s := newTestStorage(t)

	t.Run("no rows touched means not found", func(t *testing.T) {
		t.Parallel()

		res, err := s.DB.ExecContext(context.Background(), `DELETE FROM users WHERE id = 999999`)
		require.NoError(t, err)

		found, err := affected(res)
		require.NoError(t, err)
		require.False(t, found)
	})

	t.Run("a touched row means found", func(t *testing.T) {
		t.Parallel()

		createFakeUser(t, s)

		res, err := s.DB.ExecContext(context.Background(), `DELETE FROM users`)
		require.NoError(t, err)

		found, err := affected(res)
		require.NoError(t, err)
		require.True(t, found)
	})
}

// TestWrapUnique covers the one error shape the repositories are allowed to recognize as "taken": a
// UNIQUE violation, normalized to storageError.UniqueViolationError with the driver error still
// wrapped inside. Every other error (including nil) has to pass through untouched.
func TestWrapUnique(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		err          error
		assertResult func(t *testing.T, in, got error)
	}{
		{
			name: "nil passes through",
			err:  nil,
			assertResult: func(t *testing.T, in, got error) {
				require.NoError(t, got)
			},
		},
		{
			name: "an unrelated error passes through",
			err:  errStub,
			assertResult: func(t *testing.T, in, got error) {
				require.ErrorIs(t, got, in)
				require.NotErrorIs(t, got, storageError.UniqueViolationError)
			},
		},
		{
			name: "a unique violation is wrapped",
			err:  sqliteUniqueErr(),
			assertResult: func(t *testing.T, in, got error) {
				require.ErrorIs(t, got, in)
				require.ErrorIs(t, got, storageError.UniqueViolationError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := wrapUnique(tt.err)
			tt.assertResult(t, tt.err, got)
		})
	}
}

// TestEnsureDatabase covers the one thing it promises: the parent directory exists afterwards.
func TestEnsureDatabase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setup     func(t *testing.T, tmp string)
		path      func(tmp string) string
		assertErr func(t *testing.T, err error)
	}{
		{
			name: "a missing parent directory is created",
			path: func(tmp string) string { return filepath.Join(tmp, "data", "nested", "app.db") },
			assertErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "an existing parent directory is left alone",
			path: func(tmp string) string { return filepath.Join(tmp, "app.db") },
			assertErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "a bare filename has no directory to create",
			path: func(tmp string) string { return "app.db" },
			assertErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "a file where the parent directory should be is an error",
			setup: func(t *testing.T, tmp string) {
				t.Helper()

				require.NoError(t, os.WriteFile(filepath.Join(tmp, "data"), nil, 0o600))
			},
			path: func(tmp string) string { return filepath.Join(tmp, "data", "app.db") },
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmp := t.TempDir()
			if tt.setup != nil {
				tt.setup(t, tmp)
			}

			path := tt.path(tmp)

			err := EnsureDatabase(Config{Path: path})
			tt.assertErr(t, err)

			if err != nil {
				return
			}

			info, statErr := os.Stat(filepath.Dir(path))
			require.NoError(t, statErr)
			require.True(t, info.IsDir())
		})
	}
}

// TestOpen checks that a usable pool comes back for a workable path, that the single-connection cap
// is applied, and that an unusable path fails at Open rather than at the first query.
func TestOpen(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		path      func(tmp string) string
		assertErr func(t *testing.T, err error)
	}{
		{
			name: "creates and opens the database file",
			path: func(tmp string) string { return filepath.Join(tmp, "app.db") },
			assertErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "a directory in place of the file is an error",
			path: func(tmp string) string { return tmp },
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
		{
			name: "a missing parent directory is an error",
			path: func(tmp string) string { return filepath.Join(tmp, "missing", "app.db") },
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := tt.path(t.TempDir())

			s, err := Open(Config{
				Path:            path,
				ConnMaxLifetime: time.Minute,
			})
			tt.assertErr(t, err)

			if err != nil {
				return
			}

			defer func() { _ = s.Close() }()

			require.Equal(t, 1, s.Stats().MaxOpenConnections)

			_, statErr := os.Stat(path)
			require.NoError(t, statErr, "Open() did not create the file")
		})
	}
}

// TestMigrate checks that the schema is actually usable afterwards, and that running it twice
// (goose's own idempotency) doesn't error.
func TestMigrate(t *testing.T) {
	t.Parallel()

	s := newTestStorage(t)

	require.NoError(t, Migrate(context.Background(), s))

	_, err := s.DB.ExecContext(context.Background(), `SELECT id FROM users LIMIT 1`)
	require.NoError(t, err)
	_, err = s.DB.ExecContext(context.Background(), `SELECT token FROM sessions LIMIT 1`)
	require.NoError(t, err)
	_, err = s.DB.ExecContext(context.Background(), `SELECT id FROM notes LIMIT 1`)
	require.NoError(t, err)
}
