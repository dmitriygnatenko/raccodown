package create

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"

	"raccodown/internal/domain/usecase"
)

// Input is what CreateNote needs to create a new note.
type Input struct {
	Title   string
	Content string
}

// Validate rejects structurally invalid input before any repository call.
func (i Input) Validate() error {
	return validation.ValidateStruct(&i,
		validation.Field(&i.Title, usecase.NoteTitleRules()...),
	)
}
