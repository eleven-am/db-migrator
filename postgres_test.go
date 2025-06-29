package introspect

import (
	"database/sql"
	"testing"

	"github.com/eleven-am/db-migrator/generator"
	dbtest "github.com/eleven-am/db-migrator/testing"

	_ "github.com/lib/pq"
)

func TestPostgreSQLIntrospector_GetEnhancedIndexes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := dbtest.NewTestDB(t)
	defer testDB.Cleanup()

	// Setup test schema
	setupSQL := `
		CREATE TABLE users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) NOT NULL UNIQUE,
			team_id UUID NOT NULL,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT NOW()
		);

		CREATE INDEX idx_users_team_id ON users(team_id);
		CREATE INDEX idx_users_active ON users(is_active) WHERE is_active = true;
		CREATE UNIQUE INDEX uk_users_team_email ON users(team_id, email);

		CREATE TABLE teams (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL UNIQUE,
			created_at TIMESTAMP DEFAULT NOW()
		);

		ALTER TABLE users ADD CONSTRAINT fk_users_team_id 
			FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;
	`

	if err := testDB.ExecuteSQL(setupSQL); err != nil {
		t.Fatalf("Failed to setup test schema: %v", err)
	}

	introspector := NewPostgreSQLIntrospector(testDB.DB)

	tests := []struct {
		name      string
		tableName string
		validate  func([]generator.IndexDefinition) error
	}{
		{
			name:      "users table indexes",
			tableName: "users",
			validate: func(indexes []generator.IndexDefinition) error {
				expectedIndexes := map[string]struct {
					isUnique  bool
					isPrimary bool
					columns   []string
					hasWhere  bool
				}{
					"users_pkey":           {isUnique: true, isPrimary: true, columns: []string{"id"}},
					"users_email_key":      {isUnique: true, isPrimary: false, columns: []string{"email"}},
					"idx_users_team_id":    {isUnique: false, isPrimary: false, columns: []string{"team_id"}},
					"idx_users_active":     {isUnique: false, isPrimary: false, columns: []string{"is_active"}, hasWhere: true},
					"uk_users_team_email":  {isUnique: true, isPrimary: false, columns: []string{"team_id", "email"}},
				}

				if len(indexes) != len(expectedIndexes) {
					t.Errorf("Expected %d indexes, got %d", len(expectedIndexes), len(indexes))
					for _, idx := range indexes {
						t.Logf("Found index: %s (unique: %v, primary: %v, columns: %v, where: %s)", 
							idx.Name, idx.IsUnique, idx.IsPrimary, idx.Columns, idx.Where)
					}
				}

				for _, idx := range indexes {
					expected, exists := expectedIndexes[idx.Name]
					if !exists {
						t.Errorf("Unexpected index: %s", idx.Name)
						continue
					}

					if idx.IsUnique != expected.isUnique {
						t.Errorf("Index %s: expected IsUnique=%v, got %v", idx.Name, expected.isUnique, idx.IsUnique)
					}

					if idx.IsPrimary != expected.isPrimary {
						t.Errorf("Index %s: expected IsPrimary=%v, got %v", idx.Name, expected.isPrimary, idx.IsPrimary)
					}

					if len(idx.Columns) != len(expected.columns) {
						t.Errorf("Index %s: expected %d columns, got %d", idx.Name, len(expected.columns), len(idx.Columns))
					} else {
						for i, col := range expected.columns {
							if idx.Columns[i] != col {
								t.Errorf("Index %s: expected column %d to be %s, got %s", idx.Name, i, col, idx.Columns[i])
							}
						}
					}

					if expected.hasWhere && idx.Where == "" {
						t.Errorf("Index %s: expected WHERE clause but got none", idx.Name)
					}

					if !expected.hasWhere && idx.Where != "" {
						t.Errorf("Index %s: expected no WHERE clause but got: %s", idx.Name, idx.Where)
					}

					// Verify signature is generated
					if idx.Signature == "" {
						t.Errorf("Index %s: missing signature", idx.Name)
					}
				}

				return nil
			},
		},
		{
			name:      "teams table indexes",
			tableName: "teams",
			validate: func(indexes []generator.IndexDefinition) error {
				expectedCount := 2 // primary key + unique name

				if len(indexes) != expectedCount {
					t.Errorf("Expected %d indexes for teams table, got %d", expectedCount, len(indexes))
				}

				hasPK := false
				hasUnique := false
				for _, idx := range indexes {
					if idx.IsPrimary {
						hasPK = true
					}
					if idx.IsUnique && !idx.IsPrimary && idx.Name == "teams_name_key" {
						hasUnique = true
					}
				}

				if !hasPK {
					t.Error("Missing primary key index for teams table")
				}
				if !hasUnique {
					t.Error("Missing unique name index for teams table")
				}

				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indexes, err := introspector.GetEnhancedIndexes(tt.tableName)
			if err != nil {
				t.Fatalf("GetEnhancedIndexes() error = %v", err)
			}

			if err := tt.validate(indexes); err != nil {
				t.Errorf("Validation failed: %v", err)
			}
		})
	}
}

func TestPostgreSQLIntrospector_GetEnhancedForeignKeys(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := dbtest.NewTestDB(t)
	defer testDB.Cleanup()

	// Setup test schema with foreign keys
	setupSQL := `
		CREATE TABLE teams (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL
		);

		CREATE TABLE users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			team_id UUID NOT NULL
		);

		CREATE TABLE projects (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			team_id UUID NOT NULL,
			owner_id UUID
		);

		-- Add foreign keys with different actions
		ALTER TABLE users ADD CONSTRAINT fk_users_team_id 
			FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;

		ALTER TABLE projects ADD CONSTRAINT fk_projects_team_id 
			FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE RESTRICT;

		ALTER TABLE projects ADD CONSTRAINT fk_projects_owner_id 
			FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE SET NULL;
	`

	if err := testDB.ExecuteSQL(setupSQL); err != nil {
		t.Fatalf("Failed to setup test schema: %v", err)
	}

	introspector := NewPostgreSQLIntrospector(testDB.DB)

	tests := []struct {
		name      string
		tableName string
		validate  func([]generator.ForeignKeyDefinition) error
	}{
		{
			name:      "users table foreign keys",
			tableName: "users",
			validate: func(fks []generator.ForeignKeyDefinition) error {
				if len(fks) != 1 {
					t.Errorf("Expected 1 foreign key, got %d", len(fks))
					return nil
				}

				fk := fks[0]
				if fk.Name != "fk_users_team_id" {
					t.Errorf("Expected FK name 'fk_users_team_id', got '%s'", fk.Name)
				}

				if fk.ReferencedTable != "teams" {
					t.Errorf("Expected referenced table 'teams', got '%s'", fk.ReferencedTable)
				}

				if len(fk.Columns) != 1 || fk.Columns[0] != "team_id" {
					t.Errorf("Expected columns [team_id], got %v", fk.Columns)
				}

				if len(fk.ReferencedColumns) != 1 || fk.ReferencedColumns[0] != "id" {
					t.Errorf("Expected referenced columns [id], got %v", fk.ReferencedColumns)
				}

				if fk.OnDelete != "CASCADE" {
					t.Errorf("Expected OnDelete 'CASCADE', got '%s'", fk.OnDelete)
				}

				// Verify signature is generated
				if fk.Signature == "" {
					t.Error("Missing foreign key signature")
				}

				return nil
			},
		},
		{
			name:      "projects table foreign keys",
			tableName: "projects",
			validate: func(fks []generator.ForeignKeyDefinition) error {
				if len(fks) != 2 {
					t.Errorf("Expected 2 foreign keys, got %d", len(fks))
					return nil
				}

				var teamFK, ownerFK *generator.ForeignKeyDefinition
				for i := range fks {
					if fks[i].Name == "fk_projects_team_id" {
						teamFK = &fks[i]
					} else if fks[i].Name == "fk_projects_owner_id" {
						ownerFK = &fks[i]
					}
				}

				if teamFK == nil {
					t.Error("Missing team foreign key")
				} else {
					if teamFK.OnDelete != "RESTRICT" {
						t.Errorf("Expected team FK OnDelete 'RESTRICT', got '%s'", teamFK.OnDelete)
					}
				}

				if ownerFK == nil {
					t.Error("Missing owner foreign key")
				} else {
					if ownerFK.OnDelete != "SET NULL" {
						t.Errorf("Expected owner FK OnDelete 'SET NULL', got '%s'", ownerFK.OnDelete)
					}
				}

				return nil
			},
		},
		{
			name:      "teams table (no foreign keys)",
			tableName: "teams",
			validate: func(fks []generator.ForeignKeyDefinition) error {
				if len(fks) != 0 {
					t.Errorf("Expected 0 foreign keys for teams table, got %d", len(fks))
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fks, err := introspector.GetEnhancedForeignKeys(tt.tableName)
			if err != nil {
				t.Fatalf("GetEnhancedForeignKeys() error = %v", err)
			}

			if err := tt.validate(fks); err != nil {
				t.Errorf("Validation failed: %v", err)
			}
		})
	}
}

func TestPostgreSQLIntrospector_GetTableNames(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := dbtest.NewTestDB(t)
	defer testDB.Cleanup()

	// Create test tables
	setupSQL := `
		CREATE TABLE users (id UUID PRIMARY KEY);
		CREATE TABLE teams (id UUID PRIMARY KEY);
		CREATE TABLE projects (id UUID PRIMARY KEY);
		CREATE TABLE temp_table (id UUID PRIMARY KEY);
	`

	if err := testDB.ExecuteSQL(setupSQL); err != nil {
		t.Fatalf("Failed to setup test schema: %v", err)
	}

	introspector := NewPostgreSQLIntrospector(testDB.DB)

	tableNames, err := introspector.GetTableNames()
	if err != nil {
		t.Fatalf("GetTableNames() error = %v", err)
	}

	expectedTables := []string{"users", "teams", "projects", "temp_table"}
	
	if len(tableNames) < len(expectedTables) {
		t.Errorf("Expected at least %d tables, got %d", len(expectedTables), len(tableNames))
	}

	// Check that all expected tables are present
	tableMap := make(map[string]bool)
	for _, name := range tableNames {
		tableMap[name] = true
	}

	for _, expected := range expectedTables {
		if !tableMap[expected] {
			t.Errorf("Expected table '%s' not found in results", expected)
		}
	}
}

func TestPostgreSQLIntrospector_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test with invalid database connection
	invalidDB, err := sql.Open("postgres", "postgres://invalid:invalid@localhost/nonexistent?sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to create invalid connection: %v", err)
	}
	defer invalidDB.Close()

	introspector := NewPostgreSQLIntrospector(invalidDB)

	// Test error handling for non-existent table
	testDB := dbtest.NewTestDB(t)
	defer testDB.Cleanup()

	validIntrospector := NewPostgreSQLIntrospector(testDB.DB)

	_, err = validIntrospector.GetEnhancedIndexes("nonexistent_table")
	if err == nil {
		t.Error("Expected error for non-existent table, got none")
	}

	_, err = validIntrospector.GetEnhancedForeignKeys("nonexistent_table")
	if err == nil {
		t.Error("Expected error for non-existent table, got none")
	}
}

func TestPostgreSQLIntrospector_ComplexIndexes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := dbtest.NewTestDB(t)
	defer testDB.Cleanup()

	// Setup complex indexes
	setupSQL := `
		CREATE TABLE orders (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL,
			status VARCHAR(50) NOT NULL,
			priority INTEGER DEFAULT 0,
			is_active BOOLEAN DEFAULT true,
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMP DEFAULT NOW()
		);

		-- Various index types
		CREATE INDEX idx_orders_user_id ON orders(user_id);
		CREATE INDEX idx_orders_status_priority ON orders(status, priority);
		CREATE UNIQUE INDEX uk_orders_user_status ON orders(user_id, status) WHERE is_active = true;
		CREATE INDEX idx_orders_metadata_gin ON orders USING gin(metadata);
		CREATE INDEX idx_orders_created_btree ON orders USING btree(created_at);
		CREATE INDEX idx_orders_active_partial ON orders(status) WHERE is_active = true AND status != 'deleted';
	`

	if err := testDB.ExecuteSQL(setupSQL); err != nil {
		t.Fatalf("Failed to setup test schema: %v", err)
	}

	introspector := NewPostgreSQLIntrospector(testDB.DB)

	indexes, err := introspector.GetEnhancedIndexes("orders")
	if err != nil {
		t.Fatalf("GetEnhancedIndexes() error = %v", err)
	}

	// Verify complex index properties
	indexMap := make(map[string]generator.IndexDefinition)
	for _, idx := range indexes {
		indexMap[idx.Name] = idx
	}

	// Check composite index
	if composite, exists := indexMap["idx_orders_status_priority"]; exists {
		expectedCols := []string{"status", "priority"}
		if len(composite.Columns) != len(expectedCols) {
			t.Errorf("Expected composite index to have %d columns, got %d", len(expectedCols), len(composite.Columns))
		}
		for i, col := range expectedCols {
			if i < len(composite.Columns) && composite.Columns[i] != col {
				t.Errorf("Expected column %d to be %s, got %s", i, col, composite.Columns[i])
			}
		}
	} else {
		t.Error("Missing composite index idx_orders_status_priority")
	}

	// Check partial unique index
	if partialUnique, exists := indexMap["uk_orders_user_status"]; exists {
		if !partialUnique.IsUnique {
			t.Error("Expected uk_orders_user_status to be unique")
		}
		if partialUnique.Where == "" {
			t.Error("Expected uk_orders_user_status to have WHERE clause")
		}
	} else {
		t.Error("Missing partial unique index uk_orders_user_status")
	}

	// Check GIN index
	if ginIndex, exists := indexMap["idx_orders_metadata_gin"]; exists {
		if ginIndex.Method != "gin" {
			t.Errorf("Expected GIN index method, got %s", ginIndex.Method)
		}
	} else {
		t.Error("Missing GIN index idx_orders_metadata_gin")
	}

	// Check complex partial index
	if complexPartial, exists := indexMap["idx_orders_active_partial"]; exists {
		if complexPartial.Where == "" {
			t.Error("Expected complex partial index to have WHERE clause")
		}
		// The WHERE clause should be normalized
		if !contains(complexPartial.Where, "is_active") || !contains(complexPartial.Where, "status") {
			t.Errorf("Expected WHERE clause to contain is_active and status, got: %s", complexPartial.Where)
		}
	} else {
		t.Error("Missing complex partial index idx_orders_active_partial")
	}
}

// Helper function for string contains check
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

// Benchmark tests
func BenchmarkPostgreSQLIntrospector_GetEnhancedIndexes(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	testDB := dbtest.NewTestDB(&testing.T{})
	defer testDB.Cleanup()

	// Setup test table with multiple indexes
	setupSQL := `
		CREATE TABLE benchmark_table (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) UNIQUE,
			team_id UUID,
			status VARCHAR(50),
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT NOW()
		);

		CREATE INDEX idx_benchmark_team_id ON benchmark_table(team_id);
		CREATE INDEX idx_benchmark_status ON benchmark_table(status);
		CREATE INDEX idx_benchmark_active ON benchmark_table(is_active) WHERE is_active = true;
		CREATE INDEX idx_benchmark_composite ON benchmark_table(team_id, status, created_at);
	`

	if err := testDB.ExecuteSQL(setupSQL); err != nil {
		b.Fatalf("Failed to setup benchmark schema: %v", err)
	}

	introspector := NewPostgreSQLIntrospector(testDB.DB)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := introspector.GetEnhancedIndexes("benchmark_table")
		if err != nil {
			b.Fatal(err)
		}
	}
}