// Package login is the LoginUser use case: raccodown is single-user (see the demo account seeded by
// internal/app.seedDemoUser at startup), so this only ever has one row to check credentials against
// — unlike raccounting's version, it doesn't also auto-provision on first login, since startup
// seeding already guarantees an account exists before the server ever accepts a request.
package login

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"raccodown/internal/domain/entity"
	domainerror "raccodown/internal/domain/error"
	"raccodown/internal/domain/usecase"
	"raccodown/internal/port"
)

const sessionDuration = 30 * 24 * time.Hour

// incorrectCredentials is returned for both an unknown username and a wrong password — the two are
// deliberately not distinguished to callers.
var incorrectCredentials = &domainerror.UnauthorizedError{Message: "Incorrect username or password"}

// UseCase implements LoginUser.
type UseCase struct {
	userRepository    port.UserRepository
	sessionRepository port.SessionRepository
	passwordHasher    port.PasswordHasher
	tokenGenerator    port.TokenGenerator
}

// New builds a UseCase from its dependencies.
func New(
	userRepository port.UserRepository,
	sessionRepository port.SessionRepository,
	passwordHasher port.PasswordHasher,
	tokenGenerator port.TokenGenerator,
) *UseCase {
	return &UseCase{
		userRepository:    userRepository,
		sessionRepository: sessionRepository,
		passwordHasher:    passwordHasher,
		tokenGenerator:    tokenGenerator,
	}
}

// Execute verifies the given credentials and starts a new session on success.
func (uc *UseCase) Execute(ctx context.Context, input Input) (Output, error) {
	if err := input.Validate(); err != nil {
		slog.InfoContext(ctx, "login: validation", "error", err)

		return Output{}, domainerror.ToValidationError(err)
	}

	username := usecase.NormalizeUsername(input.Username)

	user, err := uc.userRepository.FindByUsername(ctx, username)
	if err != nil {
		if !domainerror.IsNotFoundError(err) {
			slog.ErrorContext(ctx, "login: find user", "error", err)
		}

		return Output{}, incorrectCredentials
	}

	if !uc.passwordHasher.Compare(user.PasswordHash, input.Password) {
		slog.InfoContext(ctx, "login: incorrect password")

		return Output{}, incorrectCredentials
	}

	if user.Settings.Language == "" && input.Language != "" {
		uc.saveLanguage(ctx, &user, input.Language)
	}

	token, err := uc.tokenGenerator.NewToken()
	if err != nil {
		slog.ErrorContext(ctx, "login: generate token", "error", err)

		return Output{}, errors.New("Failed to generate a token")
	}

	session := entity.Session{
		Token:     token,
		UserID:    user.ID,
		ExpiresAt: time.Now().UTC().Add(sessionDuration),
	}
	if err = uc.sessionRepository.Create(ctx, session); err != nil {
		slog.ErrorContext(ctx, "login: create session", "error", err)

		return Output{}, errors.New("Failed to start a session")
	}

	return Output{
		User:    user.Public(),
		Session: session,
	}, nil
}

// saveLanguage persists language as the user's UI language preference and reflects it onto user, so
// the response built from it afterward is already up to date. It's best-effort: a failure here
// shouldn't fail the login itself, just leave the language unsaved for next time.
func (uc *UseCase) saveLanguage(ctx context.Context, user *entity.User, language string) {
	settings := entity.UserSettings{
		Language: language,
		Theme:    user.Settings.Theme,
	}

	if err := uc.userRepository.UpdateSettings(ctx, user.ID, settings); err != nil {
		slog.ErrorContext(ctx, "login: save language", "error", err)

		return
	}

	user.Settings = settings
}
