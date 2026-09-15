package create

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

	t.Run("a blank title resolves to the default and tags are derived from content", func(t *testing.T) {
		t.Parallel()

		uc, repo := newUseCase(t)
		content := "buy milk #shopping"

		want := entity.Note{ID: 1, Title: "Untitled", Content: content, Tags: []string{"shopping"}}
		repo.EXPECT().
			Create(context.Background(), port.NoteCreateRequest{
				Title:   "Untitled",
				Content: content,
				Tags:    []string{"shopping"},
			}).
			Return(want, nil)

		out, err := uc.Execute(context.Background(), Input{Title: "  ", Content: content})
		require.NoError(t, err)
		require.Equal(t, want, out.Note)
	})

	t.Run("a title over the length limit fails validation", func(t *testing.T) {
		t.Parallel()

		uc, _ := newUseCase(t)

		longTitle := ""
		for range entity.MaxTitleLength + 1 {
			longTitle += "x"
		}

		_, err := uc.Execute(context.Background(), Input{Title: longTitle})

		require.True(t, domainerror.IsValidationError(err))
	})

	t.Run("a repository failure is reported as a generic save error", func(t *testing.T) {
		t.Parallel()

		uc, repo := newUseCase(t)
		repo.EXPECT().Create(gomock.Any(), gomock.Any()).
			Return(entity.Note{}, errors.New(gofakeit.Sentence()))

		_, err := uc.Execute(context.Background(), Input{Title: "t"})
		require.Error(t, err)
		require.False(t, domainerror.IsValidationError(err))
	})
}
