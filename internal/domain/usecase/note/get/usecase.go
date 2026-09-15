// Package get is the GetNote use case: find one note by id.
package get

import (
	"context"
	"errors"
	"log/slog"

	domainError "raccodown/internal/domain/error"
	"raccodown/internal/port"
)

// UseCase implements GetNote.
type UseCase struct {
	noteRepository port.NoteRepository
}

// New builds a UseCase from its dependencies.
func New(noteRepository port.NoteRepository) *UseCase {
	return &UseCase{
		noteRepository: noteRepository,
	}
}

// Execute returns the note with the given id.
func (uc *UseCase) Execute(ctx context.Context, input Input) (Output, error) {
	note, err := uc.noteRepository.FindByID(ctx, input.ID)
	if err != nil {
		if domainError.IsNotFoundError(err) {
			return Output{}, &domainError.NotFoundError{Message: "Note not found"}
		}

		slog.ErrorContext(ctx, "get note: load", "error", err)

		return Output{}, errors.New("Failed to load note")
	}

	return Output{Note: note}, nil
}
