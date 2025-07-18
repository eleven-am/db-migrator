package orm

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

// DBExecutor represents an interface that can execute database operations.
// It can be satisfied by both *sqlx.DB and *sqlx.Tx, allowing repositories
// to work with either regular connections or transactions.
type DBExecutor interface {
	// Query execution methods from sqlx.ExtContext
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error

	// Additional sqlx methods
	QueryxContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error)
	QueryRowxContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row

	// Named query support
	NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error)
	BindNamed(query string, arg interface{}) (string, []interface{}, error)

	// Prepared statements
	PreparexContext(ctx context.Context, query string) (*sqlx.Stmt, error)
	PrepareNamedContext(ctx context.Context, query string) (*sqlx.NamedStmt, error)

	// Rebind for driver-specific placeholders
	Rebind(query string) string

	// DriverName returns the driverName passed to the Open function for this DB.
	DriverName() string
}

// Compile-time checks to ensure both sqlx.DB and sqlx.Tx implement DBExecutor
var (
	_ DBExecutor = (*sqlx.DB)(nil)
	_ DBExecutor = (*sqlx.Tx)(nil)
)

// DBWrapper provides additional database-specific operations
// that are only available on *sqlx.DB (not on transactions)
type DBWrapper interface {
	DBExecutor
	BeginTxx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error)
	Close() error
	Ping() error
	PingContext(ctx context.Context) error
	Stats() sql.DBStats
}

// Ensure *sqlx.DB implements DBWrapper
var _ DBWrapper = (*sqlx.DB)(nil)
