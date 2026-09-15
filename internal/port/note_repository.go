package port

//go:generate go tool mockgen -source=note_repository.go -destination=mocks/note_repository_mock.go -package=mocks

import (
	"context"

	"raccodown/internal/domain/entity"
)

// NoteListFilter narrows NoteRepository.List. A blank field means "don't filter on this".
type NoteListFilter struct {
	// Query matches against title and content, case-insensitively.
	Query string
	// Tag matches a single lowercased tag exactly.
	Tag string
}

// NoteCreateRequest bundles the NoteRepository.Create parameters.
type NoteCreateRequest struct {
	Title   string
	Content string
	Tags    []string
}

// NoteUpdateRequest bundles the NoteRepository.Update parameters.
type NoteUpdateRequest struct {
	ID      uint64
	Title   string
	Content string
	Tags    []string
	// KnownChecksum is the Checksum the caller last read. A blank value skips the
	// optimistic-concurrency check entirely (an unconditional update).
	KnownChecksum string
}

// NoteRepository persists Notes.
type NoteRepository interface {
	List(ctx context.Context, filter NoteListFilter) ([]entity.Note, error)
	// FindByID returns a *domainerror.NotFoundError if no note with this id exists.
	FindByID(ctx context.Context, id uint64) (entity.Note, error)
	Create(ctx context.Context, req NoteCreateRequest) (entity.Note, error)
	// Update returns a *domainerror.NotFoundError if no note with this id exists, and a
	// *domainerror.ConflictError if req.KnownChecksum is set and doesn't match the note's current
	// checksum.
	Update(ctx context.Context, req NoteUpdateRequest) (entity.Note, error)
	// Delete returns a *domainerror.NotFoundError if no note with this id exists.
	Delete(ctx context.Context, id uint64) error
}
