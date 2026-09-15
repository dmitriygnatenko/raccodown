package updatecredentials

import "raccodown/internal/domain/entity"

// Output is the account after the requested changes were applied.
type Output struct {
	User entity.PublicUser
}
