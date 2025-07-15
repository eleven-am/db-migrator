package orm

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// Common errors
var (
	ErrNotFound          = errors.New("record not found")
	ErrInvalidStruct     = errors.New("invalid struct type")
	ErrNoPrimaryKey      = errors.New("no primary key defined")
	ErrInvalidQuery      = errors.New("invalid query")
	ErrInvalidCondition  = errors.New("invalid condition")
	ErrTransactionFailed = errors.New("transaction failed")
	ErrMigrationFailed   = errors.New("migration failed")
	ErrHookFailed        = errors.New("hook execution failed")
	ErrCacheFailed       = errors.New("cache operation failed")
	ErrDuplicateKey      = errors.New("duplicate key violation")
	ErrForeignKey        = errors.New("foreign key violation")
	ErrCheckConstraint   = errors.New("check constraint violation")
	ErrNotNull           = errors.New("not null constraint violation")
	ErrInvalidType       = errors.New("invalid type")
	ErrConnectionFailed  = errors.New("database connection failed")
	ErrTimeout           = errors.New("operation timeout")
	ErrCanceled          = errors.New("operation canceled")
	ErrMultipleRecords   = errors.New("multiple records found")
	ErrTransactionNil    = errors.New("transaction is nil")
)

// Error provides detailed error information
type Error struct {
	Op         string        // Operation that failed
	Table      string        // Table involved
	Err        error         // Underlying error
	Query      string        // SQL query (if applicable)
	Args       []interface{} // Query arguments (if applicable)
	Constraint string        // Constraint name (if applicable)
	Column     string        // Column name (if applicable)
	Retryable  bool          // Whether the operation can be retried
}

func (e *Error) Error() string {
	var parts []string

	parts = append(parts, fmt.Sprintf("orm: %s", e.Op))

	if e.Table != "" {
		parts = append(parts, fmt.Sprintf("table=%s", e.Table))
	}

	if e.Column != "" {
		parts = append(parts, fmt.Sprintf("column=%s", e.Column))
	}

	if e.Constraint != "" {
		parts = append(parts, fmt.Sprintf("constraint=%s", e.Constraint))
	}

	if e.Err != nil {
		parts = append(parts, e.Err.Error())
	}

	return strings.Join(parts, ": ")
}

func (e *Error) Unwrap() error {
	return e.Err
}

// Is implements errors.Is for Error type
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return errors.Is(e.Err, target)
	}

	// Match if operations are the same
	if t.Op != "" && e.Op == t.Op {
		return true
	}

	// Match if underlying errors match
	return errors.Is(e.Err, t.Err)
}

// ParsePostgreSQLError converts PostgreSQL errors to ORM errors
func ParsePostgreSQLError(err error, op, table string) error {
	if err == nil {
		return nil
	}

	// Check for common errors
	if errors.Is(err, sql.ErrNoRows) {
		return &Error{
			Op:    op,
			Table: table,
			Err:   ErrNotFound,
		}
	}

	// Parse PostgreSQL error codes
	errStr := err.Error()

	// Duplicate key violation
	if strings.Contains(errStr, "duplicate key value violates unique constraint") {
		constraint := extractConstraintName(errStr)
		return &Error{
			Op:         op,
			Table:      table,
			Err:        ErrDuplicateKey,
			Constraint: constraint,
			Retryable:  false,
		}
	}

	// Foreign key violation
	if strings.Contains(errStr, "violates foreign key constraint") {
		constraint := extractConstraintName(errStr)
		return &Error{
			Op:         op,
			Table:      table,
			Err:        ErrForeignKey,
			Constraint: constraint,
			Retryable:  false,
		}
	}

	// Not null violation
	if strings.Contains(errStr, "violates not-null constraint") {
		column := extractColumnName(errStr)
		return &Error{
			Op:        op,
			Table:     table,
			Err:       ErrNotNull,
			Column:    column,
			Retryable: false,
		}
	}

	// Check constraint violation
	if strings.Contains(errStr, "violates check constraint") {
		constraint := extractConstraintName(errStr)
		return &Error{
			Op:         op,
			Table:      table,
			Err:        ErrCheckConstraint,
			Constraint: constraint,
			Retryable:  false,
		}
	}

	// Timeout
	if strings.Contains(errStr, "context deadline exceeded") {
		return &Error{
			Op:        op,
			Table:     table,
			Err:       ErrTimeout,
			Retryable: true,
		}
	}

	// Canceled
	if strings.Contains(errStr, "context canceled") {
		return &Error{
			Op:        op,
			Table:     table,
			Err:       ErrCanceled,
			Retryable: false,
		}
	}

	// Connection errors
	if strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "broken pipe") {
		return &Error{
			Op:        op,
			Table:     table,
			Err:       ErrConnectionFailed,
			Retryable: true,
		}
	}

	// Default error
	return &Error{
		Op:        op,
		Table:     table,
		Err:       err,
		Retryable: false,
	}
}

// Helper functions to extract information from error messages

func extractConstraintName(errStr string) string {
	// Look for quoted constraint name
	start := strings.Index(errStr, "\"")
	if start == -1 {
		return ""
	}
	end := strings.Index(errStr[start+1:], "\"")
	if end == -1 {
		return ""
	}
	return errStr[start+1 : start+1+end]
}

func extractColumnName(errStr string) string {
	// Look for "column \"name\""
	columnIdx := strings.Index(errStr, "column \"")
	if columnIdx == -1 {
		return ""
	}
	start := columnIdx + 8
	end := strings.Index(errStr[start:], "\"")
	if end == -1 {
		return ""
	}
	return errStr[start : start+end]
}

// ValidationError represents validation errors
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation failed for %s: %s", e.Field, e.Message)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 1 {
		return e[0].Error()
	}

	var messages []string
	for _, err := range e {
		messages = append(messages, err.Error())
	}
	return fmt.Sprintf("validation failed: %s", strings.Join(messages, "; "))
}

// IsRetryable checks if an error is retryable
func IsRetryable(err error) bool {
	var ormErr *Error
	if errors.As(err, &ormErr) {
		return ormErr.Retryable
	}
	return false
}

// IsConstraintError checks if an error is a constraint violation
func IsConstraintError(err error) bool {
	return errors.Is(err, ErrDuplicateKey) ||
		errors.Is(err, ErrForeignKey) ||
		errors.Is(err, ErrCheckConstraint) ||
		errors.Is(err, ErrNotNull)
}

// GetConstraintName extracts the constraint name from an error
func GetConstraintName(err error) string {
	var ormErr *Error
	if errors.As(err, &ormErr) {
		return ormErr.Constraint
	}
	return ""
}

// GetColumnName extracts the column name from an error
func GetColumnName(err error) string {
	var ormErr *Error
	if errors.As(err, &ormErr) {
		return ormErr.Column
	}
	return ""
}
