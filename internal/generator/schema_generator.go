package generator

import (
	"fmt"
	parser2 "github.com/eleven-am/db-migrator/internal/parser"
	"sort"
	"strings"
)

// SchemaColumn represents a column in the target database schema
type SchemaColumn struct {
	Name            string
	Type            string
	IsNullable      bool
	DefaultValue    *string
	IsPrimaryKey    bool
	IsUnique        bool
	IsAutoIncrement bool
	ForeignKey      *ForeignKeyRef
	CheckConstraint *string
}

// ForeignKeyRef represents a foreign key reference
type ForeignKeyRef struct {
	ReferencedTable  string
	ReferencedColumn string
	OnDelete         string
	OnUpdate         string
}

// SchemaTable represents a table in the target database schema
type SchemaTable struct {
	Name        string
	Columns     []SchemaColumn
	Indexes     []SchemaIndex
	Constraints []SchemaConstraint
}

// SchemaIndex represents a database index
type SchemaIndex struct {
	Name      string
	Columns   []string
	IsUnique  bool
	IsPrimary bool   // Added to identify primary key indexes
	Type      string // e.g., "gin", "btree", "hash"
	Where     string // Partial index condition
}

// SchemaConstraint represents a table constraint
type SchemaConstraint struct {
	Name       string
	Type       string // CHECK, UNIQUE, PRIMARY KEY, FOREIGN KEY
	Definition string
	Columns    []string
}

// DatabaseSchema represents the complete target database schema
type DatabaseSchema struct {
	Tables map[string]SchemaTable
}

// SchemaGenerator converts parsed struct definitions to database schema
type SchemaGenerator struct {
	tagParser *parser2.TagParser
}

// NewSchemaGenerator creates a new schema generator
func NewSchemaGenerator() *SchemaGenerator {
	return &SchemaGenerator{
		tagParser: parser2.NewTagParser(),
	}
}

// GenerateSchema converts table definitions to database schema
func (g *SchemaGenerator) GenerateSchema(tables []parser2.TableDefinition) (*DatabaseSchema, error) {
	schema := &DatabaseSchema{
		Tables: make(map[string]SchemaTable),
	}

	for _, tableDef := range tables {
		schemaTable, err := g.generateTable(tableDef)
		if err != nil {
			return nil, fmt.Errorf("failed to generate schema for table %s: %w", tableDef.TableName, err)
		}
		schema.Tables[schemaTable.Name] = schemaTable
	}

	return schema, nil
}

// generateTable converts a single table definition to schema table
func (g *SchemaGenerator) generateTable(tableDef parser2.TableDefinition) (SchemaTable, error) {
	table := SchemaTable{
		Name:        tableDef.TableName,
		Columns:     make([]SchemaColumn, 0),
		Indexes:     make([]SchemaIndex, 0),
		Constraints: make([]SchemaConstraint, 0),
	}

	// Process each field
	for _, field := range tableDef.Fields {
		column, err := g.generateColumn(field)
		if err != nil {
			return table, fmt.Errorf("failed to generate column %s: %w", field.Name, err)
		}
		table.Columns = append(table.Columns, column)
	}

	// Process table-level definitions
	err := g.processTableLevel(tableDef.TableLevel, &table)
	if err != nil {
		return table, fmt.Errorf("failed to process table-level definitions: %w", err)
	}

	// Add implicit constraints and indexes
	g.addImplicitConstraints(&table)

	return table, nil
}

// generateColumn converts a field definition to a schema column
func (g *SchemaGenerator) generateColumn(field parser2.FieldDefinition) (SchemaColumn, error) {
	column := SchemaColumn{
		Name: field.DBName,
	}

	// Determine PostgreSQL type
	pgType, err := g.mapGoTypeToPostgreSQL(field.Type, field.DBDef)
	if err != nil {
		return column, fmt.Errorf("failed to map type for field %s: %w", field.Name, err)
	}
	column.Type = pgType

	// Check nullability
	column.IsNullable = field.IsPointer || !g.tagParser.HasFlag(field.DBDef, "not_null")

	// Set primary key
	column.IsPrimaryKey = g.tagParser.HasFlag(field.DBDef, "primary_key")
	if column.IsPrimaryKey {
		column.IsNullable = false // Primary keys are always NOT NULL
	}

	// Set unique constraint
	column.IsUnique = g.tagParser.HasFlag(field.DBDef, "unique")

	// Set auto increment
	column.IsAutoIncrement = g.tagParser.HasFlag(field.DBDef, "auto_increment") ||
		strings.Contains(strings.ToLower(column.Type), "serial")

	// Set default value
	if defaultVal := g.tagParser.GetDefault(field.DBDef); defaultVal != "" {
		column.DefaultValue = &defaultVal
	}

	// Set foreign key
	if fkRef := g.tagParser.GetForeignKey(field.DBDef); fkRef != "" {
		fk, err := g.parseForeignKeyRef(fkRef)
		if err != nil {
			return column, fmt.Errorf("invalid foreign key reference: %w", err)
		}
		column.ForeignKey = fk
	}

	// Set check constraint
	if checkExpr, exists := field.DBDef["check"]; exists {
		column.CheckConstraint = &checkExpr
	}

	return column, nil
}

// mapGoTypeToPostgreSQL maps Go types to PostgreSQL types
func (g *SchemaGenerator) mapGoTypeToPostgreSQL(goType string, dbDef map[string]string) (string, error) {
	// If type is explicitly specified in dbdef, use it
	if pgType := g.tagParser.GetType(dbDef); pgType != "" {
		// Special handling for CUID types
		switch strings.ToLower(pgType) {
		case "cuid":
			return "CHAR(25)", nil
		case "cuid2":
			return "VARCHAR(32)", nil // CUID2 uses variable length up to 32 chars
		}
		return pgType, nil
	}

	// Default type mappings based on Go type
	switch goType {
	case "string":
		return "TEXT", nil
	case "int", "int32":
		return "INTEGER", nil
	case "int64":
		return "BIGINT", nil
	case "int16":
		return "SMALLINT", nil
	case "float32":
		return "REAL", nil
	case "float64":
		return "DOUBLE PRECISION", nil
	case "bool":
		return "BOOLEAN", nil
	case "time.Time":
		return "TIMESTAMPTZ", nil
	case "[]byte":
		return "BYTEA", nil
	case "pq.StringArray":
		return "TEXT[]", nil
	case "json.RawMessage", "JSONB":
		return "JSONB", nil
	case "cuid.CUID", "CUID":
		// CUID is typically stored as CHAR(25) or VARCHAR(25)
		return "CHAR(25)", nil
	default:
		// For custom types, default to TEXT and warn
		fmt.Printf("Warning: unknown Go type '%s', defaulting to TEXT\n", goType)
		return "TEXT", nil
	}
}

// parseForeignKeyRef parses foreign key reference string
func (g *SchemaGenerator) parseForeignKeyRef(fkRef string) (*ForeignKeyRef, error) {
	parts := strings.Split(fkRef, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("foreign key must be in format 'table.column', got: %s", fkRef)
	}

	return &ForeignKeyRef{
		ReferencedTable:  strings.TrimSpace(parts[0]),
		ReferencedColumn: strings.TrimSpace(parts[1]),
		OnDelete:         "NO ACTION", // Default
		OnUpdate:         "NO ACTION", // Default
	}, nil
}

// processTableLevel processes table-level dbdef attributes
func (g *SchemaGenerator) processTableLevel(tableLevelDef map[string]string, table *SchemaTable) error {
	for key, value := range tableLevelDef {
		switch key {
		case "table":
			// Table name override - already handled
			continue
		case "index":
			// Parse index definition
			indexes, err := g.parseIndexDefinition(value, table.Name)
			if err != nil {
				return fmt.Errorf("failed to parse index definition: %w", err)
			}
			table.Indexes = append(table.Indexes, indexes...)
		case "unique":
			// Check if this is a partial unique constraint (has WHERE clause)
			if strings.Contains(value, "where:") || strings.Contains(value, "WHERE:") {
				// Parse as a unique index instead
				parts := strings.Split(value, ",")
				if len(parts) < 2 {
					return fmt.Errorf("unique constraint must have name and columns: %s", value)
				}

				indexName := strings.TrimSpace(parts[0])
				var columns []string
				var whereClause string

				for i := 1; i < len(parts); i++ {
					col := strings.TrimSpace(parts[i])
					// Check if this part contains a WHERE clause
					if strings.Contains(col, " where:") || strings.Contains(col, " WHERE:") {
						// Split on where: to separate column name from where clause
						subParts := strings.SplitN(col, " where:", 2)
						if len(subParts) == 2 {
							columns = append(columns, strings.TrimSpace(subParts[0]))
							whereClause = strings.TrimSpace(subParts[1])
						} else {
							subParts = strings.SplitN(col, " WHERE:", 2)
							if len(subParts) == 2 {
								columns = append(columns, strings.TrimSpace(subParts[0]))
								whereClause = strings.TrimSpace(subParts[1])
							}
						}
					} else if strings.HasPrefix(col, "where:") || strings.HasPrefix(col, "WHERE:") {
						whereClause = strings.TrimPrefix(strings.TrimPrefix(col, "where:"), "WHERE:")
					} else if col != "" {
						columns = append(columns, col)
					}
				}

				index := SchemaIndex{
					Name:     indexName,
					Columns:  columns,
					IsUnique: true,
					Where:    whereClause,
				}
				table.Indexes = append(table.Indexes, index)
			} else {
				// Parse as a regular unique constraint
				constraint, err := g.parseUniqueConstraint(value, table.Name)
				if err != nil {
					// If parsing fails, skip this constraint
					fmt.Printf("Warning: failed to parse unique constraint: %v\n", err)
					continue
				}
				table.Constraints = append(table.Constraints, constraint)
			}
		case "check":
			// Parse check constraint
			constraint, err := g.parseCheckConstraint(value, table.Name)
			if err != nil {
				return fmt.Errorf("failed to parse check constraint: %w", err)
			}
			table.Constraints = append(table.Constraints, constraint)
		default:
			fmt.Printf("Warning: unknown table-level attribute '%s'\n", key)
		}
	}

	return nil
}

// parseIndexDefinition parses index definition from dbdef
// Formats supported:
// - Basic: "idx_name,column1,column2"
// - With options: "idx_name,column1,column2 desc"
// - With type: "idx_name,column using:gin"
// - With where clause: "idx_name,column where:status='active'"
// - Functional: "idx_name,LOWER(column)"
func (g *SchemaGenerator) parseIndexDefinition(indexDef, tableName string) ([]SchemaIndex, error) {
	var indexes []SchemaIndex

	// Split multiple index definitions by semicolon
	indexDefs := strings.Split(indexDef, ";")

	for _, def := range indexDefs {
		def = strings.TrimSpace(def)
		if def == "" {
			continue
		}

		// Check for WHERE clause
		var whereClause string
		if whereIdx := strings.Index(def, " where:"); whereIdx != -1 {
			whereClause = def[whereIdx+7:]
			def = def[:whereIdx]
		}

		// Check for USING clause
		var indexType string
		if usingIdx := strings.Index(def, " using:"); usingIdx != -1 {
			indexType = def[usingIdx+7:]
			def = def[:usingIdx]
		}

		parts := strings.Split(def, ",")
		if len(parts) < 2 {
			return nil, fmt.Errorf("index definition must have at least name and one column: %s", def)
		}

		index := SchemaIndex{
			Name:     strings.TrimSpace(parts[0]),
			Columns:  make([]string, 0),
			IsUnique: false,
		}

		// Store additional properties
		if whereClause != "" {
			index.Where = whereClause
		}
		if indexType != "" {
			index.Type = indexType
		}

		// Process columns and flags
		for i := 1; i < len(parts); i++ {
			part := strings.TrimSpace(parts[i])

			// Skip empty parts
			if part == "" {
				continue
			}

			// Check for flags first
			if strings.ToLower(part) == "unique" {
				index.IsUnique = true
				continue
			}

			// Check for column options (desc, asc)
			column := part
			if strings.HasSuffix(strings.ToLower(part), " desc") {
				column = part[:len(part)-5] + " DESC"
			} else if strings.HasSuffix(strings.ToLower(part), " asc") {
				column = part[:len(part)-4] + " ASC"
			}

			index.Columns = append(index.Columns, column)
		}

		if len(index.Columns) == 0 {
			return nil, fmt.Errorf("index must have at least one column: %s", def)
		}

		indexes = append(indexes, index)
	}

	return indexes, nil
}

// parseUniqueConstraint parses unique constraint definition
func (g *SchemaGenerator) parseUniqueConstraint(uniqueDef, tableName string) (SchemaConstraint, error) {
	parts := strings.Split(uniqueDef, ",")
	if len(parts) < 2 {
		return SchemaConstraint{}, fmt.Errorf("unique constraint must have name and columns: %s", uniqueDef)
	}

	constraint := SchemaConstraint{
		Name:    strings.TrimSpace(parts[0]),
		Type:    "UNIQUE",
		Columns: make([]string, 0),
	}

	// Check if this is a partial unique constraint (has WHERE clause)
	var hasWhere bool
	for i := 1; i < len(parts); i++ {
		col := strings.TrimSpace(parts[i])
		if strings.HasPrefix(col, "where:") || strings.HasPrefix(col, "WHERE:") {
			// This is a partial unique constraint - should be created as an index instead
			hasWhere = true
			break
		}
		if col != "" {
			constraint.Columns = append(constraint.Columns, col)
		}
	}

	// If it has a WHERE clause, return empty constraint so it's handled as an index
	if hasWhere {
		return SchemaConstraint{}, fmt.Errorf("partial unique constraints should be created as indexes")
	}

	return constraint, nil
}

// parseCheckConstraint parses check constraint definition
func (g *SchemaGenerator) parseCheckConstraint(checkDef, tableName string) (SchemaConstraint, error) {
	parts := strings.SplitN(checkDef, ",", 2)
	if len(parts) != 2 {
		return SchemaConstraint{}, fmt.Errorf("check constraint must have name and expression: %s", checkDef)
	}

	return SchemaConstraint{
		Name:       strings.TrimSpace(parts[0]),
		Type:       "CHECK",
		Definition: strings.TrimSpace(parts[1]),
	}, nil
}

// addImplicitConstraints adds constraints that are implied by column definitions
func (g *SchemaGenerator) addImplicitConstraints(table *SchemaTable) {
	var primaryKeyColumns []string

	// Find primary key columns
	for _, column := range table.Columns {
		if column.IsPrimaryKey {
			primaryKeyColumns = append(primaryKeyColumns, column.Name)
		}
	}

	// Add primary key constraint if we have primary key columns
	if len(primaryKeyColumns) > 0 {
		pkConstraintName := fmt.Sprintf("%s_pkey", table.Name)
		constraint := SchemaConstraint{
			Name:    pkConstraintName,
			Type:    "PRIMARY KEY",
			Columns: primaryKeyColumns,
		}
		table.Constraints = append(table.Constraints, constraint)

		// Also add the implicit primary key index
		pkIndex := SchemaIndex{
			Name:      pkConstraintName,
			Columns:   primaryKeyColumns,
			IsUnique:  true,
			IsPrimary: true,
		}
		table.Indexes = append(table.Indexes, pkIndex)
	}

	// Add unique constraints for unique columns
	// Skip this - we already handle UNIQUE in column definition
	// Only add named constraints that were explicitly defined in table-level constraints
}

// GetTableNames returns sorted list of table names in the schema
func (s *DatabaseSchema) GetTableNames() []string {
	// First, collect all table names
	var names []string
	for name := range s.Tables {
		names = append(names, name)
	}

	// Sort by dependencies (topological sort)
	sorted := s.sortTablesByDependencies(names)
	return sorted
}

// sortTablesByDependencies performs topological sort on tables based on foreign key dependencies
func (s *DatabaseSchema) sortTablesByDependencies(tables []string) []string {
	// Build dependency graph
	dependencies := make(map[string][]string) // table -> tables that depend on it
	dependents := make(map[string][]string)   // table -> tables it depends on

	// Initialize
	for _, table := range tables {
		dependencies[table] = []string{}
		dependents[table] = []string{}
	}

	// Build dependency relationships
	for _, tableName := range tables {
		table := s.Tables[tableName]
		for _, col := range table.Columns {
			if col.ForeignKey != nil {
				refTable := col.ForeignKey.ReferencedTable
				// tableName depends on refTable
				dependents[tableName] = append(dependents[tableName], refTable)
				// refTable has tableName as a dependent
				dependencies[refTable] = append(dependencies[refTable], tableName)
			}
		}
	}

	// Topological sort using DFS
	visited := make(map[string]bool)
	visiting := make(map[string]bool)
	var result []string

	var visit func(string) bool
	visit = func(table string) bool {
		if visiting[table] {
			// Circular dependency detected
			return false
		}
		if visited[table] {
			return true
		}

		visiting[table] = true

		// Visit all tables this table depends on first
		for _, dep := range dependents[table] {
			if !visit(dep) {
				return false
			}
		}

		visiting[table] = false
		visited[table] = true
		result = append(result, table)
		return true
	}

	// Visit all tables
	for _, table := range tables {
		if !visited[table] {
			if !visit(table) {
				// Circular dependency, fall back to alphabetical sort
				sort.Strings(tables)
				return tables
			}
		}
	}

	return result
}

// HasTable checks if a table exists in the schema
func (s *DatabaseSchema) HasTable(tableName string) bool {
	_, exists := s.Tables[tableName]
	return exists
}

// GetTable retrieves a table from the schema
func (s *DatabaseSchema) GetTable(tableName string) (SchemaTable, bool) {
	table, exists := s.Tables[tableName]
	return table, exists
}
