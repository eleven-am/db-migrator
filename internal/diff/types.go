package diff

import (
	"github.com/eleven-am/db-migrator/internal/generator"
)

// ChangeType represents the type of schema change
type ChangeType string

const (
	ChangeTypeCreateTable ChangeType = "CREATE_TABLE"
	ChangeTypeDropTable   ChangeType = "DROP_TABLE"
	ChangeTypeAlterTable  ChangeType = "ALTER_TABLE"
	ChangeTypeRenameTable ChangeType = "RENAME_TABLE"

	ChangeTypeAddColumn    ChangeType = "ADD_COLUMN"
	ChangeTypeDropColumn   ChangeType = "DROP_COLUMN"
	ChangeTypeAlterColumn  ChangeType = "ALTER_COLUMN"
	ChangeTypeRenameColumn ChangeType = "RENAME_COLUMN"

	ChangeTypeCreateIndex ChangeType = "CREATE_INDEX"
	ChangeTypeDropIndex   ChangeType = "DROP_INDEX"
	ChangeTypeAlterIndex  ChangeType = "ALTER_INDEX"

	ChangeTypeAddConstraint  ChangeType = "ADD_CONSTRAINT"
	ChangeTypeDropConstraint ChangeType = "DROP_CONSTRAINT"
)

// Change represents a single schema change
type Change struct {
	Type        ChangeType
	TableName   string
	ColumnName  string
	OldValue    interface{} // Previous state
	NewValue    interface{} // New state
	SQL         string      // Generated SQL for this change
	IsUnsafe    bool        // Whether this change could cause data loss
	SafetyNotes string      // Explanation of why change is unsafe
}

// DiffResult represents the complete difference between two schemas
type DiffResult struct {
	Changes          []Change
	HasUnsafeChanges bool
	Summary          string
}

// TableDiff represents differences for a specific table
type TableDiff struct {
	TableName         string
	OldTable          *generator.SchemaTable
	NewTable          *generator.SchemaTable
	ColumnChanges     []ColumnChange
	IndexChanges      []IndexChange
	ConstraintChanges []ConstraintChange
	IsRenamed         bool
	OldName           string // If renamed, the previous name
}

// ColumnChange represents a change to a column
type ColumnChange struct {
	Type       ChangeType
	ColumnName string
	OldColumn  *generator.SchemaColumn
	NewColumn  *generator.SchemaColumn
	IsRenamed  bool
	OldName    string // If renamed, the previous name
}

// IndexChange represents a change to an index
type IndexChange struct {
	Type      ChangeType
	IndexName string
	OldIndex  *generator.SchemaIndex
	NewIndex  *generator.SchemaIndex
}

// ConstraintChange represents a change to a constraint
type ConstraintChange struct {
	Type           ChangeType
	ConstraintName string
	OldConstraint  *generator.SchemaConstraint
	NewConstraint  *generator.SchemaConstraint
}

// RenameHint provides hints for detecting renames
type RenameHint struct {
	OldName string
	NewName string
	Type    string // "table", "column"
}
