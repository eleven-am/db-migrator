package diff

import (
	"github.com/eleven-am/db-migrator/internal/generator"
	"testing"
)

func TestEngine_Compare_CreateTable(t *testing.T) {
	oldSchema := &generator.DatabaseSchema{
		Tables: map[string]generator.SchemaTable{},
	}

	newSchema := &generator.DatabaseSchema{
		Tables: map[string]generator.SchemaTable{
			"users": {
				Name: "users",
				Columns: []generator.SchemaColumn{
					{Name: "id", Type: "UUID", IsPrimaryKey: true},
					{Name: "email", Type: "VARCHAR(255)", IsNullable: false},
				},
			},
		},
	}

	engine := NewEngine(oldSchema, newSchema)
	result, err := engine.Compare()

	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}

	if len(result.Changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(result.Changes))
	}

	change := result.Changes[0]
	if change.Type != ChangeTypeCreateTable {
		t.Errorf("Expected CREATE_TABLE, got %s", change.Type)
	}

	if change.TableName != "users" {
		t.Errorf("Expected table name 'users', got %s", change.TableName)
	}
}

func TestEngine_Compare_DropTable(t *testing.T) {
	oldSchema := &generator.DatabaseSchema{
		Tables: map[string]generator.SchemaTable{
			"users": {
				Name: "users",
				Columns: []generator.SchemaColumn{
					{Name: "id", Type: "UUID", IsPrimaryKey: true},
				},
			},
		},
	}

	newSchema := &generator.DatabaseSchema{
		Tables: map[string]generator.SchemaTable{},
	}

	engine := NewEngine(oldSchema, newSchema)
	result, err := engine.Compare()

	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}

	if len(result.Changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(result.Changes))
	}

	change := result.Changes[0]
	if change.Type != ChangeTypeDropTable {
		t.Errorf("Expected DROP_TABLE, got %s", change.Type)
	}

	if !change.IsUnsafe {
		t.Error("DROP_TABLE should be marked as unsafe")
	}

	if !result.HasUnsafeChanges {
		t.Error("Result should indicate unsafe changes")
	}
}

func TestEngine_Compare_AddColumn(t *testing.T) {
	oldSchema := &generator.DatabaseSchema{
		Tables: map[string]generator.SchemaTable{
			"users": {
				Name: "users",
				Columns: []generator.SchemaColumn{
					{Name: "id", Type: "UUID", IsPrimaryKey: true},
				},
			},
		},
	}

	newSchema := &generator.DatabaseSchema{
		Tables: map[string]generator.SchemaTable{
			"users": {
				Name: "users",
				Columns: []generator.SchemaColumn{
					{Name: "id", Type: "UUID", IsPrimaryKey: true},
					{Name: "email", Type: "VARCHAR(255)", IsNullable: false},
				},
			},
		},
	}

	engine := NewEngine(oldSchema, newSchema)
	result, err := engine.Compare()

	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}

	if len(result.Changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(result.Changes))
	}

	change := result.Changes[0]
	if change.Type != ChangeTypeAddColumn {
		t.Errorf("Expected ADD_COLUMN, got %s", change.Type)
	}

	if change.ColumnName != "email" {
		t.Errorf("Expected column name 'email', got %s", change.ColumnName)
	}
}

func TestEngine_Compare_AlterColumn(t *testing.T) {
	oldSchema := &generator.DatabaseSchema{
		Tables: map[string]generator.SchemaTable{
			"users": {
				Name: "users",
				Columns: []generator.SchemaColumn{
					{Name: "id", Type: "UUID", IsPrimaryKey: true},
					{Name: "name", Type: "VARCHAR(50)", IsNullable: true},
				},
			},
		},
	}

	newSchema := &generator.DatabaseSchema{
		Tables: map[string]generator.SchemaTable{
			"users": {
				Name: "users",
				Columns: []generator.SchemaColumn{
					{Name: "id", Type: "UUID", IsPrimaryKey: true},
					{Name: "name", Type: "VARCHAR(100)", IsNullable: false},
				},
			},
		},
	}

	engine := NewEngine(oldSchema, newSchema)
	result, err := engine.Compare()

	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}

	if len(result.Changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(result.Changes))
	}

	change := result.Changes[0]
	if change.Type != ChangeTypeAlterColumn {
		t.Errorf("Expected ALTER_COLUMN, got %s", change.Type)
	}

	// Making nullable column NOT NULL without default should be unsafe
	if !change.IsUnsafe {
		t.Error("Making nullable column NOT NULL should be unsafe")
	}
}

func TestEngine_Compare_RenameColumn(t *testing.T) {
	oldSchema := &generator.DatabaseSchema{
		Tables: map[string]generator.SchemaTable{
			"users": {
				Name: "users",
				Columns: []generator.SchemaColumn{
					{Name: "id", Type: "UUID", IsPrimaryKey: true},
					{Name: "username", Type: "VARCHAR(50)"},
				},
			},
		},
	}

	newSchema := &generator.DatabaseSchema{
		Tables: map[string]generator.SchemaTable{
			"users": {
				Name: "users",
				Columns: []generator.SchemaColumn{
					{Name: "id", Type: "UUID", IsPrimaryKey: true},
					{Name: "user_name", Type: "VARCHAR(50)"},
				},
			},
		},
	}

	engine := NewEngine(oldSchema, newSchema)
	engine.AddRenameHint("username", "user_name", "column")

	result, err := engine.Compare()

	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}

	if len(result.Changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(result.Changes))
	}

	change := result.Changes[0]
	if change.Type != ChangeTypeRenameColumn {
		t.Errorf("Expected RENAME_COLUMN, got %s", change.Type)
	}
}

func TestEngine_Compare_CreateIndex(t *testing.T) {
	oldSchema := &generator.DatabaseSchema{
		Tables: map[string]generator.SchemaTable{
			"users": {
				Name: "users",
				Columns: []generator.SchemaColumn{
					{Name: "id", Type: "UUID", IsPrimaryKey: true},
					{Name: "email", Type: "VARCHAR(255)"},
				},
				Indexes: []generator.SchemaIndex{},
			},
		},
	}

	newSchema := &generator.DatabaseSchema{
		Tables: map[string]generator.SchemaTable{
			"users": {
				Name: "users",
				Columns: []generator.SchemaColumn{
					{Name: "id", Type: "UUID", IsPrimaryKey: true},
					{Name: "email", Type: "VARCHAR(255)"},
				},
				Indexes: []generator.SchemaIndex{
					{Name: "idx_users_email", Columns: []string{"email"}, IsUnique: true},
				},
			},
		},
	}

	engine := NewEngine(oldSchema, newSchema)
	result, err := engine.Compare()

	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}

	if len(result.Changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(result.Changes))
	}

	change := result.Changes[0]
	if change.Type != ChangeTypeCreateIndex {
		t.Errorf("Expected CREATE_INDEX, got %s", change.Type)
	}
}

func TestEngine_UnsafeTypeChanges(t *testing.T) {
	tests := []struct {
		name     string
		oldType  string
		newType  string
		isUnsafe bool
	}{
		{"safe: varchar to text", "VARCHAR(50)", "TEXT", false},
		{"safe: smallint to integer", "SMALLINT", "INTEGER", false},
		{"safe: integer to bigint", "INTEGER", "BIGINT", false},
		{"safe: real to double", "REAL", "DOUBLE PRECISION", false},
		{"unsafe: text to varchar", "TEXT", "VARCHAR(50)", true},
		{"unsafe: bigint to integer", "BIGINT", "INTEGER", true},
		{"unsafe: varchar to integer", "VARCHAR(50)", "INTEGER", true},
		{"same type", "VARCHAR(50)", "VARCHAR(50)", false},
	}

	engine := NewEngine(&generator.DatabaseSchema{}, &generator.DatabaseSchema{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.isUnsafeTypeChange(tt.oldType, tt.newType)
			if result != tt.isUnsafe {
				t.Errorf("isUnsafeTypeChange(%s, %s) = %v, want %v",
					tt.oldType, tt.newType, result, tt.isUnsafe)
			}
		})
	}
}

func TestEngine_Compare_NoChanges(t *testing.T) {
	schema := &generator.DatabaseSchema{
		Tables: map[string]generator.SchemaTable{
			"users": {
				Name: "users",
				Columns: []generator.SchemaColumn{
					{Name: "id", Type: "UUID", IsPrimaryKey: true},
					{Name: "email", Type: "VARCHAR(255)", IsNullable: false},
				},
			},
		},
	}

	engine := NewEngine(schema, schema)
	result, err := engine.Compare()

	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}

	if len(result.Changes) != 0 {
		t.Errorf("Expected no changes, got %d", len(result.Changes))
	}

	if result.Summary != "No changes detected" {
		t.Errorf("Expected 'No changes detected', got %s", result.Summary)
	}
}
