package postgres

import (
	"context"
	"database/sql"
	"time"

	"raccodown/internal/storage/model"
)

const userColumns = `id, username, password_hash, settings, created_at, updated_at`

// scanUser reads the user column list, in the order every user query below selects it.
func scanUser(row *sql.Row) (model.User, error) {
	var m model.User
	if err := row.Scan(&m.ID, &m.Username, &m.PasswordHash, &m.Settings, &m.CreatedAt, &m.UpdatedAt); err != nil {
		return model.User{}, err
	}

	return m, nil
}

// FindUserByUsername looks up a user by their username, returning sql.ErrNoRows when there's no
// match.
func (s *Storage) FindUserByUsername(ctx context.Context, username string) (model.User, error) {
	return scanUser(s.DB.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE username = $1`, username,
	))
}

// FindUserByID looks up a user by id, returning sql.ErrNoRows when there's no match.
func (s *Storage) FindUserByID(ctx context.Context, id uint64) (model.User, error) {
	return scanUser(s.DB.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE id = $1`, id,
	))
}

// CreateUser inserts a user row and returns its new, database-assigned id. A taken username comes
// back wrapped in storageError.UniqueViolationError.
func (s *Storage) CreateUser(ctx context.Context, username, passwordHash string, createdAt, updatedAt time.Time) (uint64, error) {
	id, err := s.insertReturningID(ctx,
		`INSERT INTO users (username, password_hash, created_at, updated_at) VALUES ($1, $2, $3, $4) RETURNING id`,
		username, passwordHash, createdAt, updatedAt,
	)

	return id, wrapUnique(err)
}

// UpdateUserCredentials changes a user's username and/or password hash in one statement — a blank
// username or hash leaves that column unchanged. found is false if no user with this id existed. A
// taken username comes back wrapped in storageError.UniqueViolationError.
func (s *Storage) UpdateUserCredentials(
	ctx context.Context, id uint64, username, passwordHash string, updatedAt time.Time,
) (bool, error) {
	var (
		res sql.Result
		err error
	)

	switch {
	case username != "" && passwordHash != "":
		res, err = s.DB.ExecContext(ctx,
			`UPDATE users SET username = $1, password_hash = $2, updated_at = $3 WHERE id = $4`,
			username, passwordHash, updatedAt, id,
		)
	case username != "":
		res, err = s.DB.ExecContext(ctx,
			`UPDATE users SET username = $1, updated_at = $2 WHERE id = $3`, username, updatedAt, id,
		)
	case passwordHash != "":
		res, err = s.DB.ExecContext(ctx,
			`UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3`, passwordHash, updatedAt, id,
		)
	default:
		return true, nil
	}

	if err != nil {
		return false, wrapUnique(err)
	}

	return affected(res)
}

// CountUsers returns the total number of users.
func (s *Storage) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)

	return n, err
}

// GetUserSettings returns a user's settings JSON column, returning sql.ErrNoRows when there's no
// match.
func (s *Storage) GetUserSettings(ctx context.Context, id uint64) (model.UserSettings, error) {
	var settings model.UserSettings

	err := s.DB.QueryRowContext(ctx, `SELECT settings FROM users WHERE id = $1`, id).Scan(&settings)
	if err != nil {
		return model.UserSettings{}, err
	}

	return settings, nil
}

// UpdateUserSettings overwrites a user's settings JSON column. found is false if no user with this
// id existed.
func (s *Storage) UpdateUserSettings(
	ctx context.Context, id uint64, settings model.UserSettings, updatedAt time.Time,
) (bool, error) {
	res, err := s.DB.ExecContext(ctx,
		`UPDATE users SET settings = $1, updated_at = $2 WHERE id = $3`, settings, updatedAt, id,
	)
	if err != nil {
		return false, err
	}

	return affected(res)
}
