package orm

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// TransactionOptions configures transaction behavior
type TransactionOptions struct {
	Isolation sql.IsolationLevel
	ReadOnly  bool
}

// DefaultTransactionOptions returns sensible defaults
func DefaultTransactionOptions() *TransactionOptions {
	return &TransactionOptions{
		Isolation: sql.LevelDefault,
		ReadOnly:  false,
	}
}

// ToTxOptions converts TransactionOptions to sql.TxOptions
func (o *TransactionOptions) ToTxOptions() *sql.TxOptions {
	if o == nil {
		return nil
	}
	return &sql.TxOptions{
		Isolation: o.Isolation,
		ReadOnly:  o.ReadOnly,
	}
}

// TransactionManager provides utilities for managing transactions across repositories
type TransactionManager struct {
	db *sqlx.DB
}

// NewTransactionManager creates a new transaction manager
func NewTransactionManager(db *sqlx.DB) *TransactionManager {
	return &TransactionManager{db: db}
}

// WithTransaction executes a function within a transaction
func (tm *TransactionManager) WithTransaction(ctx context.Context, fn func(*sqlx.Tx) error) error {
	return tm.WithTransactionOptions(ctx, nil, fn)
}

// WithTransactionOptions executes a function within a transaction with options
func (tm *TransactionManager) WithTransactionOptions(ctx context.Context, opts *TransactionOptions, fn func(*sqlx.Tx) error) error {
	if opts == nil {
		opts = DefaultTransactionOptions()
	}

	txOpts := &sql.TxOptions{
		Isolation: opts.Isolation,
		ReadOnly:  opts.ReadOnly,
	}

	tx, err := tm.db.BeginTxx(ctx, txOpts)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	err = fn(tx)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}

// GetTransactionManager returns a transaction manager for a repository
// This is a convenience method to create a TransactionManager from a Repository
func (r *Repository[T]) GetTransactionManager() (*TransactionManager, error) {
	db, ok := r.db.(*sqlx.DB)
	if !ok {
		return nil, fmt.Errorf("cannot create transaction manager: repository is already using a transaction")
	}
	return NewTransactionManager(db), nil
}

// WithinTransaction is a helper method that executes a function within a transaction
// using the repository's database connection
func (r *Repository[T]) WithinTransaction(ctx context.Context, fn func(*sqlx.Tx) error) error {
	tm, err := r.GetTransactionManager()
	if err != nil {
		return err
	}
	return tm.WithTransaction(ctx, fn)
}

// IsTransaction returns true if the repository is using a transaction
func (r *Repository[T]) IsTransaction() bool {
	_, ok := r.db.(*sqlx.Tx)
	return ok
}
