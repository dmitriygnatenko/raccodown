package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"raccodown/internal/domain/entity"
)

func TestHandleLogin(t *testing.T) {
	t.Parallel()

	t.Run("correct credentials set the session cookie", func(t *testing.T) {
		t.Parallel()

		mux, deps := newTestServer()
		deps.users.findByUsernameFn = func(context.Context, string) (entity.User, error) {
			return entity.User{ID: 1, Username: "raccoon", PasswordHash: "hashed"}, nil
		}
		deps.hasher.compareFn = func(hash, password string) bool {
			return hash == "hashed" && password == "correct-horse"
		}
		deps.tokens.newTokenFn = func() (string, error) { return "new-token", nil }
		deps.sessions.createFn = func(context.Context, entity.Session) error { return nil }

		body := `{"username":"raccoon","password":"correct-horse"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Contains(t, rec.Body.String(), `"raccoon"`)

		cookies := rec.Result().Cookies()
		require.Len(t, cookies, 1)
		require.Equal(t, sessionCookieName, cookies[0].Name)
		require.Equal(t, "new-token", cookies[0].Value)
		require.True(t, cookies[0].HttpOnly)
	})

	t.Run("wrong password is a 401 without revealing which field was wrong", func(t *testing.T) {
		t.Parallel()

		mux, deps := newTestServer()
		deps.users.findByUsernameFn = func(context.Context, string) (entity.User, error) {
			return entity.User{ID: 1, Username: "raccoon", PasswordHash: "hashed"}, nil
		}
		deps.hasher.compareFn = func(string, string) bool { return false }

		body := `{"username":"raccoon","password":"wrong-password"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestHandleLogout(t *testing.T) {
	t.Parallel()

	mux, deps := newTestServer()
	stubAuthenticated(deps, entity.User{ID: 1, Username: "raccoon"})
	deps.sessions.deleteFn = func(_ context.Context, token string) error {
		require.Equal(t, testToken, token)
		return nil
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.AddCookie(authCookie())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)

	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)
	require.Equal(t, sessionCookieName, cookies[0].Name)
	require.Negative(t, cookies[0].MaxAge)
}

func TestHandleMe(t *testing.T) {
	t.Parallel()

	t.Run("authenticated request returns the current user", func(t *testing.T) {
		t.Parallel()

		mux, deps := newTestServer()
		stubAuthenticated(deps, entity.User{ID: 1, Username: "raccoon"})

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req.AddCookie(authCookie())
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Contains(t, rec.Body.String(), `"raccoon"`)
	})

	t.Run("no session cookie is a 401", func(t *testing.T) {
		t.Parallel()

		mux, _ := newTestServer()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("expired session is a 401", func(t *testing.T) {
		t.Parallel()

		mux, deps := newTestServer()
		deps.sessions.findByTokenFn = func(context.Context, string) (entity.Session, error) {
			return entity.Session{Token: testToken, UserID: 1, ExpiresAt: time.Now().UTC().Add(-time.Hour)}, nil
		}
		deps.sessions.deleteFn = func(context.Context, string) error { return nil }

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req.AddCookie(authCookie())
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestHandleUpdateCredentials(t *testing.T) {
	t.Parallel()

	t.Run("correct current password updates the password", func(t *testing.T) {
		t.Parallel()

		mux, deps := newTestServer()
		stubAuthenticated(deps, entity.User{ID: 1, Username: "raccoon"})
		deps.users.findByIDFn = func(context.Context, uint64) (entity.User, error) {
			return entity.User{ID: 1, Username: "raccoon", PasswordHash: "hashed"}, nil
		}
		deps.hasher.compareFn = func(hash, password string) bool {
			return hash == "hashed" && password == "correct-horse"
		}
		deps.hasher.hashFn = func(string) (string, error) { return "new-hash", nil }
		deps.users.updateCredentialsFn = func(context.Context, uint64, string, string) error { return nil }

		body := `{"currentPassword":"correct-horse","newPassword":"battery-staple"}`
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/credentials", strings.NewReader(body))
		req.AddCookie(authCookie())
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("incorrect current password is a 401", func(t *testing.T) {
		t.Parallel()

		mux, deps := newTestServer()
		stubAuthenticated(deps, entity.User{ID: 1, Username: "raccoon"})
		deps.users.findByIDFn = func(context.Context, uint64) (entity.User, error) {
			return entity.User{ID: 1, Username: "raccoon", PasswordHash: "hashed"}, nil
		}
		deps.hasher.compareFn = func(string, string) bool { return false }

		body := `{"currentPassword":"wrong","newPassword":"battery-staple"}`
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/credentials", strings.NewReader(body))
		req.AddCookie(authCookie())
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}
