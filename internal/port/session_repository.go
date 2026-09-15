package port

//go:generate go tool mockgen -source=session_repository.go -destination=mocks/session_repository_mock.go -package=mocks

import (
	"context"
	"time"

	"raccodown/internal/domain/entity"
)

// SessionRepository persists login sessions.
type SessionRepository interface {
	Create(ctx context.Context, session entity.Session) error
	// FindByToken returns a *domainerror.NotFoundError if token doesn't map to a live session.
	FindByToken(ctx context.Context, token string) (entity.Session, error)
	// Delete is a no-op (not an error) if token doesn't map to a live session — see
	// usecase/auth/logout, which is best-effort by design.
	Delete(ctx context.Context, token string) error
	// DeleteExpired removes every session whose expiry is before now, returning how many it removed.
	DeleteExpired(ctx context.Context, now time.Time) (int, error)
}
