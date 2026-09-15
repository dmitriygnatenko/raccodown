package entity

import (
	"encoding/json"
	"time"
)

// MinUsernameLength/MaxUsernameLength bound a username; MinPasswordLength bounds a password. There's
// no upper password bound — bcrypt truncates at 72 bytes, which is the hasher's problem, not
// validation's.
const (
	MinUsernameLength = 3
	MaxUsernameLength = 255
	MinPasswordLength = 4
)

// User is the single account raccodown runs as — there's no multi-tenant model, just one signed-in
// user and their notes.
type User struct {
	ID           uint64
	Username     string
	PasswordHash string
	Settings     UserSettings
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Public strips the password hash — the only shape of User that ever leaves the backend.
func (u User) Public() PublicUser {
	return PublicUser{
		ID:       u.ID,
		Username: u.Username,
		Settings: u.Settings,
	}
}

// UserSettings is a user's saved UI preferences — by analogy with raccounting's entity.UserSettings,
// the same two fields (language, theme), persisted server-side instead of in the browser only.
// Either field can be blank: a fresh account has no saved preference yet, and the frontend falls back
// to browser/OS detection until one is explicitly chosen (see PATCH /api/v1/settings).
type UserSettings struct {
	Language string
	Theme    string
}

type userSettingsJSON struct {
	Language string `json:"language,omitempty"`
	Theme    string `json:"theme,omitempty"`
}

func (s UserSettings) MarshalJSON() ([]byte, error) {
	return json.Marshal(userSettingsJSON(s))
}

func (s *UserSettings) UnmarshalJSON(data []byte) error {
	var j userSettingsJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}

	*s = UserSettings(j)

	return nil
}

// PublicUser is what the API and the request context (see adapter/http middleware) carry — never
// the password hash.
type PublicUser struct {
	ID       uint64
	Username string
	Settings UserSettings
}

type publicUserJSON struct {
	ID       uint64       `json:"id"`
	Username string       `json:"username"`
	Settings UserSettings `json:"settings"`
}

func (u PublicUser) MarshalJSON() ([]byte, error) {
	return json.Marshal(publicUserJSON(u))
}

func (u *PublicUser) UnmarshalJSON(data []byte) error {
	var j publicUserJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}

	*u = PublicUser(j)

	return nil
}
