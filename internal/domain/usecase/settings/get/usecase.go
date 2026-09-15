// Package get is the GetSettings use case: it returns the signed-in user's saved UI settings.
package get

import (
	"context"
	"errors"
	"log/slog"

	"raccodown/internal/domain/entity"
	domainerror "raccodown/internal/domain/error"
	"raccodown/internal/port"
)

// UseCase implements GetSettings.
type UseCase struct {
	userRepository port.UserRepository
}

// New builds a UseCase from its dependencies.
func New(userRepository port.UserRepository) *UseCase {
	return &UseCase{userRepository: userRepository}
}

// Execute returns the signed-in user's saved settings.
func (uc *UseCase) Execute(ctx context.Context, userID uint64) (entity.UserSettings, error) {
	settings, err := uc.userRepository.GetSettings(ctx, userID)
	if err != nil {
		if domainerror.IsNotFoundError(err) {
			return entity.UserSettings{}, &domainerror.NotFoundError{Message: "User not found"}
		}

		slog.ErrorContext(ctx, "get settings: load", "error", err)

		return entity.UserSettings{}, errors.New("Failed to load settings")
	}

	return settings, nil
}
