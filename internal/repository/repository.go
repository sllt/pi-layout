package repository

import (
	"context"
	"errors"

	"github.com/sllt/pi-layout/pkg/log"
	piSQL "github.com/sllt/pi/pkg/pi/datasource/sql"
	"github.com/sllt/pi/pkg/pi/infra"
)

// txKey is a typed context key to avoid collisions with other packages.
type txKey struct{}

type Repository struct {
	db     infra.DB
	logger *log.Logger
}

func NewRepository(logger *log.Logger, db infra.DB) *Repository {
	return &Repository{
		db:     db,
		logger: logger,
	}
}

type Transaction interface {
	Transaction(ctx context.Context, fn func(ctx context.Context) error) error
}

func NewTransaction(r *Repository) Transaction {
	return r
}

// GetQuerier returns the transaction from context if available, otherwise the DB.
func (r *Repository) GetQuerier(ctx context.Context) piSQL.Executor {
	if tx, ok := ctx.Value(txKey{}).(*piSQL.Tx); ok {
		return tx
	}
	return r.db
}

// Transaction executes fn within a database transaction.
// If ctx already carries a transaction, fn runs in that existing transaction (no nesting).
func (r *Repository) Transaction(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	// Reuse existing transaction — avoids partial-commit on nested calls.
	if _, ok := ctx.Value(txKey{}).(*piSQL.Tx); ok {
		return fn(ctx)
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	txCtx := context.WithValue(ctx, txKey{}, tx)

	defer func() {
		if p := recover(); p != nil {
			if rbErr := tx.Rollback(); rbErr != nil && r.logger != nil {
				r.logger.Errorf("rollback after panic failed: %v", rbErr)
			}
			panic(p) // re-panic to preserve stack trace
		}
	}()

	if err := fn(txCtx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return errors.Join(err, rbErr)
		}
		return err
	}

	return tx.Commit()
}
