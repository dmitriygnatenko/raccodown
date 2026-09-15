// Package usecase holds the small pieces every use case package shares: validation rule sets, tag
// extraction, username normalization — kept in one place so they stay consistent across note, tag
// and auth use cases.
package usecase

import (
	"fmt"
	"regexp"
	"strings"

	validation "github.com/go-ozzo/ozzo-validation/v4"

	"raccodown/internal/domain/entity"
)

// NoteTitleRules is the ozzo-validation rule set for a note title field. Not Required — a blank
// title is a normal note, resolved to a default at the use case (see note/create).
func NoteTitleRules() []validation.Rule {
	return []validation.Rule{
		validation.Length(0, entity.MaxTitleLength).
			Error(fmt.Sprintf("Title must be at most %d characters", entity.MaxTitleLength)),
	}
}

// DefaultNoteTitle is used whenever a note is created with a blank title.
const DefaultNoteTitle = "Untitled"

// ResolveTitle trims the given title, falling back to DefaultNoteTitle when blank.
func ResolveTitle(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return DefaultNoteTitle
	}

	return title
}

// ExtractTags returns every #hashtag in content, lowercased, in first-seen order, deduplicated.
// A run of bare "#" characters (a Markdown heading marker: "#", "##", ...) is not a tag.
func ExtractTags(content string) []string {
	seen := make(map[string]bool)
	tags := []string{} // never nil: entity.Note.MarshalJSON aside, callers shouldn't have to guard

	for word := range strings.FieldsSeq(content) {
		word = strings.TrimRight(word, ".,!?;:")

		if !strings.HasPrefix(word, "#") {
			continue
		}

		tag := strings.ToLower(strings.TrimLeft(word, "#"))
		if tag == "" {
			continue // markdown heading marker (#, ##, ...), not a tag
		}

		if !seen[tag] {
			seen[tag] = true
			tags = append(tags, tag)
		}
	}

	return tags
}

// NormalizeUsername lower-cases and trims a username, so "Admin " and "admin" are the same account.
func NormalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

// UsernameRules is the ozzo-validation rule set for a username field: required, and between
// entity.MinUsernameLength and entity.MaxUsernameLength characters.
func UsernameRules() []validation.Rule {
	return []validation.Rule{
		validation.Required.Error("Please enter a username"),
		validation.Length(entity.MinUsernameLength, entity.MaxUsernameLength).
			Error(fmt.Sprintf(
				"Username must be between %d and %d characters",
				entity.MinUsernameLength, entity.MaxUsernameLength,
			)),
	}
}

// PasswordRules is the ozzo-validation rule set for a password field: required, and at least
// entity.MinPasswordLength characters. Required is chained in front of Length with its own message
// — Length alone treats an empty value as valid, which would let a blank password slip through.
func PasswordRules() []validation.Rule {
	return []validation.Rule{
		validation.Required.Error("Please enter a password"),
		validation.Length(entity.MinPasswordLength, 0).
			Error(fmt.Sprintf("Password must be at least %d characters", entity.MinPasswordLength)),
	}
}

// languageRE matches a BCP 47-ish UI language tag: 2-3 lowercase letters, optionally followed by a
// dash and an uppercase region (e.g. "en", "ru", "pt-BR") — by analogy with raccounting's own
// languageRE, this covers the frontend's SUPPORTED_LANGUAGES without hard-coding that specific list
// here.
var languageRE = regexp.MustCompile(`^[a-z]{2,3}(-[A-Z]{2})?$`)

// LanguageRules is the ozzo-validation rule set for a UI language field.
func LanguageRules() []validation.Rule {
	return []validation.Rule{
		validation.Required.Error("Please choose a language"),
		validation.Match(languageRE).Error(`Language must be a valid language code, e.g. "en"`),
	}
}

// ThemeRules is the ozzo-validation rule set for a UI theme field: required, and either "light" or
// "dark".
func ThemeRules() []validation.Rule {
	return []validation.Rule{
		validation.Required.Error("Please choose a theme"),
		validation.In("light", "dark").Error(`Theme must be "light" or "dark"`),
	}
}
