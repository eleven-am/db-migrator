package introspect

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// Column represents a database column definition
type Column struct {
	Name         string
	Type         string
	IsNullable   bool
	DefaultValue *string
	IsPrimaryKey bool
	IsUnique     bool
	IsAutoIncrement bool
	ForeignKey   *ForeignKeyInfo
}

// ForeignKeyInfo represents foreign key constraint information
type ForeignKeyInfo struct {
	ReferencedTable  string
	ReferencedColumn string
	ConstraintName   string
	OnDelete         string
	OnUpdate         string
}

// Index represents a database index
type Index struct {
	Name      string
	TableName string
	Columns   []string
	IsUnique  bool
	IsPrimary bool
}

// IndexDefinition represents an enhanced index definition with signature-based matching
type IndexDefinition struct {
	Name         string
	TableName    string
	Columns      []string
	IsUnique     bool
	IsPrimary    bool
	Method       string // btree, hash, gist, etc.
	Where        string // partial index condition
	Definition   string // full CREATE INDEX statement
	Signature    string // computed signature for comparison
}

// ForeignKeyDefinition represents an enhanced foreign key definition
type ForeignKeyDefinition struct {
	Name              string
	TableName         string
	Columns           []string
	ReferencedTable   string
	ReferencedColumns []string
	OnDelete          string
	OnUpdate          string
	Definition        string // full constraint definition
	Signature         string // computed signature for comparison
}

// Table represents a complete database table structure
type Table struct {
	Name        string
	Columns     []Column
	Indexes     []Index
	Constraints []Constraint
}

// Constraint represents a table constraint
type Constraint struct {
	Name       string
	Type       string // CHECK, UNIQUE, PRIMARY KEY, FOREIGN KEY
	Definition string
	Columns    []string
}

// PostgreSQLIntrospector handles introspection of PostgreSQL databases
type PostgreSQLIntrospector struct {
	db *sql.DB
}

// NewPostgreSQLIntrospector creates a new PostgreSQL introspector
func NewPostgreSQLIntrospector(db *sql.DB) *PostgreSQLIntrospector {
	return &PostgreSQLIntrospector{db: db}
}

// GetTables retrieves all tables from the database
func (i *PostgreSQLIntrospector) GetTables() ([]Table, error) {
	query := `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_type = 'BASE TABLE'
		AND table_name != 'schema_migrations'
		ORDER BY table_name
	`

	rows, err := i.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []Table
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, fmt.Errorf("failed to scan table name: %w", err)
		}

		table, err := i.GetTable(tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to get table %s: %w", tableName, err)
		}

		tables = append(tables, *table)
	}

	return tables, nil
}

// GetTable retrieves detailed information about a specific table
func (i *PostgreSQLIntrospector) GetTable(tableName string) (*Table, error) {
	table := &Table{Name: tableName}

	// Get columns
	columns, err := i.getColumns(tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get columns for table %s: %w", tableName, err)
	}
	table.Columns = columns

	// Get indexes
	indexes, err := i.getIndexes(tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get indexes for table %s: %w", tableName, err)
	}
	table.Indexes = indexes

	// Get constraints
	constraints, err := i.getConstraints(tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get constraints for table %s: %w", tableName, err)
	}
	table.Constraints = constraints

	return table, nil
}

// getColumns retrieves column information for a table
func (i *PostgreSQLIntrospector) getColumns(tableName string) ([]Column, error) {
	query := `
		SELECT 
			a.attname as column_name,
			format_type(a.atttypid, a.atttypmod) as data_type,
			NOT a.attnotnull as is_nullable,
			pg_get_expr(d.adbin, d.adrelid) as column_default,
			CASE WHEN pg_get_expr(d.adbin, d.adrelid) LIKE 'nextval%' THEN true ELSE false END as is_auto_increment,
			CASE WHEN pk.column_name IS NOT NULL THEN true ELSE false END as is_primary_key,
			CASE WHEN uk.column_name IS NOT NULL THEN true ELSE false END as is_unique,
			fk.foreign_table_name,
			fk.foreign_column_name,
			fk.constraint_name as fk_constraint_name,
			fk.delete_rule,
			fk.update_rule
		FROM pg_attribute a
		JOIN pg_class c ON c.oid = a.attrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		LEFT JOIN pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
		LEFT JOIN (
			SELECT kcu.column_name, kcu.table_name
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage kcu 
				ON tc.constraint_name = kcu.constraint_name
				AND tc.table_schema = kcu.table_schema
			WHERE tc.constraint_type = 'PRIMARY KEY'
				AND tc.table_schema = 'public'
		) pk ON a.attname = pk.column_name AND c.relname = pk.table_name
		LEFT JOIN (
			SELECT kcu.column_name, kcu.table_name
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage kcu 
				ON tc.constraint_name = kcu.constraint_name
				AND tc.table_schema = kcu.table_schema
			WHERE tc.constraint_type = 'UNIQUE'
				AND tc.table_schema = 'public'
		) uk ON a.attname = uk.column_name AND c.relname = uk.table_name
		LEFT JOIN (
			SELECT 
				kcu.column_name,
				kcu.table_name,
				ccu.table_name AS foreign_table_name,
				ccu.column_name AS foreign_column_name,
				tc.constraint_name,
				rc.delete_rule,
				rc.update_rule
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage kcu 
				ON tc.constraint_name = kcu.constraint_name
				AND tc.table_schema = kcu.table_schema
			JOIN information_schema.constraint_column_usage ccu 
				ON ccu.constraint_name = tc.constraint_name
				AND ccu.table_schema = tc.table_schema
			JOIN information_schema.referential_constraints rc
				ON tc.constraint_name = rc.constraint_name
				AND tc.table_schema = rc.constraint_schema
			WHERE tc.constraint_type = 'FOREIGN KEY'
				AND tc.table_schema = 'public'
		) fk ON a.attname = fk.column_name AND c.relname = fk.table_name
		WHERE c.relname = $1 
			AND n.nspname = 'public'
			AND a.attnum > 0
			AND NOT a.attisdropped
		ORDER BY a.attnum
	`

	rows, err := i.db.Query(query, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns: %w", err)
	}
	defer rows.Close()

	var columns []Column
	for rows.Next() {
		var col Column
		var defaultValue sql.NullString
		var foreignTableName, foreignColumnName, fkConstraintName sql.NullString
		var deleteRule, updateRule sql.NullString

		err := rows.Scan(
			&col.Name,
			&col.Type,
			&col.IsNullable,
			&defaultValue,
			&col.IsAutoIncrement,
			&col.IsPrimaryKey,
			&col.IsUnique,
			&foreignTableName,
			&foreignColumnName,
			&fkConstraintName,
			&deleteRule,
			&updateRule,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}
		
		if defaultValue.Valid {
			col.DefaultValue = &defaultValue.String
		}

		// Set foreign key info if present
		if foreignTableName.Valid && foreignColumnName.Valid {
			col.ForeignKey = &ForeignKeyInfo{
				ReferencedTable:  foreignTableName.String,
				ReferencedColumn: foreignColumnName.String,
				ConstraintName:   fkConstraintName.String,
				OnDelete:         deleteRule.String,
				OnUpdate:         updateRule.String,
			}
		}

		columns = append(columns, col)
	}

	return columns, nil
}

// getIndexes retrieves index information for a table using enhanced pg_catalog queries
func (i *PostgreSQLIntrospector) getIndexes(tableName string) ([]Index, error) {
	query := `
		SELECT 
			idx.indexname,
			idx.tablename,
			ic.indisunique as is_unique,
			ic.indisprimary as is_primary,
			string_agg(a.attname, ',' ORDER BY ic2.ordinality) as columns,
			idx.indexdef
		FROM pg_indexes idx
		JOIN pg_class tc ON tc.relname = idx.tablename AND tc.relnamespace = (SELECT oid FROM pg_namespace WHERE nspname = 'public')
		JOIN pg_class ic ON ic.relname = idx.indexname
		JOIN pg_index i ON i.indexrelid = ic.oid AND i.indrelid = tc.oid
		JOIN unnest(i.indkey) WITH ORDINALITY AS ic2(attnum, ordinality) ON true
		JOIN pg_attribute a ON a.attrelid = tc.oid AND a.attnum = ic2.attnum
		WHERE idx.tablename = $1
			AND idx.schemaname = 'public'
			-- Exclude system-generated indexes for foreign keys unless they're also user-defined
			AND NOT (idx.indexname ~ '_fkey$' AND ic.indisunique = false AND ic.indisprimary = false)
		GROUP BY idx.indexname, idx.tablename, ic.indisunique, ic.indisprimary, idx.indexdef
		ORDER BY idx.indexname
	`

	rows, err := i.db.Query(query, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query indexes: %w", err)
	}
	defer rows.Close()

	var indexes []Index
	for rows.Next() {
		var idx Index
		var columnsStr string
		var indexDef string

		err := rows.Scan(
			&idx.Name,
			&idx.TableName,
			&idx.IsUnique,
			&idx.IsPrimary,
			&columnsStr,
			&indexDef,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan index: %w", err)
		}

		// Parse column names from aggregated string
		if columnsStr != "" {
			idx.Columns = strings.Split(columnsStr, ",")
			for i := range idx.Columns {
				idx.Columns[i] = strings.TrimSpace(idx.Columns[i])
			}
		}

		indexes = append(indexes, idx)
	}

	return indexes, nil
}

// getConstraints retrieves constraint information for a table using enhanced pg_catalog queries
func (i *PostgreSQLIntrospector) getConstraints(tableName string) ([]Constraint, error) {
	query := `
		WITH constraint_info AS (
			-- Get all constraints from pg_catalog for more accurate information
			SELECT 
				c.conname as constraint_name,
				CASE c.contype
					WHEN 'p' THEN 'PRIMARY KEY'
					WHEN 'u' THEN 'UNIQUE'
					WHEN 'f' THEN 'FOREIGN KEY'
					WHEN 'c' THEN 'CHECK'
					WHEN 'x' THEN 'EXCLUDE'
					WHEN 't' THEN 'TRIGGER'
					ELSE 'UNKNOWN'
				END as constraint_type,
				pg_get_constraintdef(c.oid) as definition,
				array_agg(a.attname ORDER BY array_position(c.conkey, a.attnum)) as columns
			FROM pg_constraint c
			JOIN pg_class t ON t.oid = c.conrelid
			JOIN pg_namespace n ON n.oid = t.relnamespace
			LEFT JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(c.conkey)
			WHERE t.relname = $1
				AND n.nspname = 'public'
				AND c.contype != 'x'  -- Exclude exclusion constraints
				-- Filter out system-generated NOT NULL constraints
				AND NOT (c.contype = 'c' AND pg_get_constraintdef(c.oid) ~ '^\(\w+\s+IS\s+NOT\s+NULL\)$')
			GROUP BY c.conname, c.contype, c.oid
		)
		SELECT 
			constraint_name,
			constraint_type,
			definition,
			array_to_string(columns, ',') as columns_str
		FROM constraint_info
		WHERE array_length(columns, 1) > 0  -- Ensure we have columns
		ORDER BY constraint_name
	`

	rows, err := i.db.Query(query, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query constraints: %w", err)
	}
	defer rows.Close()

	var constraints []Constraint
	for rows.Next() {
		var constraint Constraint
		var columnsStr string

		err := rows.Scan(
			&constraint.Name,
			&constraint.Type,
			&constraint.Definition,
			&columnsStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan constraint: %w", err)
		}

		// Parse columns from comma-separated string
		if columnsStr != "" {
			constraint.Columns = strings.Split(columnsStr, ",")
			for i := range constraint.Columns {
				constraint.Columns[i] = strings.TrimSpace(constraint.Columns[i])
			}
		}

		constraints = append(constraints, constraint)
	}

	return constraints, nil
}


// TableExists checks if a table exists in the database
func (i *PostgreSQLIntrospector) TableExists(tableName string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 
			FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = $1
		)
	`

	var exists bool
	err := i.db.QueryRow(query, tableName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check table existence: %w", err)
	}

	return exists, nil
}

// GetSchema retrieves the complete database schema
func (i *PostgreSQLIntrospector) GetSchema() (map[string]Table, error) {
	tables, err := i.GetTables()
	if err != nil {
		return nil, err
	}

	schema := make(map[string]Table)
	for _, table := range tables {
		schema[table.Name] = table
	}

	return schema, nil
}

// GetEnhancedIndexes retrieves enhanced index information with signatures
func (i *PostgreSQLIntrospector) GetEnhancedIndexes(tableName string) ([]IndexDefinition, error) {
	query := `
		SELECT 
			i.indexname,
			i.tablename,
			ic.indisunique,
			ic.indisprimary,
			am.amname as method,
			pg_get_expr(ic.indpred, ic.indrelid) as where_clause,
			i.indexdef,
			string_agg(a.attname, ',' ORDER BY ic2.ordinality) as columns
		FROM pg_indexes i
		JOIN pg_class tc ON tc.relname = i.tablename AND tc.relnamespace = (SELECT oid FROM pg_namespace WHERE nspname = 'public')
		JOIN pg_class idc ON idc.relname = i.indexname
		JOIN pg_index ic ON ic.indexrelid = idc.oid AND ic.indrelid = tc.oid
		JOIN pg_am am ON am.oid = idc.relam
		JOIN unnest(ic.indkey) WITH ORDINALITY AS ic2(attnum, ordinality) ON true
		JOIN pg_attribute a ON a.attrelid = tc.oid AND a.attnum = ic2.attnum
		WHERE i.tablename = $1
			AND i.schemaname = 'public'
		GROUP BY i.indexname, i.tablename, ic.indisunique, ic.indisprimary, am.amname, ic.indpred, ic.indrelid, i.indexdef
		ORDER BY i.indexname
	`

	rows, err := i.db.Query(query, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query enhanced indexes: %w", err)
	}
	defer rows.Close()

	var indexes []IndexDefinition
	for rows.Next() {
		var idx IndexDefinition
		var whereClause sql.NullString
		var columnsStr string

		err := rows.Scan(
			&idx.Name,
			&idx.TableName,
			&idx.IsUnique,
			&idx.IsPrimary,
			&idx.Method,
			&whereClause,
			&idx.Definition,
			&columnsStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan enhanced index: %w", err)
		}

		if whereClause.Valid {
			idx.Where = whereClause.String
		}

		// Parse columns
		if columnsStr != "" {
			idx.Columns = strings.Split(columnsStr, ",")
			for i := range idx.Columns {
				idx.Columns[i] = strings.TrimSpace(idx.Columns[i])
			}
		}

		// Generate signature for comparison using normalizer
		normalizer := NewSQLNormalizer()
		idx.Signature = normalizer.GenerateCanonicalSignature(
			idx.TableName,
			idx.Columns,
			idx.IsUnique,
			idx.IsPrimary,
			idx.Method,
			idx.Where,
		)

		indexes = append(indexes, idx)
	}

	return indexes, nil
}

// GetEnhancedForeignKeys retrieves enhanced foreign key information with signatures
func (i *PostgreSQLIntrospector) GetEnhancedForeignKeys(tableName string) ([]ForeignKeyDefinition, error) {
	query := `
		SELECT 
			c.conname as constraint_name,
			tc.relname as table_name,
			string_agg(a.attname, ',' ORDER BY array_position(c.conkey, a.attnum)) as columns,
			ft.relname as referenced_table,
			string_agg(af.attname, ',' ORDER BY array_position(c.confkey, af.attnum)) as referenced_columns,
			c.confdeltype as delete_action,
			c.confupdtype as update_action,
			pg_get_constraintdef(c.oid) as definition
		FROM pg_constraint c
		JOIN pg_class tc ON tc.oid = c.conrelid
		JOIN pg_namespace n ON n.oid = tc.relnamespace
		JOIN pg_class ft ON ft.oid = c.confrelid
		JOIN pg_attribute a ON a.attrelid = tc.oid AND a.attnum = ANY(c.conkey)
		JOIN pg_attribute af ON af.attrelid = ft.oid AND af.attnum = ANY(c.confkey)
		WHERE tc.relname = $1
			AND n.nspname = 'public'
			AND c.contype = 'f'
		GROUP BY c.conname, tc.relname, ft.relname, c.confdeltype, c.confupdtype, c.oid
		ORDER BY c.conname
	`

	rows, err := i.db.Query(query, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query enhanced foreign keys: %w", err)
	}
	defer rows.Close()

	var foreignKeys []ForeignKeyDefinition
	for rows.Next() {
		var fk ForeignKeyDefinition
		var deleteAction, updateAction string
		var columnsStr, refColumnsStr string

		err := rows.Scan(
			&fk.Name,
			&fk.TableName,
			&columnsStr,
			&fk.ReferencedTable,
			&refColumnsStr,
			&deleteAction,
			&updateAction,
			&fk.Definition,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan enhanced foreign key: %w", err)
		}

		// Parse columns
		if columnsStr != "" {
			fk.Columns = strings.Split(columnsStr, ",")
			for i := range fk.Columns {
				fk.Columns[i] = strings.TrimSpace(fk.Columns[i])
			}
		}

		// Parse referenced columns
		if refColumnsStr != "" {
			fk.ReferencedColumns = strings.Split(refColumnsStr, ",")
			for i := range fk.ReferencedColumns {
				fk.ReferencedColumns[i] = strings.TrimSpace(fk.ReferencedColumns[i])
			}
		}

		// Convert action codes to readable names
		fk.OnDelete = convertActionCode(deleteAction)
		fk.OnUpdate = convertActionCode(updateAction)

		// Generate signature for comparison
		fk.Signature = generateForeignKeySignatureInternal(fk)

		foreignKeys = append(foreignKeys, fk)
	}

	return foreignKeys, nil
}

// generateForeignKeySignatureInternal creates a canonical signature for foreign key comparison
func generateForeignKeySignatureInternal(fk ForeignKeyDefinition) string {
	normalizer := NewSQLNormalizer()
	var parts []string
	
	// Table and columns (normalized)
	parts = append(parts, "table:"+strings.ToLower(strings.TrimSpace(fk.TableName)))
	normalizedCols := normalizer.NormalizeColumnList(fk.Columns, true)
	parts = append(parts, "cols:"+strings.Join(normalizedCols, ","))
	
	// Referenced table and columns (normalized)
	parts = append(parts, "ref_table:"+strings.ToLower(strings.TrimSpace(fk.ReferencedTable)))
	normalizedRefCols := normalizer.NormalizeColumnList(fk.ReferencedColumns, true)
	parts = append(parts, "ref_cols:"+strings.Join(normalizedRefCols, ","))
	
	// Actions (normalized and defaulted)
	onDelete := strings.ToUpper(strings.TrimSpace(fk.OnDelete))
	if onDelete == "" {
		onDelete = "NO ACTION"
	}
	onUpdate := strings.ToUpper(strings.TrimSpace(fk.OnUpdate))
	if onUpdate == "" {
		onUpdate = "NO ACTION"
	}
	
	parts = append(parts, "on_delete:"+onDelete)
	parts = append(parts, "on_update:"+onUpdate)
	
	return strings.Join(parts, "|")
}

// convertActionCode converts PostgreSQL action codes to readable names
func convertActionCode(code string) string {
	switch code {
	case "a":
		return "NO ACTION"
	case "r":
		return "RESTRICT"
	case "c":
		return "CASCADE"
	case "n":
		return "SET NULL"
	case "d":
		return "SET DEFAULT"
	default:
		return "NO ACTION"
	}
}