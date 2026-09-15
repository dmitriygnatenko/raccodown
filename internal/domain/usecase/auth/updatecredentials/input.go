package updatecredentials

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"

	"raccodown/internal/domain/entity"
	"raccodown/internal/domain/usecase"
)

// Input is what UpdateCredentials needs to change the signed-in user's username and/or password.
// NewUsername/NewPassword are optional — a blank one leaves that field unchanged.
type Input struct {
	User            entity.PublicUser
	CurrentPassword string
	NewUsername     string
	NewPassword     string
}

// Validate rejects a blank current password before any repository lookup, and a non-blank new
// username/password that fails its own rules. NewUsername is checked after the same
// trim/lowercase normalization the use case persists it under, so a value like " a " can't pass
// validation on its raw length and then get stored below the minimum once normalized.
func (i Input) Validate() error {
	return validation.ValidateStruct(&i,
		validation.Field(&i.CurrentPassword, validation.Required.Error("Please enter your current password")),
		validation.Field(&i.NewUsername, validation.When(i.NewUsername != "",
			validation.By(func(value any) error {
				return validation.Validate(usecase.NormalizeUsername(value.(string)), usecase.UsernameRules()...)
			}),
		)),
		validation.Field(&i.NewPassword, validation.When(i.NewPassword != "", usecase.PasswordRules()...)),
	)
}
