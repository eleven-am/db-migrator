package testing

import (
	diff2 "github.com/eleven-am/db-migrator/internal/diff"
	"github.com/eleven-am/db-migrator/internal/generator"
	"strings"
	"testing"
)

func TestMigrationSnapshot_CreateTable(t *testing.T) {
	oldSchema := &generator.DatabaseSchema{
		Tables: map[string]generator.SchemaTable{},
	}

	newSchema := &generator.DatabaseSchema{
		Tables: map[string]generator.SchemaTable{
			"users": {
				Name: "users",
				Columns: []generator.SchemaColumn{
					{
						Name:         "id",
						Type:         "UUID",
						IsPrimaryKey: true,
						DefaultValue: strPtr("gen_random_uuid()"),
					},
					{
						Name:       "email",
						Type:       "VARCHAR(255)",
						IsNullable: false,
						IsUnique:   true,
					},
					{
						Name:         "created_at",
						Type:         "TIMESTAMP",
						IsNullable:   false,
						DefaultValue: strPtr("now()"),
					},
				},
				Indexes: []generator.SchemaIndex{
					{
						Name:     "idx_users_email",
						Columns:  []string{"email"},
						IsUnique: true,
					},
				},
			},
		},
	}

	engine := diff2.NewEngine(oldSchema, newSchema)
	result, err := engine.Compare()
	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}

	migGen := diff2.NewMigrationSQLGenerator()
	upSQL, downSQL, err := migGen.GenerateMigration(result)
	if err != nil {
		t.Fatalf("Generate migration failed: %v", err)
	}

	// Normalize whitespace for comparison
	upSQL = normalizeSQL(upSQL)
	downSQL = normalizeSQL(downSQL)

	expectedUp := normalizeSQL(`
CREATE TABLE users (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    PRIMARY KEY (id)
);`)

	expectedDown := normalizeSQL(`DROP TABLE IF EXISTS users CASCADE;`)

	if !strings.Contains(upSQL, expectedUp) {
		t.Errorf("UP migration does not match expected:\nGot:\n%s\n\nExpected to contain:\n%s", upSQL, expectedUp)
	}

	if downSQL != expectedDown {
		t.Errorf("DOWN migration does not match expected:\nGot:\n%s\n\nExpected:\n%s", downSQL, expectedDown)
	}
}

func TestMigrationSnapshot_AlterTable(t *testing.T) {
	oldSchema := &generator.DatabaseSchema{
		Tables: map[string]generator.SchemaTable{
			"users": {
				Name: "users",
				Columns: []generator.SchemaColumn{
					{Name: "id", Type: "UUID", IsPrimaryKey: true},
					{Name: "email", Type: "VARCHAR(255)", IsNullable: false},
					{Name: "old_field", Type: "TEXT", IsNullable: true},
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
					{Name: "name", Type: "VARCHAR(100)", IsNullable: false, DefaultValue: strPtr("'Anonymous'")},
					{Name: "is_active", Type: "BOOLEAN", IsNullable: false, DefaultValue: strPtr("true")},
				},
			},
		},
	}

	engine := diff2.NewEngine(oldSchema, newSchema)
	result, err := engine.Compare()
	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}

	migGen := diff2.NewMigrationSQLGenerator()
	upSQL, downSQL, err := migGen.GenerateMigration(result)
	if err != nil {
		t.Fatalf("Generate migration failed: %v", err)
	}

	// Check UP migration
	sqlAssert := NewAssertSQL(t)
	sqlAssert.Contains(upSQL, "ALTER TABLE users ADD COLUMN name VARCHAR(100) NOT NULL DEFAULT 'Anonymous'")
	sqlAssert.Contains(upSQL, "ALTER TABLE users ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT true")
	sqlAssert.Contains(upSQL, "-- ALTER TABLE users DROP COLUMN old_field") // Should be commented (unsafe)

	// Check DOWN migration
	sqlAssert.Contains(downSQL, "ALTER TABLE users DROP COLUMN is_active")
	sqlAssert.Contains(downSQL, "ALTER TABLE users DROP COLUMN name")
	sqlAssert.Contains(downSQL, "ALTER TABLE users ADD COLUMN old_field TEXT")
}

func TestMigrationSnapshot_ComplexChanges(t *testing.T) {
	oldSchema := &generator.DatabaseSchema{
		Tables: map[string]generator.SchemaTable{
			"users": {
				Name: "users",
				Columns: []generator.SchemaColumn{
					{Name: "id", Type: "INTEGER", IsPrimaryKey: true},
					{Name: "username", Type: "VARCHAR(50)", IsNullable: false},
				},
			},
		},
	}

	newSchema := &generator.DatabaseSchema{
		Tables: map[string]generator.SchemaTable{
			"users": {
				Name: "users",
				Columns: []generator.SchemaColumn{
					{Name: "id", Type: "BIGINT", IsPrimaryKey: true}, // Safe type change
					{Name: "email", Type: "VARCHAR(255)", IsNullable: false, IsUnique: true},
				},
				Indexes: []generator.SchemaIndex{
					{Name: "idx_users_email", Columns: []string{"email"}, IsUnique: true},
				},
			},
			"teams": {
				Name: "teams",
				Columns: []generator.SchemaColumn{
					{Name: "id", Type: "UUID", IsPrimaryKey: true, DefaultValue: strPtr("gen_random_uuid()")},
					{Name: "name", Type: "VARCHAR(100)", IsNullable: false},
					{
						Name: "owner_id",
						Type: "BIGINT",
						ForeignKey: &generator.ForeignKeyRef{
							ReferencedTable:  "users",
							ReferencedColumn: "id",
							OnDelete:         "CASCADE",
						},
					},
				},
			},
		},
	}

	engine := diff2.NewEngine(oldSchema, newSchema)
	// Add rename hint
	engine.AddRenameHint("username", "email", "column")

	result, err := engine.Compare()
	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}

	migGen := diff2.NewMigrationSQLGenerator()
	upSQL, _, err := migGen.GenerateMigration(result)
	if err != nil {
		t.Fatalf("Generate migration failed: %v", err)
	}

	// Debug: print the generated SQL
	t.Logf("Generated SQL:\n%s", upSQL)

	// Verify operation order
	operations := []string{
		"CREATE TABLE teams",                     // Create new tables first
		"ALTER TABLE users ALTER COLUMN id TYPE", // Then alter columns
		"ALTER TABLE users RENAME COLUMN",        // Then rename
	}

	// Check if there's a CREATE UNIQUE INDEX (might not exist if UNIQUE is inline)
	if strings.Contains(upSQL, "CREATE UNIQUE INDEX") {
		operations = append(operations, "CREATE UNIQUE INDEX")
	}

	lastPos := 0
	for _, op := range operations {
		pos := strings.Index(upSQL, op)
		if pos == -1 {
			t.Errorf("Operation not found: %s", op)
			continue
		}
		if pos < lastPos {
			t.Errorf("Operations out of order: %s should come after previous operations", op)
		}
		lastPos = pos
	}
}

// Helper functions
func normalizeSQL(sql string) string {
	// Remove extra whitespace and normalize line endings
	lines := strings.Split(sql, "\n")
	var normalized []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			normalized = append(normalized, line)
		}
	}
	return strings.Join(normalized, "\n")
}

func strPtr(s string) *string {
	return &s
}
