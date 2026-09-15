package list

import "raccodown/internal/domain/entity"

// Output is every note matching Input, newest-updated first.
type Output struct {
	Notes []entity.Note
}
