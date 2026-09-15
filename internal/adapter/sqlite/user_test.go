package sqlite

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/require"

	storageError "raccodown/internal/storage/error"
	"raccodown/internal/storage/model"
)

func fakeUsername() string { return gofakeit.Username() }
func fakeHash() string     { return gofakeit.LetterN(60) }

// createFakeUser inserts a user with random username/password and returns the full row (including
// its database-assigned id), so tests can exercise lookups/updates against a real one.
func createFakeUser(t *testing.T, s *Storage) model.User {
	t.Helper()

	now := gofakeit.Date().UTC().Truncate(time.Second)
	username := fakeUsername()
	hash := fakeHash()

	id, err := s.CreateUser(context.Background(), username, hash, now, now)
	require.NoError(t, err)

	return model.User{
		ID:           id,
		Username:     username,
		PasswordHash: hash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// TestUser_CreateAndFind covers the round trip through both lookups, and the sql.ErrNoRows miss.
func TestUser_CreateAndFind(t *testing.T) {
	t.Parallel()

	s := newTestStorage(t)

	user := createFakeUser(t, s)

	byUsername, err := s.FindUserByUsername(context.Background(), user.Username)
	require.NoError(t, err)
	require.Equal(t, user, byUsername)

	byID, err := s.FindUserByID(context.Background(), user.ID)
	require.NoError(t, err)
	require.Equal(t, user, byID)

	_, err = s.FindUserByUsername(context.Background(), "does-not-exist")
	require.ErrorIs(t, err, sql.ErrNoRows)

	_, err = s.FindUserByID(context.Background(), 999999)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// TestUser_CreateDuplicateUsername covers the UNIQUE(username) constraint surfacing as
// storageError.UniqueViolationError.
func TestUser_CreateDuplicateUsername(t *testing.T) {
	t.Parallel()

	s := newTestStorage(t)
	ctx := context.Background()

	first := createFakeUser(t, s)

	now := gofakeit.Date().UTC().Truncate(time.Second)
	_, err := s.CreateUser(ctx, first.Username, fakeHash(), now, now)
	require.ErrorIs(t, err, storageError.UniqueViolationError)
}

// TestUser_UpdateUserCredentials covers renaming, changing the password hash, changing both in one
// call, the "not found" false, and the duplicate-username error.
func TestUser_UpdateUserCredentials(t *testing.T) {
	t.Parallel()

	s := newTestStorage(t)
	ctx := context.Background()

	t.Run("changes only the username", func(t *testing.T) {
		user := createFakeUser(t, s)
		newUsername := fakeUsername()
		newUpdatedAt := user.UpdatedAt.Add(time.Hour)

		found, err := s.UpdateUserCredentials(ctx, user.ID, newUsername, "", newUpdatedAt)
		require.NoError(t, err)
		require.True(t, found)

		got, err := s.FindUserByID(ctx, user.ID)
		require.NoError(t, err)
		require.Equal(t, newUsername, got.Username)
		require.Equal(t, user.PasswordHash, got.PasswordHash)
		require.Equal(t, newUpdatedAt, got.UpdatedAt)
	})

	t.Run("changes only the password hash", func(t *testing.T) {
		user := createFakeUser(t, s)
		newHash := fakeHash()
		newUpdatedAt := user.UpdatedAt.Add(time.Hour)

		found, err := s.UpdateUserCredentials(ctx, user.ID, "", newHash, newUpdatedAt)
		require.NoError(t, err)
		require.True(t, found)

		got, err := s.FindUserByID(ctx, user.ID)
		require.NoError(t, err)
		require.Equal(t, user.Username, got.Username)
		require.Equal(t, newHash, got.PasswordHash)
		require.Equal(t, newUpdatedAt, got.UpdatedAt)
	})

	t.Run("changes both in one call", func(t *testing.T) {
		user := createFakeUser(t, s)
		newUsername := fakeUsername()
		newHash := fakeHash()
		newUpdatedAt := user.UpdatedAt.Add(time.Hour)

		found, err := s.UpdateUserCredentials(ctx, user.ID, newUsername, newHash, newUpdatedAt)
		require.NoError(t, err)
		require.True(t, found)

		got, err := s.FindUserByID(ctx, user.ID)
		require.NoError(t, err)
		require.Equal(t, newUsername, got.Username)
		require.Equal(t, newHash, got.PasswordHash)
		require.Equal(t, newUpdatedAt, got.UpdatedAt)
	})

	t.Run("an unknown id is not found", func(t *testing.T) {
		found, err := s.UpdateUserCredentials(ctx, 999999, fakeUsername(), fakeHash(), time.Now().UTC())
		require.NoError(t, err)
		require.False(t, found)
	})

	t.Run("a taken username is a unique violation", func(t *testing.T) {
		user := createFakeUser(t, s)
		other := createFakeUser(t, s)

		_, err := s.UpdateUserCredentials(ctx, other.ID, user.Username, "", time.Now().UTC())
		require.ErrorIs(t, err, storageError.UniqueViolationError)
	})
}

// TestUser_CountUsers covers the count the first-run seeding decision is made on.
func TestUser_CountUsers(t *testing.T) {
	t.Parallel()

	s := newTestStorage(t)
	ctx := context.Background()

	n, err := s.CountUsers(ctx)
	require.NoError(t, err)
	require.Zero(t, n)

	createFakeUser(t, s)
	createFakeUser(t, s)

	n, err = s.CountUsers(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, n)
}

// TestUser_UpdateAndGetSettings covers the settings JSON column round trip, including a fresh
// account's settings decoding to the zero value, and the "not found" false.
func TestUser_UpdateAndGetSettings(t *testing.T) {
	t.Parallel()

	s := newTestStorage(t)
	ctx := context.Background()

	user := createFakeUser(t, s)

	settings, err := s.GetUserSettings(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, model.UserSettings{}, settings)

	newSettings := model.UserSettings{Language: "fr", Theme: "dark"}
	newUpdatedAt := user.UpdatedAt.Add(time.Hour)

	found, err := s.UpdateUserSettings(ctx, user.ID, newSettings, newUpdatedAt)
	require.NoError(t, err)
	require.True(t, found)

	got, err := s.GetUserSettings(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, newSettings, got)

	row, err := s.FindUserByID(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, newUpdatedAt, row.UpdatedAt)

	found, err = s.UpdateUserSettings(ctx, 999999, newSettings, time.Now().UTC())
	require.NoError(t, err)
	require.False(t, found)
}

// TestUser_GetSettings_NotFound covers the sql.ErrNoRows miss.
func TestUser_GetSettings_NotFound(t *testing.T) {
	t.Parallel()

	s := newTestStorage(t)

	_, err := s.GetUserSettings(context.Background(), 999999)
	require.ErrorIs(t, err, sql.ErrNoRows)
}
