package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"raccodown/internal/storage/model"
)

// noteScanner is satisfied by both *sql.Row and *sql.Rows.
type noteScanner interface {
	Scan(dest ...any) error
}

const noteColumns = `id, title, content, checksum, created_at, updated_at`

// scanNote reads the note column list, in the order every note query below selects it. Tags aren't
// among them — they live in the tags/note_tags tables and are filled in separately by the caller.
func scanNote(row noteScanner) (model.Note, error) {
	var m model.Note

	if err := row.Scan(&m.ID, &m.Title, &m.Content, &m.Checksum, &m.CreatedAt, &m.UpdatedAt); err != nil {
		return model.Note{}, err
	}

	return m, nil
}

// ListNotes returns every note, tags included. Filtering and sorting happen in the repository
// layer, not here — raccodown is single-user with a note count small enough that reading the whole
// table and filtering in Go is simpler than building the equivalent SQL.
func (s *Storage) ListNotes(ctx context.Context) ([]model.Note, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT `+noteColumns+` FROM notes`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []model.Note

	for rows.Next() {
		m, err := scanNote(rows)
		if err != nil {
			return nil, err
		}

		notes = append(notes, m)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	ids := make([]uint64, len(notes))
	for i, note := range notes {
		ids[i] = note.ID
	}

	tagsByNote, err := s.tagsForNotes(ctx, ids)
	if err != nil {
		return nil, err
	}

	for i := range notes {
		notes[i].Tags = tagsByNote[notes[i].ID]
	}

	return notes, nil
}

// FindNoteByID looks up a note by id, returning sql.ErrNoRows when there's no match.
func (s *Storage) FindNoteByID(ctx context.Context, id uint64) (model.Note, error) {
	note, err := scanNote(s.DB.QueryRowContext(ctx, `SELECT `+noteColumns+` FROM notes WHERE id = $1`, id))
	if err != nil {
		return model.Note{}, err
	}

	tags, err := s.tagsForNote(ctx, id)
	if err != nil {
		return model.Note{}, err
	}

	note.Tags = tags

	return note, nil
}

// CreateNote inserts a note row and its tag associations in one transaction, returning the row's
// new, database-assigned id.
func (s *Storage) CreateNote(ctx context.Context, note model.Note) (uint64, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op once Commit succeeds; only matters on the error paths above it

	var id uint64
	err = tx.QueryRowContext(ctx,
		`INSERT INTO notes (title, content, checksum, created_at, updated_at) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		note.Title, note.Content, note.Checksum, note.CreatedAt, note.UpdatedAt,
	).Scan(&id)
	if err != nil {
		return 0, err
	}

	if err := setNoteTags(ctx, tx, id, note.Tags); err != nil {
		return 0, err
	}

	return id, tx.Commit()
}

// UpdateNote overwrites a note's editable fields and replaces its tag associations wholesale, in one
// transaction. found is false if no note with this id existed.
func (s *Storage) UpdateNote(ctx context.Context, note model.Note) (bool, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op once Commit succeeds; only matters on the error paths above it

	res, err := tx.ExecContext(ctx,
		`UPDATE notes SET title = $1, content = $2, checksum = $3, updated_at = $4 WHERE id = $5`,
		note.Title, note.Content, note.Checksum, note.UpdatedAt, note.ID,
	)
	if err != nil {
		return false, err
	}

	found, err := affected(res)
	if err != nil || !found {
		return found, err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM note_tags WHERE note_id = $1`, note.ID); err != nil {
		return false, err
	}

	if err := setNoteTags(ctx, tx, note.ID, note.Tags); err != nil {
		return false, err
	}

	if err := pruneUnusedTags(ctx, tx); err != nil {
		return false, err
	}

	return true, tx.Commit()
}

// DeleteNote removes a note row; its note_tags rows go with it via ON DELETE CASCADE. Any tag left
// with no note_tags rows referencing it afterward is pruned too, so the tags table never accumulates
// dead rows for tags no note carries anymore. found is false if no note with this id existed.
func (s *Storage) DeleteNote(ctx context.Context, id uint64) (bool, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op once Commit succeeds; only matters on the error paths above it

	res, err := tx.ExecContext(ctx, `DELETE FROM notes WHERE id = $1`, id)
	if err != nil {
		return false, err
	}

	found, err := affected(res)
	if err != nil || !found {
		return found, err
	}

	if err := pruneUnusedTags(ctx, tx); err != nil {
		return false, err
	}

	return true, tx.Commit()
}

// setNoteTags upserts each of tags into the tags table (a tag already in use by another note is
// left alone) and links noteID to all of them. Runs against a transaction the caller controls
// (CreateNote/UpdateNote), so the note row and its tags land atomically.
func setNoteTags(ctx context.Context, tx *sql.Tx, noteID uint64, tags []string) error {
	now := time.Now().UTC()

	for _, tag := range tags {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO tags (name, created_at) VALUES ($1, $2) ON CONFLICT (name) DO NOTHING`, tag, now,
		); err != nil {
			return fmt.Errorf("upserting tag %q: %w", tag, err)
		}

		if _, err := tx.ExecContext(ctx,
			`INSERT INTO note_tags (note_id, tag_name) VALUES ($1, $2)`, noteID, tag,
		); err != nil {
			return fmt.Errorf("linking tag %q: %w", tag, err)
		}
	}

	return nil
}

// pruneUnusedTags deletes any tags row no note_tags entry references anymore — called after
// UpdateNote replaces a note's tags and after DeleteNote removes a note, either of which can leave a
// tag with no notes carrying it. Runs against the caller's transaction.
func pruneUnusedTags(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx,
		`DELETE FROM tags WHERE name NOT IN (SELECT DISTINCT tag_name FROM note_tags)`,
	)

	return err
}

// tagsForNote returns noteID's tags, alphabetically.
func (s *Storage) tagsForNote(ctx context.Context, noteID uint64) ([]string, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT tag_name FROM note_tags WHERE note_id = $1 ORDER BY tag_name`, noteID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string

	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}

		tags = append(tags, tag)
	}

	return tags, rows.Err()
}

// tagsForNotes returns a note id -> tags map covering every id in noteIDs, each alphabetically.
// Fetching every note_tags row for the batch in one query and grouping in Go, rather than one query
// per note, avoids an N+1 query for ListNotes.
func (s *Storage) tagsForNotes(ctx context.Context, noteIDs []uint64) (map[uint64][]string, error) {
	tagsByNote := make(map[uint64][]string, len(noteIDs))
	if len(noteIDs) == 0 {
		return tagsByNote, nil
	}

	placeholders := make([]string, len(noteIDs))
	args := make([]any, len(noteIDs))

	for i, id := range noteIDs {
		placeholders[i] = "$" + strconv.Itoa(i+1)
		args[i] = id
	}

	query := `SELECT note_id, tag_name FROM note_tags WHERE note_id IN (` +
		strings.Join(placeholders, ",") + `) ORDER BY note_id, tag_name`

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var noteID uint64
		var tag string
		if err := rows.Scan(&noteID, &tag); err != nil {
			return nil, err
		}

		tagsByNote[noteID] = append(tagsByNote[noteID], tag)
	}

	return tagsByNote, rows.Err()
}
