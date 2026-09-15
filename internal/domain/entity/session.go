package entity

import "time"

// Session is a signed-in user's server-side session record — the value behind the session cookie
// (see adapter/http/cookies.go).
type Session struct {
	Token     string
	UserID    uint64
	ExpiresAt time.Time
	CreatedAt time.Time
}
