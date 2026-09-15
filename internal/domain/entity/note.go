package entity

import (
	"encoding/json"
	"time"
)

// MaxTitleLength is the maximum accepted length for a note title.
const MaxTitleLength = 255

// Note is a single Markdown/plain-text note. Tags are derived from the content (see
// usecase.ExtractTags), not entered separately, so they always match whatever #hashtags currently
// appear in Content.
type Note struct {
	ID        uint64
	Title     string
	Content   string
	Tags      []string
	CreatedAt time.Time
	UpdatedAt time.Time
	// Checksum is a content hash used for optimistic concurrency: a client must echo back the
	// Checksum it last read for Update to succeed (see NoteRepository.Update).
	Checksum string
}

// noteJSON is Note's wire shape.
type noteJSON struct {
	ID        uint64    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Checksum  string    `json:"checksum"`
}

// MarshalJSON encodes Tags as [] rather than null when there are none, which is what every consumer
// (the frontend included) expects.
func (n Note) MarshalJSON() ([]byte, error) {
	tags := n.Tags
	if tags == nil {
		tags = []string{}
	}

	return json.Marshal(noteJSON{
		ID:        n.ID,
		Title:     n.Title,
		Content:   n.Content,
		Tags:      tags,
		CreatedAt: n.CreatedAt,
		UpdatedAt: n.UpdatedAt,
		Checksum:  n.Checksum,
	})
}

// UnmarshalJSON is noteJSON's inverse.
func (n *Note) UnmarshalJSON(data []byte) error {
	var j noteJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}

	*n = Note(j)

	return nil
}
