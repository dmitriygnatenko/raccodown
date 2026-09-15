package http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"raccodown/internal/domain/entity"
	"raccodown/internal/domain/usecase/auth/authenticate"
)

// WithCORS exists for local iteration against a frontend served some other way than this binary's
// own go:embed (the normal case doesn't need it — same origin, same port). NOTE: auth is
// cookie-based. If the frontend is ever actually served from a different origin, "*" here won't
// work for it — browsers refuse to send credentialed requests to a wildcard origin. In that case,
// replace "*" with the frontend's exact origin and add Access-Control-Allow-Credentials: true.
func WithCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, If-Match")
		w.Header().Set("Access-Control-Expose-Headers", "ETag")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// WithLogging writes a short line per request, at debug level (quiet by default — see
// config.LoadLog).
func WithLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		slog.Debug(fmt.Sprintf("%s %s %s", r.Method, r.URL.Path, time.Since(start)))
	})
}

type ctxKey string

const userContextKey ctxKey = "currentUser"

// userFromContext returns the user requireAuth attached to the request context, or nil if none is
// present.
func userFromContext(r *http.Request) *entity.PublicUser {
	u, _ := r.Context().Value(userContextKey).(*entity.PublicUser)
	return u
}

// requireAuth is the middleware every route but POST /api/v1/auth/login is wrapped with: without a
// valid session, it never reaches next.
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := s.Auth.Authenticate.Execute(r.Context(), authenticate.Input{Token: sessionToken(r)})
		if err != nil {
			writeError(w, http.StatusUnauthorized, "Authentication required")
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, &out.User)

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
