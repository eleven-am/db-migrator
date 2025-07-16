package introspect

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
)

func (i *Inspector) getPostgreSQLSchema(ctx context.Context) (*DatabaseSchema, error) {
	schema := &DatabaseSchema{
		Tables:    make(map[string]*TableSchema),
		Views:     make(map[string]*ViewSchema),
		Enums:     make(map[string]*EnumSchema),
		Functions: make(map[string]*FunctionSchema),
		Sequences: make(map[string]*SequenceSchema),
	}

	var dbName string
	err := i.db.QueryRowContext(ctx, "SELECT current_database()").Scan(&dbName)
	if err != nil {
		return nil, fmt.Errorf("failed to get database name: %w", err)
	}
	schema.Name = dbName

	metadata, err := i.getPostgreSQLMetadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}
	schema.Metadata = *metadata

	tables, err := i.getPostgreSQLTables(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tables: %w", err)
	}
	for _, table := range tables {
		schema.Tables[table.Name] = table
	}

	schema.Views, err = i.getPostgreSQLViews(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get views: %w", err)
	}

	schema.Enums, err = i.getPostgreSQLEnums(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get enums: %w", err)
	}

	schema.Functions, err = i.getPostgreSQLFunctions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get functions: %w", err)
	}

	schema.Sequences, err = i.getPostgreSQLSequences(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get sequences: %w", err)
	}

	return schema, nil
}

func (i *Inspector) getPostgreSQLMetadata(ctx context.Context) (*DatabaseMetadata, error) {
	metadata := &DatabaseMetadata{
		InspectedAt: time.Now(),
	}

	err := i.db.QueryRowContext(ctx, "SELECT version()").Scan(&metadata.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	query := `
		SELECT pg_encoding_to_char(encoding), datcollate
		FROM pg_database
		WHERE datname = current_database()
	`
	err = i.db.QueryRowContext(ctx, query).Scan(&metadata.Encoding, &metadata.Collation)
	if err != nil {
		return nil, fmt.Errorf("failed to get encoding: %w", err)
	}

	err = i.db.QueryRowContext(ctx, "SELECT pg_database_size(current_database())").Scan(&metadata.Size)
	if err != nil {
		return nil, fmt.Errorf("failed to get database size: %w", err)
	}

	err = i.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM information_schema.tables 
		WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
		AND table_type = 'BASE TABLE'
	`).Scan(&metadata.TableCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get table count: %w", err)
	}

	err = i.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM pg_indexes 
		WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
	`).Scan(&metadata.IndexCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get index count: %w", err)
	}

	err = i.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM information_schema.table_constraints
		WHERE constraint_schema NOT IN ('pg_catalog', 'information_schema')
	`).Scan(&metadata.ConstraintCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get constraint count: %w", err)
	}

	return metadata, nil
}

func (i *Inspector) getPostgreSQLTables(ctx context.Context) ([]*TableSchema, error) {
	query := `
		SELECT 
			t.table_schema,
			t.table_name,
			obj_description(c.oid, 'pg_class') as table_comment
		FROM information_schema.tables t
		JOIN pg_class c ON c.relname = t.table_name
		JOIN pg_namespace n ON n.oid = c.relnamespace AND n.nspname = t.table_schema
		WHERE t.table_schema NOT IN ('pg_catalog', 'information_schema')
		AND t.table_type = 'BASE TABLE'
		ORDER BY t.table_schema, t.table_name
	`

	rows, err := i.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []*TableSchema
	for rows.Next() {
		var schema, name string
		var comment sql.NullString

		if err := rows.Scan(&schema, &name, &comment); err != nil {
			return nil, fmt.Errorf("failed to scan table: %w", err)
		}

		table, err := i.getPostgreSQLTable(ctx, schema, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get table %s.%s: %w", schema, name, err)
		}

		if comment.Valid {
			table.Comment = comment.String
		}

		tables = append(tables, table)
	}

	return tables, rows.Err()
}

func (i *Inspector) getPostgreSQLTable(ctx context.Context, schemaName, tableName string) (*TableSchema, error) {
	table := &TableSchema{
		Name:        tableName,
		Schema:      schemaName,
		Columns:     make([]*ColumnSchema, 0),
		ForeignKeys: make([]*ForeignKeySchema, 0),
		Indexes:     make([]*IndexSchema, 0),
		Constraints: make([]*ConstraintSchema, 0),
		Triggers:    make([]*TriggerSchema, 0),
	}

	columns, err := i.getPostgreSQLColumns(ctx, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}
	table.Columns = columns

	pk, err := i.getPostgreSQLPrimaryKey(ctx, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get primary key: %w", err)
	}
	table.PrimaryKey = pk

	fks, err := i.getPostgreSQLForeignKeys(ctx, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get foreign keys: %w", err)
	}
	table.ForeignKeys = fks

	indexes, err := i.getPostgreSQLIndexes(ctx, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get indexes: %w", err)
	}
	table.Indexes = indexes

	constraints, err := i.getPostgreSQLConstraints(ctx, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get constraints: %w", err)
	}
	table.Constraints = constraints

	triggers, err := i.getPostgreSQLTriggers(ctx, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get triggers: %w", err)
	}
	table.Triggers = triggers

	stats, err := i.getPostgreSQLTableStatistics(ctx, schemaName, tableName)
	if err == nil {
		table.RowCount = stats.RowCount
		table.SizeBytes = stats.TotalSizeBytes
	}

	return table, nil
}

func (i *Inspector) getPostgreSQLColumns(ctx context.Context, schemaName, tableName string) ([]*ColumnSchema, error) {
	query := `
		SELECT 
			c.column_name,
			c.ordinal_position,
			c.data_type,
			c.udt_name,
			c.is_nullable = 'YES' as is_nullable,
			c.column_default,
			c.character_maximum_length,
			c.numeric_precision,
			c.numeric_scale,
			c.is_identity = 'YES' as is_identity,
			c.is_generated = 'ALWAYS' as is_generated,
			c.generation_expression,
			col_description(pgc.oid, c.ordinal_position) as column_comment
		FROM information_schema.columns c
		JOIN pg_class pgc ON pgc.relname = c.table_name
		JOIN pg_namespace n ON n.oid = pgc.relnamespace AND n.nspname = c.table_schema
		WHERE c.table_schema = $1 AND c.table_name = $2
		ORDER BY c.ordinal_position
	`

	rows, err := i.db.QueryContext(ctx, query, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns: %w", err)
	}
	defer rows.Close()

	var columns []*ColumnSchema
	for rows.Next() {
		col := &ColumnSchema{}
		var defaultValue, generationExpr, comment sql.NullString
		var charMaxLength, numericPrecision, numericScale sql.NullInt64

		err := rows.Scan(
			&col.Name,
			&col.OrdinalPosition,
			&col.DataType,
			&col.UDTName,
			&col.IsNullable,
			&defaultValue,
			&charMaxLength,
			&numericPrecision,
			&numericScale,
			&col.IsIdentity,
			&col.IsGenerated,
			&generationExpr,
			&comment,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}

		if defaultValue.Valid {
			col.DefaultValue = &defaultValue.String
		}
		if charMaxLength.Valid {
			val := int(charMaxLength.Int64)
			col.CharMaxLength = &val
		}
		if numericPrecision.Valid {
			val := int(numericPrecision.Int64)
			col.NumericPrecision = &val
		}
		if numericScale.Valid {
			val := int(numericScale.Int64)
			col.NumericScale = &val
		}
		if generationExpr.Valid {
			col.GenerationExpr = &generationExpr.String
		}
		if comment.Valid {
			col.Comment = comment.String
		}

		columns = append(columns, col)
	}

	return columns, rows.Err()
}

func (i *Inspector) getPostgreSQLPrimaryKey(ctx context.Context, schemaName, tableName string) (*PrimaryKeySchema, error) {
	query := `
		SELECT 
			tc.constraint_name,
			array_agg(kcu.column_name ORDER BY kcu.ordinal_position) as columns
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu 
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
			AND tc.table_name = kcu.table_name
		WHERE tc.constraint_type = 'PRIMARY KEY'
		AND tc.table_schema = $1
		AND tc.table_name = $2
		GROUP BY tc.constraint_name
	`

	var pk PrimaryKeySchema
	var columns pq.StringArray

	err := i.db.QueryRowContext(ctx, query, schemaName, tableName).Scan(&pk.Name, &columns)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query primary key: %w", err)
	}

	pk.Columns = []string(columns)
	return &pk, nil
}

func (i *Inspector) getPostgreSQLForeignKeys(ctx context.Context, schemaName, tableName string) ([]*ForeignKeySchema, error) {
	query := `
		SELECT 
			tc.constraint_name,
			array_agg(kcu.column_name ORDER BY kcu.ordinal_position) as columns,
			ccu.table_schema as referenced_schema,
			ccu.table_name as referenced_table,
			array_agg(ccu.column_name ORDER BY kcu.ordinal_position) as referenced_columns,
			rc.delete_rule,
			rc.update_rule
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
			AND tc.table_name = kcu.table_name
		JOIN information_schema.constraint_column_usage ccu
			ON tc.constraint_name = ccu.constraint_name
			AND tc.table_schema = ccu.constraint_schema
		JOIN information_schema.referential_constraints rc
			ON tc.constraint_name = rc.constraint_name
			AND tc.table_schema = rc.constraint_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
		AND tc.table_schema = $1
		AND tc.table_name = $2
		GROUP BY tc.constraint_name, ccu.table_schema, ccu.table_name, rc.delete_rule, rc.update_rule
	`

	rows, err := i.db.QueryContext(ctx, query, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query foreign keys: %w", err)
	}
	defer rows.Close()

	var foreignKeys []*ForeignKeySchema
	for rows.Next() {
		fk := &ForeignKeySchema{}
		var columns, refColumns pq.StringArray

		err := rows.Scan(
			&fk.Name,
			&columns,
			&fk.ReferencedSchema,
			&fk.ReferencedTable,
			&refColumns,
			&fk.OnDelete,
			&fk.OnUpdate,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan foreign key: %w", err)
		}

		fk.Columns = []string(columns)
		fk.ReferencedColumns = []string(refColumns)
		foreignKeys = append(foreignKeys, fk)
	}

	return foreignKeys, rows.Err()
}

func (i *Inspector) getPostgreSQLIndexes(ctx context.Context, schemaName, tableName string) ([]*IndexSchema, error) {
	query := `
		SELECT 
			i.relname as index_name,
			idx.indisunique as is_unique,
			idx.indisprimary as is_primary,
			idx.indpred IS NOT NULL as is_partial,
			pg_get_expr(idx.indpred, idx.indrelid) as where_clause,
			am.amname as index_type,
			ARRAY(
				SELECT pg_get_indexdef(idx.indexrelid, k + 1, true)
				FROM generate_subscripts(idx.indkey, 1) as k
				ORDER BY k
			) as columns,
			ts.spcname as tablespace
		FROM pg_index idx
		JOIN pg_class i ON i.oid = idx.indexrelid
		JOIN pg_class t ON t.oid = idx.indrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		JOIN pg_am am ON am.oid = i.relam
		LEFT JOIN pg_tablespace ts ON ts.oid = i.reltablespace
		WHERE n.nspname = $1
		AND t.relname = $2
		AND NOT idx.indisprimary
		ORDER BY i.relname
	`

	rows, err := i.db.QueryContext(ctx, query, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query indexes: %w", err)
	}
	defer rows.Close()

	var indexes []*IndexSchema
	for rows.Next() {
		idx := &IndexSchema{
			Columns: make([]IndexColumn, 0),
		}
		var whereClause sql.NullString
		var tablespace sql.NullString
		var columnExprs pq.StringArray

		err := rows.Scan(
			&idx.Name,
			&idx.IsUnique,
			&idx.IsPrimary,
			&idx.IsPartial,
			&whereClause,
			&idx.Type,
			&columnExprs,
			&tablespace,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan index: %w", err)
		}

		if whereClause.Valid {
			idx.Where = whereClause.String
		}
		if tablespace.Valid {
			idx.TableSpace = tablespace.String
		}

		for _, expr := range columnExprs {
			col := IndexColumn{
				Expression: expr,
			}

			if !strings.Contains(expr, "(") {
				parts := strings.Fields(expr)
				if len(parts) > 0 {
					col.Name = strings.Trim(parts[0], `"`)
					if len(parts) > 1 {
						col.Order = parts[1]
					}
				}
			}
			idx.Columns = append(idx.Columns, col)
		}

		indexes = append(indexes, idx)
	}

	return indexes, rows.Err()
}

func (i *Inspector) getPostgreSQLConstraints(ctx context.Context, schemaName, tableName string) ([]*ConstraintSchema, error) {
	query := `
		SELECT 
			tc.constraint_name,
			tc.constraint_type,
			pg_get_constraintdef(c.oid) as definition,
			COALESCE(array_agg(kcu.column_name ORDER BY kcu.ordinal_position) FILTER (WHERE kcu.column_name IS NOT NULL), '{}') as columns
		FROM information_schema.table_constraints tc
		JOIN pg_constraint c ON c.conname = tc.constraint_name
		JOIN pg_namespace n ON n.oid = c.connamespace AND n.nspname = tc.constraint_schema
		LEFT JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
			AND tc.table_name = kcu.table_name
		WHERE tc.table_schema = $1
		AND tc.table_name = $2
		AND tc.constraint_type IN ('CHECK', 'UNIQUE', 'EXCLUDE')
		GROUP BY tc.constraint_name, tc.constraint_type, c.oid
		ORDER BY tc.constraint_name
	`

	rows, err := i.db.QueryContext(ctx, query, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query constraints: %w", err)
	}
	defer rows.Close()

	var constraints []*ConstraintSchema
	for rows.Next() {
		c := &ConstraintSchema{}
		var columns pq.StringArray

		err := rows.Scan(&c.Name, &c.Type, &c.Definition, &columns)
		if err != nil {
			return nil, fmt.Errorf("failed to scan constraint: %w", err)
		}

		c.Columns = []string(columns)
		constraints = append(constraints, c)
	}

	return constraints, rows.Err()
}

func (i *Inspector) getPostgreSQLTriggers(ctx context.Context, schemaName, tableName string) ([]*TriggerSchema, error) {
	query := `
		SELECT 
			t.tgname as trigger_name,
			CASE t.tgtype & 2 WHEN 2 THEN 'BEFORE' ELSE 'AFTER' END as timing,
			ARRAY_REMOVE(ARRAY[
				CASE t.tgtype & 4 WHEN 4 THEN 'INSERT' END,
				CASE t.tgtype & 8 WHEN 8 THEN 'DELETE' END,
				CASE t.tgtype & 16 WHEN 16 THEN 'UPDATE' END
			], NULL) as events,
			CASE t.tgtype & 1 WHEN 1 THEN 'ROW' ELSE 'STATEMENT' END as level,
			p.proname as function_name,
			pg_get_triggerdef(t.oid) as definition,
			t.tgenabled != 'D' as is_enabled
		FROM pg_trigger t
		JOIN pg_class c ON c.oid = t.tgrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		JOIN pg_proc p ON p.oid = t.tgfoid
		WHERE n.nspname = $1
		AND c.relname = $2
		AND NOT t.tgisinternal
		ORDER BY t.tgname
	`

	rows, err := i.db.QueryContext(ctx, query, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query triggers: %w", err)
	}
	defer rows.Close()

	var triggers []*TriggerSchema
	for rows.Next() {
		tr := &TriggerSchema{}
		var events pq.StringArray

		err := rows.Scan(
			&tr.Name,
			&tr.Timing,
			&events,
			&tr.Level,
			&tr.Function,
			&tr.Definition,
			&tr.IsEnabled,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trigger: %w", err)
		}

		tr.Events = []string(events)
		triggers = append(triggers, tr)
	}

	return triggers, rows.Err()
}

func (i *Inspector) getPostgreSQLTableStatistics(ctx context.Context, schemaName, tableName string) (*TableStatistics, error) {
	query := `
		SELECT 
			n_live_tup as live_tuples,
			n_dead_tup as dead_tuples,
			pg_total_relation_size(c.oid) as total_size,
			pg_relation_size(c.oid) as data_size,
			pg_indexes_size(c.oid) as index_size,
			COALESCE(pg_relation_size(c.reltoastrelid), 0) as toast_size,
			last_vacuum,
			last_autovacuum,
			last_analyze
		FROM pg_stat_user_tables s
		JOIN pg_class c ON c.oid = s.relid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = $1
		AND c.relname = $2
	`

	stats := &TableStatistics{
		TableName: tableName,
	}

	var lastVacuum, lastAutoVacuum, lastAnalyze sql.NullTime

	err := i.db.QueryRowContext(ctx, query, schemaName, tableName).Scan(
		&stats.LiveTuples,
		&stats.DeadTuples,
		&stats.TotalSizeBytes,
		&stats.DataSizeBytes,
		&stats.IndexSizeBytes,
		&stats.ToastSizeBytes,
		&lastVacuum,
		&lastAutoVacuum,
		&lastAnalyze,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query table statistics: %w", err)
	}

	stats.RowCount = stats.LiveTuples

	if lastVacuum.Valid {
		stats.LastVacuum = &lastVacuum.Time
	}
	if lastAutoVacuum.Valid {
		stats.LastAutoVacuum = &lastAutoVacuum.Time
	}
	if lastAnalyze.Valid {
		stats.LastAnalyze = &lastAnalyze.Time
	}

	return stats, nil
}

func (i *Inspector) getPostgreSQLViews(ctx context.Context) (map[string]*ViewSchema, error) {
	query := `
		SELECT 
			v.table_schema,
			v.table_name,
			v.view_definition,
			obj_description(c.oid, 'pg_class') as view_comment
		FROM information_schema.views v
		JOIN pg_class c ON c.relname = v.table_name
		JOIN pg_namespace n ON n.oid = c.relnamespace AND n.nspname = v.table_schema
		WHERE v.table_schema NOT IN ('pg_catalog', 'information_schema')
		ORDER BY v.table_schema, v.table_name
	`

	rows, err := i.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query views: %w", err)
	}
	defer rows.Close()

	views := make(map[string]*ViewSchema)
	for rows.Next() {
		view := &ViewSchema{}
		var comment sql.NullString

		err := rows.Scan(&view.Schema, &view.Name, &view.Definition, &comment)
		if err != nil {
			return nil, fmt.Errorf("failed to scan view: %w", err)
		}

		if comment.Valid {
			view.Comment = comment.String
		}

		view.Columns, err = i.getPostgreSQLColumns(ctx, view.Schema, view.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to get columns for view %s.%s: %w", view.Schema, view.Name, err)
		}

		views[fmt.Sprintf("%s.%s", view.Schema, view.Name)] = view
	}

	return views, rows.Err()
}

func (i *Inspector) getPostgreSQLEnums(ctx context.Context) (map[string]*EnumSchema, error) {
	query := `
		SELECT 
			n.nspname as schema,
			t.typname as name,
			array_agg(e.enumlabel ORDER BY e.enumsortorder) as values
		FROM pg_type t
		JOIN pg_namespace n ON n.oid = t.typnamespace
		JOIN pg_enum e ON e.enumtypid = t.oid
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema')
		GROUP BY n.nspname, t.typname
		ORDER BY n.nspname, t.typname
	`

	rows, err := i.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query enums: %w", err)
	}
	defer rows.Close()

	enums := make(map[string]*EnumSchema)
	for rows.Next() {
		enum := &EnumSchema{}
		var values pq.StringArray

		err := rows.Scan(&enum.Schema, &enum.Name, &values)
		if err != nil {
			return nil, fmt.Errorf("failed to scan enum: %w", err)
		}

		enum.Values = []string(values)
		enums[fmt.Sprintf("%s.%s", enum.Schema, enum.Name)] = enum
	}

	return enums, rows.Err()
}

func (i *Inspector) getPostgreSQLFunctions(ctx context.Context) (map[string]*FunctionSchema, error) {
	query := `
		SELECT 
			n.nspname as schema,
			p.proname as name,
			pg_get_function_result(p.oid) as return_type,
			l.lanname as language,
			p.prosrc as definition,
			p.provolatile = 'v' as is_volatile
		FROM pg_proc p
		JOIN pg_namespace n ON n.oid = p.pronamespace
		JOIN pg_language l ON l.oid = p.prolang
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema')
		AND p.prokind IN ('f', 'p') -- functions and procedures
		AND NOT EXISTS (
			SELECT 1 FROM pg_trigger t WHERE t.tgfoid = p.oid
		)
		ORDER BY n.nspname, p.proname
	`

	rows, err := i.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query functions: %w", err)
	}
	defer rows.Close()

	functions := make(map[string]*FunctionSchema)
	for rows.Next() {
		fn := &FunctionSchema{
			Arguments: make([]FunctionArgument, 0),
		}

		err := rows.Scan(
			&fn.Schema,
			&fn.Name,
			&fn.ReturnType,
			&fn.Language,
			&fn.Definition,
			&fn.IsVolatile,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan function: %w", err)
		}

		functions[fmt.Sprintf("%s.%s", fn.Schema, fn.Name)] = fn
	}

	return functions, rows.Err()
}

func (i *Inspector) getPostgreSQLSequences(ctx context.Context) (map[string]*SequenceSchema, error) {
	query := `
		SELECT 
			n.nspname as schema,
			c.relname as name,
			s.seqtypid::regtype as data_type,
			s.seqstart as start_value,
			s.seqmin as min_value,
			s.seqmax as max_value,
			s.seqincrement as increment,
			s.seqcycle as cycle_option,
			pg_get_serial_sequence(dc.table_schema||'.'||dc.table_name, dc.column_name) as owned_by
		FROM pg_sequence s
		JOIN pg_class c ON c.oid = s.seqrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		LEFT JOIN information_schema.columns dc 
			ON pg_get_serial_sequence(dc.table_schema||'.'||dc.table_name, dc.column_name) = n.nspname||'.'||c.relname
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema')
		ORDER BY n.nspname, c.relname
	`

	rows, err := i.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query sequences: %w", err)
	}
	defer rows.Close()

	sequences := make(map[string]*SequenceSchema)
	for rows.Next() {
		seq := &SequenceSchema{}
		var ownedBy sql.NullString

		err := rows.Scan(
			&seq.Schema,
			&seq.Name,
			&seq.DataType,
			&seq.StartValue,
			&seq.MinValue,
			&seq.MaxValue,
			&seq.Increment,
			&seq.CycleOption,
			&ownedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sequence: %w", err)
		}

		if ownedBy.Valid {
			seq.OwnedBy = ownedBy.String
		}

		sequences[fmt.Sprintf("%s.%s", seq.Schema, seq.Name)] = seq
	}

	return sequences, rows.Err()
}
