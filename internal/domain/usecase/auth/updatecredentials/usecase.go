// Package updatecredentials is the UpdateCredentials use case: it lets the signed-in user change
// their username and/or password, after confirming their current password.
package updatecredentials

import (
	"context"
	"errors"
	"log/slog"

	domainerror "raccodown/internal/domain/error"
	"raccodown/internal/domain/usecase"
	"raccodown/internal/port"
)

// UseCase implements UpdateCredentials.
type UseCase struct {
	userRepository port.UserRepository
	passwordHasher port.PasswordHasher
}

// New builds a UseCase from its dependencies.
func New(userRepository port.UserRepository, passwordHasher port.PasswordHasher) *UseCase {
	return &UseCase{userRepository: userRepository, passwordHasher: passwordHasher}
}

// Execute changes the signed-in user's username and/or password, after confirming their current
// password. The username and password hash are written in a single UpdateCredentials call, so a
// request that changes both either lands both or neither — never just one.
func (uc *UseCase) Execute(ctx context.Context, input Input) (Output, error) {
	if err := input.Validate(); err != nil {
		slog.InfoContext(ctx, "update credentials: validation", "error", err)

		return Output{}, domainerror.ToValidationError(err)
	}

	current, err := uc.userRepository.FindByID(ctx, input.User.ID)
	if err != nil {
		if domainerror.IsNotFoundError(err) {
			return Output{}, &domainerror.NotFoundError{Message: "User not found"}
		}

		slog.ErrorContext(ctx, "update credentials: find user", "error", err)

		return Output{}, errors.New("Failed to verify current password")
	}

	if !uc.passwordHasher.Compare(current.PasswordHash, input.CurrentPassword) {
		slog.InfoContext(ctx, "update credentials: incorrect current password", "user_id", input.User.ID)

		return Output{}, &domainerror.UnauthorizedError{Message: "Incorrect current password"}
	}

	updated := input.User

	newUsername := usecase.NormalizeUsername(input.NewUsername)
	if newUsername == updated.Username {
		newUsername = "" // no-op rename; leave the column untouched
	}

	var newHash string

	if input.NewPassword != "" {
		newHash, err = uc.passwordHasher.Hash(input.NewPassword)
		if err != nil {
			slog.ErrorContext(ctx, "update credentials: hash password", "error", err)

			return Output{}, errors.New("Failed to process password")
		}
	}

	if newUsername != "" || newHash != "" {
		if err := uc.userRepository.UpdateCredentials(ctx, updated.ID, newUsername, newHash); err != nil {
			if domainerror.IsConflictError(err) {
				return Output{}, &domainerror.ConflictError{Message: "A user with this username is already registered"}
			}

			if domainerror.IsNotFoundError(err) {
				return Output{}, &domainerror.NotFoundError{Message: "User not found"}
			}

			slog.ErrorContext(ctx, "update credentials: save", "error", err)

			return Output{}, errors.New("Failed to update credentials")
		}

		if newUsername != "" {
			updated.Username = newUsername
		}
	}

	return Output{User: updated}, nil
}
