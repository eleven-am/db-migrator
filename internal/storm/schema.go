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
	config *ststorm.Config
	logger ststorm.Logger
}

func NewSchemaInspector(db *sqlx.DB, config *ststorm.Config, logger ststorm.Logger) *SchemaInspectorImpl {
	return &SchemaInspectorImpl{
		db:     db,
		config: config,
		logger: logger,
	}
}

func (s *SchemaInspectorImpl) Inspect(ctx context.Context) (*ststorm.Schema, error) {
	s.logger.Info("Inspecting database schema...")

	inspector := introspect.NewInspector(s.db.DB, "postgres")

	dbSchema, err := inspector.GetSchema(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to introspect database schema: %w", err)
	}

	stormSchema := s.convertIntrospectSchemaToStorm(dbSchema)

	s.logger.Info("Schema inspection completed", "tables", len(stormSchema.Tables))
	return stormSchema, nil
}

func (s *SchemaInspectorImpl) Compare(ctx context.Context, from, to *ststorm.Schema) (*ststorm.SchemaDiff, error) {
	s.logger.Info("Comparing schemas...")

	diff := &ststorm.SchemaDiff{
		AddedTables:    make(map[string]*ststorm.Table),
		DroppedTables:  make(map[string]*ststorm.Table),
		ModifiedTables: make(map[string]*ststorm.TableDiff),
	}

	for name, toTable := range to.Tables {
		if fromTable, exists := from.Tables[name]; exists {
			tableDiff := s.compareTable(fromTable, toTable)
			if !tableDiff.IsEmpty() {
				diff.ModifiedTables[name] = tableDiff
			}
		} else {
			diff.AddedTables[name] = toTable
		}
	}

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

func (s *SchemaInspectorImpl) ExportSQL(ctx context.Context) (string, error) {
	s.logger.Info("Exporting schema as SQL...")

	inspector := introspect.NewInspector(s.db.DB, "postgres")

	dbSchema, err := inspector.GetSchema(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to introspect database schema: %w", err)
	}

	sqlExport, err := inspector.ExportSchema(dbSchema, introspect.ExportFormatSQL)
	if err != nil {
		return "", fmt.Errorf("failed to export schema as SQL: %w", err)
	}

	return string(sqlExport), nil
}

func (s *SchemaInspectorImpl) ExportGo(ctx context.Context) (string, error) {
	s.logger.Info("Exporting schema as Go structs...")

	inspector := introspect.NewInspector(s.db.DB, "postgres")

	dbSchema, err := inspector.GetSchema(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to introspect database schema: %w", err)
	}

	structGen := introspect.NewStructGenerator(dbSchema, "models")
	goCode, err := structGen.GenerateStructs()
	if err != nil {
		return "", fmt.Errorf("failed to generate Go structs: %w", err)
	}

	return goCode, nil
}

func (s *SchemaInspectorImpl) convertIntrospectSchemaToStorm(dbSchema *introspect.DatabaseSchema) *ststorm.Schema {
	stormSchema := &ststorm.Schema{
		Tables: make(map[string]*ststorm.Table),
	}

	for tableName, table := range dbSchema.Tables {
		stormTable := &ststorm.Table{
			Name:    table.Name,
			Columns: make(map[string]*ststorm.Column),
		}

		for _, col := range table.Columns {
			stormCol := &ststorm.Column{
				Name:     col.Name,
				Type:     col.DataType,
				Nullable: col.IsNullable,
			}
			if col.DefaultValue != nil {
				stormCol.Default = *col.DefaultValue
			}
			stormTable.Columns[col.Name] = stormCol
		}

		if table.PrimaryKey != nil {
			stormTable.PrimaryKey = &ststorm.PrimaryKey{
				Name:    table.PrimaryKey.Name,
				Columns: table.PrimaryKey.Columns,
			}
		}

		for _, fk := range table.ForeignKeys {
			stormFK := &ststorm.ForeignKey{
				Name:           fk.Name,
				Columns:        fk.Columns,
				ForeignTable:   fk.ReferencedTable,
				ForeignColumns: fk.ReferencedColumns,
			}
			stormTable.ForeignKeys = append(stormTable.ForeignKeys, stormFK)
		}

		for _, idx := range table.Indexes {
			columns := make([]string, len(idx.Columns))
			for i, col := range idx.Columns {
				columns[i] = col.Name
			}
			stormIdx := &ststorm.Index{
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

func (s *SchemaInspectorImpl) compareTable(from, to *ststorm.Table) *ststorm.TableDiff {
	diff := &ststorm.TableDiff{
		AddedColumns:    make(map[string]*ststorm.Column),
		DroppedColumns:  make(map[string]*ststorm.Column),
		ModifiedColumns: make(map[string]*ststorm.ColumnDiff),
	}

	for name, toColumn := range to.Columns {
		if fromColumn, exists := from.Columns[name]; exists {
			columnDiff := s.compareColumn(fromColumn, toColumn)
			if !columnDiff.IsEmpty() {
				diff.ModifiedColumns[name] = columnDiff
			}
		} else {
			diff.AddedColumns[name] = toColumn
		}
	}

	for name, fromColumn := range from.Columns {
		if _, exists := to.Columns[name]; !exists {
			diff.DroppedColumns[name] = fromColumn
		}
	}

	return diff
}

func (s *SchemaInspectorImpl) compareColumn(from, to *ststorm.Column) *ststorm.ColumnDiff {
	diff := &ststorm.ColumnDiff{}

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
