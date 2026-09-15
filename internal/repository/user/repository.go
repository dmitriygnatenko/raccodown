// Package user implements port.UserRepository on top of the users table: it converts row models
// into domain entities and turns the storage layer's raw errors into domain errors — a missing row
// into *domainerror.NotFoundError, a username collision into *domainerror.ConflictError. Both are
// message-less; the use case supplies the text.
package user

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"raccodown/internal/domain/entity"
	domainerror "raccodown/internal/domain/error"
	"raccodown/internal/port"
	storageError "raccodown/internal/storage/error"
	"raccodown/internal/storage/model"
)

//go:generate go tool mockgen -source=repository.go -destination=mocks/storage_mock.go -package=mocks

// Storage is the slice of a driver adapter this repository uses — the users table and nothing
// else.
type Storage interface {
	// FindUserByUsername returns sql.ErrNoRows when no user has this (already-normalized) username.
	FindUserByUsername(ctx context.Context, username string) (model.User, error)
	// FindUserByID returns sql.ErrNoRows when no user has this id.
	FindUserByID(ctx context.Context, id uint64) (model.User, error)
	// CreateUser inserts a user row and returns its new, database-assigned id. A taken username
	// comes back wrapped in storageError.UniqueViolationError.
	CreateUser(ctx context.Context, username, passwordHash string, createdAt, updatedAt time.Time) (id uint64, err error)
	// UpdateUserCredentials changes a user's username and/or password hash in one statement — a
	// blank username or hash leaves that column unchanged. found is false if no user with this id
	// existed. A taken username comes back wrapped in storageError.UniqueViolationError.
	UpdateUserCredentials(
		ctx context.Context, id uint64, username, passwordHash string, updatedAt time.Time,
	) (found bool, err error)
	// GetUserSettings returns sql.ErrNoRows when no user has this id.
	GetUserSettings(ctx context.Context, id uint64) (model.UserSettings, error)
	// UpdateUserSettings overwrites a user's settings JSON column. found is false if no user with
	// this id existed.
	UpdateUserSettings(
		ctx context.Context, id uint64, settings model.UserSettings, updatedAt time.Time,
	) (found bool, err error)
	CountUsers(ctx context.Context) (int, error)
}

// Repository implements port.UserRepository.
type Repository struct {
	storage Storage
}

// New builds a Repository against s.
func New(s Storage) *Repository {
	return &Repository{storage: s}
}

// FindByUsername returns a message-less *domainerror.NotFoundError if no user with this username
// exists.
func (r *Repository) FindByUsername(ctx context.Context, username string) (entity.User, error) {
	m, err := r.storage.FindUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.User{}, &domainerror.NotFoundError{}
		}

		return entity.User{}, err
	}

	return m.ToEntity(), nil
}

// FindByID returns a message-less *domainerror.NotFoundError if no user with this id exists.
func (r *Repository) FindByID(ctx context.Context, id uint64) (entity.User, error) {
	m, err := r.storage.FindUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.User{}, &domainerror.NotFoundError{}
		}

		return entity.User{}, err
	}

	return m.ToEntity(), nil
}

// Create inserts a new user, reporting a message-less *domainerror.ConflictError if the username is
// already taken. The id is assigned by the database (autoincrement), not generated here.
func (r *Repository) Create(ctx context.Context, req port.UserCreateRequest) (entity.User, error) {
	now := time.Now().UTC()

	id, err := r.storage.CreateUser(ctx, req.Username, req.PasswordHash, now, now)
	if err != nil {
		if errors.Is(err, storageError.UniqueViolationError) {
			return entity.User{}, &domainerror.ConflictError{}
		}

		return entity.User{}, err
	}

	return entity.User{
		ID:           id,
		Username:     req.Username,
		PasswordHash: req.PasswordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// UpdateCredentials changes a user's username and/or password hash in one atomic operation — a
// blank username or hash leaves that column unchanged. Reports a message-less
// *domainerror.NotFoundError for an unknown id or a message-less *domainerror.ConflictError if the
// new username is already taken.
func (r *Repository) UpdateCredentials(ctx context.Context, id uint64, username, passwordHash string) error {
	found, err := r.storage.UpdateUserCredentials(ctx, id, username, passwordHash, time.Now().UTC())
	if err != nil {
		if errors.Is(err, storageError.UniqueViolationError) {
			return &domainerror.ConflictError{}
		}

		return err
	}

	if !found {
		return &domainerror.NotFoundError{}
	}

	return nil
}

// Count returns the total number of users.
func (r *Repository) Count(ctx context.Context) (int, error) {
	return r.storage.CountUsers(ctx)
}

// GetSettings returns a message-less *domainerror.NotFoundError if no user with this id exists.
func (r *Repository) GetSettings(ctx context.Context, id uint64) (entity.UserSettings, error) {
	m, err := r.storage.GetUserSettings(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.UserSettings{}, &domainerror.NotFoundError{}
		}

		return entity.UserSettings{}, err
	}

	return m.ToEntity(), nil
}

// UpdateSettings overwrites a user's saved UI settings, reporting a message-less
// *domainerror.NotFoundError for an unknown id.
func (r *Repository) UpdateSettings(ctx context.Context, id uint64, settings entity.UserSettings) error {
	found, err := r.storage.UpdateUserSettings(ctx, id, model.UserSettings{
		Language: settings.Language,
		Theme:    settings.Theme,
	}, time.Now().UTC())
	if err != nil {
		return err
	}

	if !found {
		return &domainerror.NotFoundError{}
	}

	return nil
}
