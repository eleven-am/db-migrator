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

func NewStorm(db *sqlx.DB) *Storm {
	storm := &Storm{
		db:           db,
		executor:     db,
		repositories: make(map[string]interface{}),
	}

	storm.initializeRepositories()

	return storm
}

func newStormWithExecutor(db *sqlx.DB, executor DBExecutor) *Storm {
	storm := &Storm{
		db:           db,
		executor:     executor,
		repositories: make(map[string]interface{}),
	}

	storm.initializeRepositories()
	return storm
}

func (s *Storm) WithTransaction(ctx context.Context, fn func(*Storm) error) error {
	if _, isTransaction := s.executor.(*sqlx.Tx); isTransaction {

		return fn(s)
	}

	db, ok := s.db.(*sqlx.DB)
	if !ok {
		return fmt.Errorf("cannot start transaction: executor is not a database connection")
	}

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			if rbErr := tx.Rollback(); rbErr != nil && rbErr.Error() != "sql: transaction has already been committed or rolled back" {
				// Only log non-"tx closed" errors
			}
		}
	}()

	txStorm := newStormWithExecutor(db, tx)
	if err := fn(txStorm); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	committed = true

	return nil
}

func (s *Storm) WithTransactionOptions(ctx context.Context, opts *TransactionOptions, fn func(*Storm) error) error {
	if _, isTransaction := s.executor.(*sqlx.Tx); isTransaction {

		return fn(s)
	}

	db, ok := s.db.(*sqlx.DB)
	if !ok {
		return fmt.Errorf("cannot start transaction: executor is not a database connection")
	}

	txOpts := opts.ToTxOptions()
	tx, err := db.BeginTxx(ctx, txOpts)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			if rbErr := tx.Rollback(); rbErr != nil && rbErr.Error() != "sql: transaction has already been committed or rolled back" {
				// Only log non-"tx closed" errors
			}
		}
	}()

	txStorm := newStormWithExecutor(db, tx)
	if err := fn(txStorm); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	committed = true

	return nil
}

func (s *Storm) GetExecutor() DBExecutor {
	return s.executor
}

func And(conditions ...Condition) Condition {
	sqlizers := make([]squirrel.Sqlizer, len(conditions))
	for i, c := range conditions {
		sqlizers[i] = c.condition
	}
	return Condition{squirrel.And(sqlizers)}
}

func Or(conditions ...Condition) Condition {
	sqlizers := make([]squirrel.Sqlizer, len(conditions))
	for i, c := range conditions {
		sqlizers[i] = c.condition
	}
	return Condition{squirrel.Or(sqlizers)}
}

func Not(condition Condition) Condition {
	return Condition{squirrel.Expr("NOT (?)", condition.ToSqlizer())}
}

func (s *Storm) GetDB() *sqlx.DB {
	if db, ok := s.db.(*sqlx.DB); ok {
		return db
	}
	return nil
}

func (s *Storm) initializeRepositories() {

}
