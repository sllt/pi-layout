package repository

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"sync"

	"github.com/sllt/pi-layout/pkg/log"
	piSQL "github.com/sllt/pi/pkg/pi/datasource/sql"
	"github.com/sllt/pi/pkg/pi/infra"
)

var ErrCrossDatabaseTransaction = errors.New("transaction belongs to a different database")
var ErrRollbackOnly = errors.New("transaction is rollback-only")
var ErrTransactionOptions = errors.New("nested transaction cannot change options")

type txKey struct{}
type transactionScope struct {
	tx      *piSQL.Tx
	owner   infra.DB
	options sql.TxOptions
	mu      sync.Mutex
	failure error
}

func (s *transactionScope) fail(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failure = errors.Join(s.failure, err)
}
func (s *transactionScope) failureError() error { s.mu.Lock(); defer s.mu.Unlock(); return s.failure }

type Repository struct {
	db     infra.DB
	logger *log.Logger
}

func NewRepository(logger *log.Logger, db infra.DB) *Repository {
	return &Repository{db: db, logger: logger}
}

type Transaction interface {
	Transaction(context.Context, func(context.Context) error) error
}

// TransactionWithOptions is an optional extension; existing service mocks remain compatible.
type TransactionWithOptions interface {
	TransactionWithOptions(context.Context, *sql.TxOptions, func(context.Context) error) error
}

func NewTransaction(r *Repository) Transaction { return r }

func sameDB(a, b infra.DB) bool {
	return a != nil && b != nil && reflect.TypeOf(a) == reflect.TypeOf(b) && reflect.TypeOf(a).Comparable() && a == b
}
func (r *Repository) GetQuerier(ctx context.Context) piSQL.Executor {
	if scope, ok := ctx.Value(txKey{}).(*transactionScope); ok {
		if !sameDB(scope.owner, r.db) {
			scope.fail(ErrCrossDatabaseTransaction)
			return piSQL.ErrorExecutor(ErrCrossDatabaseTransaction)
		}
		return scope.tx
	}
	return r.db
}
func (r *Repository) Transaction(ctx context.Context, fn func(context.Context) error) error {
	return r.TransactionWithOptions(ctx, nil, fn)
}
func (r *Repository) TransactionWithOptions(ctx context.Context, opts *sql.TxOptions, fn func(context.Context) error) (err error) {
	if err = ctx.Err(); err != nil {
		return err
	}
	if fn == nil {
		return errors.New("transaction callback is nil")
	}
	var normalized sql.TxOptions
	if opts != nil {
		normalized = *opts
	}
	if scope, ok := ctx.Value(txKey{}).(*transactionScope); ok {
		if !sameDB(scope.owner, r.db) {
			scope.fail(ErrCrossDatabaseTransaction)
			return ErrCrossDatabaseTransaction
		}
		if opts != nil && normalized != scope.options {
			scope.fail(ErrTransactionOptions)
			return ErrTransactionOptions
		}
		defer func() {
			if p := recover(); p != nil {
				scope.fail(ErrRollbackOnly)
				panic(p)
			}
			if err != nil {
				scope.fail(err)
			}
		}()
		return fn(ctx)
	}
	starter, ok := r.db.(interface {
		BeginTxContext(context.Context, *sql.TxOptions) (*piSQL.Tx, error)
	})
	if !ok {
		return errors.New("database adapter must implement BeginTxContext for context-aware transactions")
	}
	tx, err := starter.BeginTxContext(ctx, opts)
	if err != nil {
		return err
	}
	// Also covers runtime.Goexit inside a callback. Normal paths below retain
	// meaningful rollback errors; a finished transaction returns ErrTxDone here.
	defer tx.Rollback()
	scope := &transactionScope{tx: tx, owner: r.db, options: normalized}
	rollback := func() error {
		e := tx.Rollback()
		if errors.Is(e, sql.ErrTxDone) {
			return nil
		}
		return e
	}
	defer func() {
		if p := recover(); p != nil {
			if rbErr := rollback(); rbErr != nil && r.logger != nil {
				r.logger.Errorf("rollback after panic: %v", rbErr)
			}
			panic(p)
		}
	}()
	if err = fn(context.WithValue(ctx, txKey{}, scope)); err != nil {
		return errors.Join(err, rollback())
	}
	if cause := scope.failureError(); cause != nil {
		return errors.Join(ErrRollbackOnly, cause, rollback())
	}
	if err = ctx.Err(); err != nil {
		return errors.Join(err, rollback())
	}
	if err = tx.Commit(); err != nil {
		return errors.Join(err, rollback())
	}
	return nil
}
