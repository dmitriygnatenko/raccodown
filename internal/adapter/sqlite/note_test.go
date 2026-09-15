package sqlite

import (
	"context"
	"database/sql"
	"sort"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/require"

	"raccodown/internal/storage/model"
)

// fakeTags returns n distinct tag names, sorted — real storage always returns a note's tags sorted
// (see tagsForNote/tagsForNotes), so fixtures are pre-sorted to make round-trip comparisons exact.
func fakeTags(n int) []string {
	seen := make(map[string]bool, n)
	tags := make([]string, 0, n)

	for len(tags) < n {
		tag := gofakeit.LoremIpsumWord()
		if seen[tag] {
			continue
		}

		seen[tag] = true
		tags = append(tags, tag)
	}

	sort.Strings(tags)

	return tags
}

// fakeNoteModel builds a note ready to insert — ID is left zero, since it's assigned by the
// database on CreateNote, not generated here.
func fakeNoteModel() model.Note {
	now := gofakeit.Date().UTC().Truncate(time.Second)

	return model.Note{
		Title:     gofakeit.Sentence(3),
		Content:   gofakeit.Paragraph(1, 3, 8, " "),
		Tags:      fakeTags(2),
		Checksum:  gofakeit.LetterN(16),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// createFakeNote inserts note and returns the full row, including its database-assigned id, so
// tests can exercise lookups/updates against a real one.
func createFakeNote(t *testing.T, s *Storage, note model.Note) model.Note {
	t.Helper()

	id, err := s.CreateNote(context.Background(), note)
	require.NoError(t, err)

	note.ID = id

	return note
}

// TestNote_CreateAndFind covers the round trip, including the tags/note_tags tables, and the
// sql.ErrNoRows miss.
func TestNote_CreateAndFind(t *testing.T) {
	t.Parallel()

	s := newTestStorage(t)
	ctx := context.Background()

	note := createFakeNote(t, s, fakeNoteModel())

	got, err := s.FindNoteByID(ctx, note.ID)
	require.NoError(t, err)
	require.Equal(t, note, got)

	_, err = s.FindNoteByID(ctx, 999999999)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// TestNote_CreateWithoutTags covers a note with no tags round-tripping as an empty slice rather than
// an error.
func TestNote_CreateWithoutTags(t *testing.T) {
	t.Parallel()

	s := newTestStorage(t)
	ctx := context.Background()

	fixture := fakeNoteModel()
	fixture.Tags = nil
	note := createFakeNote(t, s, fixture)

	got, err := s.FindNoteByID(ctx, note.ID)
	require.NoError(t, err)
	require.Empty(t, got.Tags)
}

// TestNote_CreateReusesExistingTag covers two notes sharing a tag: the tags table gets one row, not
// two, and both notes still report it.
func TestNote_CreateReusesExistingTag(t *testing.T) {
	t.Parallel()

	s := newTestStorage(t)
	ctx := context.Background()

	shared := gofakeit.LoremIpsumWord()

	aFixture := fakeNoteModel()
	aFixture.Tags = []string{shared}
	a := createFakeNote(t, s, aFixture)

	bFixture := fakeNoteModel()
	bFixture.Tags = []string{shared}
	b := createFakeNote(t, s, bFixture)

	var tagRows int
	require.NoError(t, s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM tags WHERE name = ?`, shared).Scan(&tagRows))
	require.Equal(t, 1, tagRows)

	gotA, err := s.FindNoteByID(ctx, a.ID)
	require.NoError(t, err)
	require.Equal(t, []string{shared}, gotA.Tags)

	gotB, err := s.FindNoteByID(ctx, b.ID)
	require.NoError(t, err)
	require.Equal(t, []string{shared}, gotB.Tags)
}

// TestNote_ListNotes covers reading back every row, unfiltered — filtering happens in the repository
// layer.
func TestNote_ListNotes(t *testing.T) {
	t.Parallel()

	s := newTestStorage(t)
	ctx := context.Background()

	a := createFakeNote(t, s, fakeNoteModel())
	b := createFakeNote(t, s, fakeNoteModel())

	got, err := s.ListNotes(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, []model.Note{a, b}, got)
}

// TestNote_UpdateNote covers rewriting a note's editable fields and replacing its tags wholesale,
// leaving CreatedAt untouched, and the "not found" false.
func TestNote_UpdateNote(t *testing.T) {
	t.Parallel()

	s := newTestStorage(t)
	ctx := context.Background()

	note := createFakeNote(t, s, fakeNoteModel())

	updated := note
	updated.Title = gofakeit.Sentence(3)
	updated.Content = gofakeit.Paragraph(1, 3, 8, " ")
	updated.Tags = fakeTags(1)
	updated.Checksum = gofakeit.LetterN(16)
	updated.UpdatedAt = note.UpdatedAt.Add(time.Hour)

	found, err := s.UpdateNote(ctx, updated)
	require.NoError(t, err)
	require.True(t, found)

	got, err := s.FindNoteByID(ctx, note.ID)
	require.NoError(t, err)
	require.Equal(t, updated, got)
	require.Equal(t, note.CreatedAt, got.CreatedAt)

	// The old tag association is gone, not just superseded — a tag dropped by an update shouldn't
	// linger linked to the note it no longer appears in.
	var oldLinks int
	require.NoError(t, s.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM note_tags WHERE note_id = ? AND tag_name = ?`, note.ID, note.Tags[0],
	).Scan(&oldLinks))
	require.Zero(t, oldLinks)

	missing := fakeNoteModel()
	missing.ID = 999999999
	found, err = s.UpdateNote(ctx, missing)
	require.NoError(t, err)
	require.False(t, found)
}

// TestNote_DeleteNote covers removal (including its note_tags rows, via ON DELETE CASCADE) and the
// "not found" false.
func TestNote_DeleteNote(t *testing.T) {
	t.Parallel()

	s := newTestStorage(t)
	ctx := context.Background()

	note := createFakeNote(t, s, fakeNoteModel())

	found, err := s.DeleteNote(ctx, note.ID)
	require.NoError(t, err)
	require.True(t, found)

	_, err = s.FindNoteByID(ctx, note.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)

	var links int
	require.NoError(t, s.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM note_tags WHERE note_id = ?`, note.ID,
	).Scan(&links))
	require.Zero(t, links)

	found, err = s.DeleteNote(ctx, note.ID)
	require.NoError(t, err)
	require.False(t, found)
}
