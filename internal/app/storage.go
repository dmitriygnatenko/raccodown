package app

import (
	"context"
	"fmt"
	"log/slog"

	"raccodown/internal/adapter/mysql"
	"raccodown/internal/adapter/postgres"
	"raccodown/internal/adapter/sqlite"
	"raccodown/internal/config"
	noteRepo "raccodown/internal/repository/note"
	sessionRepo "raccodown/internal/repository/session"
	userRepo "raccodown/internal/repository/user"
)

// storage is where every repository's separate expectations of a driver adapter meet: each names
// only the table group it touches, and this is the one place that asks a single adapter to satisfy
// all of them at once — so a driver missing an operation fails to compile here, at the composition
// root, rather than anywhere downstream.
type storage interface {
	noteRepo.Storage
	userRepo.Storage
	sessionRepo.Storage

	Close() error
}

// openStorage picks the driver adapter named by cfg.Driver, makes sure the target database exists,
// opens the connection and runs migrations. Only this composition-root function knows about the
// concrete adapters — everything downstream depends on the interface it needs.
func openStorage(ctx context.Context, cfg config.DBConfig) (storage, error) {
	switch cfg.Driver {
	case config.DriverSQLite:
		sqliteCfg := sqlite.Config{
			Path:            cfg.SQLitePath,
			ConnMaxLifetime: cfg.ConnMaxLifetime,
		}

		slog.InfoContext(ctx, "opening database", "driver", cfg.Driver, "path", cfg.SQLitePath)

		return openAndMigrate(ctx,
			func() error { return sqlite.EnsureDatabase(sqliteCfg) },
			func() (*sqlite.Storage, error) { return sqlite.Open(sqliteCfg) },
			sqlite.Migrate,
		)

	case config.DriverMySQL:
		mysqlCfg := mysql.Config{
			Host:            cfg.Host,
			Port:            cfg.Port,
			User:            cfg.User,
			Password:        cfg.Password,
			Name:            cfg.Name,
			MaxOpenConns:    cfg.MaxOpenConns,
			MaxIdleConns:    cfg.MaxIdleConns,
			ConnMaxLifetime: cfg.ConnMaxLifetime,
			ConnTimeout:     cfg.ConnTimeout,
		}

		slog.InfoContext(ctx, "opening database",
			"driver", cfg.Driver, "host", cfg.Host, "port", cfg.Port, "name", cfg.Name)

		return openAndMigrate(ctx,
			func() error { return mysql.EnsureDatabase(mysqlCfg) },
			func() (*mysql.Storage, error) { return mysql.Open(mysqlCfg) },
			mysql.Migrate,
		)

	case config.DriverPostgres:
		postgresCfg := postgres.Config{
			Host:            cfg.Host,
			Port:            cfg.Port,
			User:            cfg.User,
			Password:        cfg.Password,
			Name:            cfg.Name,
			MaxOpenConns:    cfg.MaxOpenConns,
			MaxIdleConns:    cfg.MaxIdleConns,
			ConnMaxLifetime: cfg.ConnMaxLifetime,
		}

		slog.InfoContext(ctx, "opening database",
			"driver", cfg.Driver, "host", cfg.Host, "port", cfg.Port, "name", cfg.Name)

		return openAndMigrate(ctx,
			func() error { return postgres.EnsureDatabase(postgresCfg) },
			func() (*postgres.Storage, error) { return postgres.Open(postgresCfg) },
			postgres.Migrate,
		)

	default:
		return nil, fmt.Errorf("unsupported DB_DRIVER %q", cfg.Driver)
	}
}

// openAndMigrate runs the three steps every driver follows to get a ready connection: best-effort
// auto-create the target database, open it, then run migrations. ensure's failure is only logged,
// not returned — the database usually already exists (which needs no creating) and looks identical
// from here to creation having failed, so treating it as fatal would break the common case.
func openAndMigrate[T any](
	ctx context.Context,
	ensure func() error,
	open func() (T, error),
	migrate func(context.Context, T) error,
) (T, error) {
	var zero T

	if err := ensure(); err != nil {
		slog.WarnContext(
			ctx,
			"failed to auto-create the database; ignore if it already exists",
			"error", err,
		)
	}

	store, err := open()
	if err != nil {
		return zero, fmt.Errorf("failed to connect to the database: %w", err)
	}

	if err = migrate(ctx, store); err != nil {
		return zero, fmt.Errorf("failed to run migrations: %w", err)
	}

	slog.InfoContext(ctx, "migrations applied")

	return store, nil
}
