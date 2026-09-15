// Package update is the UpdateNote use case.
package update

import (
	"context"
	"errors"
	"log/slog"

	domainError "raccodown/internal/domain/error"
	"raccodown/internal/domain/usecase"
	"raccodown/internal/port"
)

// UseCase implements UpdateNote.
type UseCase struct {
	noteRepository port.NoteRepository
}

// New builds a UseCase from its dependencies.
func New(noteRepository port.NoteRepository) *UseCase {
	return &UseCase{
		noteRepository: noteRepository,
	}
}

// Execute validates and rewrites an existing note. Tags are re-derived from content on every
// update, so editing #tags in the text is the only way to change them.
func (uc *UseCase) Execute(ctx context.Context, input Input) (Output, error) {
	if err := input.Validate(); err != nil {
		slog.InfoContext(ctx, "update note: validation", "error", err)

		return Output{}, domainError.ToValidationError(err)
	}

	note, err := uc.noteRepository.Update(ctx, port.NoteUpdateRequest{
		ID:            input.ID,
		Title:         usecase.ResolveTitle(input.Title),
		Content:       input.Content,
		Tags:          usecase.ExtractTags(input.Content),
		KnownChecksum: input.KnownChecksum,
	})
	if err != nil {
		if domainError.IsNotFoundError(err) {
			return Output{}, &domainError.NotFoundError{Message: "Note not found"}
		}

		if domainError.IsConflictError(err) {
			return uc.conflict(ctx, input.ID)
		}

		slog.ErrorContext(ctx, "update note: save", "error", err)

		return Output{}, errors.New("Failed to save note")
	}

	return Output{Note: note}, nil
}

// conflict loads the note's current state so ConflictError can carry it back to the caller. If the
// note was deleted between the failed Update and this re-fetch, that's reported as a 404 like any
// other missing-note path, not as a generic save failure.
func (uc *UseCase) conflict(ctx context.Context, id uint64) (Output, error) {
	current, err := uc.noteRepository.FindByID(ctx, id)
	if err != nil {
		if domainError.IsNotFoundError(err) {
			return Output{}, &domainError.NotFoundError{Message: "Note not found"}
		}

		slog.ErrorContext(ctx, "update note: load current after conflict", "error", err)

		return Output{}, errors.New("Failed to save note")
	}

	return Output{}, &ConflictError{Current: current}
}
