package authenticate

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"raccodown/internal/domain/entity"
	domainerror "raccodown/internal/domain/error"
	"raccodown/internal/port/mocks"
)

func newUseCase(t *testing.T) (*UseCase, *mocks.MockSessionRepository, *mocks.MockUserRepository) {
	t.Helper()

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	sessions := mocks.NewMockSessionRepository(mc)
	users := mocks.NewMockUserRepository(mc)

	return New(sessions, users), sessions, users
}

func TestUseCase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("a valid session resolves to its user", func(t *testing.T) {
		t.Parallel()

		uc, sessions, users := newUseCase(t)
		session := entity.Session{Token: "tok", UserID: 1, ExpiresAt: time.Now().UTC().Add(time.Hour)}
		user := entity.User{ID: 1, Username: "raccoon"}

		sessions.EXPECT().FindByToken(context.Background(), "tok").Return(session, nil)
		users.EXPECT().FindByID(context.Background(), uint64(1)).Return(user, nil)

		out, err := uc.Execute(context.Background(), Input{Token: "tok"})
		require.NoError(t, err)
		require.Equal(t, user.Public(), out.User)
	})

	t.Run("a blank token is unauthorized without touching the repository", func(t *testing.T) {
		t.Parallel()

		uc, _, _ := newUseCase(t)

		_, err := uc.Execute(context.Background(), Input{Token: ""})

		var unauthorized *domainerror.UnauthorizedError
		require.ErrorAs(t, err, &unauthorized)
	})

	t.Run("an unknown token is unauthorized", func(t *testing.T) {
		t.Parallel()

		uc, sessions, _ := newUseCase(t)
		sessions.EXPECT().FindByToken(context.Background(), "tok").Return(entity.Session{}, &domainerror.NotFoundError{})

		_, err := uc.Execute(context.Background(), Input{Token: "tok"})

		var unauthorized *domainerror.UnauthorizedError
		require.ErrorAs(t, err, &unauthorized)
	})

	t.Run("an expired session is unauthorized and gets deleted", func(t *testing.T) {
		t.Parallel()

		uc, sessions, _ := newUseCase(t)
		session := entity.Session{Token: "tok", UserID: 1, ExpiresAt: time.Now().UTC().Add(-time.Hour)}

		sessions.EXPECT().FindByToken(context.Background(), "tok").Return(session, nil)
		sessions.EXPECT().Delete(context.Background(), "tok").Return(nil)

		_, err := uc.Execute(context.Background(), Input{Token: "tok"})

		var unauthorized *domainerror.UnauthorizedError
		require.ErrorAs(t, err, &unauthorized)
	})

	t.Run("a session pointing at a since-deleted user is unauthorized", func(t *testing.T) {
		t.Parallel()

		uc, sessions, users := newUseCase(t)
		session := entity.Session{Token: "tok", UserID: 1, ExpiresAt: time.Now().UTC().Add(time.Hour)}

		sessions.EXPECT().FindByToken(context.Background(), "tok").Return(session, nil)
		users.EXPECT().FindByID(context.Background(), uint64(1)).Return(entity.User{}, &domainerror.NotFoundError{})

		_, err := uc.Execute(context.Background(), Input{Token: "tok"})

		var unauthorized *domainerror.UnauthorizedError
		require.ErrorAs(t, err, &unauthorized)
	})
}
