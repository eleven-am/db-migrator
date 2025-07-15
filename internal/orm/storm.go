package orm

import (
	"context"
	"fmt"
	"github.com/Masterminds/squirrel"

	"github.com/jmoiron/sqlx"
)

// Storm is the main entry point for all ORM operations
// It holds all repositories and manages database connections
type Storm struct {
	db       DBExecutor
	executor DBExecutor // Current executor (DB or TX)

	// Repository registry - will be populated by code generation
	repositories map[string]interface{}
}

// NewStorm creates a new Storm instance with the given database connection
func NewStorm(db *sqlx.DB) *Storm {
	storm := &Storm{
		db:           db,
		executor:     db,
		repositories: make(map[string]interface{}),
	}

	// Repository initialization will be handled by generated code
	storm.initializeRepositories()

	return storm
}

// newStormWithExecutor creates a Storm instance with a specific executor (for transactions)
func newStormWithExecutor(db *sqlx.DB, executor DBExecutor) *Storm {
	storm := &Storm{
		db:           db,
		executor:     executor,
		repositories: make(map[string]interface{}),
	}

	// Repository initialization will be handled by generated code
	storm.initializeRepositories()

	return storm
}

// WithTransaction executes a function within a database transaction
// It returns a transaction-aware Storm instance to the callback
func (s *Storm) WithTransaction(ctx context.Context, fn func(*Storm) error) error {
	// Only start a new transaction if we're not already in one
	if _, isTransaction := s.executor.(*sqlx.Tx); isTransaction {
		// Already in a transaction, just use the current Storm
		return fn(s)
	}

	// Get the actual DB connection
	db, ok := s.db.(*sqlx.DB)
	if !ok {
		return fmt.Errorf("cannot start transaction: executor is not a database connection")
	}

	// Start transaction
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Create transaction-aware Storm
	txStorm := newStormWithExecutor(db, tx)

	// Execute the function with the transaction Storm
	if err := fn(txStorm); err != nil {
		// Rollback on error
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("failed to rollback transaction: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	// Commit on success
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// WithTransactionOptions executes a function within a database transaction with specific options
func (s *Storm) WithTransactionOptions(ctx context.Context, opts *TransactionOptions, fn func(*Storm) error) error {
	// Only start a new transaction if we're not already in one
	if _, isTransaction := s.executor.(*sqlx.Tx); isTransaction {
		// Already in a transaction, just use the current Storm
		return fn(s)
	}

	// Get the actual DB connection
	db, ok := s.db.(*sqlx.DB)
	if !ok {
		return fmt.Errorf("cannot start transaction: executor is not a database connection")
	}

	// Convert options to TxOptions
	txOpts := opts.ToTxOptions()

	// Start transaction with options
	tx, err := db.BeginTxx(ctx, txOpts)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Create transaction-aware Storm
	txStorm := newStormWithExecutor(db, tx)

	// Execute the function with the transaction Storm
	if err := fn(txStorm); err != nil {
		// Rollback on error
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("failed to rollback transaction: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	// Commit on success
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetExecutor returns the current database executor
// This is useful for raw queries or custom operations
func (s *Storm) GetExecutor() DBExecutor {
	return s.executor
}

// And combines multiple conditions with AND
func (s *Storm) And(conditions ...Condition) Condition {
	sqlizers := make([]squirrel.Sqlizer, len(conditions))
	for i, c := range conditions {
		sqlizers[i] = c.condition
	}
	return Condition{squirrel.And(sqlizers)}
}

// Or combines multiple conditions with OR
func (s *Storm) Or(conditions ...Condition) Condition {
	sqlizers := make([]squirrel.Sqlizer, len(conditions))
	for i, c := range conditions {
		sqlizers[i] = c.condition
	}
	return Condition{squirrel.Or(sqlizers)}
}

// Not negates a condition(s *Storm)
func (s *Storm) Not(condition Condition) Condition {
	return Condition{squirrel.Expr("NOT (?)", condition.ToSqlizer())}
}

// GetDB returns the underlying database connection
// This is useful when you need the actual *sqlx.DB
func (s *Storm) GetDB() *sqlx.DB {
	if db, ok := s.db.(*sqlx.DB); ok {
		return db
	}
	return nil
}

// initializeRepositories is a placeholder that will be replaced by generated code
// The generated code will initialize all repository fields
func (s *Storm) initializeRepositories() {
	// This method will be overridden by the generated code
}
