// Package list is the ListNotes use case: it returns every note matching an optional
// query/tag filter, newest-updated first.
package list

import (
	"context"
	"errors"
	"log/slog"

	"raccodown/internal/domain/entity"
	"raccodown/internal/port"
)

// UseCase implements ListNotes.
type UseCase struct {
	noteRepository port.NoteRepository
}

// New builds a UseCase from its dependencies.
func New(noteRepository port.NoteRepository) *UseCase {
	return &UseCase{
		noteRepository: noteRepository,
	}
}

// Execute returns every note matching input's filter.
func (uc *UseCase) Execute(ctx context.Context, input Input) (Output, error) {
	notes, err := uc.noteRepository.List(ctx, port.NoteListFilter{
		Query: input.Query,
		Tag:   input.Tag,
	})
	if err != nil {
		slog.ErrorContext(ctx, "list notes: load", "error", err)

		return Output{}, errors.New("Failed to load notes")
	}

	if notes == nil {
		notes = []entity.Note{}
	}

	return Output{Notes: notes}, nil
}
