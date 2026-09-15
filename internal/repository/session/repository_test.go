package session

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"raccodown/internal/domain/entity"
	domainerror "raccodown/internal/domain/error"
	"raccodown/internal/repository/session/mocks"
	"raccodown/internal/storage/model"
)

// newRepo returns a Repository wired to a fresh MockStorage; any call a test doesn't stub via
// EXPECT() fails it.
func newRepo(t *testing.T) (*Repository, *mocks.MockStorage) {
	t.Helper()

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	m := mocks.NewMockStorage(mc)

	return New(m), m
}

func fakeToken() string  { return gofakeit.UUID() }
func fakeUserID() uint64 { return uint64(gofakeit.Number(1, 1_000_000)) }

var errStub = errors.New(gofakeit.Sentence())

// TestRepository_Create covers the insert, including stamping CreatedAt.
func TestRepository_Create(t *testing.T) {
	t.Parallel()

	session := entity.Session{
		Token:     fakeToken(),
		UserID:    fakeUserID(),
		ExpiresAt: gofakeit.Date().UTC(),
	}

	t.Run("stores the session, stamping CreatedAt", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)

		var stored model.Session
		m.EXPECT().CreateSession(context.Background(), gomock.Any()).
			DoAndReturn(func(_ context.Context, s model.Session) error {
				stored = s
				return nil
			})

		err := r.Create(context.Background(), session)
		require.NoError(t, err)

		require.Equal(t, session.Token, stored.Token)
		require.Equal(t, session.UserID, stored.UserID)
		require.Equal(t, session.ExpiresAt, stored.ExpiresAt)
		require.WithinDuration(t, time.Now().UTC(), stored.CreatedAt, time.Minute)
	})

	t.Run("a storage error is propagated", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		m.EXPECT().CreateSession(context.Background(), gomock.Any()).Return(errStub)

		err := r.Create(context.Background(), session)
		require.ErrorIs(t, err, errStub)
	})
}

// TestRepository_FindByToken covers the lookup, including the token -> NotFoundError translation.
func TestRepository_FindByToken(t *testing.T) {
	t.Parallel()

	token := fakeToken()

	t.Run("finds the session", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		row := model.Session{
			Token:     token,
			UserID:    fakeUserID(),
			ExpiresAt: gofakeit.Date().UTC(),
			CreatedAt: gofakeit.Date().UTC(),
		}
		m.EXPECT().FindSessionByToken(context.Background(), token).Return(row, nil)

		got, err := r.FindByToken(context.Background(), token)
		require.NoError(t, err)
		require.Equal(t, row.ToEntity(), got)
	})

	t.Run("an unknown token becomes a message-less NotFoundError", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		m.EXPECT().FindSessionByToken(context.Background(), token).Return(model.Session{}, sql.ErrNoRows)

		_, err := r.FindByToken(context.Background(), token)

		var notFound *domainerror.NotFoundError
		require.ErrorAs(t, err, &notFound)
	})

	t.Run("a storage error is propagated", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		m.EXPECT().FindSessionByToken(context.Background(), token).Return(model.Session{}, errStub)

		_, err := r.FindByToken(context.Background(), token)
		require.ErrorIs(t, err, errStub)
	})
}

// TestRepository_Delete covers the plain, best-effort delegation to storage.
func TestRepository_Delete(t *testing.T) {
	t.Parallel()

	token := fakeToken()

	t.Run("delegates to storage", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		m.EXPECT().DeleteSession(context.Background(), token).Return(nil)

		require.NoError(t, r.Delete(context.Background(), token))
	})

	t.Run("a storage error is propagated", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		m.EXPECT().DeleteSession(context.Background(), token).Return(errStub)

		err := r.Delete(context.Background(), token)
		require.ErrorIs(t, err, errStub)
	})
}

// TestRepository_DeleteExpired covers the plain delegation to storage.
func TestRepository_DeleteExpired(t *testing.T) {
	t.Parallel()

	now := gofakeit.Date().UTC()
	want := gofakeit.Number(0, 100)

	t.Run("delegates to storage", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		m.EXPECT().DeleteExpiredSessions(context.Background(), now).Return(want, nil)

		got, err := r.DeleteExpired(context.Background(), now)
		require.NoError(t, err)
		require.Equal(t, want, got)
	})

	t.Run("a storage error is propagated", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		m.EXPECT().DeleteExpiredSessions(context.Background(), now).Return(0, errStub)

		_, err := r.DeleteExpired(context.Background(), now)
		require.ErrorIs(t, err, errStub)
	})
}
