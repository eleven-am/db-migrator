package storm

import (
	"context"
	"fmt"

	"github.com/eleven-am/storm/internal/introspect"
	"github.com/eleven-am/storm/pkg/storm"
	"github.com/jmoiron/sqlx"
)

// SchemaInspectorImpl implements schema introspection
type SchemaInspectorImpl struct {
	db     *sqlx.DB
	config *storm.Config
	logger storm.Logger
}

// NewSchemaInspector creates a new schema inspector
func NewSchemaInspector(db *sqlx.DB, config *storm.Config, logger storm.Logger) *SchemaInspectorImpl {
	return &SchemaInspectorImpl{
		db:     db,
		config: config,
		logger: logger,
	}
}

// Inspect analyzes the database schema
func (s *SchemaInspectorImpl) Inspect(ctx context.Context) (*storm.Schema, error) {
	s.logger.Info("Inspecting database schema...")

	// Use the existing working introspect functionality
	inspector := introspect.NewInspector(s.db.DB, "postgres")

	// Get the database schema
	dbSchema, err := inspector.GetSchema(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to introspect database schema: %w", err)
	}

	// Convert to Storm schema format
	stormSchema := s.convertIntrospectSchemaToStorm(dbSchema)

	s.logger.Info("Schema inspection completed", "tables", len(stormSchema.Tables))
	return stormSchema, nil
}

// Compare compares two schemas and returns differences
func (s *SchemaInspectorImpl) Compare(ctx context.Context, from, to *storm.Schema) (*storm.SchemaDiff, error) {
	s.logger.Info("Comparing schemas...")

	diff := &storm.SchemaDiff{
		AddedTables:    make(map[string]*storm.Table),
		DroppedTables:  make(map[string]*storm.Table),
		ModifiedTables: make(map[string]*storm.TableDiff),
	}

	// Find added and modified tables
	for name, toTable := range to.Tables {
		if fromTable, exists := from.Tables[name]; exists {
			// Table exists in both schemas, check for modifications
			tableDiff := s.compareTable(fromTable, toTable)
			if !tableDiff.IsEmpty() {
				diff.ModifiedTables[name] = tableDiff
			}
		} else {
			// Table only exists in 'to' schema
			diff.AddedTables[name] = toTable
		}
	}

	// Find dropped tables
	for name, fromTable := range from.Tables {
		if _, exists := to.Tables[name]; !exists {
			diff.DroppedTables[name] = fromTable
		}
	}

	s.logger.Info("Schema comparison completed",
		"added", len(diff.AddedTables),
		"dropped", len(diff.DroppedTables),
		"modified", len(diff.ModifiedTables))

	return diff, nil
}

// ExportSQL exports the schema as SQL DDL
func (s *SchemaInspectorImpl) ExportSQL(ctx context.Context) (string, error) {
	s.logger.Info("Exporting schema as SQL...")

	// Use the existing working introspect functionality
	inspector := introspect.NewInspector(s.db.DB, "postgres")

	// Get the database schema
	dbSchema, err := inspector.GetSchema(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to introspect database schema: %w", err)
	}

	// Export as SQL using the existing functionality
	sqlExport, err := inspector.ExportSchema(dbSchema, introspect.ExportFormatSQL)
	if err != nil {
		return "", fmt.Errorf("failed to export schema as SQL: %w", err)
	}

	return string(sqlExport), nil
}

// ExportGo exports the schema as Go structs
func (s *SchemaInspectorImpl) ExportGo(ctx context.Context) (string, error) {
	s.logger.Info("Exporting schema as Go structs...")

	// Use the existing working introspect functionality
	inspector := introspect.NewInspector(s.db.DB, "postgres")

	// Get the database schema
	dbSchema, err := inspector.GetSchema(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to introspect database schema: %w", err)
	}

	// Use the existing struct generator
	structGen := introspect.NewStructGenerator(dbSchema, "models")
	goCode, err := structGen.GenerateStructs()
	if err != nil {
		return "", fmt.Errorf("failed to generate Go structs: %w", err)
	}

	return goCode, nil
}

// Helper methods

// convertIntrospectSchemaToStorm converts introspect.DatabaseSchema to storm.Schema
func (s *SchemaInspectorImpl) convertIntrospectSchemaToStorm(dbSchema *introspect.DatabaseSchema) *storm.Schema {
	stormSchema := &storm.Schema{
		Tables: make(map[string]*storm.Table),
	}

	for tableName, table := range dbSchema.Tables {
		stormTable := &storm.Table{
			Name:    table.Name,
			Columns: make(map[string]*storm.Column),
		}

		// Convert columns
		for _, col := range table.Columns {
			stormCol := &storm.Column{
				Name:     col.Name,
				Type:     col.DataType,
				Nullable: col.IsNullable,
			}
			if col.DefaultValue != nil {
				stormCol.Default = *col.DefaultValue
			}
			stormTable.Columns[col.Name] = stormCol
		}

		// Convert primary key
		if table.PrimaryKey != nil {
			stormTable.PrimaryKey = &storm.PrimaryKey{
				Name:    table.PrimaryKey.Name,
				Columns: table.PrimaryKey.Columns,
			}
		}

		// Convert foreign keys
		for _, fk := range table.ForeignKeys {
			stormFK := &storm.ForeignKey{
				Name:           fk.Name,
				Columns:        fk.Columns,
				ForeignTable:   fk.ReferencedTable,
				ForeignColumns: fk.ReferencedColumns,
			}
			stormTable.ForeignKeys = append(stormTable.ForeignKeys, stormFK)
		}

		// Convert indexes
		for _, idx := range table.Indexes {
			columns := make([]string, len(idx.Columns))
			for i, col := range idx.Columns {
				columns[i] = col.Name
			}
			stormIdx := &storm.Index{
				Name:    idx.Name,
				Columns: columns,
				Unique:  idx.IsUnique,
			}
			stormTable.Indexes = append(stormTable.Indexes, stormIdx)
		}

		stormSchema.Tables[tableName] = stormTable
	}

	return stormSchema
}

func (s *SchemaInspectorImpl) compareTable(from, to *storm.Table) *storm.TableDiff {
	diff := &storm.TableDiff{
		AddedColumns:    make(map[string]*storm.Column),
		DroppedColumns:  make(map[string]*storm.Column),
		ModifiedColumns: make(map[string]*storm.ColumnDiff),
	}

	// Compare columns
	for name, toColumn := range to.Columns {
		if fromColumn, exists := from.Columns[name]; exists {
			// Column exists in both, check for modifications
			columnDiff := s.compareColumn(fromColumn, toColumn)
			if !columnDiff.IsEmpty() {
				diff.ModifiedColumns[name] = columnDiff
			}
		} else {
			// Column only exists in 'to' schema
			diff.AddedColumns[name] = toColumn
		}
	}

	// Find dropped columns
	for name, fromColumn := range from.Columns {
		if _, exists := to.Columns[name]; !exists {
			diff.DroppedColumns[name] = fromColumn
		}
	}

	return diff
}

func (s *SchemaInspectorImpl) compareColumn(from, to *storm.Column) *storm.ColumnDiff {
	diff := &storm.ColumnDiff{}

	if from.Type != to.Type {
		diff.TypeChanged = true
		diff.OldType = from.Type
		diff.NewType = to.Type
	}

	if from.Nullable != to.Nullable {
		diff.NullableChanged = true
		diff.OldNullable = from.Nullable
		diff.NewNullable = to.Nullable
	}

	if from.Default != to.Default {
		diff.DefaultChanged = true
		diff.OldDefault = from.Default
		diff.NewDefault = to.Default
	}

	return diff
}
