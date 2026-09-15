package http

import (
	"net/http"

	"raccodown/internal/domain/usecase/auth/login"
	"raccodown/internal/domain/usecase/auth/logout"
	"raccodown/internal/domain/usecase/auth/updatecredentials"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	// Language is the frontend's current UI language — saved as the user's preference the first
	// time they ever log in with one on record (see login.UseCase.Execute). Optional.
	Language string `json:"language"`
}

// handleLogin handles POST /api/v1/auth/login.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body loginRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	out, err := s.Auth.Login.Execute(r.Context(), login.Input{
		Username: body.Username,
		Password: body.Password,
		Language: body.Language,
	})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	s.setSessionCookie(w, out.Session)
	writeJSON(w, http.StatusOK, out.User)
}

// handleLogout handles POST /api/v1/auth/logout.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.Auth.Logout.Execute(r.Context(), logout.Input{Token: sessionToken(r)})

	s.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// handleMe handles GET /api/v1/auth/me — reports the current session's user. requireAuth has
// already verified the session and attached the user to the request context, so this just returns
// it (used by the frontend to restore a session after a page reload).
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	current := userFromContext(r)
	if current == nil {
		writeError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	writeJSON(w, http.StatusOK, current)
}

type updateCredentialsRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewUsername     string `json:"newUsername"`
	NewPassword     string `json:"newPassword"`
}

// handleUpdateCredentials handles PATCH /api/v1/auth/credentials.
func (s *Server) handleUpdateCredentials(w http.ResponseWriter, r *http.Request) {
	current := userFromContext(r)
	if current == nil {
		writeError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	var body updateCredentialsRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	out, err := s.Auth.UpdateCredentials.Execute(r.Context(), updatecredentials.Input{
		User:            *current,
		CurrentPassword: body.CurrentPassword,
		NewUsername:     body.NewUsername,
		NewPassword:     body.NewPassword,
	})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, out.User)
}
