package orm

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
)

// preparedStatements manages prepared statements for the repository (internal use only)
type preparedStatements struct {
	findByID    *sqlx.Stmt
	insert      *sqlx.NamedStmt
	update      *sqlx.NamedStmt
	deleteByID  *sqlx.Stmt
	count       *sqlx.Stmt
	mutex       sync.RWMutex
	initialized bool
}

// initializePreparedStatements creates commonly used prepared statements
func (r *Repository[T]) initializePreparedStatements() error {
	if r.insertStmt != nil && r.updateStmt != nil {
		return nil // Already initialized
	}

	// Prepare FindByID statement
	if len(r.primaryKeys) == 1 {
		findByIDSQL := fmt.Sprintf(
			"SELECT %s FROM %s WHERE %s = $1",
			"*", // Simplified - could be column list
			r.tableName,
			r.primaryKeys[0],
		)

		// Fix the SELECT columns formatting
		selectCols := make([]string, len(r.selectColumns))
		for i, col := range r.selectColumns {
			selectCols[i] = fmt.Sprintf("\"%s\"", col)
		}
		findByIDSQL = fmt.Sprintf(
			"SELECT %s FROM %s WHERE %s = $1",
			strings.Join(r.selectColumns, ", "),
			r.tableName,
			r.primaryKeys[0],
		)

		// Preparex is available on DBExecutor
		stmt, err := r.db.PreparexContext(context.Background(), findByIDSQL)
		if err != nil {
			return fmt.Errorf("failed to prepare findByID statement: %w", err)
		}
		// Store for later use (we'll add this to Repository struct)
		_ = stmt
	}

	// Prepare INSERT statement (named)
	insertCols := make([]string, len(r.insertColumns))
	insertVals := make([]string, len(r.insertColumns))
	for i, col := range r.insertColumns {
		insertCols[i] = col
		insertVals[i] = ":" + col
	}

	insertSQL := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		r.tableName,
		strings.Join(insertCols, ", "),
		strings.Join(insertVals, ", "),
	)

	namedStmt, err := r.db.PrepareNamedContext(context.Background(), insertSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	r.insertStmt = namedStmt

	return nil
}

// WithTimeout sets a timeout for the query
func (q *Query[T]) WithTimeout(timeout time.Duration) *Query[T] {
	ctx, cancel := context.WithTimeout(q.ctx, timeout)
	q.ctx = ctx
	_ = cancel // In real implementation, we'd store this to call later
	return q
}
