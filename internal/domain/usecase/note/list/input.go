package list

// Input narrows ListNotes. A blank field means "don't filter on this".
type Input struct {
	Query string
	Tag   string
}
