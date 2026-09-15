// Package list is the ListTags use case.
package list

import (
	"context"
	"errors"
	"log/slog"

	"raccodown/internal/port"
)

// UseCase implements ListTags.
type UseCase struct {
	noteRepository port.NoteRepository
}

// New builds a UseCase from its dependencies.
func New(noteRepository port.NoteRepository) *UseCase {
	return &UseCase{
		noteRepository: noteRepository,
	}
}

// Execute returns every distinct tag currently in use, sorted alphabetically.
func (uc *UseCase) Execute(ctx context.Context) (Output, error) {
	tags, err := uc.noteRepository.ListTags(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "list tags: load", "error", err)

		return Output{}, errors.New("Failed to load tags")
	}

	return Output{Tags: tags}, nil
}
