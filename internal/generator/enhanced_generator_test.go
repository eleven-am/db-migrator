package generator

import (
	"github.com/eleven-am/db-migrator/internal/parser"
	"reflect"
	"testing"
)

func TestEnhancedGenerator_GenerateIndexDefinitions(t *testing.T) {
	generator := NewEnhancedGenerator()

	tests := []struct {
		name         string
		tableDefn    parser.TableDefinition
		expectedLen  int
		validateFunc func([]IndexDefinition) error
	}{
		{
			name: "simple table with primary key",
			tableDefn: parser.TableDefinition{
				StructName: "User",
				TableName:  "users",
				Fields: []parser.FieldDefinition{
					{
						Name:   "ID",
						DBName: "id",
						DBDef: map[string]string{
							"type":        "cuid",
							"primary_key": "",
						},
					},
				},
				TableLevel: map[string]string{},
			},
			expectedLen: 1,
			validateFunc: func(indexes []IndexDefinition) error {
				idx := indexes[0]
				if idx.Name != "users_pkey" {
					t.Errorf("Expected primary key name 'users_pkey', got '%s'", idx.Name)
				}
				if !idx.IsPrimary || !idx.IsUnique {
					t.Error("Primary key should be both primary and unique")
				}
				if len(idx.Columns) != 1 || idx.Columns[0] != "id" {
					t.Errorf("Expected columns [id], got %v", idx.Columns)
				}
				return nil
			},
		},
		{
			name: "table with field-level unique constraint",
			tableDefn: parser.TableDefinition{
				StructName: "User",
				TableName:  "users",
				Fields: []parser.FieldDefinition{
					{
						Name:   "ID",
						DBName: "id",
						DBDef: map[string]string{
							"type":        "cuid",
							"primary_key": "",
						},
					},
					{
						Name:   "Email",
						DBName: "email",
						DBDef: map[string]string{
							"type":   "varchar(255)",
							"unique": "",
						},
					},
				},
				TableLevel: map[string]string{},
			},
			expectedLen: 2,
			validateFunc: func(indexes []IndexDefinition) error {
				// Should have primary key + unique constraint
				var pkIndex, uniqueIndex *IndexDefinition
				for i := range indexes {
					if indexes[i].IsPrimary {
						pkIndex = &indexes[i]
					} else if indexes[i].IsUnique {
						uniqueIndex = &indexes[i]
					}
				}

				if pkIndex == nil {
					t.Error("Primary key index not found")
				}
				if uniqueIndex == nil {
					t.Error("Unique index not found")
				}

				if uniqueIndex != nil {
					if uniqueIndex.Name != "idx_users_email" {
						t.Errorf("Expected unique index name 'idx_users_email', got '%s'", uniqueIndex.Name)
					}
					if len(uniqueIndex.Columns) != 1 || uniqueIndex.Columns[0] != "email" {
						t.Errorf("Expected columns [email], got %v", uniqueIndex.Columns)
					}
				}
				return nil
			},
		},
		{
			name: "table with table-level indexes",
			tableDefn: parser.TableDefinition{
				StructName: "AuditLog",
				TableName:  "audit_logs",
				Fields: []parser.FieldDefinition{
					{
						Name:   "ID",
						DBName: "id",
						DBDef: map[string]string{
							"type":        "cuid",
							"primary_key": "",
						},
					},
					{
						Name:   "EntityType",
						DBName: "entity_type",
						DBDef: map[string]string{
							"type": "varchar(50)",
						},
					},
					{
						Name:   "EntityID",
						DBName: "entity_id",
						DBDef: map[string]string{
							"type": "cuid",
						},
					},
				},
				TableLevel: map[string]string{
					"index":  "idx_audit_logs_entity,entity_type,entity_id",
					"unique": "uk_audit_logs_unique,entity_type,entity_id",
				},
			},
			expectedLen: 3, // primary key + index + unique
			validateFunc: func(indexes []IndexDefinition) error {
				var pkIndex, regIndex, uniqueIndex *IndexDefinition
				for i := range indexes {
					if indexes[i].IsPrimary {
						pkIndex = &indexes[i]
					} else if indexes[i].IsUnique && indexes[i].Name == "uk_audit_logs_unique" {
						uniqueIndex = &indexes[i]
					} else if !indexes[i].IsUnique && indexes[i].Name == "idx_audit_logs_entity" {
						regIndex = &indexes[i]
					}
				}

				if pkIndex == nil {
					t.Error("Primary key index not found")
				}
				if regIndex == nil {
					t.Error("Regular index not found")
				}
				if uniqueIndex == nil {
					t.Error("Unique index not found")
				}

				if regIndex != nil {
					expectedCols := []string{"entity_type", "entity_id"}
					if !reflect.DeepEqual(regIndex.Columns, expectedCols) {
						t.Errorf("Expected regular index columns %v, got %v", expectedCols, regIndex.Columns)
					}
				}

				if uniqueIndex != nil {
					expectedCols := []string{"entity_type", "entity_id"}
					if !reflect.DeepEqual(uniqueIndex.Columns, expectedCols) {
						t.Errorf("Expected unique index columns %v, got %v", expectedCols, uniqueIndex.Columns)
					}
				}
				return nil
			},
		},
		{
			name: "table with partial index",
			tableDefn: parser.TableDefinition{
				StructName: "Order",
				TableName:  "orders",
				Fields: []parser.FieldDefinition{
					{
						Name:   "ID",
						DBName: "id",
						DBDef: map[string]string{
							"type":        "cuid",
							"primary_key": "",
						},
					},
					{
						Name:   "IsActive",
						DBName: "is_active",
						DBDef: map[string]string{
							"type": "boolean",
						},
					},
				},
				TableLevel: map[string]string{
					"index": "idx_orders_active,is_active where:is_active=true",
				},
			},
			expectedLen: 2, // primary key + partial index
			validateFunc: func(indexes []IndexDefinition) error {
				var partialIndex *IndexDefinition
				for i := range indexes {
					if !indexes[i].IsPrimary {
						partialIndex = &indexes[i]
						break
					}
				}

				if partialIndex == nil {
					t.Error("Partial index not found")
				} else {
					if partialIndex.Where != "is_active=true" {
						t.Errorf("Expected WHERE clause 'is_active=true', got '%s'", partialIndex.Where)
					}
					if len(partialIndex.Columns) != 1 || partialIndex.Columns[0] != "is_active" {
						t.Errorf("Expected columns [is_active], got %v", partialIndex.Columns)
					}
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indexes, err := generator.GenerateIndexDefinitions(tt.tableDefn)
			if err != nil {
				t.Fatalf("GenerateIndexDefinitions() error = %v", err)
			}

			if len(indexes) != tt.expectedLen {
				t.Errorf("Expected %d indexes, got %d", tt.expectedLen, len(indexes))
			}

			if tt.validateFunc != nil {
				if err := tt.validateFunc(indexes); err != nil {
					t.Errorf("Validation failed: %v", err)
				}
			}

			// Verify all indexes have signatures
			for i, idx := range indexes {
				if idx.Signature == "" {
					t.Errorf("Index %d missing signature", i)
				}
			}
		})
	}
}

func TestEnhancedGenerator_parseTableLevelIndex(t *testing.T) {
	generator := NewEnhancedGenerator()

	tests := []struct {
		name        string
		tableName   string
		indexDefStr string
		isUnique    bool
		expected    IndexDefinition
		expectError bool
	}{
		{
			name:        "simple index",
			tableName:   "users",
			indexDefStr: "idx_users_email,email",
			isUnique:    false,
			expected: IndexDefinition{
				Name:      "idx_users_email",
				TableName: "users",
				Columns:   []string{"email"},
				IsUnique:  false,
				Method:    "btree",
				Where:     "",
			},
		},
		{
			name:        "unique constraint",
			tableName:   "users",
			indexDefStr: "uk_users_email,email",
			isUnique:    true,
			expected: IndexDefinition{
				Name:      "uk_users_email",
				TableName: "users",
				Columns:   []string{"email"},
				IsUnique:  true,
				Method:    "btree",
				Where:     "",
			},
		},
		{
			name:        "composite index",
			tableName:   "audit_logs",
			indexDefStr: "idx_audit_entity,entity_type,entity_id",
			isUnique:    false,
			expected: IndexDefinition{
				Name:      "idx_audit_entity",
				TableName: "audit_logs",
				Columns:   []string{"entity_type", "entity_id"},
				IsUnique:  false,
				Method:    "btree",
				Where:     "",
			},
		},
		{
			name:        "partial index",
			tableName:   "orders",
			indexDefStr: "idx_orders_active,is_active where:is_active=true",
			isUnique:    false,
			expected: IndexDefinition{
				Name:      "idx_orders_active",
				TableName: "orders",
				Columns:   []string{"is_active"},
				IsUnique:  false,
				Method:    "btree",
				Where:     "is_active=true",
			},
		},
		{
			name:        "partial index with spaces",
			tableName:   "products",
			indexDefStr: "idx_products_category,category where:category IS NOT NULL",
			isUnique:    false,
			expected: IndexDefinition{
				Name:      "idx_products_category",
				TableName: "products",
				Columns:   []string{"category"},
				IsUnique:  false,
				Method:    "btree",
				Where:     "category IS NOT NULL",
			},
		},
		{
			name:        "malformed - no columns",
			tableName:   "users",
			indexDefStr: "idx_users_empty",
			isUnique:    false,
			expectError: true,
		},
		{
			name:        "malformed - empty name",
			tableName:   "users",
			indexDefStr: ",email",
			isUnique:    false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := generator.parseTableLevelIndex(tt.tableName, tt.indexDefStr, tt.isUnique)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.Name != tt.expected.Name {
				t.Errorf("Expected name '%s', got '%s'", tt.expected.Name, result.Name)
			}
			if result.TableName != tt.expected.TableName {
				t.Errorf("Expected table '%s', got '%s'", tt.expected.TableName, result.TableName)
			}
			if !reflect.DeepEqual(result.Columns, tt.expected.Columns) {
				t.Errorf("Expected columns %v, got %v", tt.expected.Columns, result.Columns)
			}
			if result.IsUnique != tt.expected.IsUnique {
				t.Errorf("Expected IsUnique %v, got %v", tt.expected.IsUnique, result.IsUnique)
			}
			if result.Method != tt.expected.Method {
				t.Errorf("Expected method '%s', got '%s'", tt.expected.Method, result.Method)
			}
			if result.Where != tt.expected.Where {
				t.Errorf("Expected where '%s', got '%s'", tt.expected.Where, result.Where)
			}
		})
	}
}

func TestEnhancedGenerator_GenerateForeignKeyDefinitions(t *testing.T) {
	generator := NewEnhancedGenerator()

	tests := []struct {
		name         string
		tableDefn    parser.TableDefinition
		expectedLen  int
		validateFunc func([]ForeignKeyDefinition) error
	}{
		{
			name: "table with foreign key",
			tableDefn: parser.TableDefinition{
				StructName: "Project",
				TableName:  "projects",
				Fields: []parser.FieldDefinition{
					{
						Name:   "ID",
						DBName: "id",
						DBDef: map[string]string{
							"type":        "cuid",
							"primary_key": "",
						},
					},
					{
						Name:   "TeamID",
						DBName: "team_id",
						DBDef: map[string]string{
							"type":        "cuid",
							"foreign_key": "teams.id",
						},
					},
				},
			},
			expectedLen: 1,
			validateFunc: func(fks []ForeignKeyDefinition) error {
				fk := fks[0]
				if fk.Name != "projects_team_id_fkey" {
					t.Errorf("Expected FK name 'projects_team_id_fkey', got '%s'", fk.Name)
				}
				if fk.ReferencedTable != "teams" {
					t.Errorf("Expected referenced table 'teams', got '%s'", fk.ReferencedTable)
				}
				if len(fk.ReferencedColumns) != 1 || fk.ReferencedColumns[0] != "id" {
					t.Errorf("Expected referenced columns [id], got %v", fk.ReferencedColumns)
				}
				if fk.OnDelete != "NO ACTION" {
					t.Errorf("Expected OnDelete 'NO ACTION', got '%s'", fk.OnDelete)
				}
				return nil
			},
		},
		{
			name: "table with foreign key and actions",
			tableDefn: parser.TableDefinition{
				StructName: "Pipeline",
				TableName:  "pipelines",
				Fields: []parser.FieldDefinition{
					{
						Name:   "ProjectID",
						DBName: "project_id",
						DBDef: map[string]string{
							"type":        "cuid",
							"foreign_key": "projects.id",
							"on_delete":   "CASCADE",
							"on_update":   "RESTRICT",
						},
					},
				},
			},
			expectedLen: 1,
			validateFunc: func(fks []ForeignKeyDefinition) error {
				fk := fks[0]
				if fk.OnDelete != "CASCADE" {
					t.Errorf("Expected OnDelete 'CASCADE', got '%s'", fk.OnDelete)
				}
				if fk.OnUpdate != "RESTRICT" {
					t.Errorf("Expected OnUpdate 'RESTRICT', got '%s'", fk.OnUpdate)
				}
				return nil
			},
		},
		{
			name: "table with no foreign keys",
			tableDefn: parser.TableDefinition{
				StructName: "User",
				TableName:  "users",
				Fields: []parser.FieldDefinition{
					{
						Name:   "ID",
						DBName: "id",
						DBDef: map[string]string{
							"type":        "cuid",
							"primary_key": "",
						},
					},
					{
						Name:   "Email",
						DBName: "email",
						DBDef: map[string]string{
							"type": "varchar(255)",
						},
					},
				},
			},
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fks, err := generator.GenerateForeignKeyDefinitions(tt.tableDefn)
			if err != nil {
				t.Fatalf("GenerateForeignKeyDefinitions() error = %v", err)
			}

			if len(fks) != tt.expectedLen {
				t.Errorf("Expected %d foreign keys, got %d", tt.expectedLen, len(fks))
			}

			if tt.validateFunc != nil {
				if err := tt.validateFunc(fks); err != nil {
					t.Errorf("Validation failed: %v", err)
				}
			}

			// Verify all foreign keys have signatures
			for i, fk := range fks {
				if fk.Signature == "" {
					t.Errorf("Foreign key %d missing signature", i)
				}
			}
		})
	}
}

func TestEnhancedGenerator_CompareSchemas(t *testing.T) {
	generator := NewEnhancedGenerator()

	// Create test data
	structIndexes := []IndexDefinition{
		{
			Name:      "users_pkey",
			TableName: "users",
			Columns:   []string{"id"},
			IsUnique:  true,
			IsPrimary: true,
			Method:    "btree",
			Signature: "table:users|cols:id|primary:true|unique:true|method:btree",
		},
		{
			Name:      "idx_users_email",
			TableName: "users",
			Columns:   []string{"email"},
			IsUnique:  true,
			IsPrimary: false,
			Method:    "btree",
			Signature: "table:users|cols:email|unique:true|method:btree",
		},
		{
			Name:      "idx_users_team_id",
			TableName: "users",
			Columns:   []string{"team_id"},
			IsUnique:  false,
			IsPrimary: false,
			Method:    "btree",
			Signature: "table:users|cols:team_id|method:btree",
		},
	}

	dbIndexes := []IndexDefinition{
		{
			Name:      "users_pkey",
			TableName: "users",
			Columns:   []string{"id"},
			IsUnique:  true,
			IsPrimary: true,
			Method:    "btree",
			Signature: "table:users|cols:id|primary:true|unique:true|method:btree",
		},
		{
			Name:      "users_email_key", // Different name but same signature
			TableName: "users",
			Columns:   []string{"email"},
			IsUnique:  true,
			IsPrimary: false,
			Method:    "btree",
			Signature: "table:users|cols:email|unique:true|method:btree",
		},
		{
			Name:      "idx_users_old_field",
			TableName: "users",
			Columns:   []string{"old_field"},
			IsUnique:  false,
			IsPrimary: false,
			Method:    "btree",
			Signature: "table:users|cols:old_field|method:btree",
		},
	}

	comparison, err := generator.CompareSchemas(structIndexes, []ForeignKeyDefinition{}, dbIndexes, []ForeignKeyDefinition{})
	if err != nil {
		t.Fatalf("CompareSchemas() error = %v", err)
	}

	// Should create idx_users_team_id (in struct but not in DB)
	if len(comparison.IndexesToCreate) != 1 {
		t.Errorf("Expected 1 index to create, got %d", len(comparison.IndexesToCreate))
	} else {
		if comparison.IndexesToCreate[0].Name != "idx_users_team_id" {
			t.Errorf("Expected to create 'idx_users_team_id', got '%s'", comparison.IndexesToCreate[0].Name)
		}
	}

	// Should drop idx_users_old_field (in DB but not in struct)
	if len(comparison.IndexesToDrop) != 1 {
		t.Errorf("Expected 1 index to drop, got %d", len(comparison.IndexesToDrop))
	} else {
		if comparison.IndexesToDrop[0].Name != "idx_users_old_field" {
			t.Errorf("Expected to drop 'idx_users_old_field', got '%s'", comparison.IndexesToDrop[0].Name)
		}
	}

	// No foreign key changes expected
	if len(comparison.ForeignKeysToCreate) != 0 {
		t.Errorf("Expected 0 foreign keys to create, got %d", len(comparison.ForeignKeysToCreate))
	}
	if len(comparison.ForeignKeysToDrop) != 0 {
		t.Errorf("Expected 0 foreign keys to drop, got %d", len(comparison.ForeignKeysToDrop))
	}
}

func TestEnhancedGenerator_IsSafeOperation(t *testing.T) {
	generator := NewEnhancedGenerator()

	tests := []struct {
		name       string
		comparison *SchemaComparison
		expected   bool
	}{
		{
			name: "safe - only creates indexes",
			comparison: &SchemaComparison{
				IndexesToCreate: []IndexDefinition{
					{Name: "idx_new", IsUnique: false, IsPrimary: false},
				},
			},
			expected: true,
		},
		{
			name: "unsafe - drops unique index",
			comparison: &SchemaComparison{
				IndexesToDrop: []IndexDefinition{
					{Name: "uk_unique", IsUnique: true, IsPrimary: false},
				},
			},
			expected: false,
		},
		{
			name: "unsafe - drops primary key",
			comparison: &SchemaComparison{
				IndexesToDrop: []IndexDefinition{
					{Name: "table_pkey", IsUnique: true, IsPrimary: true},
				},
			},
			expected: false,
		},
		{
			name: "unsafe - drops foreign key",
			comparison: &SchemaComparison{
				ForeignKeysToDrop: []ForeignKeyDefinition{
					{Name: "fk_constraint"},
				},
			},
			expected: false,
		},
		{
			name: "safe - drops regular index",
			comparison: &SchemaComparison{
				IndexesToDrop: []IndexDefinition{
					{Name: "idx_regular", IsUnique: false, IsPrimary: false},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.IsSafeOperation(tt.comparison)
			if result != tt.expected {
				t.Errorf("IsSafeOperation() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEnhancedGenerator_GenerateSafeSQL(t *testing.T) {
	generator := NewEnhancedGenerator()

	comparison := &SchemaComparison{
		IndexesToCreate: []IndexDefinition{
			{
				Name:      "idx_users_email",
				TableName: "users",
				Columns:   []string{"email"},
				IsUnique:  true,
				Method:    "btree",
			},
		},
		IndexesToDrop: []IndexDefinition{
			{
				Name:      "idx_old_index",
				TableName: "users",
				Columns:   []string{"old_field"},
				IsUnique:  false,
				Method:    "btree",
			},
		},
		ForeignKeysToCreate: []ForeignKeyDefinition{
			{
				Name:              "fk_projects_team",
				TableName:         "projects",
				Columns:           []string{"team_id"},
				ReferencedTable:   "teams",
				ReferencedColumns: []string{"id"},
				OnDelete:          "CASCADE",
			},
		},
	}

	tests := []struct {
		name               string
		allowDestructive   bool
		expectedUpCount    int
		expectedDownCount  int
		shouldIncludeDrops bool
	}{
		{
			name:               "safe mode - no destructive operations",
			allowDestructive:   false,
			expectedUpCount:    2, // CREATE INDEX + CREATE FK
			expectedDownCount:  2, // DROP INDEX + DROP FK
			shouldIncludeDrops: false,
		},
		{
			name:               "destructive mode - include all operations",
			allowDestructive:   true,
			expectedUpCount:    3, // CREATE INDEX + CREATE FK + DROP INDEX
			expectedDownCount:  3, // DROP INDEX + DROP FK + CREATE INDEX
			shouldIncludeDrops: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upSQL, downSQL, err := generator.GenerateSafeSQL(comparison, tt.allowDestructive)
			if err != nil {
				t.Fatalf("GenerateSafeSQL() error = %v", err)
			}

			if len(upSQL) != tt.expectedUpCount {
				t.Errorf("Expected %d up statements, got %d", tt.expectedUpCount, len(upSQL))
			}

			if len(downSQL) != tt.expectedDownCount {
				t.Errorf("Expected %d down statements, got %d", tt.expectedDownCount, len(downSQL))
			}

			// Check for CREATE statements (should always be present)
			hasCreateIndex := false
			hasCreateFK := false
			for _, stmt := range upSQL {
				if contains(stmt, "CREATE UNIQUE INDEX idx_users_email") {
					hasCreateIndex = true
				}
				if contains(stmt, "ADD CONSTRAINT fk_projects_team") {
					hasCreateFK = true
				}
			}

			if !hasCreateIndex {
				t.Error("Missing CREATE INDEX statement")
			}
			if !hasCreateFK {
				t.Error("Missing CREATE FOREIGN KEY statement")
			}

			// Check for DROP statements (should only be present if destructive allowed)
			hasDropIndex := false
			for _, stmt := range upSQL {
				if contains(stmt, "DROP INDEX") {
					hasDropIndex = true
					break
				}
			}

			if tt.shouldIncludeDrops && !hasDropIndex {
				t.Error("Expected DROP INDEX statement in destructive mode")
			}
			if !tt.shouldIncludeDrops && hasDropIndex {
				t.Error("Unexpected DROP INDEX statement in safe mode")
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			indexOf(s, substr) >= 0)
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
func BenchmarkEnhancedGenerator_GenerateIndexDefinitions(b *testing.B) {
	generator := NewEnhancedGenerator()
	tableDefn := parser.TableDefinition{
		StructName: "ComplexTable",
		TableName:  "complex_tables",
		Fields: []parser.FieldDefinition{
			{Name: "ID", DBName: "id", DBDef: map[string]string{"type": "cuid", "primary_key": ""}},
			{Name: "Email", DBName: "email", DBDef: map[string]string{"type": "varchar(255)", "unique": ""}},
			{Name: "TeamID", DBName: "team_id", DBDef: map[string]string{"type": "cuid"}},
		},
		TableLevel: map[string]string{
			"index":  "idx_complex_team,team_id;idx_complex_multi,team_id,email",
			"unique": "uk_complex_unique,email,team_id",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := generator.GenerateIndexDefinitions(tableDefn)
		if err != nil {
			b.Fatal(err)
		}
	}
}
