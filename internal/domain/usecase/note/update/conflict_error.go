package update

import "raccodown/internal/domain/entity"

// ConflictError signals that Input.KnownChecksum no longer matched the note's checksum — someone
// else's write got there first. Current is the note's present state, so the caller (the HTTP
// adapter) can offer the user a merge instead of just reporting failure.
//
// This carries a payload domainerror.ConflictError doesn't, so it's its own type rather than that
// one. adapter/http checks for it specifically (via errors.As) before falling back to the generic
// domain-error-to-HTTP-status mapping, so it can put Current in the response body alongside the
// 409.
type ConflictError struct {
	Current entity.Note
}

// Error implements the error interface.
func (e *ConflictError) Error() string {
	return "Note changed since last read"
}
