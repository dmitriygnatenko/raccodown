package updatecredentials

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

func newUseCase(t *testing.T) (*UseCase, *mocks.MockUserRepository, *mocks.MockPasswordHasher) {
	t.Helper()

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	users := mocks.NewMockUserRepository(mc)
	hasher := mocks.NewMockPasswordHasher(mc)

	return New(users, hasher), users, hasher
}

func TestUseCase_Execute(t *testing.T) {
	t.Parallel()

	current := entity.User{ID: 1, Username: "raccoon", PasswordHash: "hashed"}
	base := Input{User: current.Public(), CurrentPassword: "correct-horse"}

	t.Run("username and password change in a single UpdateCredentials call", func(t *testing.T) {
		t.Parallel()

		uc, users, hasher := newUseCase(t)
		users.EXPECT().FindByID(context.Background(), uint64(1)).Return(current, nil)
		hasher.EXPECT().Compare("hashed", "correct-horse").Return(true)
		hasher.EXPECT().Hash("battery-staple").Return("new-hash", nil)
		users.EXPECT().UpdateCredentials(context.Background(), uint64(1), "newname", "new-hash").Return(nil)

		in := base
		in.NewUsername = "NewName"
		in.NewPassword = "battery-staple"

		out, err := uc.Execute(context.Background(), in)
		require.NoError(t, err)
		require.Equal(t, "newname", out.User.Username)
	})

	t.Run("incorrect current password is an UnauthorizedError", func(t *testing.T) {
		t.Parallel()

		uc, users, hasher := newUseCase(t)
		users.EXPECT().FindByID(context.Background(), uint64(1)).Return(current, nil)
		hasher.EXPECT().Compare("hashed", "correct-horse").Return(false)

		_, err := uc.Execute(context.Background(), base)

		var unauthorized *domainerror.UnauthorizedError
		require.ErrorAs(t, err, &unauthorized)
	})

	t.Run("an unknown user id is a NotFoundError", func(t *testing.T) {
		t.Parallel()

		uc, users, _ := newUseCase(t)
		users.EXPECT().FindByID(context.Background(), uint64(1)).Return(entity.User{}, &domainerror.NotFoundError{})

		_, err := uc.Execute(context.Background(), base)

		var notFound *domainerror.NotFoundError
		require.ErrorAs(t, err, &notFound)
	})

	t.Run("a taken username is a ConflictError", func(t *testing.T) {
		t.Parallel()

		uc, users, hasher := newUseCase(t)
		users.EXPECT().FindByID(context.Background(), uint64(1)).Return(current, nil)
		hasher.EXPECT().Compare("hashed", "correct-horse").Return(true)
		users.EXPECT().UpdateCredentials(context.Background(), uint64(1), "taken", "").
			Return(&domainerror.ConflictError{})

		in := base
		in.NewUsername = "taken"

		_, err := uc.Execute(context.Background(), in)

		var conflict *domainerror.ConflictError
		require.ErrorAs(t, err, &conflict)
	})

	t.Run("the account being deleted mid-request surfaces as a NotFoundError", func(t *testing.T) {
		t.Parallel()

		uc, users, hasher := newUseCase(t)
		users.EXPECT().FindByID(context.Background(), uint64(1)).Return(current, nil)
		hasher.EXPECT().Compare("hashed", "correct-horse").Return(true)
		users.EXPECT().UpdateCredentials(context.Background(), uint64(1), "newname", "").
			Return(&domainerror.NotFoundError{})

		in := base
		in.NewUsername = "NewName"

		_, err := uc.Execute(context.Background(), in)

		var notFound *domainerror.NotFoundError
		require.ErrorAs(t, err, &notFound)
	})

	t.Run("only the username changes: UpdateCredentials gets a blank password hash and Hash is never called", func(t *testing.T) {
		t.Parallel()

		uc, users, hasher := newUseCase(t)
		users.EXPECT().FindByID(context.Background(), uint64(1)).Return(current, nil)
		hasher.EXPECT().Compare("hashed", "correct-horse").Return(true)
		users.EXPECT().UpdateCredentials(context.Background(), uint64(1), "newname", "").Return(nil)

		in := base
		in.NewUsername = "NewName"

		out, err := uc.Execute(context.Background(), in)
		require.NoError(t, err)
		require.Equal(t, "newname", out.User.Username)
	})

	t.Run("renaming to the same (normalized) username is a no-op: the repository is never called", func(t *testing.T) {
		t.Parallel()

		uc, users, hasher := newUseCase(t)
		users.EXPECT().FindByID(context.Background(), uint64(1)).Return(current, nil)
		hasher.EXPECT().Compare("hashed", "correct-horse").Return(true)

		in := base
		in.NewUsername = " Raccoon "

		out, err := uc.Execute(context.Background(), in)
		require.NoError(t, err)
		require.Equal(t, "raccoon", out.User.Username)
	})

	t.Run("a password hash failure happens before any repository write", func(t *testing.T) {
		t.Parallel()

		uc, users, hasher := newUseCase(t)
		users.EXPECT().FindByID(context.Background(), uint64(1)).Return(current, nil)
		hasher.EXPECT().Compare("hashed", "correct-horse").Return(true)
		hasher.EXPECT().Hash("battery-staple").Return("", errors.New(gofakeit.Sentence()))

		in := base
		in.NewUsername = "NewName"
		in.NewPassword = "battery-staple"

		_, err := uc.Execute(context.Background(), in)
		require.Error(t, err)
	})

	t.Run("input that fails validation never reaches the repository", func(t *testing.T) {
		t.Parallel()

		uc, _, _ := newUseCase(t)

		in := base
		in.NewUsername = "a" // below entity.MinUsernameLength once normalized

		_, err := uc.Execute(context.Background(), in)
		require.True(t, domainerror.IsValidationError(err))
	})

	t.Run("a username that only normalizes below the minimum length fails validation", func(t *testing.T) {
		t.Parallel()

		uc, _, _ := newUseCase(t)

		in := base
		in.NewUsername = " a " // 3 raw characters, 1 once trimmed

		_, err := uc.Execute(context.Background(), in)
		require.True(t, domainerror.IsValidationError(err))
	})
}
