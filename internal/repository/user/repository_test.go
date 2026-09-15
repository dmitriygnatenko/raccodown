package user

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	domainerror "raccodown/internal/domain/error"
	"raccodown/internal/port"
	"raccodown/internal/repository/user/mocks"
	storageError "raccodown/internal/storage/error"
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

func fakeID() uint64       { return uint64(gofakeit.Number(1, 1_000_000)) }
func fakeUsername() string { return gofakeit.Username() }
func fakeHash() string     { return gofakeit.LetterN(60) }
func fakeLanguage() string { return gofakeit.LanguageAbbreviation() }

func fakeUserSettingsModel() model.UserSettings {
	return model.UserSettings{Language: fakeLanguage(), Theme: "dark"}
}

func fakeUserModel() model.User {
	now := gofakeit.Date().UTC()

	return model.User{
		ID:           fakeID(),
		Username:     fakeUsername(),
		PasswordHash: fakeHash(),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

var errStub = errors.New(gofakeit.Sentence())

// TestRepository_FindByUsername covers the lookup, including the username -> NotFoundError
// translation.
func TestRepository_FindByUsername(t *testing.T) { //nolint:dupl // mirrors TestRepository_FindByID
	t.Parallel()

	username := fakeUsername()

	t.Run("finds the user", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		row := fakeUserModel()
		row.Username = username
		m.EXPECT().FindUserByUsername(context.Background(), username).Return(row, nil)

		got, err := r.FindByUsername(context.Background(), username)
		require.NoError(t, err)
		require.Equal(t, row.ToEntity(), got)
	})

	t.Run("an unknown username becomes a message-less NotFoundError", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		m.EXPECT().FindUserByUsername(context.Background(), username).Return(model.User{}, sql.ErrNoRows)

		_, err := r.FindByUsername(context.Background(), username)

		var notFound *domainerror.NotFoundError
		require.ErrorAs(t, err, &notFound)
		require.Empty(t, notFound.Message)
	})

	t.Run("a storage error is propagated", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		m.EXPECT().FindUserByUsername(context.Background(), username).Return(model.User{}, errStub)

		_, err := r.FindByUsername(context.Background(), username)
		require.ErrorIs(t, err, errStub)
	})
}

// TestRepository_FindByID covers the lookup, including the id -> NotFoundError translation.
func TestRepository_FindByID(t *testing.T) { //nolint:dupl // mirrors TestRepository_FindByUsername
	t.Parallel()

	id := fakeID()

	t.Run("finds the user", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		row := fakeUserModel()
		row.ID = id
		m.EXPECT().FindUserByID(context.Background(), id).Return(row, nil)

		got, err := r.FindByID(context.Background(), id)
		require.NoError(t, err)
		require.Equal(t, row.ToEntity(), got)
	})

	t.Run("an unknown id becomes a message-less NotFoundError", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		m.EXPECT().FindUserByID(context.Background(), id).Return(model.User{}, sql.ErrNoRows)

		_, err := r.FindByID(context.Background(), id)

		var notFound *domainerror.NotFoundError
		require.ErrorAs(t, err, &notFound)
	})

	t.Run("a storage error is propagated", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		m.EXPECT().FindUserByID(context.Background(), id).Return(model.User{}, errStub)

		_, err := r.FindByID(context.Background(), id)
		require.ErrorIs(t, err, errStub)
	})
}

// TestRepository_Create covers the insert, including the database-assigned id and the username ->
// ConflictError translation.
func TestRepository_Create(t *testing.T) {
	t.Parallel()

	req := port.UserCreateRequest{
		Username:     fakeUsername(),
		PasswordHash: fakeHash(),
	}

	t.Run("stores the user and returns it with its new id", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		wantID := fakeID()

		m.EXPECT().
			CreateUser(context.Background(), req.Username, req.PasswordHash, gomock.Any(), gomock.Any()).
			Return(wantID, nil)

		got, err := r.Create(context.Background(), req)
		require.NoError(t, err)

		require.Equal(t, wantID, got.ID)
		require.Equal(t, req.Username, got.Username)
		require.Equal(t, req.PasswordHash, got.PasswordHash)
		require.WithinDuration(t, time.Now().UTC(), got.CreatedAt, time.Minute)
		require.Equal(t, got.CreatedAt, got.UpdatedAt)
	})

	t.Run("a taken username becomes a message-less ConflictError", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		m.EXPECT().CreateUser(context.Background(), req.Username, req.PasswordHash, gomock.Any(), gomock.Any()).
			Return(uint64(0), storageError.UniqueViolationError)

		_, err := r.Create(context.Background(), req)

		var conflict *domainerror.ConflictError
		require.ErrorAs(t, err, &conflict)
		require.Empty(t, conflict.Message)
	})

	t.Run("any other storage error is propagated", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		m.EXPECT().CreateUser(context.Background(), req.Username, req.PasswordHash, gomock.Any(), gomock.Any()).
			Return(uint64(0), errStub)

		_, err := r.Create(context.Background(), req)
		require.ErrorIs(t, err, errStub)
	})
}

// TestRepository_UpdateCredentials covers the username/password-hash change, including the
// not-found and conflict translations.
func TestRepository_UpdateCredentials(t *testing.T) {
	t.Parallel()

	id := fakeID()
	username := fakeUsername()
	hash := fakeHash()

	tests := []struct {
		name      string
		mock      func(m *mocks.MockStorage)
		assertErr func(t *testing.T, err error)
	}{
		{
			name: "changes the username and password hash",
			mock: func(m *mocks.MockStorage) {
				m.EXPECT().UpdateUserCredentials(context.Background(), id, username, hash, gomock.Any()).
					Return(true, nil)
			},
			assertErr: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name: "an unknown id becomes a message-less NotFoundError",
			mock: func(m *mocks.MockStorage) {
				m.EXPECT().UpdateUserCredentials(context.Background(), id, username, hash, gomock.Any()).
					Return(false, nil)
			},
			assertErr: func(t *testing.T, err error) {
				var notFound *domainerror.NotFoundError
				require.ErrorAs(t, err, &notFound)
			},
		},
		{
			name: "a taken username becomes a message-less ConflictError",
			mock: func(m *mocks.MockStorage) {
				m.EXPECT().UpdateUserCredentials(context.Background(), id, username, hash, gomock.Any()).
					Return(false, storageError.UniqueViolationError)
			},
			assertErr: func(t *testing.T, err error) {
				var conflict *domainerror.ConflictError
				require.ErrorAs(t, err, &conflict)
			},
		},
		{
			name: "any other storage error is propagated",
			mock: func(m *mocks.MockStorage) {
				m.EXPECT().UpdateUserCredentials(context.Background(), id, username, hash, gomock.Any()).
					Return(false, errStub)
			},
			assertErr: func(t *testing.T, err error) { require.ErrorIs(t, err, errStub) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r, m := newRepo(t)
			tt.mock(m)

			err := r.UpdateCredentials(context.Background(), id, username, hash)
			tt.assertErr(t, err)
		})
	}
}

// TestRepository_GetSettings covers the lookup, including the id -> NotFoundError translation.
func TestRepository_GetSettings(t *testing.T) {
	t.Parallel()

	id := fakeID()

	t.Run("returns the stored settings", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		settings := fakeUserSettingsModel()
		m.EXPECT().GetUserSettings(context.Background(), id).Return(settings, nil)

		got, err := r.GetSettings(context.Background(), id)
		require.NoError(t, err)
		require.Equal(t, settings.ToEntity(), got)
	})

	t.Run("an unknown id becomes a message-less NotFoundError", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		m.EXPECT().GetUserSettings(context.Background(), id).Return(model.UserSettings{}, sql.ErrNoRows)

		_, err := r.GetSettings(context.Background(), id)

		var notFound *domainerror.NotFoundError
		require.ErrorAs(t, err, &notFound)
	})

	t.Run("a storage error is propagated", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		m.EXPECT().GetUserSettings(context.Background(), id).Return(model.UserSettings{}, errStub)

		_, err := r.GetSettings(context.Background(), id)
		require.ErrorIs(t, err, errStub)
	})
}

// TestRepository_UpdateSettings covers overwriting the settings, including the not-found
// translation.
func TestRepository_UpdateSettings(t *testing.T) {
	t.Parallel()

	id := fakeID()
	settings := fakeUserSettingsModel().ToEntity()

	tests := []struct {
		name      string
		mock      func(m *mocks.MockStorage)
		assertErr func(t *testing.T, err error)
	}{
		{
			name: "overwrites the settings",
			mock: func(m *mocks.MockStorage) {
				m.EXPECT().
					UpdateUserSettings(context.Background(), id, model.UserSettings{Language: settings.Language, Theme: settings.Theme}, gomock.Any()).
					Return(true, nil)
			},
			assertErr: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name: "an unknown id becomes a message-less NotFoundError",
			mock: func(m *mocks.MockStorage) {
				m.EXPECT().UpdateUserSettings(context.Background(), id, gomock.Any(), gomock.Any()).Return(false, nil)
			},
			assertErr: func(t *testing.T, err error) {
				var notFound *domainerror.NotFoundError
				require.ErrorAs(t, err, &notFound)
			},
		},
		{
			name: "a storage error is propagated",
			mock: func(m *mocks.MockStorage) {
				m.EXPECT().UpdateUserSettings(context.Background(), id, gomock.Any(), gomock.Any()).Return(false, errStub)
			},
			assertErr: func(t *testing.T, err error) { require.ErrorIs(t, err, errStub) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r, m := newRepo(t)
			tt.mock(m)

			err := r.UpdateSettings(context.Background(), id, settings)
			tt.assertErr(t, err)
		})
	}
}

// TestRepository_Count covers the plain delegation to storage.
func TestRepository_Count(t *testing.T) {
	t.Parallel()

	want := gofakeit.Number(0, 500)

	t.Run("delegates to storage", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		m.EXPECT().CountUsers(context.Background()).Return(want, nil)

		got, err := r.Count(context.Background())
		require.NoError(t, err)
		require.Equal(t, want, got)
	})

	t.Run("a storage error is propagated", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		m.EXPECT().CountUsers(context.Background()).Return(0, errStub)

		_, err := r.Count(context.Background())
		require.ErrorIs(t, err, errStub)
	})
}
