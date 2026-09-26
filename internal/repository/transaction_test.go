package repository

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/sllt/pi/pkg/pi/config"
	piSQL "github.com/sllt/pi/pkg/pi/datasource/sql"
	"github.com/sllt/pi/pkg/pi/logging"
	"github.com/stretchr/testify/require"
)

func transactionDB(t *testing.T) (*Repository, *piSQL.DB) {
	t.Helper()
	cfg := config.NewSnapshot(map[string]string{"DB_DIALECT": "sqlite", "DB_NAME": filepath.Join(t.TempDir(), "tx.db"), "DB_MAX_OPEN_CONNECTION": "1"})
	db, err := piSQL.OpenContext(t.Context(), cfg, logging.NewWriterLogger(logging.ERROR, io.Discard, io.Discard), nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	_, err = db.ExecContext(t.Context(), "CREATE TABLE items (id INTEGER PRIMARY KEY, value TEXT UNIQUE)")
	require.NoError(t, err)
	return NewRepository(nil, db), db
}
func countItems(t *testing.T, db *piSQL.DB) int {
	t.Helper()
	var count int
	require.NoError(t, db.QueryRowContext(t.Context(), "SELECT count(*) FROM items").Scan(&count))
	return count
}

func TestTransactionSQLiteRollbackAndNestedScope(t *testing.T) {
	r, db := transactionDB(t)
	boom := errors.New("nested operation failed")
	err := r.Transaction(t.Context(), func(ctx context.Context) error {
		_, err := r.GetQuerier(ctx).ExecContext(ctx, "INSERT INTO items VALUES (1, 'first')")
		require.NoError(t, err)
		err = r.Transaction(ctx, func(nested context.Context) error {
			_, err := r.GetQuerier(nested).ExecContext(nested, "INSERT INTO items VALUES (2, 'second')")
			require.NoError(t, err)
			return boom
		})
		require.ErrorIs(t, err, boom)
		return nil
	})
	require.ErrorIs(t, err, ErrRollbackOnly)
	require.ErrorIs(t, err, boom)
	require.Zero(t, countItems(t, db))
	require.Panics(t, func() {
		_ = r.Transaction(t.Context(), func(ctx context.Context) error {
			_, err := r.GetQuerier(ctx).ExecContext(ctx, "INSERT INTO items VALUES (3, 'panic')")
			require.NoError(t, err)
			panic("abort")
		})
	})
	require.Zero(t, countItems(t, db))
	require.NoError(t, r.Transaction(t.Context(), func(ctx context.Context) error {
		return r.Transaction(ctx, func(nested context.Context) error {
			_, err := r.GetQuerier(nested).ExecContext(nested, "INSERT INTO items VALUES (4, 'committed')")
			return err
		})
	}))
	require.Equal(t, 1, countItems(t, db))
}

func TestTransactionRejectsCrossDatabaseAndChangedOptions(t *testing.T) {
	a, dbA := transactionDB(t)
	b, dbB := transactionDB(t)
	err := a.Transaction(t.Context(), func(ctx context.Context) error {
		require.ErrorIs(t, b.Transaction(ctx, func(context.Context) error { t.Fatal("cross DB callback executed"); return nil }), ErrCrossDatabaseTransaction)
		_, err := b.GetQuerier(ctx).ExecContext(ctx, "INSERT INTO items VALUES (1,'wrong db')")
		require.ErrorIs(t, err, ErrCrossDatabaseTransaction)
		var id int
		require.ErrorIs(t, b.GetQuerier(ctx).QueryRowContext(ctx, "SELECT 1").Scan(&id), ErrCrossDatabaseTransaction)
		return nil
	})
	require.ErrorIs(t, err, ErrCrossDatabaseTransaction)
	require.Zero(t, countItems(t, dbA))
	require.Zero(t, countItems(t, dbB))
	err = a.Transaction(t.Context(), func(ctx context.Context) error {
		_ = a.TransactionWithOptions(ctx, &sql.TxOptions{ReadOnly: true}, func(context.Context) error { t.Fatal("changed nested options accepted"); return nil })
		return nil
	})
	require.ErrorIs(t, err, ErrTransactionOptions)
	owned, err := dbA.BeginTxContext(t.Context(), nil)
	require.NoError(t, err)
	require.True(t, dbA.Owns(owned))
	require.False(t, dbB.Owns(owned))
	require.NoError(t, owned.Rollback())
}

func TestTransactionCallerCancellationAndPoolWait(t *testing.T) {
	r, db := transactionDB(t)
	held, err := db.BeginTxContext(t.Context(), nil)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	err = r.Transaction(ctx, func(context.Context) error { t.Fatal("callback ran without a connection"); return nil })
	cancel()
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.NoError(t, held.Rollback())
	ctx, cancel = context.WithCancel(t.Context())
	defer cancel()
	err = r.Transaction(ctx, func(ctx context.Context) error {
		_, err := r.GetQuerier(ctx).ExecContext(ctx, "INSERT INTO items VALUES(1,'cancel')")
		require.NoError(t, err)
		cancel()
		return nil
	})
	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, countItems(t, db))
}

func TestTransactionCommitFailureIsNotRetried(t *testing.T) {
	raw, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer raw.Close()
	r := NewRepository(nil, &piSQL.DB{DB: raw})
	boom := errors.New("commit result unknown")
	mock.ExpectBegin()
	mock.ExpectExec("INSERT").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(boom)
	calls := 0
	err = r.TransactionWithOptions(t.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable}, func(ctx context.Context) error {
		calls++
		_, err := r.GetQuerier(ctx).ExecContext(ctx, "INSERT INTO items VALUES (1,'commit')")
		return err
	})
	require.ErrorIs(t, err, boom)
	require.Equal(t, 1, calls)
	require.NoError(t, mock.ExpectationsWereMet())
}
