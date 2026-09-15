package mysql

import (
	"context"
	"database/sql"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/require"

	"raccodown/internal/storage/model"
)

func fakeNoteModel() model.Note {
	now := fakeTime()

	return model.Note{
		ID:        fakeID(),
		Title:     gofakeit.Sentence(3),
		Content:   gofakeit.Paragraph(1, 3, 8, " "),
		Tags:      []string{gofakeit.LoremIpsumWord()},
		Checksum:  gofakeit.LetterN(16),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

const noteSelect = `SELECT ` + noteColumns + ` FROM notes`

// TestFindNoteByID covers the row scan plus the tags/note_tags follow-up query, and the
// sql.ErrNoRows miss.
func TestFindNoteByID(t *testing.T) {
	t.Parallel()

	note := fakeNoteModel()

	tests := []struct {
		name         string
		mock         func(mock sqlmock.Sqlmock)
		assertResult func(t *testing.T, got model.Note)
		assertErr    func(t *testing.T, err error)
	}{
		{
			name: "finds the row and its tags",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(noteSelect + ` WHERE id = ?`).WithArgs(note.ID).WillReturnRows(
					sqlmock.NewRows([]string{"id", "title", "content", "checksum", "created_at", "updated_at"}).
						AddRow(note.ID, note.Title, note.Content, note.Checksum, note.CreatedAt, note.UpdatedAt),
				)
				mock.ExpectQuery(`SELECT tag_name FROM note_tags WHERE note_id = ? ORDER BY tag_name`).
					WithArgs(note.ID).
					WillReturnRows(sqlmock.NewRows([]string{"tag_name"}).AddRow(note.Tags[0]))
			},
			assertResult: func(t *testing.T, got model.Note) { require.Equal(t, note, got) },
			assertErr:    func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name: "an unknown id is sql.ErrNoRows",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(noteSelect + ` WHERE id = ?`).WithArgs(note.ID).WillReturnError(sql.ErrNoRows)
			},
			assertResult: func(t *testing.T, got model.Note) {},
			assertErr:    func(t *testing.T, err error) { require.ErrorIs(t, err, sql.ErrNoRows) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s, mock := newMock(t)
			tt.mock(mock)

			got, err := s.FindNoteByID(context.Background(), note.ID)
			tt.assertErr(t, err)
			tt.assertResult(t, got)
		})
	}
}

// TestListNotes covers reading back every row, unfiltered, with tags attached in one batched
// follow-up query rather than one per note.
func TestListNotes(t *testing.T) {
	t.Parallel()

	a, b := fakeNoteModel(), fakeNoteModel()

	s, mock := newMock(t)

	mock.ExpectQuery(noteSelect).WillReturnRows(
		sqlmock.NewRows([]string{"id", "title", "content", "checksum", "created_at", "updated_at"}).
			AddRow(a.ID, a.Title, a.Content, a.Checksum, a.CreatedAt, a.UpdatedAt).
			AddRow(b.ID, b.Title, b.Content, b.Checksum, b.CreatedAt, b.UpdatedAt),
	)
	mock.ExpectQuery(`SELECT note_id, tag_name FROM note_tags WHERE note_id IN (?,?) ORDER BY note_id, tag_name`).
		WithArgs(a.ID, b.ID).
		WillReturnRows(sqlmock.NewRows([]string{"note_id", "tag_name"}).
			AddRow(a.ID, a.Tags[0]).
			AddRow(b.ID, b.Tags[0]))

	got, err := s.ListNotes(context.Background())
	require.NoError(t, err)
	require.ElementsMatch(t, []model.Note{a, b}, got)
}

// TestListNotes_Empty covers the case ListNotes skips the tags follow-up query entirely — there are
// no note ids to look up.
func TestListNotes_Empty(t *testing.T) {
	t.Parallel()

	s, mock := newMock(t)

	mock.ExpectQuery(noteSelect).WillReturnRows(
		sqlmock.NewRows([]string{"id", "title", "content", "checksum", "created_at", "updated_at"}),
	)

	got, err := s.ListNotes(context.Background())
	require.NoError(t, err)
	require.Empty(t, got)
}

// TestCreateNote covers the note row insert plus its tag links landing in one transaction, the
// database-assigned id coming back via LastInsertId, and that a failure midway rolls the whole thing
// back.
func TestCreateNote(t *testing.T) {
	t.Parallel()

	insertNote := `INSERT INTO notes (title, content, checksum, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`
	insertTag := `INSERT IGNORE INTO tags (name, created_at) VALUES (?, ?)`
	linkTag := `INSERT INTO note_tags (note_id, tag_name) VALUES (?, ?)`

	t.Run("inserts the note and its tags, returning the new id", func(t *testing.T) {
		t.Parallel()

		note := fakeNoteModel()
		wantID := int64(fakeID())
		s, mock := newMock(t)

		mock.ExpectBegin()
		mock.ExpectExec(insertNote).
			WithArgs(note.Title, note.Content, note.Checksum, note.CreatedAt, note.UpdatedAt).
			WillReturnResult(sqlmock.NewResult(wantID, 1))
		mock.ExpectExec(insertTag).WithArgs(note.Tags[0], sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(linkTag).WithArgs(uint64(wantID), note.Tags[0]).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		got, err := s.CreateNote(context.Background(), note)
		require.NoError(t, err)
		require.Equal(t, uint64(wantID), got)
	})

	t.Run("a note with no tags skips the tag statements", func(t *testing.T) {
		t.Parallel()

		note := fakeNoteModel()
		note.Tags = nil
		wantID := int64(fakeID())
		s, mock := newMock(t)

		mock.ExpectBegin()
		mock.ExpectExec(insertNote).
			WithArgs(note.Title, note.Content, note.Checksum, note.CreatedAt, note.UpdatedAt).
			WillReturnResult(sqlmock.NewResult(wantID, 1))
		mock.ExpectCommit()

		got, err := s.CreateNote(context.Background(), note)
		require.NoError(t, err)
		require.Equal(t, uint64(wantID), got)
	})

	t.Run("a failing insert rolls back and returns the error", func(t *testing.T) {
		t.Parallel()

		note := fakeNoteModel()
		s, mock := newMock(t)

		mock.ExpectBegin()
		mock.ExpectExec(insertNote).
			WithArgs(note.Title, note.Content, note.Checksum, note.CreatedAt, note.UpdatedAt).
			WillReturnError(errStub)
		mock.ExpectRollback()

		got, err := s.CreateNote(context.Background(), note)
		require.ErrorIs(t, err, errStub)
		require.Zero(t, got)
	})
}

// TestUpdateNote covers the row overwrite, replacing tags wholesale, pruning tags left unused, and
// the "not found" false — which skips the tag statements entirely, since there's nothing to update.
func TestUpdateNote(t *testing.T) {
	t.Parallel()

	updateNote := `UPDATE notes SET title = ?, content = ?, checksum = ?, updated_at = ? WHERE id = ?`
	deleteLinks := `DELETE FROM note_tags WHERE note_id = ?`
	insertTag := `INSERT IGNORE INTO tags (name, created_at) VALUES (?, ?)`
	linkTag := `INSERT INTO note_tags (note_id, tag_name) VALUES (?, ?)`
	pruneTags := `DELETE FROM tags WHERE name NOT IN (SELECT DISTINCT tag_name FROM note_tags)`

	t.Run("overwrites the row and replaces its tags", func(t *testing.T) {
		t.Parallel()

		note := fakeNoteModel()
		s, mock := newMock(t)

		mock.ExpectBegin()
		mock.ExpectExec(updateNote).
			WithArgs(note.Title, note.Content, note.Checksum, note.UpdatedAt, note.ID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(deleteLinks).WithArgs(note.ID).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(insertTag).WithArgs(note.Tags[0], sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(linkTag).WithArgs(note.ID, note.Tags[0]).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(pruneTags).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		found, err := s.UpdateNote(context.Background(), note)
		require.NoError(t, err)
		require.True(t, found)
	})

	t.Run("an unknown id is not found and touches nothing else", func(t *testing.T) {
		t.Parallel()

		note := fakeNoteModel()
		s, mock := newMock(t)

		mock.ExpectBegin()
		mock.ExpectExec(updateNote).
			WithArgs(note.Title, note.Content, note.Checksum, note.UpdatedAt, note.ID).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectRollback()

		found, err := s.UpdateNote(context.Background(), note)
		require.NoError(t, err)
		require.False(t, found)
	})
}

// TestDeleteNote covers removal (note_tags rows go with it via ON DELETE CASCADE, so the adapter
// only has to prune tags left unused) and the "not found" false.
func TestDeleteNote(t *testing.T) {
	t.Parallel()

	deleteNote := `DELETE FROM notes WHERE id = ?`
	pruneTags := `DELETE FROM tags WHERE name NOT IN (SELECT DISTINCT tag_name FROM note_tags)`

	t.Run("removes the note and prunes unused tags", func(t *testing.T) {
		t.Parallel()

		id := fakeID()
		s, mock := newMock(t)

		mock.ExpectBegin()
		mock.ExpectExec(deleteNote).WithArgs(id).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(pruneTags).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		found, err := s.DeleteNote(context.Background(), id)
		require.NoError(t, err)
		require.True(t, found)
	})

	t.Run("an unknown id is not found and skips pruning", func(t *testing.T) {
		t.Parallel()

		id := fakeID()
		s, mock := newMock(t)

		mock.ExpectBegin()
		mock.ExpectExec(deleteNote).WithArgs(id).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectRollback()

		found, err := s.DeleteNote(context.Background(), id)
		require.NoError(t, err)
		require.False(t, found)
	})
}
