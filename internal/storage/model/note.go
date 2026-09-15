package model

import (
	"time"

	"raccodown/internal/domain/entity"
)

// Note is the shape of a row in the notes table. Tags aren't a column — they live in the tags/
// note_tags tables (see adapter/mysql/note.go and adapter/sqlite/note.go) and are populated
// separately after the row scan.
type Note struct {
	ID        uint64
	Title     string
	Content   string
	Tags      []string
	Checksum  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ToEntity converts the stored row into a domain entity.Note.
func (m Note) ToEntity() entity.Note {
	return entity.Note{
		ID:        m.ID,
		Title:     m.Title,
		Content:   m.Content,
		Tags:      m.Tags,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		Checksum:  m.Checksum,
	}
}
