package http

import "net/http"

// handleListTags handles GET /api/v1/tags — every distinct tag currently in use.
func (s *Server) handleListTags(w http.ResponseWriter, r *http.Request) {
	out, err := s.Tags.List.Execute(r.Context())
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, out.Tags)
}
