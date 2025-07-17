package orm

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"reflect"
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

func (r *Repository[T]) Query(ctx context.Context) *Query[T] {
	return &Query[T]{
		repo: r,
		builder: squirrel.Select(r.Columns()...).
			From(r.metadata.TableName).
			PlaceholderFormat(squirrel.Dollar),
		ctx:         ctx,
		whereClause: squirrel.And{},
		joins:       make([]join, 0),
		includes:    make([]include, 0),
	}
}

func (q *Query[T]) WithTx(tx *sqlx.Tx) *Query[T] {
	q.tx = tx
	return q
}

func (q *Query[T]) Where(condition Condition) *Query[T] {
	if q.err != nil {
		return q
	}
	q.whereClause = append(q.whereClause, condition.ToSqlizer())
	return q
}

func (q *Query[T]) OrderBy(expressions ...string) *Query[T] {
	if q.err != nil {
		return q
	}
	q.orderBy = append(q.orderBy, expressions...)
	return q
}

func (q *Query[T]) Limit(limit uint64) *Query[T] {
	if q.err != nil {
		return q
	}
	q.limit = &limit
	return q
}

func (q *Query[T]) Offset(offset uint64) *Query[T] {
	if q.err != nil {
		return q
	}
	q.offset = &offset
	return q
}

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

func (q *Query[T]) InnerJoin(table, condition string) *Query[T] {
	return q.Join(InnerJoin, table, condition)
}

func (q *Query[T]) LeftJoin(table, condition string) *Query[T] {
	return q.Join(LeftJoin, table, condition)
}

func (q *Query[T]) RightJoin(table, condition string) *Query[T] {
	return q.Join(RightJoin, table, condition)
}

func (q *Query[T]) FullJoin(table, condition string) *Query[T] {
	return q.Join(FullJoin, table, condition)
}

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

func (q *Query[T]) buildQuery() (string, []interface{}, error) {
	if q.err != nil {
		return "", nil, q.err
	}

	builder := q.builder

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

	if len(q.whereClause) > 0 {
		builder = builder.Where(q.whereClause)
	}

	for _, orderBy := range q.orderBy {
		builder = builder.OrderBy(orderBy)
	}

	if q.limit != nil {
		builder = builder.Limit(*q.limit)
	}

	if q.offset != nil {
		builder = builder.Offset(*q.offset)
	}

	baseSQL, baseArgs, err := builder.ToSql()
	if err != nil {
		return "", nil, err
	}

	return baseSQL, baseArgs, nil
}

func (q *Query[T]) Find() ([]T, error) {
	if len(q.includes) > 0 {
		return q.findWithRelationships()
	}

	finalBuilder := q.builder

	for _, join := range q.joins {
		switch join.Type {
		case InnerJoin:
			finalBuilder = finalBuilder.InnerJoin(fmt.Sprintf("%s ON %s", join.Table, join.Condition))
		case LeftJoin:
			finalBuilder = finalBuilder.LeftJoin(fmt.Sprintf("%s ON %s", join.Table, join.Condition))
		case RightJoin:
			finalBuilder = finalBuilder.RightJoin(fmt.Sprintf("%s ON %s", join.Table, join.Condition))
		case FullJoin:
			finalBuilder = finalBuilder.Join(fmt.Sprintf("FULL OUTER JOIN %s ON %s", join.Table, join.Condition))
		}
	}

	if len(q.whereClause) > 0 {
		finalBuilder = finalBuilder.Where(q.whereClause)
	}

	for _, orderBy := range q.orderBy {
		finalBuilder = finalBuilder.OrderBy(orderBy)
	}

	if q.limit != nil {
		finalBuilder = finalBuilder.Limit(*q.limit)
	}

	if q.offset != nil {
		finalBuilder = finalBuilder.Offset(*q.offset)
	}

	var records []T
	err := q.repo.executeQueryMiddleware(OpQuery, q.ctx, nil, finalBuilder, func(middlewareCtx *MiddlewareContext) error {
		finalQuery := middlewareCtx.QueryBuilder.(squirrel.SelectBuilder)

		sqlQuery, args, err := finalQuery.ToSql()
		if err != nil {
			return &Error{
				Op:    "find",
				Table: q.repo.metadata.TableName,
				Err:   fmt.Errorf("failed to build query: %w", err),
			}
		}

		var execErr error
		if q.tx != nil {
			execErr = q.tx.SelectContext(q.ctx, &records, sqlQuery, args...)
		} else {
			execErr = q.repo.db.SelectContext(q.ctx, &records, sqlQuery, args...)
		}

		if execErr != nil {
			return &Error{
				Op:    "find",
				Table: q.repo.metadata.TableName,
				Err:   fmt.Errorf("failed to execute query: %w", execErr),
			}
		}

		return nil
	})

	return records, err
}

func (q *Query[T]) First() (*T, error) {
	q.Limit(1)
	records, err := q.Find()
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, &Error{
			Op:    "first",
			Table: q.repo.metadata.TableName,
			Err:   ErrNotFound,
		}
	}

	return &records[0], nil
}

func (q *Query[T]) Count() (int64, error) {
	countBuilder := squirrel.Select("COUNT(*)").
		From(q.repo.metadata.TableName).
		PlaceholderFormat(squirrel.Dollar)

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

	if len(q.whereClause) > 0 {
		countBuilder = countBuilder.Where(q.whereClause)
	}

	var count int64
	err := q.repo.executeQueryMiddleware(OpQuery, q.ctx, nil, countBuilder, func(middlewareCtx *MiddlewareContext) error {
		finalQuery := middlewareCtx.QueryBuilder.(squirrel.SelectBuilder)

		sqlQuery, args, err := finalQuery.ToSql()
		if err != nil {
			return &Error{
				Op:    "count",
				Table: q.repo.metadata.TableName,
				Err:   fmt.Errorf("failed to build count query: %w", err),
			}
		}

		var execErr error
		if q.tx != nil {
			execErr = q.tx.GetContext(q.ctx, &count, sqlQuery, args...)
		} else {
			execErr = q.repo.db.GetContext(q.ctx, &count, sqlQuery, args...)
		}

		if execErr != nil {
			return &Error{
				Op:    "count",
				Table: q.repo.metadata.TableName,
				Err:   fmt.Errorf("failed to execute count query: %w", execErr),
			}
		}

		return nil
	})

	return count, err
}

func (q *Query[T]) Exists() (bool, error) {
	count, err := q.Count()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (q *Query[T]) Delete() (int64, error) {
	deleteBuilder := squirrel.Delete(q.repo.metadata.TableName).
		PlaceholderFormat(squirrel.Dollar)

	if len(q.whereClause) > 0 {
		deleteBuilder = deleteBuilder.Where(q.whereClause)
	}

	var rowsAffected int64
	err := q.repo.executeQueryMiddleware(OpDelete, q.ctx, nil, deleteBuilder, func(middlewareCtx *MiddlewareContext) error {
		finalQuery := middlewareCtx.QueryBuilder.(squirrel.DeleteBuilder)

		sqlQuery, args, err := finalQuery.ToSql()
		if err != nil {
			return &Error{
				Op:    "delete",
				Table: q.repo.metadata.TableName,
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
			return parsePostgreSQLError(err, "delete", q.repo.metadata.TableName)
		}

		rowsAffected, err = result.RowsAffected()
		if err != nil {
			return &Error{
				Op:    "delete",
				Table: q.repo.metadata.TableName,
				Err:   fmt.Errorf("failed to get rows affected: %w", err),
			}
		}

		return nil
	})

	return rowsAffected, err
}

func (q *Query[T]) Update(updates map[string]interface{}) (int64, error) {
	if len(updates) == 0 {
		return 0, &Error{
			Op:    "update",
			Table: q.repo.metadata.TableName,
			Err:   fmt.Errorf("no updates provided"),
		}
	}

	updateBuilder := squirrel.Update(q.repo.metadata.TableName).
		PlaceholderFormat(squirrel.Dollar)

	for column, value := range updates {
		updateBuilder = updateBuilder.Set(column, value)
	}

	if len(q.whereClause) > 0 {
		updateBuilder = updateBuilder.Where(q.whereClause)
	}

	var rowsAffected int64
	err := q.repo.executeQueryMiddleware(OpUpdateMany, q.ctx, updates, updateBuilder, func(middlewareCtx *MiddlewareContext) error {
		finalQuery := middlewareCtx.QueryBuilder.(squirrel.UpdateBuilder)

		sqlQuery, args, err := finalQuery.ToSql()
		if err != nil {
			return &Error{
				Op:    "update",
				Table: q.repo.metadata.TableName,
				Err:   fmt.Errorf("failed to build update query: %w", err),
			}
		}

		middlewareCtx.Query = sqlQuery
		middlewareCtx.Args = args

		var result sql.Result
		if q.tx != nil {
			result, err = q.tx.ExecContext(q.ctx, sqlQuery, args...)
		} else {
			result, err = q.repo.db.ExecContext(q.ctx, sqlQuery, args...)
		}

		if err != nil {
			return parsePostgreSQLError(err, "update", q.repo.metadata.TableName)
		}

		rowsAffected, err = result.RowsAffected()
		if err != nil {
			return &Error{
				Op:    "update",
				Table: q.repo.metadata.TableName,
				Err:   fmt.Errorf("failed to get rows affected: %w", err),
			}
		}

		return nil
	})

	return rowsAffected, err
}

func (q *Query[T]) findWithRelationships() ([]T, error) {

	originalIncludes := q.includes
	q.includes = nil

	records, err := q.Find()
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return records, nil
	}

	for _, include := range originalIncludes {
		if err := q.loadRelationship(records, include); err != nil {
			return nil, fmt.Errorf("failed to load relationship %s: %w", include.name, err)
		}
	}

	return records, nil
}

func (q *Query[T]) loadRelationship(records []T, include include) error {
	if len(records) == 0 {
		return nil
	}

	relationship := q.repo.relationshipManager.getRelationship(include.name)
	if relationship == nil {
		return fmt.Errorf("relationship %s not found", include.name)
	}

	if relationship.SetValue == nil {
		return fmt.Errorf("relationship %s does not have SetValue function", include.name)
	}

	switch relationship.Type {
	case "belongs_to":
		return q.loadBelongsToRelationship(records, relationship)
	case "has_one":
		return q.loadHasOneRelationship(records, relationship)
	case "has_many":
		return q.loadHasManyRelationship(records, relationship)
	case "has_many_through":
		return q.loadHasManyThroughRelationship(records, relationship)
	default:
		return fmt.Errorf("unsupported relationship type: %s", relationship.Type)
	}
}

func (q *Query[T]) loadBelongsToRelationship(records []T, relationship *relationshipDef) error {

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

	relatedQuery := fmt.Sprintf("SELECT * FROM %s WHERE %s = ANY($1)", relationship.Target, relationship.TargetKey)

	var zero T
	zeroValue := reflect.ValueOf(zero)
	relField := zeroValue.FieldByName(relationship.FieldName)
	if !relField.IsValid() {
		return fmt.Errorf("relationship field %s not found", relationship.FieldName)
	}

	fieldType := relField.Type()
	if fieldType.Kind() == reflect.Ptr {
		fieldType = fieldType.Elem()
	}

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

	relatedMap := make(map[interface{}]reflect.Value)
	for i := 0; i < relatedRecords.Len(); i++ {
		relatedRecord := relatedRecords.Index(i)
		pkField := relatedRecord.FieldByName(relationship.TargetKey)
		if pkField.IsValid() {
			relatedMap[pkField.Interface()] = relatedRecord
		}
	}

	for fkValue, recordIndices := range keyToRecordIndices {
		if relatedRecord, exists := relatedMap[fkValue]; exists {
			for _, idx := range recordIndices {
				relationship.SetValue(&records[idx], relatedRecord.Interface())
			}
		}
	}

	return nil
}

func (q *Query[T]) loadHasOneRelationship(records []T, relationship *relationshipDef) error {

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

	relatedQuery := fmt.Sprintf("SELECT * FROM %s WHERE %s = ANY($1)", relationship.Target, relationship.ForeignKey)

	var zero T
	zeroValue := reflect.ValueOf(zero)
	relField := zeroValue.FieldByName(relationship.FieldName)
	if !relField.IsValid() {
		return fmt.Errorf("relationship field %s not found", relationship.FieldName)
	}

	fieldType := relField.Type()
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

	for i := 0; i < relatedRecords.Len(); i++ {
		relatedRecord := relatedRecords.Index(i)
		fkField := relatedRecord.FieldByName(relationship.ForeignKey)
		if fkField.IsValid() {
			fkValue := fkField.Interface()
			if recordIdx, exists := keyToRecordIndex[fkValue]; exists {
				relationship.SetValue(&records[recordIdx], relatedRecord.Interface())
			}
		}
	}

	return nil
}

func (q *Query[T]) loadHasManyRelationship(records []T, relationship *relationshipDef) error {

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

	relatedQuery := fmt.Sprintf("SELECT * FROM %s WHERE %s = ANY($1)", relationship.Target, relationship.ForeignKey)

	var zero T
	zeroValue := reflect.ValueOf(zero)
	relField := zeroValue.FieldByName(relationship.FieldName)
	if !relField.IsValid() {
		return fmt.Errorf("relationship field %s not found", relationship.FieldName)
	}

	fieldType := relField.Type()
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

	relatedGroups := make(map[interface{}][]reflect.Value)
	for i := 0; i < relatedRecords.Len(); i++ {
		relatedRecord := relatedRecords.Index(i)
		fkField := relatedRecord.FieldByName(relationship.ForeignKey)
		if fkField.IsValid() {
			fkValue := fkField.Interface()
			relatedGroups[fkValue] = append(relatedGroups[fkValue], relatedRecord)
		}
	}

	for sourceValue, recordIndices := range keyToRecordIndices {
		if relatedGroup, exists := relatedGroups[sourceValue]; exists {

			sliceValue := reflect.MakeSlice(relField.Type(), len(relatedGroup), len(relatedGroup))
			for i, relatedRecord := range relatedGroup {
				sliceValue.Index(i).Set(relatedRecord)
			}

			for _, idx := range recordIndices {
				relationship.SetValue(&records[idx], sliceValue.Interface())
			}
		}
	}

	return nil
}

func (q *Query[T]) loadHasManyThroughRelationship(records []T, relationship *relationshipDef) error {

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

	relatedQuery := fmt.Sprintf(`
		SELECT t.* FROM %s t
		INNER JOIN %s jt ON t.%s = jt.%s
		WHERE jt.%s = ANY($1)
	`, relationship.Target, relationship.JoinTable,
		relationship.TargetKey, relationship.TargetFK,
		relationship.SourceFK)

	var zero T
	zeroValue := reflect.ValueOf(zero)
	relField := zeroValue.FieldByName(relationship.FieldName)
	if !relField.IsValid() {
		return fmt.Errorf("relationship field %s not found", relationship.FieldName)
	}

	fieldType := relField.Type()
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

	sourceToTargets := make(map[interface{}][]interface{})
	for _, junction := range junctionRecords {
		sourceToTargets[junction.SourceKey] = append(sourceToTargets[junction.SourceKey], junction.TargetKey)
	}

	targetToRecord := make(map[interface{}]reflect.Value)
	for i := 0; i < relatedRecords.Len(); i++ {
		relatedRecord := relatedRecords.Index(i)
		targetField := relatedRecord.FieldByName(relationship.TargetKey)
		if targetField.IsValid() {
			targetToRecord[targetField.Interface()] = relatedRecord
		}
	}

	for sourceValue, recordIndices := range keyToRecordIndices {
		if targetKeys, exists := sourceToTargets[sourceValue]; exists {

			var relatedGroup []reflect.Value
			for _, targetKey := range targetKeys {
				if relatedRecord, exists := targetToRecord[targetKey]; exists {
					relatedGroup = append(relatedGroup, relatedRecord)
				}
			}

			if len(relatedGroup) > 0 {

				sliceValue := reflect.MakeSlice(relField.Type(), len(relatedGroup), len(relatedGroup))
				for i, relatedRecord := range relatedGroup {
					sliceValue.Index(i).Set(relatedRecord)
				}

				for _, idx := range recordIndices {
					relationship.SetValue(&records[idx], sliceValue.Interface())
				}
			}
		}
	}

	return nil
}

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
			Table: q.repo.metadata.TableName,
			Err:   fmt.Errorf("failed to execute raw query: %w", err),
		}
	}

	return records, nil
}

func (q *Query[T]) buildFinalQuery(query string, args []interface{}) (string, []interface{}) {
	return query, args
}
