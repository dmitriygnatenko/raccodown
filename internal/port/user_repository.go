package port

//go:generate go tool mockgen -source=user_repository.go -destination=mocks/user_repository_mock.go -package=mocks

import (
	"context"

	"raccodown/internal/domain/entity"
)

// UserCreateRequest bundles the UserRepository.Create parameters.
type UserCreateRequest struct {
	Username     string
	PasswordHash string
}

// UserRepository persists the single account raccodown runs as.
type UserRepository interface {
	// FindByUsername returns a *domainerror.NotFoundError if no user with this (already-normalized)
	// username exists.
	FindByUsername(ctx context.Context, username string) (entity.User, error)
	// FindByID returns a *domainerror.NotFoundError if no user with this id exists.
	FindByID(ctx context.Context, id uint64) (entity.User, error)
	// Create returns a *domainerror.ConflictError if the username is already taken.
	Create(ctx context.Context, req UserCreateRequest) (entity.User, error)
	// UpdateCredentials changes a user's username and/or password hash in one atomic operation — a
	// blank username or hash leaves that column unchanged. Returns a *domainerror.ConflictError if a
	// non-blank username is already taken, and a *domainerror.NotFoundError if no user with this id
	// exists.
	UpdateCredentials(ctx context.Context, id uint64, username, passwordHash string) error
	// GetSettings returns a *domainerror.NotFoundError if no user with this id exists.
	GetSettings(ctx context.Context, id uint64) (entity.UserSettings, error)
	// UpdateSettings overwrites a user's saved UI settings (language, theme). Returns a
	// *domainerror.NotFoundError if no user with this id exists.
	UpdateSettings(ctx context.Context, id uint64, settings entity.UserSettings) error
	// Count returns the total number of users — used to decide whether to seed the demo account on
	// startup.
	Count(ctx context.Context) (int, error)
}
