package app

import (
	noteRepo "raccodown/internal/repository/note"
	sessionRepo "raccodown/internal/repository/session"
	userRepo "raccodown/internal/repository/user"
)

// repositories bundles every repository the use cases depend on. It exists so use-case wiring can
// take just the repositories it needs without Run growing a long, error-prone parameter list of its
// own.
type repositories struct {
	Notes    *noteRepo.Repository
	Users    *userRepo.Repository
	Sessions *sessionRepo.Repository
}

// newRepositories builds every repository against the one open database connection — see
// internal/adapter/mysql and internal/adapter/sqlite for the driver adapters and
// internal/app/storage.go for how one of them is opened.
func newRepositories(store storage) repositories {
	return repositories{
		Notes:    noteRepo.New(store),
		Users:    userRepo.New(store),
		Sessions: sessionRepo.New(store),
	}
}
