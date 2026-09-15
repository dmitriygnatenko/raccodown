// Package delete is the DeleteNote use case.
package delete

import (
	"context"
	"errors"
	"log/slog"

	domainError "raccodown/internal/domain/error"
	"raccodown/internal/port"
)

// UseCase implements DeleteNote.
type UseCase struct {
	noteRepository port.NoteRepository
}

// New builds a UseCase from its dependencies.
func New(noteRepository port.NoteRepository) *UseCase {
	return &UseCase{
		noteRepository: noteRepository,
	}
}

// Execute removes the note with the given id.
func (uc *UseCase) Execute(ctx context.Context, input Input) error {
	if err := uc.noteRepository.Delete(ctx, input.ID); err != nil {
		if domainError.IsNotFoundError(err) {
			return &domainError.NotFoundError{Message: "Note not found"}
		}

		slog.ErrorContext(ctx, "delete note: remove", "error", err)

		return errors.New("Failed to delete note")
	}

	return nil
}
