package sqlite

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/require"

	"raccodown/internal/storage/model"
)

func fakeSessionToken() string { return gofakeit.UUID() }

func fakeSessionModel(userID uint64) model.Session {
	return model.Session{
		Token:     fakeSessionToken(),
		UserID:    userID,
		ExpiresAt: gofakeit.Date().UTC().Truncate(time.Second),
		CreatedAt: gofakeit.Date().UTC().Truncate(time.Second),
	}
}

// TestSession_CreateAndFind covers the round trip, including the FOREIGN KEY(user_id) requirement
// and the sql.ErrNoRows miss.
func TestSession_CreateAndFind(t *testing.T) {
	t.Parallel()

	s := newTestStorage(t)
	ctx := context.Background()

	user := createFakeUser(t, s)

	session := fakeSessionModel(user.ID)
	require.NoError(t, s.CreateSession(ctx, session))

	got, err := s.FindSessionByToken(ctx, session.Token)
	require.NoError(t, err)
	require.Equal(t, session, got)

	_, err = s.FindSessionByToken(ctx, "does-not-exist")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// TestSession_Delete covers removal, which is best-effort: deleting an unknown token isn't an error.
func TestSession_Delete(t *testing.T) {
	t.Parallel()

	s := newTestStorage(t)
	ctx := context.Background()

	user := createFakeUser(t, s)

	session := fakeSessionModel(user.ID)
	require.NoError(t, s.CreateSession(ctx, session))

	require.NoError(t, s.DeleteSession(ctx, session.Token))

	_, err := s.FindSessionByToken(ctx, session.Token)
	require.ErrorIs(t, err, sql.ErrNoRows)

	require.NoError(t, s.DeleteSession(ctx, "does-not-exist"))
}

// TestSession_DeleteExpired covers the sweep the hourly cleanup goroutine runs: only sessions
// expired before the given time are removed, and the count reflects how many.
func TestSession_DeleteExpired(t *testing.T) {
	t.Parallel()

	s := newTestStorage(t)
	ctx := context.Background()

	user := createFakeUser(t, s)

	now := time.Now().UTC()

	expired := fakeSessionModel(user.ID)
	expired.ExpiresAt = now.Add(-time.Hour)
	require.NoError(t, s.CreateSession(ctx, expired))

	live := fakeSessionModel(user.ID)
	live.ExpiresAt = now.Add(time.Hour)
	require.NoError(t, s.CreateSession(ctx, live))

	n, err := s.DeleteExpiredSessions(ctx, now)
	require.NoError(t, err)
	require.Equal(t, 1, n)

	_, err = s.FindSessionByToken(ctx, expired.Token)
	require.ErrorIs(t, err, sql.ErrNoRows)

	_, err = s.FindSessionByToken(ctx, live.Token)
	require.NoError(t, err)
}

// TestSession_CascadesOnUserDelete covers the sessions.user_id ON DELETE CASCADE: deleting a user
// takes their sessions with them.
func TestSession_CascadesOnUserDelete(t *testing.T) {
	t.Parallel()

	s := newTestStorage(t)
	ctx := context.Background()

	user := createFakeUser(t, s)

	session := fakeSessionModel(user.ID)
	require.NoError(t, s.CreateSession(ctx, session))

	_, err := s.DB.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, user.ID)
	require.NoError(t, err)

	_, err = s.FindSessionByToken(ctx, session.Token)
	require.ErrorIs(t, err, sql.ErrNoRows)
}
