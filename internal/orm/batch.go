package orm

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

// UpsertOptions configures upsert behavior
type UpsertOptions struct {
	ConflictColumns []string          // Columns that define conflicts (ON CONFLICT)
	UpdateColumns   []string          // Columns to update on conflict (if empty, updates all non-conflict columns)
	UpdateExpr      map[string]string // Custom update expressions (column -> expression)
}

// Upsert inserts or updates a record using PostgreSQL's ON CONFLICT clause
func (r *Repository[T]) Upsert(ctx context.Context, record *T, opts UpsertOptions) error {
	if record == nil {
		return &Error{
			Op:    "upsert",
			Table: r.tableName,
			Err:   fmt.Errorf("record cannot be nil"),
		}
	}

	if len(opts.ConflictColumns) == 0 {
		return &Error{
			Op:    "upsert",
			Table: r.tableName,
			Err:   fmt.Errorf("conflict columns must be specified"),
		}
	}

	// Build insert query
	query := squirrel.Insert(r.tableName).
		PlaceholderFormat(squirrel.Dollar)

	// Extract values
	recordValue := reflect.ValueOf(record).Elem()
	recordType := recordValue.Type()

	columns := make([]string, 0, len(r.insertColumns))
	values := make([]interface{}, 0, len(r.insertColumns))

	for _, column := range r.insertColumns {
		fieldName := r.reverseMap[column]
		_, found := recordType.FieldByName(fieldName)
		if !found {
			continue
		}

		fieldValue := recordValue.FieldByName(fieldName)
		if !fieldValue.IsValid() {
			continue
		}

		columns = append(columns, column)
		values = append(values, fieldValue.Interface())
	}

	query = query.Columns(columns...).Values(values...)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return &Error{
			Op:    "upsert",
			Table: r.tableName,
			Err:   fmt.Errorf("failed to build insert query: %w", err),
		}
	}

	// Add ON CONFLICT clause
	onConflict := fmt.Sprintf(" ON CONFLICT (%s)", strings.Join(opts.ConflictColumns, ", "))

	// Determine what to update
	var updateColumns []string
	if len(opts.UpdateColumns) > 0 {
		updateColumns = opts.UpdateColumns
	} else {
		// Update all columns except conflict columns
		conflictSet := make(map[string]bool)
		for _, col := range opts.ConflictColumns {
			conflictSet[col] = true
		}

		for _, col := range columns {
			if !conflictSet[col] {
				updateColumns = append(updateColumns, col)
			}
		}
	}

	if len(updateColumns) > 0 {
		var setParts []string
		for _, col := range updateColumns {
			if expr, hasCustom := opts.UpdateExpr[col]; hasCustom {
				setParts = append(setParts, fmt.Sprintf("%s = %s", col, expr))
			} else {
				setParts = append(setParts, fmt.Sprintf("%s = EXCLUDED.%s", col, col))
			}
		}
		onConflict += " DO UPDATE SET " + strings.Join(setParts, ", ")
	} else {
		onConflict += " DO NOTHING"
	}

	// Execute upsert
	finalQuery := sqlQuery + onConflict
	_, err = r.db.ExecContext(ctx, finalQuery, args...)
	if err != nil {
		return &Error{
			Op:    "upsert",
			Table: r.tableName,
			Err:   fmt.Errorf("failed to execute upsert: %w", err),
		}
	}

	return nil
}

// UpsertMany performs bulk upsert operations
func (r *Repository[T]) UpsertMany(ctx context.Context, records []T, opts UpsertOptions) error {
	if len(records) == 0 {
		return nil
	}

	if len(opts.ConflictColumns) == 0 {
		return &Error{
			Op:    "upsertMany",
			Table: r.tableName,
			Err:   fmt.Errorf("conflict columns must be specified"),
		}
	}

	// Check if we're already in a transaction
	var executor DBExecutor
	needsCommit := false

	if _, isTransaction := r.db.(*sqlx.Tx); isTransaction {
		// Already in a transaction, use it
		executor = r.db
	} else {
		// Not in a transaction, create one
		db := r.db.(*sqlx.DB)
		tx, err := db.BeginTxx(ctx, nil)
		if err != nil {
			return &Error{
				Op:    "upsertMany",
				Table: r.tableName,
				Err:   fmt.Errorf("failed to begin transaction: %w", err),
			}
		}
		defer tx.Rollback()
		executor = tx
		needsCommit = true
	} // Will be ignored if commit succeeds

	// Build batch insert query
	query := squirrel.Insert(r.tableName).
		PlaceholderFormat(squirrel.Dollar).
		Columns(r.insertColumns...)

	for _, record := range records {
		v := reflect.ValueOf(record)
		if !v.IsValid() || (v.Kind() == reflect.Ptr && v.IsNil()) {
			continue
		}

		// Extract values for this record
		recordValue := reflect.ValueOf(record).Elem()
		recordType := recordValue.Type()

		values := make([]interface{}, 0, len(r.insertColumns))
		for _, column := range r.insertColumns {
			fieldName := r.reverseMap[column]
			_, found := recordType.FieldByName(fieldName)
			if !found {
				values = append(values, nil)
				continue
			}

			fieldValue := recordValue.FieldByName(fieldName)
			if !fieldValue.IsValid() {
				values = append(values, nil)
				continue
			}

			values = append(values, fieldValue.Interface())
		}

		query = query.Values(values...)
	}

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return &Error{
			Op:    "upsertMany",
			Table: r.tableName,
			Err:   fmt.Errorf("failed to build batch insert query: %w", err),
		}
	}

	// Add ON CONFLICT clause
	onConflict := fmt.Sprintf(" ON CONFLICT (%s)", strings.Join(opts.ConflictColumns, ", "))

	// Determine what to update
	var updateColumns []string
	if len(opts.UpdateColumns) > 0 {
		updateColumns = opts.UpdateColumns
	} else {
		// Update all columns except conflict columns
		conflictSet := make(map[string]bool)
		for _, col := range opts.ConflictColumns {
			conflictSet[col] = true
		}

		for _, col := range r.insertColumns {
			if !conflictSet[col] {
				updateColumns = append(updateColumns, col)
			}
		}
	}

	if len(updateColumns) > 0 {
		var setParts []string
		for _, col := range updateColumns {
			if expr, hasCustom := opts.UpdateExpr[col]; hasCustom {
				setParts = append(setParts, fmt.Sprintf("%s = %s", col, expr))
			} else {
				setParts = append(setParts, fmt.Sprintf("%s = EXCLUDED.%s", col, col))
			}
		}
		onConflict += " DO UPDATE SET " + strings.Join(setParts, ", ")
	} else {
		onConflict += " DO NOTHING"
	}

	// Execute batch upsert
	finalQuery := sqlQuery + onConflict
	_, err = executor.ExecContext(ctx, finalQuery, args...)
	if err != nil {
		return &Error{
			Op:    "upsertMany",
			Table: r.tableName,
			Err:   fmt.Errorf("failed to execute batch upsert: %w", err),
		}
	}

	// Commit transaction only if we created it
	if needsCommit {
		tx := executor.(*sqlx.Tx)
		if err := tx.Commit(); err != nil {
			return &Error{
				Op:    "upsertMany",
				Table: r.tableName,
				Err:   fmt.Errorf("failed to commit transaction: %w", err),
			}
		}
	}

	return nil
}

// BulkUpdateOptions configures bulk update behavior
type BulkUpdateOptions struct {
	UpdateColumns []string // Columns to update (if empty, updates all non-primary key columns)
	WhereColumns  []string // Columns to match on for WHERE clause (if empty, uses primary keys)
}

// BulkUpdate updates multiple records with different values per row using CTE
func (r *Repository[T]) BulkUpdate(ctx context.Context, records []T, opts BulkUpdateOptions) (int64, error) {
	if len(records) == 0 {
		return 0, nil
	}

	// Determine which columns to update and match on
	updateColumns := opts.UpdateColumns
	if len(updateColumns) == 0 {
		updateColumns = r.updateColumns
	}

	whereColumns := opts.WhereColumns
	if len(whereColumns) == 0 {
		whereColumns = r.primaryKeys
	}

	if len(whereColumns) == 0 {
		return 0, &Error{
			Op:    "bulkUpdate",
			Table: r.tableName,
			Err:   fmt.Errorf("no where columns specified and no primary keys found"),
		}
	}

	// Check if we're already in a transaction
	var executor DBExecutor
	needsCommit := false

	if _, isTransaction := r.db.(*sqlx.Tx); isTransaction {
		// Already in a transaction, use it
		executor = r.db
	} else {
		// Not in a transaction, create one
		db := r.db.(*sqlx.DB)
		tx, err := db.BeginTxx(ctx, nil)
		if err != nil {
			return 0, &Error{
				Op:    "bulkUpdate",
				Table: r.tableName,
				Err:   fmt.Errorf("failed to begin transaction: %w", err),
			}
		}
		defer tx.Rollback()
		executor = tx
		needsCommit = true
	}

	// Build CTE query for bulk update
	var valueParts []string
	var args []interface{}
	argIndex := 1

	allColumns := append(whereColumns, updateColumns...)

	for _, record := range records {
		v := reflect.ValueOf(record)
		if !v.IsValid() || (v.Kind() == reflect.Ptr && v.IsNil()) {
			continue
		}

		recordValue := reflect.ValueOf(record).Elem()
		recordType := recordValue.Type()

		var rowValues []string
		for _, column := range allColumns {
			fieldName := r.reverseMap[column]
			_, found := recordType.FieldByName(fieldName)
			if !found {
				rowValues = append(rowValues, "NULL")
				continue
			}

			fieldValue := recordValue.FieldByName(fieldName)
			if !fieldValue.IsValid() {
				rowValues = append(rowValues, "NULL")
				continue
			}

			rowValues = append(rowValues, fmt.Sprintf("$%d", argIndex))
			args = append(args, fieldValue.Interface())
			argIndex++
		}

		valueParts = append(valueParts, "("+strings.Join(rowValues, ", ")+")")
	}

	if len(valueParts) == 0 {
		return 0, nil
	}

	// Build the complete query
	var columnNames []string
	var columnTypes []string
	for _, column := range allColumns {
		columnNames = append(columnNames, column)
		columnTypes = append(columnTypes, "text") // Simplified - in practice you'd want proper types
	}

	cteQuery := fmt.Sprintf(`
		WITH updates(%s) AS (
			VALUES %s
		)
		UPDATE %s 
		SET %s
		FROM updates
		WHERE %s`,
		strings.Join(columnNames, ", "),
		strings.Join(valueParts, ", "),
		r.tableName,
		r.buildUpdateSetClause(updateColumns, whereColumns),
		r.buildWhereClause(whereColumns),
	)

	result, err := executor.ExecContext(ctx, cteQuery, args...)
	if err != nil {
		return 0, &Error{
			Op:    "bulkUpdate",
			Table: r.tableName,
			Err:   fmt.Errorf("failed to execute bulk update: %w", err),
		}
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, &Error{
			Op:    "bulkUpdate",
			Table: r.tableName,
			Err:   fmt.Errorf("failed to get rows affected: %w", err),
		}
	}

	// Commit transaction only if we created it
	if needsCommit {
		tx := executor.(*sqlx.Tx)
		if err := tx.Commit(); err != nil {
			return 0, &Error{
				Op:    "bulkUpdate",
				Table: r.tableName,
				Err:   fmt.Errorf("failed to commit transaction: %w", err),
			}
		}
	}

	return rowsAffected, nil
}

// Helper methods for bulk update query building
func (r *Repository[T]) buildUpdateSetClause(updateColumns, whereColumns []string) string {
	var setParts []string
	for _, col := range updateColumns {
		setParts = append(setParts, fmt.Sprintf("%s = updates.%s", col, col))
	}
	return strings.Join(setParts, ", ")
}

func (r *Repository[T]) buildWhereClause(whereColumns []string) string {
	var whereParts []string
	for _, col := range whereColumns {
		whereParts = append(whereParts, fmt.Sprintf("%s.%s = updates.%s", r.tableName, col, col))
	}
	return strings.Join(whereParts, " AND ")
}
