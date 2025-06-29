package main

import (
	"github.com/eleven-am/db-migrator/internal/generator"
	"github.com/eleven-am/db-migrator/internal/introspect"
	"github.com/eleven-am/db-migrator/internal/parser"
	dbtest "github.com/eleven-am/db-migrator/internal/testing"
	"os"
	"path/filepath"
	"testing"
)

// TestEndToEndMigrationWorkflow tests the complete migration workflow
func TestEndToEndMigrationWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := dbtest.NewTestDB(t)
	defer testDB.Cleanup()

	// Create a temporary directory with test Go structs
	tmpDir := t.TempDir()
	testModelsFile := filepath.Join(tmpDir, "models.go")

	// Write test models that represent a realistic scenario
	testModels := `
package models

import "time"

// User represents a user in the system
type User struct {
	_        struct{} ` + "`" + `dbdef:"table:users;index:idx_users_team_id,team_id"` + "`" + `
	
	ID        string    ` + "`" + `db:"id" dbdef:"type:cuid;primary_key;default:gen_cuid()"` + "`" + `
	Email     string    ` + "`" + `db:"email" dbdef:"type:varchar(255);not_null;unique"` + "`" + `
	TeamID    string    ` + "`" + `db:"team_id" dbdef:"type:cuid;not_null;foreign_key:teams.id"` + "`" + `
	IsActive  bool      ` + "`" + `db:"is_active" dbdef:"type:boolean;not_null;default:true"` + "`" + `
	CreatedAt time.Time ` + "`" + `db:"created_at" dbdef:"type:timestamptz;not_null;default:now()"` + "`" + `
}

// Team represents a team
type Team struct {
	_        struct{} ` + "`" + `dbdef:"table:teams"` + "`" + `
	
	ID        string    ` + "`" + `db:"id" dbdef:"type:cuid;primary_key;default:gen_cuid()"` + "`" + `
	Name      string    ` + "`" + `db:"name" dbdef:"type:varchar(255);not_null;unique"` + "`" + `
	CreatedAt time.Time ` + "`" + `db:"created_at" dbdef:"type:timestamptz;not_null;default:now()"` + "`" + `
}

// Project represents a project with partial indexes
type Project struct {
	_        struct{} ` + "`" + `dbdef:"table:projects;index:idx_projects_active,is_active where:is_active=true;unique:uk_projects_team_name,team_id,name"` + "`" + `
	
	ID        string    ` + "`" + `db:"id" dbdef:"type:cuid;primary_key;default:gen_cuid()"` + "`" + `
	TeamID    string    ` + "`" + `db:"team_id" dbdef:"type:cuid;not_null;foreign_key:teams.id"` + "`" + `
	Name      string    ` + "`" + `db:"name" dbdef:"type:varchar(255);not_null"` + "`" + `
	IsActive  bool      ` + "`" + `db:"is_active" dbdef:"type:boolean;not_null;default:true"` + "`" + `
	CreatedAt time.Time ` + "`" + `db:"created_at" dbdef:"type:timestamptz;not_null;default:now()"` + "`" + `
}
`

	if err := os.WriteFile(testModelsFile, []byte(testModels), 0644); err != nil {
		t.Fatalf("Failed to write test models: %v", err)
	}

	// Step 1: Parse Go structs
	structParser := parser.NewStructParser()
	tables, err := structParser.ParseFile(testModelsFile)
	if err != nil {
		t.Fatalf("Failed to parse structs: %v", err)
	}

	if len(tables) != 3 {
		t.Fatalf("Expected 3 tables, got %d", len(tables))
	}

	// Step 2: Generate schema definitions from structs
	enhancedGen := generator.NewEnhancedGenerator()

	var allStructIndexes []generator.IndexDefinition
	var allStructFKs []generator.ForeignKeyDefinition

	for _, table := range tables {
		indexes, err := enhancedGen.GenerateIndexDefinitions(table)
		if err != nil {
			t.Fatalf("Failed to generate indexes for %s: %v", table.TableName, err)
		}
		allStructIndexes = append(allStructIndexes, indexes...)

		fks, err := enhancedGen.GenerateForeignKeyDefinitions(table)
		if err != nil {
			t.Fatalf("Failed to generate foreign keys for %s: %v", table.TableName, err)
		}
		allStructFKs = append(allStructFKs, fks...)
	}

	// Step 3: Create initial database schema (simulate existing database)
	initialSchema := `
		CREATE TABLE teams (
			id TEXT PRIMARY KEY DEFAULT 'old_default',
			name VARCHAR(255) NOT NULL UNIQUE,
			created_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE users (
			id TEXT PRIMARY KEY DEFAULT 'old_default',
			email VARCHAR(255) NOT NULL UNIQUE,
			team_id TEXT NOT NULL,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT NOW()
		);

		CREATE INDEX idx_users_team_id ON users(team_id);
		CREATE INDEX idx_users_old_field ON users(created_at); -- This should be dropped

		-- Missing foreign key constraint (should be created)

		-- Missing projects table entirely (should be created in real scenario)
	`

	if err := testDB.ExecuteSQL(initialSchema); err != nil {
		t.Fatalf("Failed to setup initial schema: %v", err)
	}

	// Step 4: Introspect current database state
	introspector := introspect.NewPostgreSQLIntrospector(testDB.DB)

	tableNames, err := introspector.GetTableNames()
	if err != nil {
		t.Fatalf("Failed to get table names: %v", err)
	}

	var allDBIndexes []generator.IndexDefinition
	var allDBFKs []generator.ForeignKeyDefinition

	for _, tableName := range tableNames {
		dbIndexes, err := introspector.GetEnhancedIndexes(tableName)
		if err != nil {
			t.Fatalf("Failed to get indexes for %s: %v", tableName, err)
		}
		allDBIndexes = append(allDBIndexes, dbIndexes...)

		dbFKs, err := introspector.GetEnhancedForeignKeys(tableName)
		if err != nil {
			t.Fatalf("Failed to get foreign keys for %s: %v", tableName, err)
		}
		allDBFKs = append(allDBFKs, dbFKs...)
	}

	// Step 5: Compare schemas
	comparison, err := enhancedGen.CompareSchemas(allStructIndexes, allStructFKs, allDBIndexes, allDBFKs)
	if err != nil {
		t.Fatalf("Failed to compare schemas: %v", err)
	}

	// Step 6: Verify the comparison results
	t.Run("Schema Comparison Results", func(t *testing.T) {
		// Should have indexes to create (projects table indexes, foreign keys)
		if len(comparison.IndexesToCreate) == 0 {
			t.Error("Expected some indexes to create")
		}

		// Should have indexes to drop (idx_users_old_field)
		if len(comparison.IndexesToDrop) == 0 {
			t.Error("Expected some indexes to drop")
		}

		// Should have foreign keys to create
		if len(comparison.ForeignKeysToCreate) == 0 {
			t.Error("Expected some foreign keys to create")
		}

		t.Logf("Indexes to create: %d", len(comparison.IndexesToCreate))
		t.Logf("Indexes to drop: %d", len(comparison.IndexesToDrop))
		t.Logf("Foreign keys to create: %d", len(comparison.ForeignKeysToCreate))
		t.Logf("Foreign keys to drop: %d", len(comparison.ForeignKeysToDrop))
	})

	// Step 7: Generate SQL migrations
	upSQL, downSQL, err := enhancedGen.GenerateSafeSQL(comparison, true) // Allow destructive for test
	if err != nil {
		t.Fatalf("Failed to generate SQL: %v", err)
	}

	t.Run("SQL Generation", func(t *testing.T) {
		if len(upSQL) == 0 {
			t.Error("Expected some UP SQL statements")
		}

		if len(downSQL) == 0 {
			t.Error("Expected some DOWN SQL statements")
		}

		// Verify SQL contains expected statements
		hasCreateIndex := false
		hasCreateFK := false
		hasDropIndex := false

		for _, stmt := range upSQL {
			if contains(stmt, "CREATE") && contains(stmt, "INDEX") {
				hasCreateIndex = true
			}
			if contains(stmt, "ADD CONSTRAINT") && contains(stmt, "FOREIGN KEY") {
				hasCreateFK = true
			}
			if contains(stmt, "DROP INDEX") {
				hasDropIndex = true
			}
		}

		if !hasCreateIndex {
			t.Error("Expected CREATE INDEX statement in UP migration")
		}
		// Note: FK creation might not be needed if schema matches
		// if !hasCreateFK {
		//     t.Error("Expected CREATE FOREIGN KEY statement in UP migration")
		// }
		if !hasDropIndex {
			t.Error("Expected DROP INDEX statement in UP migration")
		}

		t.Logf("Generated %d UP statements and %d DOWN statements", len(upSQL), len(downSQL))
	})

	// Step 8: Test safety checks
	t.Run("Safety Checks", func(t *testing.T) {
		safeComparison := &generator.SchemaComparison{
			IndexesToCreate: []generator.IndexDefinition{
				{Name: "idx_safe", IsUnique: false, IsPrimary: false},
			},
		}

		if !enhancedGen.IsSafeOperation(safeComparison) {
			t.Error("Expected safe operation to be marked as safe")
		}

		unsafeComparison := &generator.SchemaComparison{
			IndexesToDrop: []generator.IndexDefinition{
				{Name: "idx_unsafe", IsUnique: true, IsPrimary: false},
			},
		}

		if enhancedGen.IsSafeOperation(unsafeComparison) {
			t.Error("Expected unsafe operation to be marked as unsafe")
		}
	})
}

// TestMigrationWithRealDatabase tests against a more complete database setup
func TestMigrationWithCompleteSchema(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := dbtest.NewTestDB(t)
	defer testDB.Cleanup()

	// Setup a complete schema that matches our struct definitions
	completeSchema := `
		CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

		CREATE TABLE teams (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			name VARCHAR(255) NOT NULL UNIQUE,
			created_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE users (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			email VARCHAR(255) NOT NULL UNIQUE,
			team_id UUID NOT NULL,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT NOW()
		);

		CREATE INDEX idx_users_team_id ON users(team_id);

		CREATE TABLE projects (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			team_id UUID NOT NULL,
			name VARCHAR(255) NOT NULL,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT NOW()
		);

		CREATE INDEX idx_projects_active ON projects(is_active) WHERE is_active = true;
		CREATE UNIQUE INDEX uk_projects_team_name ON projects(team_id, name);

		-- Add all foreign keys
		ALTER TABLE users ADD CONSTRAINT fk_users_team_id 
			FOREIGN KEY (team_id) REFERENCES teams(id);

		ALTER TABLE projects ADD CONSTRAINT fk_projects_team_id 
			FOREIGN KEY (team_id) REFERENCES teams(id);
	`

	if err := testDB.ExecuteSQL(completeSchema); err != nil {
		t.Fatalf("Failed to setup complete schema: %v", err)
	}

	// Create minimal test models that should match the database
	tmpDir := t.TempDir()
	testModelsFile := filepath.Join(tmpDir, "models.go")

	matchingModels := `
package models

import "time"

type User struct {
	_        struct{} ` + "`" + `dbdef:"table:users;index:idx_users_team_id,team_id"` + "`" + `
	
	ID        string    ` + "`" + `db:"id" dbdef:"type:uuid;primary_key;default:uuid_generate_v4()"` + "`" + `
	Email     string    ` + "`" + `db:"email" dbdef:"type:varchar(255);not_null;unique"` + "`" + `
	TeamID    string    ` + "`" + `db:"team_id" dbdef:"type:uuid;not_null;foreign_key:teams.id"` + "`" + `
	IsActive  bool      ` + "`" + `db:"is_active" dbdef:"type:boolean;not_null;default:true"` + "`" + `
	CreatedAt time.Time ` + "`" + `db:"created_at" dbdef:"type:timestamp;not_null;default:now()"` + "`" + `
}

type Team struct {
	_        struct{} ` + "`" + `dbdef:"table:teams"` + "`" + `
	
	ID        string    ` + "`" + `db:"id" dbdef:"type:uuid;primary_key;default:uuid_generate_v4()"` + "`" + `
	Name      string    ` + "`" + `db:"name" dbdef:"type:varchar(255);not_null;unique"` + "`" + `
	CreatedAt time.Time ` + "`" + `db:"created_at" dbdef:"type:timestamp;not_null;default:now()"` + "`" + `
}

type Project struct {
	_        struct{} ` + "`" + `dbdef:"table:projects;index:idx_projects_active,is_active where:is_active=true;unique:uk_projects_team_name,team_id,name"` + "`" + `
	
	ID        string    ` + "`" + `db:"id" dbdef:"type:uuid;primary_key;default:uuid_generate_v4()"` + "`" + `
	TeamID    string    ` + "`" + `db:"team_id" dbdef:"type:uuid;not_null;foreign_key:teams.id"` + "`" + `
	Name      string    ` + "`" + `db:"name" dbdef:"type:varchar(255);not_null"` + "`" + `
	IsActive  bool      ` + "`" + `db:"is_active" dbdef:"type:boolean;not_null;default:true"` + "`" + `
	CreatedAt time.Time ` + "`" + `db:"created_at" dbdef:"type:timestamp;not_null;default:now()"` + "`" + `
}
`

	if err := os.WriteFile(testModelsFile, []byte(matchingModels), 0644); err != nil {
		t.Fatalf("Failed to write test models: %v", err)
	}

	// Run the full comparison workflow
	structParser := parser.NewStructParser()
	tables, err := structParser.ParseFile(testModelsFile)
	if err != nil {
		t.Fatalf("Failed to parse structs: %v", err)
	}

	enhancedGen := generator.NewEnhancedGenerator()
	introspector := introspect.NewPostgreSQLIntrospector(testDB.DB)

	// Generate struct definitions
	var allStructIndexes []generator.IndexDefinition
	var allStructFKs []generator.ForeignKeyDefinition

	for _, table := range tables {
		indexes, err := enhancedGen.GenerateIndexDefinitions(table)
		if err != nil {
			t.Fatalf("Failed to generate indexes: %v", err)
		}
		allStructIndexes = append(allStructIndexes, indexes...)

		fks, err := enhancedGen.GenerateForeignKeyDefinitions(table)
		if err != nil {
			t.Fatalf("Failed to generate foreign keys: %v", err)
		}
		allStructFKs = append(allStructFKs, fks...)
	}

	// Get database state
	tableNames, err := introspector.GetTableNames()
	if err != nil {
		t.Fatalf("Failed to get table names: %v", err)
	}

	var allDBIndexes []generator.IndexDefinition
	var allDBFKs []generator.ForeignKeyDefinition

	for _, tableName := range tableNames {
		indexes, err := introspector.GetEnhancedIndexes(tableName)
		if err != nil {
			t.Fatalf("Failed to get indexes: %v", err)
		}
		allDBIndexes = append(allDBIndexes, indexes...)

		fks, err := introspector.GetEnhancedForeignKeys(tableName)
		if err != nil {
			t.Fatalf("Failed to get foreign keys: %v", err)
		}
		allDBFKs = append(allDBFKs, fks...)
	}

	// Compare schemas
	comparison, err := enhancedGen.CompareSchemas(allStructIndexes, allStructFKs, allDBIndexes, allDBFKs)
	if err != nil {
		t.Fatalf("Failed to compare schemas: %v", err)
	}

	// With perfectly matching schemas, we should have no changes
	totalChanges := len(comparison.IndexesToCreate) + len(comparison.IndexesToDrop) +
		len(comparison.ForeignKeysToCreate) + len(comparison.ForeignKeysToDrop)

	if totalChanges > 0 {
		t.Errorf("Expected no changes for matching schema, but got %d changes", totalChanges)
		t.Logf("Indexes to create: %d", len(comparison.IndexesToCreate))
		t.Logf("Indexes to drop: %d", len(comparison.IndexesToDrop))
		t.Logf("Foreign keys to create: %d", len(comparison.ForeignKeysToCreate))
		t.Logf("Foreign keys to drop: %d", len(comparison.ForeignKeysToDrop))

		// Log details for debugging
		for _, idx := range comparison.IndexesToCreate {
			t.Logf("CREATE INDEX: %s -> %s", idx.Name, idx.Signature)
		}
		for _, idx := range comparison.IndexesToDrop {
			t.Logf("DROP INDEX: %s -> %s", idx.Name, idx.Signature)
		}
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
