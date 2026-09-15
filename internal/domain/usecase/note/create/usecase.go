// Package create is the CreateNote use case.
package create

import (
	"context"
	"errors"
	"log/slog"

	domainError "raccodown/internal/domain/error"
	"raccodown/internal/domain/usecase"
	"raccodown/internal/port"
)

// UseCase implements CreateNote.
type UseCase struct {
	noteRepository port.NoteRepository
}

// New builds a UseCase from its dependencies.
func New(noteRepository port.NoteRepository) *UseCase {
	return &UseCase{
		noteRepository: noteRepository,
	}
}

// Execute validates and creates a new note. A blank title resolves to usecase.DefaultNoteTitle;
// tags are derived from content, not taken from Input.
func (uc *UseCase) Execute(ctx context.Context, input Input) (Output, error) {
	if err := input.Validate(); err != nil {
		slog.InfoContext(ctx, "create note: validation", "error", err)

		return Output{}, domainError.ToValidationError(err)
	}

	note, err := uc.noteRepository.Create(ctx, port.NoteCreateRequest{
		Title:   usecase.ResolveTitle(input.Title),
		Content: input.Content,
		Tags:    usecase.ExtractTags(input.Content),
	})
	if err != nil {
		slog.ErrorContext(ctx, "create note: save", "error", err)

		return Output{}, errors.New("Failed to save note")
	}

	return Output{Note: note}, nil
}
