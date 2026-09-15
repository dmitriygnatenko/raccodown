package http

import (
	"archive/zip"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"raccodown/internal/domain/usecase/note/create"
	"raccodown/internal/domain/usecase/note/delete"
	"raccodown/internal/domain/usecase/note/get"
	"raccodown/internal/domain/usecase/note/list"
	"raccodown/internal/domain/usecase/note/update"
)

type noteRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// handleListNotes handles GET /api/v1/notes?q=&tag=.
func (s *Server) handleListNotes(w http.ResponseWriter, r *http.Request) {
	out, err := s.Notes.List.Execute(r.Context(), list.Input{
		Query: r.URL.Query().Get("q"),
		Tag:   r.URL.Query().Get("tag"),
	})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"notes": out.Notes, "total": len(out.Notes)})
}

// handleCreateNote handles POST /api/v1/notes.
func (s *Server) handleCreateNote(w http.ResponseWriter, r *http.Request) {
	var body noteRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	out, err := s.Notes.Create.Execute(r.Context(), create.Input{
		Title:   body.Title,
		Content: body.Content,
	})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, out.Note)
}

// handleGetNote handles GET /api/v1/notes/{id}.
func (s *Server) handleGetNote(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUint64ID(w, r, "id")
	if !ok {
		return
	}

	out, err := s.Notes.Get.Execute(r.Context(), get.Input{ID: id})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	w.Header().Set("ETag", out.Note.Checksum)
	writeJSON(w, http.StatusOK, out.Note)
}

// handleUpdateNote handles PUT /api/v1/notes/{id}. A conflicting write (the If-Match header no
// longer matches the note's checksum) comes back as 409 with the note's current state alongside the
// error, so the client can offer the user a merge instead of just failing.
func (s *Server) handleUpdateNote(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUint64ID(w, r, "id")
	if !ok {
		return
	}

	var body noteRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	out, err := s.Notes.Update.Execute(r.Context(), update.Input{
		ID:            id,
		Title:         body.Title,
		Content:       body.Content,
		KnownChecksum: r.Header.Get("If-Match"),
	})
	if err != nil {
		var conflict *update.ConflictError
		if errors.As(err, &conflict) {
			w.Header().Set("ETag", conflict.Current.Checksum)
			writeJSON(w, http.StatusConflict, map[string]any{
				"error":   conflict.Error(),
				"current": conflict.Current,
			})
			return
		}

		writeUseCaseError(w, err)
		return
	}

	w.Header().Set("ETag", out.Note.Checksum)
	writeJSON(w, http.StatusOK, out.Note)
}

// handleExportNotes handles GET /api/v1/notes/export: every note as its own .md file (named after
// its title, content written verbatim) bundled into one zip — the plain-files-on-disk shape the
// project's real persistence layer is headed toward anyway, so an export just needs to reuse the
// List use case rather than a dedicated one of its own; building a zip is a presentation concern,
// not a business rule.
func (s *Server) handleExportNotes(w http.ResponseWriter, r *http.Request) {
	out, err := s.Notes.List.Execute(r.Context(), list.Input{})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="raccodown-export.zip"`)

	zw := zip.NewWriter(w)
	defer zw.Close()

	used := make(map[string]int)
	for _, note := range out.Notes {
		f, err := zw.Create(exportFilename(note.Title, used))
		if err != nil {
			return // headers are already sent; nothing more useful to do than stop
		}

		if _, err := f.Write([]byte(note.Content)); err != nil {
			return
		}
	}
}

// invalidFilenameChars matches characters that can't appear in a filename on at least one of
// Windows/macOS/Linux.
var invalidFilenameChars = regexp.MustCompile(`[/\\:*?"<>|]`)

// exportFilename turns a note title into a safe, unique "<title>.md" — disambiguating a repeated
// title (or a title that sanitizes to nothing) by appending " (n)", counted via used.
func exportFilename(title string, used map[string]int) string {
	name := strings.TrimSpace(invalidFilenameChars.ReplaceAllString(title, "-"))
	if name == "" {
		name = "Untitled"
	}

	used[name]++
	if n := used[name]; n > 1 {
		name = fmt.Sprintf("%s (%d)", name, n)
	}

	return name + ".md"
}

// handleDeleteNote handles DELETE /api/v1/notes/{id}.
func (s *Server) handleDeleteNote(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUint64ID(w, r, "id")
	if !ok {
		return
	}

	if err := s.Notes.Delete.Execute(r.Context(), delete.Input{ID: id}); err != nil {
		writeUseCaseError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
