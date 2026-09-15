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

func fakeToken() string { return gofakeit.UUID() }

// TestCreateSession covers the insert behind every login. CreateSession returns whatever the driver
// reports untouched — no wrapUnique call here — so a constraint violation (an unknown user, a token
// already in use) surfaces as a plain *mysqldriver.MySQLError, not one of the storage package's
// sentinels.
func TestCreateSession(t *testing.T) {
	t.Parallel()

	query := `INSERT INTO sessions (token, user_id, expires_at, created_at) VALUES (?, ?, ?, ?)`

	token := fakeToken()
	userID := fakeID()
	expiresAt, createdAt := fakeTime(), fakeTime()

	tests := []struct {
		name      string
		mock      func(mock sqlmock.Sqlmock)
		assertErr func(t *testing.T, err error)
	}{
		{
			name: "stores the session",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).WithArgs(token, userID, expiresAt, createdAt).WillReturnResult(sqlmock.NewResult(0, 1))
			},
			assertErr: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name: "a driver error is propagated",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).WithArgs(token, userID, expiresAt, createdAt).WillReturnError(errStub)
			},
			assertErr: func(t *testing.T, err error) { require.ErrorIs(t, err, errStub) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s, mock := newMock(t)
			tt.mock(mock)

			session := model.Session{Token: token, UserID: userID, ExpiresAt: expiresAt, CreatedAt: createdAt}

			err := s.CreateSession(context.Background(), session)
			tt.assertErr(t, err)
		})
	}
}

// TestFindSessionByToken covers the lookup every authenticated request starts with.
func TestFindSessionByToken(t *testing.T) {
	t.Parallel()

	query := `SELECT user_id, expires_at, created_at FROM sessions WHERE token = ?`
	token := fakeToken()
	userID := fakeID()
	expiresAt, createdAt := fakeTime(), fakeTime()

	tests := []struct {
		name         string
		mock         func(mock sqlmock.Sqlmock)
		assertResult func(t *testing.T, got model.Session)
		assertErr    func(t *testing.T, err error)
	}{
		{
			name: "finds the session",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).WithArgs(token).WillReturnRows(
					sqlmock.NewRows([]string{"user_id", "expires_at", "created_at"}).AddRow(userID, expiresAt, createdAt),
				)
			},
			assertResult: func(t *testing.T, got model.Session) {
				require.Equal(t, model.Session{Token: token, UserID: userID, ExpiresAt: expiresAt, CreatedAt: createdAt}, got)
			},
			assertErr: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name:         "an unknown token is sql.ErrNoRows",
			mock:         func(mock sqlmock.Sqlmock) { mock.ExpectQuery(query).WithArgs(token).WillReturnError(sql.ErrNoRows) },
			assertResult: func(t *testing.T, got model.Session) {},
			assertErr:    func(t *testing.T, err error) { require.ErrorIs(t, err, sql.ErrNoRows) },
		},
		{
			name:         "a driver error is propagated",
			mock:         func(mock sqlmock.Sqlmock) { mock.ExpectQuery(query).WithArgs(token).WillReturnError(errStub) },
			assertResult: func(t *testing.T, got model.Session) {},
			assertErr:    func(t *testing.T, err error) { require.ErrorIs(t, err, errStub) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s, mock := newMock(t)
			tt.mock(mock)

			got, err := s.FindSessionByToken(context.Background(), token)
			tt.assertErr(t, err)
			tt.assertResult(t, got)
		})
	}
}

// TestDeleteSession covers logout: removal is best-effort, so an unknown token isn't an error.
func TestDeleteSession(t *testing.T) {
	t.Parallel()

	query := `DELETE FROM sessions WHERE token = ?`
	token := fakeToken()

	tests := []struct {
		name      string
		mock      func(mock sqlmock.Sqlmock)
		assertErr func(t *testing.T, err error)
	}{
		{
			name: "removes the session",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).WithArgs(token).WillReturnResult(sqlmock.NewResult(0, 1))
			},
			assertErr: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name: "an unknown token is not an error",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).WithArgs(token).WillReturnResult(sqlmock.NewResult(0, 0))
			},
			assertErr: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name: "a driver error is propagated",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).WithArgs(token).WillReturnError(errStub)
			},
			assertErr: func(t *testing.T, err error) { require.ErrorIs(t, err, errStub) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s, mock := newMock(t)
			tt.mock(mock)

			err := s.DeleteSession(context.Background(), token)
			tt.assertErr(t, err)
		})
	}
}

// TestDeleteExpiredSessions covers the sweep the hourly cleanup goroutine runs: the row count
// RowsAffected reports is what gets returned to the caller.
func TestDeleteExpiredSessions(t *testing.T) {
	t.Parallel()

	query := `DELETE FROM sessions WHERE expires_at < ?`
	now := fakeTime()
	wantDeleted := gofakeit.Number(0, 100)

	tests := []struct {
		name         string
		mock         func(mock sqlmock.Sqlmock)
		assertResult func(t *testing.T, got int)
		assertErr    func(t *testing.T, err error)
	}{
		{
			name: "reports how many sessions were swept",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).WithArgs(now).WillReturnResult(sqlmock.NewResult(0, int64(wantDeleted)))
			},
			assertResult: func(t *testing.T, got int) { require.Equal(t, wantDeleted, got) },
			assertErr:    func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name:         "a driver error is propagated",
			mock:         func(mock sqlmock.Sqlmock) { mock.ExpectExec(query).WithArgs(now).WillReturnError(errStub) },
			assertResult: func(t *testing.T, got int) {},
			assertErr:    func(t *testing.T, err error) { require.ErrorIs(t, err, errStub) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s, mock := newMock(t)
			tt.mock(mock)

			got, err := s.DeleteExpiredSessions(context.Background(), now)
			tt.assertErr(t, err)
			tt.assertResult(t, got)
		})
	}
}
