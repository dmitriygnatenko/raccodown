package http

// This file bundles hand-rolled test doubles for every port.* dependency, plus a helper that wires
// them into a fully real Server (real use cases, fake repositories) — the same "fake at the
// repository boundary" style the use case tests use, just one layer up. Handler tests exercise the
// whole stack from an HTTP request down to a fake repository call, through the real mux (so routing,
// requireAuth and path-value parsing are covered too), and back up through the real JSON encoding.

import (
	"context"
	"net/http"
	"time"

	"raccodown/internal/domain/entity"
	"raccodown/internal/port"

	"raccodown/internal/domain/usecase/auth/authenticate"
	"raccodown/internal/domain/usecase/auth/login"
	"raccodown/internal/domain/usecase/auth/logout"
	"raccodown/internal/domain/usecase/auth/updatecredentials"

	noteCreate "raccodown/internal/domain/usecase/note/create"
	noteDelete "raccodown/internal/domain/usecase/note/delete"
	noteGet "raccodown/internal/domain/usecase/note/get"
	noteList "raccodown/internal/domain/usecase/note/list"
	noteUpdate "raccodown/internal/domain/usecase/note/update"

	settingsGet "raccodown/internal/domain/usecase/settings/get"
	settingsUpdate "raccodown/internal/domain/usecase/settings/update"

	tagList "raccodown/internal/domain/usecase/tag/list"
)

// testToken is the session token every authenticated test request carries.
const testToken = "test-session-token"

// fakeNoteRepository is a hand-rolled test double for port.NoteRepository.
type fakeNoteRepository struct {
	listFn     func(ctx context.Context, filter port.NoteListFilter) ([]entity.Note, error)
	listTagsFn func(ctx context.Context) ([]string, error)
	findByIDFn func(ctx context.Context, id uint64) (entity.Note, error)
	createFn   func(ctx context.Context, req port.NoteCreateRequest) (entity.Note, error)
	updateFn   func(ctx context.Context, req port.NoteUpdateRequest) (entity.Note, error)
	deleteFn   func(ctx context.Context, id uint64) error
}

func (f *fakeNoteRepository) List(ctx context.Context, filter port.NoteListFilter) ([]entity.Note, error) {
	return f.listFn(ctx, filter)
}

func (f *fakeNoteRepository) ListTags(ctx context.Context) ([]string, error) {
	return f.listTagsFn(ctx)
}

func (f *fakeNoteRepository) FindByID(ctx context.Context, id uint64) (entity.Note, error) {
	return f.findByIDFn(ctx, id)
}

func (f *fakeNoteRepository) Create(ctx context.Context, req port.NoteCreateRequest) (entity.Note, error) {
	return f.createFn(ctx, req)
}

func (f *fakeNoteRepository) Update(ctx context.Context, req port.NoteUpdateRequest) (entity.Note, error) {
	return f.updateFn(ctx, req)
}

func (f *fakeNoteRepository) Delete(ctx context.Context, id uint64) error {
	return f.deleteFn(ctx, id)
}

// fakeUserRepository is a hand-rolled test double for port.UserRepository.
type fakeUserRepository struct {
	findByUsernameFn    func(ctx context.Context, username string) (entity.User, error)
	findByIDFn          func(ctx context.Context, id uint64) (entity.User, error)
	createFn            func(ctx context.Context, req port.UserCreateRequest) (entity.User, error)
	updateCredentialsFn func(ctx context.Context, id uint64, username, passwordHash string) error
	getSettingsFn       func(ctx context.Context, id uint64) (entity.UserSettings, error)
	updateSettingsFn    func(ctx context.Context, id uint64, settings entity.UserSettings) error
	countFn             func(ctx context.Context) (int, error)
}

func (f *fakeUserRepository) FindByUsername(ctx context.Context, username string) (entity.User, error) {
	return f.findByUsernameFn(ctx, username)
}

func (f *fakeUserRepository) FindByID(ctx context.Context, id uint64) (entity.User, error) {
	return f.findByIDFn(ctx, id)
}

func (f *fakeUserRepository) Create(ctx context.Context, req port.UserCreateRequest) (entity.User, error) {
	return f.createFn(ctx, req)
}

func (f *fakeUserRepository) UpdateCredentials(ctx context.Context, id uint64, username, passwordHash string) error {
	return f.updateCredentialsFn(ctx, id, username, passwordHash)
}

func (f *fakeUserRepository) GetSettings(ctx context.Context, id uint64) (entity.UserSettings, error) {
	return f.getSettingsFn(ctx, id)
}

func (f *fakeUserRepository) UpdateSettings(ctx context.Context, id uint64, settings entity.UserSettings) error {
	return f.updateSettingsFn(ctx, id, settings)
}

func (f *fakeUserRepository) Count(ctx context.Context) (int, error) {
	return f.countFn(ctx)
}

// fakeSessionRepository is a hand-rolled test double for port.SessionRepository.
type fakeSessionRepository struct {
	createFn        func(ctx context.Context, session entity.Session) error
	findByTokenFn   func(ctx context.Context, token string) (entity.Session, error)
	deleteFn        func(ctx context.Context, token string) error
	deleteExpiredFn func(ctx context.Context, now time.Time) (int, error)
}

func (f *fakeSessionRepository) Create(ctx context.Context, session entity.Session) error {
	return f.createFn(ctx, session)
}

func (f *fakeSessionRepository) FindByToken(ctx context.Context, token string) (entity.Session, error) {
	return f.findByTokenFn(ctx, token)
}

func (f *fakeSessionRepository) Delete(ctx context.Context, token string) error {
	return f.deleteFn(ctx, token)
}

func (f *fakeSessionRepository) DeleteExpired(ctx context.Context, now time.Time) (int, error) {
	return f.deleteExpiredFn(ctx, now)
}

// fakePasswordHasher is a hand-rolled test double for port.PasswordHasher.
type fakePasswordHasher struct {
	hashFn    func(password string) (string, error)
	compareFn func(hash, password string) bool
}

func (f *fakePasswordHasher) Hash(password string) (string, error) {
	return f.hashFn(password)
}

func (f *fakePasswordHasher) Compare(hash, password string) bool {
	return f.compareFn(hash, password)
}

// fakeTokenGenerator is a hand-rolled test double for port.TokenGenerator.
type fakeTokenGenerator struct {
	newTokenFn func() (string, error)
}

func (f *fakeTokenGenerator) NewToken() (string, error) {
	return f.newTokenFn()
}

// testDeps bundles every fake dependency a fully wired Server depends on, so a test can stub just
// the ones its scenario touches.
type testDeps struct {
	notes    *fakeNoteRepository
	users    *fakeUserRepository
	sessions *fakeSessionRepository
	hasher   *fakePasswordHasher
	tokens   *fakeTokenGenerator
}

// newTestServer builds a Server wired entirely from real use cases backed by fresh, unstubbed fakes,
// and registers its routes onto a *http.ServeMux — so tests exercise routing, requireAuth and
// path-value parsing exactly as production does. Each fake method panics with a nil-function-call
// error until a test sets the matching *Fn field.
func newTestServer() (*http.ServeMux, *testDeps) {
	deps := &testDeps{
		notes:    &fakeNoteRepository{},
		users:    &fakeUserRepository{},
		sessions: &fakeSessionRepository{},
		hasher:   &fakePasswordHasher{},
		tokens:   &fakeTokenGenerator{},
	}

	s := &Server{
		Auth: AuthUseCases{
			Login:             login.New(deps.users, deps.sessions, deps.hasher, deps.tokens),
			Logout:            logout.New(deps.sessions),
			Authenticate:      authenticate.New(deps.sessions, deps.users),
			UpdateCredentials: updatecredentials.New(deps.users, deps.hasher),
		},
		Notes: NoteUseCases{
			Create: noteCreate.New(deps.notes),
			Get:    noteGet.New(deps.notes),
			List:   noteList.New(deps.notes),
			Update: noteUpdate.New(deps.notes),
			Delete: noteDelete.New(deps.notes),
		},
		Tags: TagUseCases{
			List: tagList.New(deps.notes),
		},
		Settings: SettingsUseCases{
			Get:    settingsGet.New(deps.users),
			Update: settingsUpdate.New(deps.users),
		},
	}

	mux := http.NewServeMux()
	s.RegisterRoutes(mux)

	return mux, deps
}

// stubAuthenticated makes testToken resolve, through requireAuth, to user — every route but
// POST /api/v1/auth/login requires this before a test request carrying authCookie() will reach its
// handler.
func stubAuthenticated(deps *testDeps, user entity.User) {
	deps.sessions.findByTokenFn = func(context.Context, string) (entity.Session, error) {
		return entity.Session{
			Token:     testToken,
			UserID:    user.ID,
			ExpiresAt: time.Now().UTC().Add(time.Hour),
		}, nil
	}
	deps.users.findByIDFn = func(context.Context, uint64) (entity.User, error) {
		return user, nil
	}
}

// authCookie is the session cookie a request needs to authenticate against stubAuthenticated.
func authCookie() *http.Cookie {
	return &http.Cookie{Name: sessionCookieName, Value: testToken}
}
