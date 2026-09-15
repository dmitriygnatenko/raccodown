package mysql

import (
	"context"
	"time"

	"raccodown/internal/storage/model"
)

// CreateSession inserts a session row.
func (s *Storage) CreateSession(ctx context.Context, session model.Session) error {
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO sessions (token, user_id, expires_at, created_at) VALUES (?, ?, ?, ?)`,
		session.Token, session.UserID, session.ExpiresAt, session.CreatedAt,
	)

	return err
}

// FindSessionByToken looks up a session by its token, returning sql.ErrNoRows when there's no match.
func (s *Storage) FindSessionByToken(ctx context.Context, token string) (model.Session, error) {
	m := model.Session{Token: token}

	err := s.DB.QueryRowContext(ctx,
		`SELECT user_id, expires_at, created_at FROM sessions WHERE token = ?`, token,
	).Scan(&m.UserID, &m.ExpiresAt, &m.CreatedAt)
	if err != nil {
		return model.Session{}, err
	}

	return m, nil
}

// DeleteSession removes a session by its token.
func (s *Storage) DeleteSession(ctx context.Context, token string) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM sessions WHERE token = ?`, token)

	return err
}

// DeleteExpiredSessions removes every session that expired before now, returning how many it
// removed.
func (s *Storage) DeleteExpiredSessions(ctx context.Context, now time.Time) (int, error) {
	res, err := s.DB.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < ?`, now)
	if err != nil {
		return 0, err
	}

	n, err := res.RowsAffected()

	return int(n), err
}
