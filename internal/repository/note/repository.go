// Package note implements port.NoteRepository on top of the notes table: it converts row models
// into domain entities, generates note ids, computes the optimistic-concurrency checksum, and turns
// the storage layer's raw errors into domain errors — a missing row into
// *domainerror.NotFoundError, a stale checksum into *domainerror.ConflictError. Both are
// message-less; the use case supplies the text.
package note

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"slices"
	"sort"
	"strings"
	"time"

	"raccodown/internal/domain/entity"
	domainerror "raccodown/internal/domain/error"
	"raccodown/internal/port"
	"raccodown/internal/storage/model"
)

//go:generate go tool mockgen -source=repository.go -destination=mocks/storage_mock.go -package=mocks

// Storage is the slice of a driver adapter this repository uses — the notes table and nothing
// else.
type Storage interface {
	ListNotes(ctx context.Context) ([]model.Note, error)
	// FindNoteByID returns sql.ErrNoRows when no note has this id.
	FindNoteByID(ctx context.Context, id uint64) (model.Note, error)
	// CreateNote inserts a note row and returns its new, database-assigned id.
	CreateNote(ctx context.Context, note model.Note) (id uint64, err error)
	// UpdateNote overwrites a note's editable fields. found is false if no note with this id existed.
	UpdateNote(ctx context.Context, note model.Note) (found bool, err error)
	// DeleteNote removes a note row. found is false if no note with this id existed.
	DeleteNote(ctx context.Context, id uint64) (found bool, err error)
}

// Repository implements port.NoteRepository.
type Repository struct {
	storage Storage
}

// New builds a Repository against s.
func New(s Storage) *Repository {
	return &Repository{storage: s}
}

// checksum is the optimistic-concurrency token for a note's content.
func checksum(content string) string {
	sum := sha256.Sum256([]byte(content))

	return hex.EncodeToString(sum[:])[:16]
}

// List returns every note matching filter, newest-updated first. Filtering happens here rather than
// in SQL, matching the whole table read of Storage.ListNotes — see the doc comment there.
func (r *Repository) List(ctx context.Context, filter port.NoteListFilter) ([]entity.Note, error) {
	rows, err := r.storage.ListNotes(ctx)
	if err != nil {
		return nil, err
	}

	query := strings.ToLower(filter.Query)
	tag := strings.ToLower(filter.Tag)

	notes := make([]entity.Note, 0, len(rows))

	for _, row := range rows {
		note := row.ToEntity()

		if query != "" &&
			!strings.Contains(strings.ToLower(note.Title), query) &&
			!strings.Contains(strings.ToLower(note.Content), query) {
			continue
		}

		if tag != "" && !slices.Contains(note.Tags, tag) {
			continue
		}

		notes = append(notes, note)
	}

	sort.Slice(notes, func(i, j int) bool { return notes[i].UpdatedAt.After(notes[j].UpdatedAt) })

	return notes, nil
}

// FindByID returns a message-less *domainerror.NotFoundError if no note with this id exists.
func (r *Repository) FindByID(ctx context.Context, id uint64) (entity.Note, error) {
	m, err := r.storage.FindNoteByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.Note{}, &domainerror.NotFoundError{}
		}

		return entity.Note{}, err
	}

	return m.ToEntity(), nil
}

// Create inserts a new note and returns it. The id is assigned by the database (autoincrement), not
// generated here.
func (r *Repository) Create(ctx context.Context, req port.NoteCreateRequest) (entity.Note, error) {
	now := time.Now().UTC()

	note := entity.Note{
		Title:     req.Title,
		Content:   req.Content,
		Tags:      req.Tags,
		CreatedAt: now,
		UpdatedAt: now,
		Checksum:  checksum(req.Content),
	}

	id, err := r.storage.CreateNote(ctx, toModel(note))
	if err != nil {
		return entity.Note{}, err
	}

	note.ID = id

	return note, nil
}

// Update rewrites an existing note. An unknown id is reported as a message-less
// *domainerror.NotFoundError, and a stale req.KnownChecksum as a message-less
// *domainerror.ConflictError — the use case supplies the message either way.
func (r *Repository) Update(ctx context.Context, req port.NoteUpdateRequest) (entity.Note, error) {
	existing, err := r.storage.FindNoteByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.Note{}, &domainerror.NotFoundError{}
		}

		return entity.Note{}, err
	}

	if req.KnownChecksum != "" && req.KnownChecksum != existing.Checksum {
		return entity.Note{}, &domainerror.ConflictError{}
	}

	note := entity.Note{
		ID:        req.ID,
		Title:     req.Title,
		Content:   req.Content,
		Tags:      req.Tags,
		CreatedAt: existing.CreatedAt,
		UpdatedAt: time.Now().UTC(),
		Checksum:  checksum(req.Content),
	}

	found, err := r.storage.UpdateNote(ctx, toModel(note))
	if err != nil {
		return entity.Note{}, err
	}

	if !found {
		return entity.Note{}, &domainerror.NotFoundError{}
	}

	return note, nil
}

// Delete removes a note, reporting a message-less *domainerror.NotFoundError if it doesn't exist.
func (r *Repository) Delete(ctx context.Context, id uint64) error {
	found, err := r.storage.DeleteNote(ctx, id)
	if err != nil {
		return err
	}

	if !found {
		return &domainerror.NotFoundError{}
	}

	return nil
}

func toModel(note entity.Note) model.Note {
	return model.Note{
		ID:        note.ID,
		Title:     note.Title,
		Content:   note.Content,
		Tags:      note.Tags,
		Checksum:  note.Checksum,
		CreatedAt: note.CreatedAt,
		UpdatedAt: note.UpdatedAt,
	}
}
