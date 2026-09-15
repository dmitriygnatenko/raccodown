package update

import (
	"context"
	"errors"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"raccodown/internal/domain/entity"
	domainerror "raccodown/internal/domain/error"
	"raccodown/internal/port"
	"raccodown/internal/port/mocks"
)

func newUseCase(t *testing.T) (*UseCase, *mocks.MockNoteRepository) {
	t.Helper()

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	m := mocks.NewMockNoteRepository(mc)

	return New(m), m
}

func TestUseCase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("rewrites the note, re-deriving tags from content", func(t *testing.T) {
		t.Parallel()

		uc, repo := newUseCase(t)
		want := entity.Note{ID: 1, Title: "t", Content: "c #work", Tags: []string{"work"}}

		repo.EXPECT().
			Update(context.Background(), port.NoteUpdateRequest{
				ID:            1,
				Title:         "t",
				Content:       "c #work",
				Tags:          []string{"work"},
				KnownChecksum: "abc",
			}).
			Return(want, nil)

		out, err := uc.Execute(context.Background(), Input{
			ID: 1, Title: "t", Content: "c #work", KnownChecksum: "abc",
		})
		require.NoError(t, err)
		require.Equal(t, want, out.Note)
	})

	t.Run("a checksum conflict fetches the current note and returns *ConflictError", func(t *testing.T) {
		t.Parallel()

		uc, repo := newUseCase(t)
		current := entity.Note{ID: 1, Title: "current title", Content: "current content"}

		repo.EXPECT().Update(context.Background(), gomock.Any()).
			Return(entity.Note{}, &domainerror.ConflictError{})
		repo.EXPECT().FindByID(context.Background(), uint64(1)).Return(current, nil)

		_, err := uc.Execute(context.Background(), Input{ID: 1, KnownChecksum: "stale"})

		var conflict *ConflictError
		require.ErrorAs(t, err, &conflict)
		require.Equal(t, current, conflict.Current)
	})

	t.Run("an unknown id is reported as NotFoundError", func(t *testing.T) {
		t.Parallel()

		uc, repo := newUseCase(t)
		repo.EXPECT().Update(context.Background(), gomock.Any()).
			Return(entity.Note{}, &domainerror.NotFoundError{})

		_, err := uc.Execute(context.Background(), Input{ID: 999999})

		var notFound *domainerror.NotFoundError
		require.ErrorAs(t, err, &notFound)
	})

	t.Run("the note being deleted between the conflict and the re-fetch is a NotFoundError, not a generic save error", func(t *testing.T) {
		t.Parallel()

		uc, repo := newUseCase(t)
		repo.EXPECT().Update(context.Background(), gomock.Any()).
			Return(entity.Note{}, &domainerror.ConflictError{})
		repo.EXPECT().FindByID(context.Background(), gomock.Any()).
			Return(entity.Note{}, &domainerror.NotFoundError{})

		_, err := uc.Execute(context.Background(), Input{ID: 1})

		var notFound *domainerror.NotFoundError
		require.ErrorAs(t, err, &notFound)
	})

	t.Run("a repository failure while re-fetching after a conflict is a generic save error", func(t *testing.T) {
		t.Parallel()

		uc, repo := newUseCase(t)
		repo.EXPECT().Update(context.Background(), gomock.Any()).
			Return(entity.Note{}, &domainerror.ConflictError{})
		repo.EXPECT().FindByID(context.Background(), gomock.Any()).
			Return(entity.Note{}, errors.New(gofakeit.Sentence()))

		_, err := uc.Execute(context.Background(), Input{ID: 1})

		var conflict *ConflictError
		require.False(t, errors.As(err, &conflict))
		require.Error(t, err)
	})
}
