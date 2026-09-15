// Package session implements port.SessionRepository on top of the sessions table.
package session

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"raccodown/internal/domain/entity"
	domainerror "raccodown/internal/domain/error"
	"raccodown/internal/storage/model"
)

//go:generate go tool mockgen -source=repository.go -destination=mocks/storage_mock.go -package=mocks

// Storage is the slice of a driver adapter this repository uses — the sessions table and nothing
// else.
type Storage interface {
	CreateSession(ctx context.Context, session model.Session) error
	// FindSessionByToken returns sql.ErrNoRows when no session has this token.
	FindSessionByToken(ctx context.Context, token string) (model.Session, error)
	DeleteSession(ctx context.Context, token string) error
	DeleteExpiredSessions(ctx context.Context, now time.Time) (int, error)
}

// Repository implements port.SessionRepository.
type Repository struct {
	storage Storage
}

// New builds a Repository against s.
func New(s Storage) *Repository {
	return &Repository{storage: s}
}

// Create inserts a session, stamping CreatedAt.
func (r *Repository) Create(ctx context.Context, session entity.Session) error {
	session.CreatedAt = time.Now().UTC()

	return r.storage.CreateSession(ctx, model.Session{
		Token:     session.Token,
		UserID:    session.UserID,
		ExpiresAt: session.ExpiresAt,
		CreatedAt: session.CreatedAt,
	})
}

// FindByToken returns a message-less *domainerror.NotFoundError if token doesn't map to a live
// session.
func (r *Repository) FindByToken(ctx context.Context, token string) (entity.Session, error) {
	m, err := r.storage.FindSessionByToken(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.Session{}, &domainerror.NotFoundError{}
		}

		return entity.Session{}, err
	}

	return m.ToEntity(), nil
}

// Delete removes a session. Deleting an unknown token is not an error — see usecase/auth/logout,
// which is best-effort by design.
func (r *Repository) Delete(ctx context.Context, token string) error {
	return r.storage.DeleteSession(ctx, token)
}

// DeleteExpired removes every session whose expiry is before now, returning how many it removed.
func (r *Repository) DeleteExpired(ctx context.Context, now time.Time) (int, error) {
	return r.storage.DeleteExpiredSessions(ctx, now)
}
