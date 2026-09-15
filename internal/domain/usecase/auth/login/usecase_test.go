package login

import (
	"context"
	"errors"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"raccodown/internal/domain/entity"
	domainerror "raccodown/internal/domain/error"
	"raccodown/internal/port/mocks"
)

func newUseCase(t *testing.T) (*UseCase, *mocks.MockUserRepository, *mocks.MockSessionRepository, *mocks.MockPasswordHasher, *mocks.MockTokenGenerator) {
	t.Helper()

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	users := mocks.NewMockUserRepository(mc)
	sessions := mocks.NewMockSessionRepository(mc)
	hasher := mocks.NewMockPasswordHasher(mc)
	tokens := mocks.NewMockTokenGenerator(mc)

	return New(users, sessions, hasher, tokens), users, sessions, hasher, tokens
}

func TestUseCase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("correct credentials start a session", func(t *testing.T) {
		t.Parallel()

		uc, users, sessions, hasher, tokens := newUseCase(t)
		user := entity.User{ID: 1, Username: "raccoon", PasswordHash: "hashed"}

		users.EXPECT().FindByUsername(context.Background(), "raccoon").Return(user, nil)
		hasher.EXPECT().Compare("hashed", "correct-horse").Return(true)
		tokens.EXPECT().NewToken().Return("new-token", nil)
		sessions.EXPECT().Create(context.Background(), gomock.Any()).
			DoAndReturn(func(_ context.Context, s entity.Session) error {
				require.Equal(t, "new-token", s.Token)
				require.Equal(t, uint64(1), s.UserID)
				require.False(t, s.ExpiresAt.IsZero())
				return nil
			})

		out, err := uc.Execute(context.Background(), Input{Username: " Raccoon ", Password: "correct-horse"})
		require.NoError(t, err)
		require.Equal(t, user.Public(), out.User)
		require.Equal(t, "new-token", out.Session.Token)
	})

	t.Run("unknown username is an UnauthorizedError, same as a wrong password", func(t *testing.T) {
		t.Parallel()

		uc, users, _, _, _ := newUseCase(t)
		users.EXPECT().FindByUsername(context.Background(), "raccoon").
			Return(entity.User{}, &domainerror.NotFoundError{})

		_, err := uc.Execute(context.Background(), Input{Username: "raccoon", Password: "correct-horse"})

		var unauthorized *domainerror.UnauthorizedError
		require.ErrorAs(t, err, &unauthorized)
	})

	t.Run("wrong password is an UnauthorizedError", func(t *testing.T) {
		t.Parallel()

		uc, users, _, hasher, _ := newUseCase(t)
		users.EXPECT().FindByUsername(context.Background(), "raccoon").
			Return(entity.User{ID: 1, Username: "raccoon", PasswordHash: "hashed"}, nil)
		hasher.EXPECT().Compare("hashed", "wrong-password").Return(false)

		_, err := uc.Execute(context.Background(), Input{Username: "raccoon", Password: "wrong-password"})

		var unauthorized *domainerror.UnauthorizedError
		require.ErrorAs(t, err, &unauthorized)
	})

	t.Run("input that fails validation never reaches the repository", func(t *testing.T) {
		t.Parallel()

		uc, _, _, _, _ := newUseCase(t)

		_, err := uc.Execute(context.Background(), Input{Username: "raccoon", Password: "abc"})

		require.True(t, domainerror.IsValidationError(err))
	})

	t.Run("language is saved for an account that has none yet", func(t *testing.T) {
		t.Parallel()

		uc, users, sessions, hasher, tokens := newUseCase(t)
		user := entity.User{ID: 1, Username: "raccoon", PasswordHash: "hashed"}

		users.EXPECT().FindByUsername(context.Background(), "raccoon").Return(user, nil)
		hasher.EXPECT().Compare("hashed", "correct-horse").Return(true)
		users.EXPECT().UpdateSettings(context.Background(), uint64(1), entity.UserSettings{Language: "fr"}).Return(nil)
		tokens.EXPECT().NewToken().Return("new-token", nil)
		sessions.EXPECT().Create(context.Background(), gomock.Any()).Return(nil)

		out, err := uc.Execute(context.Background(), Input{Username: "raccoon", Password: "correct-horse", Language: "fr"})
		require.NoError(t, err)
		require.Equal(t, "fr", out.User.Settings.Language)
	})

	t.Run("an already-set language is not overwritten by the login form's language", func(t *testing.T) {
		t.Parallel()

		uc, users, sessions, hasher, tokens := newUseCase(t)
		user := entity.User{
			ID: 1, Username: "raccoon", PasswordHash: "hashed",
			Settings: entity.UserSettings{Language: "ru"},
		}

		users.EXPECT().FindByUsername(context.Background(), "raccoon").Return(user, nil)
		hasher.EXPECT().Compare("hashed", "correct-horse").Return(true)
		tokens.EXPECT().NewToken().Return("new-token", nil)
		sessions.EXPECT().Create(context.Background(), gomock.Any()).Return(nil)

		out, err := uc.Execute(context.Background(), Input{Username: "raccoon", Password: "correct-horse", Language: "en"})
		require.NoError(t, err)
		require.Equal(t, "ru", out.User.Settings.Language)
	})

	t.Run("a failure saving the language doesn't fail the login", func(t *testing.T) {
		t.Parallel()

		uc, users, sessions, hasher, tokens := newUseCase(t)
		user := entity.User{ID: 1, Username: "raccoon", PasswordHash: "hashed"}

		users.EXPECT().FindByUsername(context.Background(), "raccoon").Return(user, nil)
		hasher.EXPECT().Compare("hashed", "correct-horse").Return(true)
		users.EXPECT().UpdateSettings(context.Background(), uint64(1), gomock.Any()).Return(errors.New(gofakeit.Sentence()))
		tokens.EXPECT().NewToken().Return("new-token", nil)
		sessions.EXPECT().Create(context.Background(), gomock.Any()).Return(nil)

		out, err := uc.Execute(context.Background(), Input{Username: "raccoon", Password: "correct-horse", Language: "fr"})
		require.NoError(t, err)
		require.Empty(t, out.User.Settings.Language)
	})

	t.Run("a malformed language fails validation before any lookup", func(t *testing.T) {
		t.Parallel()

		uc, _, _, _, _ := newUseCase(t)

		_, err := uc.Execute(context.Background(), Input{Username: "raccoon", Password: "correct-horse", Language: "not-a-code!"})

		require.True(t, domainerror.IsValidationError(err))
	})

	t.Run("a token generation failure is a generic error", func(t *testing.T) {
		t.Parallel()

		uc, users, _, hasher, tokens := newUseCase(t)
		users.EXPECT().FindByUsername(context.Background(), "raccoon").
			Return(entity.User{ID: 1, Username: "raccoon", PasswordHash: "hashed"}, nil)
		hasher.EXPECT().Compare("hashed", "correct-horse").Return(true)
		tokens.EXPECT().NewToken().Return("", errors.New(gofakeit.Sentence()))

		_, err := uc.Execute(context.Background(), Input{Username: "raccoon", Password: "correct-horse"})
		require.Error(t, err)
	})
}
