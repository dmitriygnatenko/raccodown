package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	raccodownWeb "raccodown/web"

	httpAPI "raccodown/internal/adapter/http"

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

// newServer wires every use case the HTTP API depends on, grouped the same way httpAPI.Server groups
// its routes. Keeping this assembly in one function makes it the single place that shows, for any
// given use case, exactly which repository feeds it.
func newServer(repos repositories, svcs services, cookieSecure bool) *httpAPI.Server {
	return &httpAPI.Server{
		Auth: httpAPI.AuthUseCases{
			Login:             authLogin.New(repos.Users, repos.Sessions, svcs.Hasher, svcs.Tokens),
			Logout:            authLogout.New(repos.Sessions),
			Authenticate:      authAuthenticate.New(repos.Sessions, repos.Users),
			UpdateCredentials: authUpdateCreds.New(repos.Users, svcs.Hasher),
		},
		Notes: httpAPI.NoteUseCases{
			Create: noteCreate.New(repos.Notes),
			Get:    noteGet.New(repos.Notes),
			List:   noteList.New(repos.Notes),
			Update: noteUpdate.New(repos.Notes),
			Delete: noteDelete.New(repos.Notes),
		},
		Tags: httpAPI.TagUseCases{
			List: tagList.New(repos.Notes),
		},
		Settings: httpAPI.SettingsUseCases{
			Get:    settingsGet.New(repos.Users),
			Update: settingsUpdate.New(repos.Users),
		},
		CookieSecure: cookieSecure,
	}
}

// newHandler registers the API routes and the embedded frontend on one mux, then wraps it with the
// CORS and logging middleware that every request — API or static file — should go through.
func newHandler(srv *httpAPI.Server) http.Handler {
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	mux.Handle("/", http.FileServer(http.FS(raccodownWeb.WebFiles)))

	return httpAPI.WithLogging(httpAPI.WithCORS(mux))
}

// newHTTPServer builds the server with timeouts tuned for this app rather than net/http's defaults.
func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:    addr,
		Handler: handler,
		// ReadHeaderTimeout guards against slow-header attacks (slowloris); ReadTimeout/WriteTimeout
		// stay generous enough to cover a slow connection without cutting it off.
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
}

// shutdownTimeout bounds how long graceful shutdown waits for in-flight requests to finish before
// giving up and forcibly closing whatever connections are still open.
const shutdownTimeout = 10 * time.Second

// runServer serves on httpServer until ctx is canceled, then drains in-flight requests via Shutdown
// instead of cutting them off. A fresh, un-canceled context is used for Shutdown itself since ctx
// just fired as the reason to stop.
func runServer(ctx context.Context, httpServer *http.Server) error {
	serveErr := make(chan error, 1)

	go func() {
		serveErr <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		return fmt.Errorf("server stopped unexpectedly: %w", err)

	case <-ctx.Done():
		slog.Info("shutting down")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("failed to shut down cleanly: %w", err)
		}

		return nil
	}
}
