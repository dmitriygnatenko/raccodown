package note

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	domainerror "raccodown/internal/domain/error"
	"raccodown/internal/port"
	"raccodown/internal/repository/note/mocks"
	"raccodown/internal/storage/model"
)

// newRepo returns a Repository wired to a fresh MockStorage; any call a test doesn't stub via
// EXPECT() fails it.
func newRepo(t *testing.T) (*Repository, *mocks.MockStorage) {
	t.Helper()

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	m := mocks.NewMockStorage(mc)

	return New(m), m
}

func fakeTitle() string   { return gofakeit.Sentence(3) }
func fakeContent() string { return gofakeit.Paragraph(1, 3, 8, " ") }
func fakeID() uint64      { return uint64(gofakeit.Number(1, 1_000_000)) }

func fakeNoteModel() model.Note {
	now := gofakeit.Date().UTC()

	return model.Note{
		ID:        fakeID(),
		Title:     fakeTitle(),
		Content:   fakeContent(),
		Tags:      []string{gofakeit.LoremIpsumWord()},
		Checksum:  fakeChecksum(fakeContent()),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func fakeChecksum(content string) string {
	sum := sha256.Sum256([]byte(content))

	return hex.EncodeToString(sum[:])[:16]
}

var errStub = errors.New(gofakeit.Sentence())

// TestRepository_List covers the in-Go filtering and sorting applied on top of Storage.ListNotes.
func TestRepository_List(t *testing.T) {
	t.Parallel()

	older := fakeNoteModel()
	older.Title = "Groceries"
	older.Content = "milk and bread"
	older.Tags = []string{"home"}
	older.UpdatedAt = time.Now().Add(-time.Hour).UTC()

	newer := fakeNoteModel()
	newer.Title = "Project plan"
	newer.Content = "ship the release"
	newer.Tags = []string{"work"}
	newer.UpdatedAt = time.Now().UTC()

	tests := []struct {
		name      string
		filter    port.NoteListFilter
		wantTitle []string
	}{
		{
			name:      "no filter returns everything, newest-updated first",
			filter:    port.NoteListFilter{},
			wantTitle: []string{"Project plan", "Groceries"},
		},
		{
			name:      "query matches title case-insensitively",
			filter:    port.NoteListFilter{Query: "GROCE"},
			wantTitle: []string{"Groceries"},
		},
		{
			name:      "query matches content case-insensitively",
			filter:    port.NoteListFilter{Query: "RELEASE"},
			wantTitle: []string{"Project plan"},
		},
		{
			name:      "tag matches exactly",
			filter:    port.NoteListFilter{Tag: "work"},
			wantTitle: []string{"Project plan"},
		},
		{
			name:      "an unmatched tag returns nothing",
			filter:    port.NoteListFilter{Tag: "nope"},
			wantTitle: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r, m := newRepo(t)
			m.EXPECT().ListNotes(context.Background()).Return([]model.Note{older, newer}, nil)

			got, err := r.List(context.Background(), tt.filter)
			require.NoError(t, err)

			gotTitles := make([]string, len(got))
			for i, n := range got {
				gotTitles[i] = n.Title
			}

			require.Equal(t, tt.wantTitle, gotTitles)
		})
	}

	t.Run("a storage error is propagated", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		m.EXPECT().ListNotes(context.Background()).Return(nil, errStub)

		_, err := r.List(context.Background(), port.NoteListFilter{})
		require.ErrorIs(t, err, errStub)
	})
}

// TestRepository_FindByID covers the lookup, including the id -> NotFoundError translation.
func TestRepository_FindByID(t *testing.T) {
	t.Parallel()

	id := fakeID()

	t.Run("finds the note", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		row := fakeNoteModel()
		row.ID = id
		m.EXPECT().FindNoteByID(context.Background(), id).Return(row, nil)

		got, err := r.FindByID(context.Background(), id)
		require.NoError(t, err)
		require.Equal(t, row.ToEntity(), got)
	})

	t.Run("an unknown id becomes a message-less NotFoundError", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		m.EXPECT().FindNoteByID(context.Background(), id).Return(model.Note{}, sql.ErrNoRows)

		_, err := r.FindByID(context.Background(), id)

		var notFound *domainerror.NotFoundError
		require.ErrorAs(t, err, &notFound)
		require.Empty(t, notFound.Message)
	})

	t.Run("a storage error is propagated", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		m.EXPECT().FindNoteByID(context.Background(), id).Return(model.Note{}, errStub)

		_, err := r.FindByID(context.Background(), id)
		require.ErrorIs(t, err, errStub)
	})
}

// TestRepository_Create covers the insert, including checksum computation and id generation.
func TestRepository_Create(t *testing.T) {
	t.Parallel()

	req := port.NoteCreateRequest{
		Title:   fakeTitle(),
		Content: fakeContent(),
		Tags:    []string{gofakeit.LoremIpsumWord()},
	}

	t.Run("stores the note and returns it with the database-assigned id", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		wantID := fakeID()

		var stored model.Note
		m.EXPECT().CreateNote(context.Background(), gomock.Any()).
			DoAndReturn(func(_ context.Context, note model.Note) (uint64, error) {
				stored = note
				return wantID, nil
			})

		got, err := r.Create(context.Background(), req)
		require.NoError(t, err)

		require.Equal(t, wantID, got.ID)
		require.Equal(t, req.Title, got.Title)
		require.Equal(t, req.Content, got.Content)
		require.Equal(t, req.Tags, got.Tags)
		require.Equal(t, fakeChecksum(req.Content), got.Checksum)
		require.WithinDuration(t, time.Now().UTC(), got.CreatedAt, time.Minute)
		require.Equal(t, got.CreatedAt, got.UpdatedAt)

		require.Zero(t, stored.ID) // not yet assigned when it reaches storage
		require.Equal(t, got.Checksum, stored.Checksum)
	})

	t.Run("a storage error is propagated", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		m.EXPECT().CreateNote(context.Background(), gomock.Any()).Return(uint64(0), errStub)

		_, err := r.Create(context.Background(), req)
		require.ErrorIs(t, err, errStub)
	})
}

// TestRepository_Update covers the read-check-write optimistic-concurrency flow: an unknown id, a
// stale checksum, and a successful rewrite.
func TestRepository_Update(t *testing.T) {
	t.Parallel()

	existing := fakeNoteModel()
	req := port.NoteUpdateRequest{
		ID:      existing.ID,
		Title:   fakeTitle(),
		Content: fakeContent(),
		Tags:    []string{gofakeit.LoremIpsumWord()},
	}

	t.Run("an unknown id becomes a message-less NotFoundError", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		m.EXPECT().FindNoteByID(context.Background(), req.ID).Return(model.Note{}, sql.ErrNoRows)

		_, err := r.Update(context.Background(), req)

		var notFound *domainerror.NotFoundError
		require.ErrorAs(t, err, &notFound)
	})

	t.Run("a lookup storage error is propagated", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		m.EXPECT().FindNoteByID(context.Background(), req.ID).Return(model.Note{}, errStub)

		_, err := r.Update(context.Background(), req)
		require.ErrorIs(t, err, errStub)
	})

	t.Run("a stale checksum becomes a message-less ConflictError", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		m.EXPECT().FindNoteByID(context.Background(), req.ID).Return(existing, nil)

		staleReq := req
		staleReq.KnownChecksum = existing.Checksum + "stale"

		_, err := r.Update(context.Background(), staleReq)

		var conflict *domainerror.ConflictError
		require.ErrorAs(t, err, &conflict)
	})

	t.Run("a matching checksum is accepted", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		m.EXPECT().FindNoteByID(context.Background(), req.ID).Return(existing, nil)
		m.EXPECT().UpdateNote(context.Background(), gomock.Any()).Return(true, nil)

		matchingReq := req
		matchingReq.KnownChecksum = existing.Checksum

		got, err := r.Update(context.Background(), matchingReq)
		require.NoError(t, err)
		require.Equal(t, req.Title, got.Title)
		require.Equal(t, existing.CreatedAt, got.CreatedAt)
		require.Equal(t, fakeChecksum(req.Content), got.Checksum)
	})

	t.Run("an unset KnownChecksum skips the concurrency check", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		m.EXPECT().FindNoteByID(context.Background(), req.ID).Return(existing, nil)
		m.EXPECT().UpdateNote(context.Background(), gomock.Any()).Return(true, nil)

		got, err := r.Update(context.Background(), req)
		require.NoError(t, err)
		require.Equal(t, req.ID, got.ID)
	})

	t.Run("a concurrent delete during update becomes a message-less NotFoundError", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		m.EXPECT().FindNoteByID(context.Background(), req.ID).Return(existing, nil)
		m.EXPECT().UpdateNote(context.Background(), gomock.Any()).Return(false, nil)

		_, err := r.Update(context.Background(), req)

		var notFound *domainerror.NotFoundError
		require.ErrorAs(t, err, &notFound)
	})

	t.Run("an update storage error is propagated", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		m.EXPECT().FindNoteByID(context.Background(), req.ID).Return(existing, nil)
		m.EXPECT().UpdateNote(context.Background(), gomock.Any()).Return(false, errStub)

		_, err := r.Update(context.Background(), req)
		require.ErrorIs(t, err, errStub)
	})
}

// TestRepository_Delete covers removal, including the id -> NotFoundError translation.
func TestRepository_Delete(t *testing.T) {
	t.Parallel()

	id := fakeID()

	t.Run("deletes the note", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		m.EXPECT().DeleteNote(context.Background(), id).Return(true, nil)

		require.NoError(t, r.Delete(context.Background(), id))
	})

	t.Run("an unknown id becomes a message-less NotFoundError", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		m.EXPECT().DeleteNote(context.Background(), id).Return(false, nil)

		err := r.Delete(context.Background(), id)

		var notFound *domainerror.NotFoundError
		require.ErrorAs(t, err, &notFound)
	})

	t.Run("a storage error is propagated", func(t *testing.T) {
		t.Parallel()

		r, m := newRepo(t)
		m.EXPECT().DeleteNote(context.Background(), id).Return(false, errStub)

		err := r.Delete(context.Background(), id)
		require.ErrorIs(t, err, errStub)
	})
}
