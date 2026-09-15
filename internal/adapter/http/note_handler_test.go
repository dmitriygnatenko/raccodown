package http

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"raccodown/internal/domain/entity"
	domainerror "raccodown/internal/domain/error"
	"raccodown/internal/port"
)

// testUser is the account stubAuthenticated signs every note/tag handler test in as — these routes
// don't care who's asking (raccodown is single-user), just that requireAuth let the request through.
var testUser = entity.User{ID: 1, Username: "raccoon"}

func TestHandleListNotes(t *testing.T) {
	t.Parallel()

	mux, deps := newTestServer()
	stubAuthenticated(deps, testUser)
	deps.notes.listFn = func(context.Context, port.NoteListFilter) ([]entity.Note, error) {
		return []entity.Note{{ID: 1, Title: "hello"}}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notes", nil)
	req.AddCookie(authCookie())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"hello"`)
}

func TestHandleListNotes_unauthenticated(t *testing.T) {
	t.Parallel()

	mux, _ := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notes", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandleCreateNote(t *testing.T) {
	t.Parallel()

	mux, deps := newTestServer()
	stubAuthenticated(deps, testUser)
	deps.notes.createFn = func(_ context.Context, req port.NoteCreateRequest) (entity.Note, error) {
		return entity.Note{ID: 1, Title: req.Title, Content: req.Content, Tags: req.Tags}, nil
	}

	body := `{"title":"t","content":"c #tag"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", strings.NewReader(body))
	req.AddCookie(authCookie())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	require.Contains(t, rec.Body.String(), `"tag"`)
}

func TestHandleGetNote(t *testing.T) {
	t.Parallel()

	t.Run("a well-formed but absent id is mapped to 404", func(t *testing.T) {
		t.Parallel()

		mux, deps := newTestServer()
		stubAuthenticated(deps, testUser)
		deps.notes.findByIDFn = func(context.Context, uint64) (entity.Note, error) {
			return entity.Note{}, &domainerror.NotFoundError{}
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/notes/999999", nil)
		req.AddCookie(authCookie())
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("a malformed id is mapped to 400 without reaching the repository", func(t *testing.T) {
		t.Parallel()

		mux, deps := newTestServer()
		stubAuthenticated(deps, testUser)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/notes/missing", nil)
		req.AddCookie(authCookie())
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("found note is returned with an ETag", func(t *testing.T) {
		t.Parallel()

		mux, deps := newTestServer()
		stubAuthenticated(deps, testUser)
		deps.notes.findByIDFn = func(context.Context, uint64) (entity.Note, error) {
			return entity.Note{ID: 1, Checksum: "abc123"}, nil
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/notes/1", nil)
		req.AddCookie(authCookie())
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "abc123", rec.Header().Get("ETag"))
	})
}

func TestHandleUpdateNote(t *testing.T) {
	t.Parallel()

	t.Run("a checksum conflict returns 409 with the current note", func(t *testing.T) {
		t.Parallel()

		mux, deps := newTestServer()
		stubAuthenticated(deps, testUser)
		deps.notes.updateFn = func(context.Context, port.NoteUpdateRequest) (entity.Note, error) {
			return entity.Note{}, &domainerror.ConflictError{}
		}
		deps.notes.findByIDFn = func(context.Context, uint64) (entity.Note, error) {
			return entity.Note{ID: 1, Title: "current", Checksum: "current-checksum"}, nil
		}

		body := `{"title":"mine","content":"c"}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/notes/1", strings.NewReader(body))
		req.Header.Set("If-Match", "stale-checksum")
		req.AddCookie(authCookie())
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusConflict, rec.Code)
		require.Equal(t, "current-checksum", rec.Header().Get("ETag"))
		require.Contains(t, rec.Body.String(), `"current"`)
	})

	t.Run("a successful update returns 200 with an ETag", func(t *testing.T) {
		t.Parallel()

		mux, deps := newTestServer()
		stubAuthenticated(deps, testUser)
		deps.notes.updateFn = func(context.Context, port.NoteUpdateRequest) (entity.Note, error) {
			return entity.Note{ID: 1, Title: "mine", Checksum: "new-checksum"}, nil
		}

		body := `{"title":"mine","content":"c"}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/notes/1", strings.NewReader(body))
		req.AddCookie(authCookie())
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "new-checksum", rec.Header().Get("ETag"))
	})
}

func TestHandleDeleteNote(t *testing.T) {
	t.Parallel()

	mux, deps := newTestServer()
	stubAuthenticated(deps, testUser)
	deps.notes.deleteFn = func(_ context.Context, id uint64) error {
		require.Equal(t, uint64(1), id)
		return nil
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/notes/1", nil)
	req.AddCookie(authCookie())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)
}

func TestHandleExportNotes(t *testing.T) {
	t.Parallel()

	mux, deps := newTestServer()
	stubAuthenticated(deps, testUser)
	deps.notes.listFn = func(context.Context, port.NoteListFilter) ([]entity.Note, error) {
		return []entity.Note{
			{ID: 1, Title: "Recipe: pad thai / extra spicy", Content: "step one"},
			{ID: 2, Title: "Recipe: pad thai / extra spicy", Content: "a duplicate title"},
			{ID: 3, Title: "  ", Content: "a blank title"},
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notes/export", nil)
	req.AddCookie(authCookie())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/zip", rec.Header().Get("Content-Type"))
	require.Contains(t, rec.Header().Get("Content-Disposition"), "attachment")

	zr, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	require.NoError(t, err)
	require.Len(t, zr.File, 3)

	got := make(map[string]string, len(zr.File))
	for _, f := range zr.File {
		rc, err := f.Open()
		require.NoError(t, err)
		content, err := io.ReadAll(rc)
		require.NoError(t, err)
		rc.Close()
		got[f.Name] = string(content)
	}

	require.Equal(t, map[string]string{
		"Recipe- pad thai - extra spicy.md":     "step one",
		"Recipe- pad thai - extra spicy (2).md": "a duplicate title",
		"Untitled.md":                           "a blank title",
	}, got)
}

func TestHandleListTags(t *testing.T) {
	t.Parallel()

	mux, deps := newTestServer()
	stubAuthenticated(deps, testUser)
	deps.notes.listFn = func(context.Context, port.NoteListFilter) ([]entity.Note, error) {
		return []entity.Note{{ID: 1, Tags: []string{"work", "home"}}, {ID: 2, Tags: []string{"work"}}}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tags", nil)
	req.AddCookie(authCookie())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `["home","work"]`, rec.Body.String())
}
