// Package error holds the errors the storage layer raises on its own behalf — the ones with no
// portable representation in database/sql, which the driver adapter (internal/adapter/mysql or
// internal/adapter/sqlite) therefore has to normalize, in its own dialect, before a repository can
// recognize them. It deliberately names itself after the predeclared type, like
// internal/domain/error does, so importers alias it (storageError) and the two never blur together
// at a call site.
package error

import "errors"

// UniqueViolationError reports that a write collided with a UNIQUE constraint (SQLite's "UNIQUE
// constraint failed", MySQL's ER_DUP_ENTRY). The adapter detects it in its own dialect and wraps it
// in this sentinel; the
// repository that recognizes it with errors.Is is what turns it into a message-less domain
// *ConflictError, leaving the use case to supply wording that fits the record in question. Nothing
// maps this one to an HTTP status — it isn't meant to travel past the repository.
var UniqueViolationError = errors.New("unique constraint violation")
