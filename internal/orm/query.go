package orm

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

// Query provides a fluent interface for building database queries with all features integrated
type Query[T any] struct {
	repo    *Repository[T]
	builder squirrel.SelectBuilder
	err     error
	ctx     context.Context

	// Query options
	limit       *uint64
	offset      *uint64
	orderBy     []string
	whereClause squirrel.And

	// Transaction support
	tx *sqlx.Tx

	// Join support
	joins    []join
	includes []include
}

// Query creates a new query builder
func (r *Repository[T]) Query() *Query[T] {
	return &Query[T]{
		repo: r,
		builder: squirrel.Select(r.selectColumns...).
			From(r.tableName).
			PlaceholderFormat(squirrel.Dollar),
		ctx:         context.Background(),
		whereClause: squirrel.And{},
		joins:       make([]join, 0),
		includes:    make([]include, 0),
	}
}

// QueryContext creates a new query builder with context
func (r *Repository[T]) QueryContext(ctx context.Context) *Query[T] {
	q := r.Query()
	q.ctx = ctx
	return q
}

// WithTx sets the transaction for this query
func (q *Query[T]) WithTx(tx *sqlx.Tx) *Query[T] {
	q.tx = tx
	return q
}

// Where adds a type-safe condition
func (q *Query[T]) Where(condition Condition) *Query[T] {
	if q.err != nil {
		return q
	}
	q.whereClause = append(q.whereClause, condition.ToSqlizer())
	return q
}

// OrderBy adds an ORDER BY clause
func (q *Query[T]) OrderBy(expressions ...string) *Query[T] {
	if q.err != nil {
		return q
	}
	q.orderBy = append(q.orderBy, expressions...)
	return q
}

// Limit sets the LIMIT clause
func (q *Query[T]) Limit(limit uint64) *Query[T] {
	if q.err != nil {
		return q
	}
	q.limit = &limit
	return q
}

// Offset sets the OFFSET clause
func (q *Query[T]) Offset(offset uint64) *Query[T] {
	if q.err != nil {
		return q
	}
	q.offset = &offset
	return q
}

// Join methods

// Join adds a join clause
func (q *Query[T]) Join(joinType JoinType, table, condition string) *Query[T] {
	if q.err != nil {
		return q
	}
	q.joins = append(q.joins, join{
		Type:      joinType,
		Table:     table,
		Condition: condition,
	})
	return q
}

// InnerJoin adds an INNER JOIN
func (q *Query[T]) InnerJoin(table, condition string) *Query[T] {
	return q.Join(InnerJoin, table, condition)
}

// LeftJoin adds a LEFT JOIN
func (q *Query[T]) LeftJoin(table, condition string) *Query[T] {
	return q.Join(LeftJoin, table, condition)
}

// RightJoin adds a RIGHT JOIN
func (q *Query[T]) RightJoin(table, condition string) *Query[T] {
	return q.Join(RightJoin, table, condition)
}

// FullJoin adds a FULL OUTER JOIN
func (q *Query[T]) FullJoin(table, condition string) *Query[T] {
	return q.Join(FullJoin, table, condition)
}

// Include adds relationships to eager load
func (q *Query[T]) Include(relationships ...string) *Query[T] {
	if q.err != nil {
		return q
	}
	for _, rel := range relationships {
		q.includes = append(q.includes, include{
			name:       rel,
			conditions: make([]Condition, 0),
		})
	}
	return q
}

// IncludeWhere adds a relationship with conditions
func (q *Query[T]) IncludeWhere(relationship string, conditions ...Condition) *Query[T] {
	if q.err != nil {
		return q
	}
	q.includes = append(q.includes, include{
		name:       relationship,
		conditions: conditions,
	})
	return q
}

// Advanced PostgreSQL features

// Build methods

// buildQuery constructs the final SQL query
func (q *Query[T]) buildQuery() (string, []interface{}, error) {
	if q.err != nil {
		return "", nil, q.err
	}

	builder := q.builder

	// Add JOINs
	for _, join := range q.joins {
		switch join.Type {
		case InnerJoin:
			builder = builder.InnerJoin(fmt.Sprintf("%s ON %s", join.Table, join.Condition))
		case LeftJoin:
			builder = builder.LeftJoin(fmt.Sprintf("%s ON %s", join.Table, join.Condition))
		case RightJoin:
			builder = builder.RightJoin(fmt.Sprintf("%s ON %s", join.Table, join.Condition))
		case FullJoin:
			builder = builder.Join(fmt.Sprintf("FULL OUTER JOIN %s ON %s", join.Table, join.Condition))
		}
	}

	// Add WHERE clauses
	if len(q.whereClause) > 0 {
		builder = builder.Where(q.whereClause)
	}

	// Add ORDER BY
	for _, orderBy := range q.orderBy {
		builder = builder.OrderBy(orderBy)
	}

	// Add LIMIT
	if q.limit != nil {
		builder = builder.Limit(*q.limit)
	}

	// Add OFFSET
	if q.offset != nil {
		builder = builder.Offset(*q.offset)
	}

	baseSQL, baseArgs, err := builder.ToSql()
	if err != nil {
		return "", nil, err
	}

	return baseSQL, baseArgs, nil
}

// Execution methods

// Find executes the query and returns all matching records
func (q *Query[T]) Find() ([]T, error) {
	// Handle relationship eager loading
	if len(q.includes) > 0 {
		return q.findWithRelationships()
	}

	sqlQuery, args, err := q.buildQuery()
	if err != nil {
		return nil, &Error{
			Op:    "find",
			Table: q.repo.tableName,
			Err:   fmt.Errorf("failed to build query: %w", err),
		}
	}

	// Execute query
	var records []T
	var execErr error

	if q.tx != nil {
		execErr = q.tx.SelectContext(q.ctx, &records, sqlQuery, args...)
	} else {
		execErr = q.repo.db.SelectContext(q.ctx, &records, sqlQuery, args...)
	}

	if execErr != nil {
		return nil, &Error{
			Op:    "find",
			Table: q.repo.tableName,
			Err:   fmt.Errorf("failed to execute query: %w", execErr),
		}
	}

	return records, nil
}

// First executes the query and returns the first matching record
func (q *Query[T]) First() (*T, error) {
	q.Limit(1)
	records, err := q.Find()
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, &Error{
			Op:    "first",
			Table: q.repo.tableName,
			Err:   ErrNotFound,
		}
	}

	return &records[0], nil
}

// Count returns the number of records matching the query
func (q *Query[T]) Count() (int64, error) {
	countBuilder := squirrel.Select("COUNT(*)").
		From(q.repo.tableName).
		PlaceholderFormat(squirrel.Dollar)

	// Add JOINs
	for _, join := range q.joins {
		switch join.Type {
		case InnerJoin:
			countBuilder = countBuilder.InnerJoin(fmt.Sprintf("%s ON %s", join.Table, join.Condition))
		case LeftJoin:
			countBuilder = countBuilder.LeftJoin(fmt.Sprintf("%s ON %s", join.Table, join.Condition))
		case RightJoin:
			countBuilder = countBuilder.RightJoin(fmt.Sprintf("%s ON %s", join.Table, join.Condition))
		case FullJoin:
			countBuilder = countBuilder.Join(fmt.Sprintf("FULL OUTER JOIN %s ON %s", join.Table, join.Condition))
		}
	}

	// Add WHERE clauses
	if len(q.whereClause) > 0 {
		countBuilder = countBuilder.Where(q.whereClause)
	}

	sqlQuery, args, err := countBuilder.ToSql()
	if err != nil {
		return 0, &Error{
			Op:    "count",
			Table: q.repo.tableName,
			Err:   fmt.Errorf("failed to build count query: %w", err),
		}
	}

	var count int64
	if q.tx != nil {
		err = q.tx.GetContext(q.ctx, &count, sqlQuery, args...)
	} else {
		err = q.repo.db.GetContext(q.ctx, &count, sqlQuery, args...)
	}

	if err != nil {
		return 0, &Error{
			Op:    "count",
			Table: q.repo.tableName,
			Err:   fmt.Errorf("failed to execute count query: %w", err),
		}
	}

	return count, nil
}

// Exists checks if any records match the query
func (q *Query[T]) Exists() (bool, error) {
	count, err := q.Count()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Delete deletes all records matching the query
func (q *Query[T]) Delete() (int64, error) {
	deleteBuilder := squirrel.Delete(q.repo.tableName).
		PlaceholderFormat(squirrel.Dollar)

	// Add WHERE clauses
	if len(q.whereClause) > 0 {
		deleteBuilder = deleteBuilder.Where(q.whereClause)
	}

	sqlQuery, args, err := deleteBuilder.ToSql()
	if err != nil {
		return 0, &Error{
			Op:    "delete",
			Table: q.repo.tableName,
			Err:   fmt.Errorf("failed to build delete query: %w", err),
		}
	}

	var result sql.Result
	if q.tx != nil {
		result, err = q.tx.ExecContext(q.ctx, sqlQuery, args...)
	} else {
		result, err = q.repo.db.ExecContext(q.ctx, sqlQuery, args...)
	}

	if err != nil {
		return 0, &Error{
			Op:    "delete",
			Table: q.repo.tableName,
			Err:   fmt.Errorf("failed to execute delete query: %w", err),
		}
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, &Error{
			Op:    "delete",
			Table: q.repo.tableName,
			Err:   fmt.Errorf("failed to get rows affected: %w", err),
		}
	}

	return rowsAffected, nil
}

// Utility methods

// findWithRelationships handles eager loading of relationships
func (q *Query[T]) findWithRelationships() ([]T, error) {
	// Store includes temporarily
	originalIncludes := q.includes
	q.includes = nil

	// Execute the main query without includes to avoid recursion
	records, err := q.Find()
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return records, nil
	}

	// Load each relationship directly on the records slice
	for _, include := range originalIncludes {
		if err := q.loadRelationship(records, include); err != nil {
			return nil, fmt.Errorf("failed to load relationship %s: %w", include.name, err)
		}
	}

	return records, nil
}

// loadRelationship loads a specific relationship for the records
func (q *Query[T]) loadRelationship(records []T, include include) error {
	if len(records) == 0 {
		return nil
	}

	relationship := q.repo.relationshipManager.GetRelationship(include.name)
	if relationship == nil {
		return fmt.Errorf("relationship %s not found", include.name)
	}

	// Use reflection to dynamically load relationships
	recordValue := reflect.ValueOf(&records[0]).Elem()
	recordType := recordValue.Type()

	// Find the relationship field in the struct
	var relationshipField reflect.StructField
	var fieldFound bool
	for i := 0; i < recordType.NumField(); i++ {
		field := recordType.Field(i)
		if strings.EqualFold(field.Name, include.name) {
			relationshipField = field
			fieldFound = true
			break
		}
	}

	if !fieldFound {
		return fmt.Errorf("relationship field %s not found in struct %s", include.name, recordType.Name())
	}

	switch relationship.Type {
	case "belongs_to":
		return q.loadBelongsToRelationship(records, relationship, relationshipField)
	case "has_one":
		return q.loadHasOneRelationship(records, relationship, relationshipField)
	case "has_many":
		return q.loadHasManyRelationship(records, relationship, relationshipField)
	case "has_many_through":
		return q.loadHasManyThroughRelationship(records, relationship, relationshipField)
	default:
		return fmt.Errorf("unsupported relationship type: %s", relationship.Type)
	}
}

// loadBelongsToRelationship loads belongs_to relationships
func (q *Query[T]) loadBelongsToRelationship(records []T, relationship *relationshipDef, field reflect.StructField) error {
	// Extract foreign key values from all records
	foreignKeys := make([]interface{}, 0, len(records))
	keyToRecordIndices := make(map[interface{}][]int)

	for i, record := range records {
		recordValue := reflect.ValueOf(record)
		fkField := recordValue.FieldByName(relationship.ForeignKey)
		if !fkField.IsValid() || fkField.IsZero() {
			continue
		}

		fkValue := fkField.Interface()
		if _, exists := keyToRecordIndices[fkValue]; !exists {
			foreignKeys = append(foreignKeys, fkValue)
		}
		keyToRecordIndices[fkValue] = append(keyToRecordIndices[fkValue], i)
	}

	if len(foreignKeys) == 0 {
		return nil
	}

	// Query the related records
	relatedQuery := fmt.Sprintf("SELECT * FROM %s WHERE %s = ANY($1)", relationship.Target, relationship.TargetKey)

	// Execute query and scan into appropriate type based on field type
	fieldType := field.Type
	if fieldType.Kind() == reflect.Ptr {
		fieldType = fieldType.Elem()
	}

	// Create slice of the target type
	sliceType := reflect.SliceOf(fieldType)
	relatedRecords := reflect.New(sliceType).Elem()

	var err error
	if q.tx != nil {
		err = q.tx.SelectContext(q.ctx, relatedRecords.Addr().Interface(), relatedQuery, foreignKeys)
	} else {
		err = q.repo.db.SelectContext(q.ctx, relatedRecords.Addr().Interface(), relatedQuery, foreignKeys)
	}

	if err != nil {
		return fmt.Errorf("failed to load belongs_to relationship %s: %w", relationship.FieldName, err)
	}

	// Create map of related records by their primary key
	relatedMap := make(map[interface{}]reflect.Value)
	for i := 0; i < relatedRecords.Len(); i++ {
		relatedRecord := relatedRecords.Index(i)
		pkField := relatedRecord.FieldByName(relationship.TargetKey)
		if pkField.IsValid() {
			relatedMap[pkField.Interface()] = relatedRecord
		}
	}

	// Assign related records to the original records
	for fkValue, recordIndices := range keyToRecordIndices {
		if relatedRecord, exists := relatedMap[fkValue]; exists {
			for _, idx := range recordIndices {
				recordValue := reflect.ValueOf(&records[idx]).Elem()
				relationshipFieldValue := recordValue.FieldByName(field.Name)

				if relationshipFieldValue.CanSet() {
					if field.Type.Kind() == reflect.Ptr {
						// Set pointer to the related record
						ptr := reflect.New(field.Type.Elem())
						ptr.Elem().Set(relatedRecord)
						relationshipFieldValue.Set(ptr)
					} else {
						// Set the value directly
						relationshipFieldValue.Set(relatedRecord)
					}
				}
			}
		}
	}

	return nil
}

// loadHasOneRelationship loads has_one relationships
func (q *Query[T]) loadHasOneRelationship(records []T, relationship *relationshipDef, field reflect.StructField) error {
	// Extract source key values
	sourceKeys := make([]interface{}, 0, len(records))
	keyToRecordIndex := make(map[interface{}]int)

	for i, record := range records {
		recordValue := reflect.ValueOf(record)
		sourceField := recordValue.FieldByName(relationship.SourceKey)
		if !sourceField.IsValid() || sourceField.IsZero() {
			continue
		}

		sourceValue := sourceField.Interface()
		sourceKeys = append(sourceKeys, sourceValue)
		keyToRecordIndex[sourceValue] = i
	}

	if len(sourceKeys) == 0 {
		return nil
	}

	// Query related records
	relatedQuery := fmt.Sprintf("SELECT * FROM %s WHERE %s = ANY($1)", relationship.Target, relationship.ForeignKey)

	fieldType := field.Type
	if fieldType.Kind() == reflect.Ptr {
		fieldType = fieldType.Elem()
	}

	sliceType := reflect.SliceOf(fieldType)
	relatedRecords := reflect.New(sliceType).Elem()

	var err error
	if q.tx != nil {
		err = q.tx.SelectContext(q.ctx, relatedRecords.Addr().Interface(), relatedQuery, sourceKeys)
	} else {
		err = q.repo.db.SelectContext(q.ctx, relatedRecords.Addr().Interface(), relatedQuery, sourceKeys)
	}

	if err != nil {
		return fmt.Errorf("failed to load has_one relationship %s: %w", relationship.FieldName, err)
	}

	// Assign related records
	for i := 0; i < relatedRecords.Len(); i++ {
		relatedRecord := relatedRecords.Index(i)
		fkField := relatedRecord.FieldByName(relationship.ForeignKey)
		if fkField.IsValid() {
			fkValue := fkField.Interface()
			if recordIdx, exists := keyToRecordIndex[fkValue]; exists {
				recordValue := reflect.ValueOf(&records[recordIdx]).Elem()
				relationshipFieldValue := recordValue.FieldByName(field.Name)

				if relationshipFieldValue.CanSet() {
					if field.Type.Kind() == reflect.Ptr {
						ptr := reflect.New(field.Type.Elem())
						ptr.Elem().Set(relatedRecord)
						relationshipFieldValue.Set(ptr)
					} else {
						relationshipFieldValue.Set(relatedRecord)
					}
				}
			}
		}
	}

	return nil
}

// loadHasManyRelationship loads has_many relationships
func (q *Query[T]) loadHasManyRelationship(records []T, relationship *relationshipDef, field reflect.StructField) error {
	// Extract source key values
	sourceKeys := make([]interface{}, 0, len(records))
	keyToRecordIndices := make(map[interface{}][]int)

	for i, record := range records {
		recordValue := reflect.ValueOf(record)
		sourceField := recordValue.FieldByName(relationship.SourceKey)
		if !sourceField.IsValid() || sourceField.IsZero() {
			continue
		}

		sourceValue := sourceField.Interface()
		if _, exists := keyToRecordIndices[sourceValue]; !exists {
			sourceKeys = append(sourceKeys, sourceValue)
		}
		keyToRecordIndices[sourceValue] = append(keyToRecordIndices[sourceValue], i)
	}

	if len(sourceKeys) == 0 {
		return nil
	}

	// Query related records
	relatedQuery := fmt.Sprintf("SELECT * FROM %s WHERE %s = ANY($1)", relationship.Target, relationship.ForeignKey)

	// Determine the element type of the slice
	fieldType := field.Type
	if fieldType.Kind() == reflect.Slice {
		fieldType = fieldType.Elem()
	}

	sliceType := reflect.SliceOf(fieldType)
	relatedRecords := reflect.New(sliceType).Elem()

	var err error
	if q.tx != nil {
		err = q.tx.SelectContext(q.ctx, relatedRecords.Addr().Interface(), relatedQuery, sourceKeys)
	} else {
		err = q.repo.db.SelectContext(q.ctx, relatedRecords.Addr().Interface(), relatedQuery, sourceKeys)
	}

	if err != nil {
		return fmt.Errorf("failed to load has_many relationship %s: %w", relationship.FieldName, err)
	}

	// Group related records by foreign key
	relatedGroups := make(map[interface{}][]reflect.Value)
	for i := 0; i < relatedRecords.Len(); i++ {
		relatedRecord := relatedRecords.Index(i)
		fkField := relatedRecord.FieldByName(relationship.ForeignKey)
		if fkField.IsValid() {
			fkValue := fkField.Interface()
			relatedGroups[fkValue] = append(relatedGroups[fkValue], relatedRecord)
		}
	}

	// Assign related record slices to original records
	for sourceValue, recordIndices := range keyToRecordIndices {
		if relatedGroup, exists := relatedGroups[sourceValue]; exists {
			// Create slice of related records
			sliceValue := reflect.MakeSlice(field.Type, len(relatedGroup), len(relatedGroup))
			for i, relatedRecord := range relatedGroup {
				sliceValue.Index(i).Set(relatedRecord)
			}

			// Assign to all records with this source key
			for _, idx := range recordIndices {
				recordValue := reflect.ValueOf(&records[idx]).Elem()
				relationshipFieldValue := recordValue.FieldByName(field.Name)

				if relationshipFieldValue.CanSet() {
					relationshipFieldValue.Set(sliceValue)
				}
			}
		}
	}

	return nil
}

// loadHasManyThroughRelationship loads has_many_through relationships
func (q *Query[T]) loadHasManyThroughRelationship(records []T, relationship *relationshipDef, field reflect.StructField) error {
	// Extract source key values
	sourceKeys := make([]interface{}, 0, len(records))
	keyToRecordIndices := make(map[interface{}][]int)

	for i, record := range records {
		recordValue := reflect.ValueOf(record)
		sourceField := recordValue.FieldByName(relationship.SourceKey)
		if !sourceField.IsValid() || sourceField.IsZero() {
			continue
		}

		sourceValue := sourceField.Interface()
		if _, exists := keyToRecordIndices[sourceValue]; !exists {
			sourceKeys = append(sourceKeys, sourceValue)
		}
		keyToRecordIndices[sourceValue] = append(keyToRecordIndices[sourceValue], i)
	}

	if len(sourceKeys) == 0 {
		return nil
	}

	// Build JOIN query through the junction table
	relatedQuery := fmt.Sprintf(`
		SELECT t.* FROM %s t
		INNER JOIN %s jt ON t.%s = jt.%s
		WHERE jt.%s = ANY($1)
	`, relationship.Target, relationship.JoinTable,
		relationship.TargetKey, relationship.TargetFK,
		relationship.SourceFK)

	// Determine the element type of the slice
	fieldType := field.Type
	if fieldType.Kind() == reflect.Slice {
		fieldType = fieldType.Elem()
	}

	sliceType := reflect.SliceOf(fieldType)
	relatedRecords := reflect.New(sliceType).Elem()

	var err error
	if q.tx != nil {
		err = q.tx.SelectContext(q.ctx, relatedRecords.Addr().Interface(), relatedQuery, sourceKeys)
	} else {
		err = q.repo.db.SelectContext(q.ctx, relatedRecords.Addr().Interface(), relatedQuery, sourceKeys)
	}

	if err != nil {
		return fmt.Errorf("failed to load has_many_through relationship %s: %w", relationship.FieldName, err)
	}

	// For has_many_through, we need another query to get the junction mapping
	junctionQuery := fmt.Sprintf("SELECT %s, %s FROM %s WHERE %s = ANY($1)",
		relationship.SourceFK, relationship.TargetFK,
		relationship.JoinTable, relationship.SourceFK)

	type junctionRecord struct {
		SourceKey interface{}
		TargetKey interface{}
	}

	var junctionRecords []junctionRecord
	if q.tx != nil {
		err = q.tx.SelectContext(q.ctx, &junctionRecords, junctionQuery, sourceKeys)
	} else {
		err = q.repo.db.SelectContext(q.ctx, &junctionRecords, junctionQuery, sourceKeys)
	}

	if err != nil {
		return fmt.Errorf("failed to load junction table for has_many_through relationship %s: %w", relationship.FieldName, err)
	}

	// Create mapping from source key to target keys
	sourceToTargets := make(map[interface{}][]interface{})
	for _, junction := range junctionRecords {
		sourceToTargets[junction.SourceKey] = append(sourceToTargets[junction.SourceKey], junction.TargetKey)
	}

	// Create mapping from target key to related record
	targetToRecord := make(map[interface{}]reflect.Value)
	for i := 0; i < relatedRecords.Len(); i++ {
		relatedRecord := relatedRecords.Index(i)
		targetField := relatedRecord.FieldByName(relationship.TargetKey)
		if targetField.IsValid() {
			targetToRecord[targetField.Interface()] = relatedRecord
		}
	}

	// Assign related records to original records
	for sourceValue, recordIndices := range keyToRecordIndices {
		if targetKeys, exists := sourceToTargets[sourceValue]; exists {
			// Collect related records for this source
			var relatedGroup []reflect.Value
			for _, targetKey := range targetKeys {
				if relatedRecord, exists := targetToRecord[targetKey]; exists {
					relatedGroup = append(relatedGroup, relatedRecord)
				}
			}

			if len(relatedGroup) > 0 {
				// Create slice of related records
				sliceValue := reflect.MakeSlice(field.Type, len(relatedGroup), len(relatedGroup))
				for i, relatedRecord := range relatedGroup {
					sliceValue.Index(i).Set(relatedRecord)
				}

				// Assign to all records with this source key
				for _, idx := range recordIndices {
					recordValue := reflect.ValueOf(&records[idx]).Elem()
					relationshipFieldValue := recordValue.FieldByName(field.Name)

					if relationshipFieldValue.CanSet() {
						relationshipFieldValue.Set(sliceValue)
					}
				}
			}
		}
	}

	return nil
}

// Advanced PostgreSQL operations

// ExecuteRaw executes a raw SQL query with CTEs and window functions
func (q *Query[T]) ExecuteRaw(query string, args ...interface{}) ([]T, error) {
	finalQuery, finalArgs := q.buildFinalQuery(query, args)

	var records []T
	var err error
	if q.tx != nil {
		err = q.tx.SelectContext(q.ctx, &records, finalQuery, finalArgs...)
	} else {
		err = q.repo.db.SelectContext(q.ctx, &records, finalQuery, finalArgs...)
	}

	if err != nil {
		return nil, &Error{
			Op:    "executeRaw",
			Table: q.repo.tableName,
			Err:   fmt.Errorf("failed to execute raw query: %w", err),
		}
	}

	return records, nil
}

// buildFinalQuery constructs the final query
func (q *Query[T]) buildFinalQuery(query string, args []interface{}) (string, []interface{}) {
	return query, args
}
