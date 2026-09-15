// Package http is the driving adapter: it exposes the application's use cases over HTTP, translating
// requests into use case Input values and use case Output/errors back into JSON responses. It owns
// nothing that looks like business logic — validation, persistence and side effects all live behind
// the use cases it calls.
package http

import (
	"net/http"

	authAuthenticate "raccodown/internal/domain/usecase/auth/authenticate"
	authLogin "raccodown/internal/domain/usecase/auth/login"
	authLogout "raccodown/internal/domain/usecase/auth/logout"
	authUpdateCreds "raccodown/internal/domain/usecase/auth/updatecredentials"

	noteCreate "raccodown/internal/domain/usecase/note/create"
	noteDelete "raccodown/internal/domain/usecase/note/delete"
	noteGet "raccodown/internal/domain/usecase/note/get"
	noteList "raccodown/internal/domain/usecase/note/list"
	noteUpdate "raccodown/internal/domain/usecase/note/update"

	settingsGet "raccodown/internal/domain/usecase/settings/get"
	settingsUpdate "raccodown/internal/domain/usecase/settings/update"

	tagList "raccodown/internal/domain/usecase/tag/list"
)

// AuthUseCases collects the use cases behind the /api/v1/auth routes.
type AuthUseCases struct {
	Login             *authLogin.UseCase
	Logout            *authLogout.UseCase
	Authenticate      *authAuthenticate.UseCase
	UpdateCredentials *authUpdateCreds.UseCase
}

// NoteUseCases collects the use cases behind the /api/v1/notes routes.
type NoteUseCases struct {
	Create *noteCreate.UseCase
	Get    *noteGet.UseCase
	List   *noteList.UseCase
	Update *noteUpdate.UseCase
	Delete *noteDelete.UseCase
}

// TagUseCases collects the use cases behind the /api/v1/tags routes.
type TagUseCases struct {
	List *tagList.UseCase
}

// SettingsUseCases collects the use cases behind the /api/v1/settings routes.
type SettingsUseCases struct {
	Get    *settingsGet.UseCase
	Update *settingsUpdate.UseCase
}

// Server holds every use case the API surfaces, plus the one setting the HTTP layer itself is
// responsible for (the session cookie's Secure flag).
type Server struct {
	Auth     AuthUseCases
	Notes    NoteUseCases
	Tags     TagUseCases
	Settings SettingsUseCases

	// CookieSecure sets the session cookie's Secure flag — true once the app is served over HTTPS.
	CookieSecure bool
}

// RegisterRoutes wires every endpoint onto mux. Every route but POST /api/v1/auth/login is wrapped
// with requireAuth.
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/health", s.handleHealth)

	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/logout", s.requireAuth(s.handleLogout))
	mux.HandleFunc("GET /api/v1/auth/me", s.requireAuth(s.handleMe))
	mux.HandleFunc("PATCH /api/v1/auth/credentials", s.requireAuth(s.handleUpdateCredentials))

	mux.HandleFunc("GET /api/v1/notes", s.requireAuth(s.handleListNotes))
	mux.HandleFunc("POST /api/v1/notes", s.requireAuth(s.handleCreateNote))
	mux.HandleFunc("GET /api/v1/notes/export", s.requireAuth(s.handleExportNotes))
	mux.HandleFunc("GET /api/v1/notes/{id}", s.requireAuth(s.handleGetNote))
	mux.HandleFunc("PUT /api/v1/notes/{id}", s.requireAuth(s.handleUpdateNote))
	mux.HandleFunc("DELETE /api/v1/notes/{id}", s.requireAuth(s.handleDeleteNote))

	mux.HandleFunc("GET /api/v1/tags", s.requireAuth(s.handleListTags))

	mux.HandleFunc("GET /api/v1/settings", s.requireAuth(s.handleGetSettings))
	mux.HandleFunc("PATCH /api/v1/settings", s.requireAuth(s.handleUpdateSettings))
}

// handleHealth handles GET /api/v1/health — a trivial liveness check. Deliberately not behind
// requireAuth: it's what an external monitor/orchestrator polls to see if the process is up at all.
func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
