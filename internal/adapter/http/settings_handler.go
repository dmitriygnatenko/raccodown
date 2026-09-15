package http

import (
	"net/http"

	"raccodown/internal/domain/usecase/settings/update"
)

// handleGetSettings handles GET /api/v1/settings.
func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	current := userFromContext(r)
	if current == nil {
		writeError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	settings, err := s.Settings.Get.Execute(r.Context(), current.ID)
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, settings)
}

type updateSettingsRequest struct {
	Language string `json:"language"`
	Theme    string `json:"theme"`
}

// handleUpdateSettings handles PATCH /api/v1/settings.
func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	current := userFromContext(r)
	if current == nil {
		writeError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	var body updateSettingsRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	out, err := s.Settings.Update.Execute(r.Context(), update.Input{
		UserID:   current.ID,
		Language: body.Language,
		Theme:    body.Theme,
	})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, out.Settings)
}
