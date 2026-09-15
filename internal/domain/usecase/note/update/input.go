package update

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"

	"raccodown/internal/domain/usecase"
)

// Input is what UpdateNote needs to rewrite an existing note.
type Input struct {
	ID      uint64
	Title   string
	Content string
	// KnownChecksum is the Checksum the caller last read. A blank value skips the
	// optimistic-concurrency check entirely (an unconditional update).
	KnownChecksum string
}

// Validate rejects structurally invalid input before any repository call.
func (i Input) Validate() error {
	return validation.ValidateStruct(&i,
		validation.Field(&i.Title, usecase.NoteTitleRules()...),
	)
}
