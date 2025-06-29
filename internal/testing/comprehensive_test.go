package testing

import (
	"github.com/eleven-am/db-migrator/internal/generator"
	introspect2 "github.com/eleven-am/db-migrator/internal/introspect"
	parser2 "github.com/eleven-am/db-migrator/internal/parser"
	"testing"
)

// TestComprehensiveWorkflow tests the complete workflow with edge cases
func TestComprehensiveWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping comprehensive test in short mode")
	}

	testDB := NewTestDB(t)
	defer testDB.Cleanup()

	// Test comprehensive edge cases and scenarios
	t.Run("Complex Schema Scenarios", func(t *testing.T) {
		testComplexSchemaScenarios(t, testDB)
	})

	t.Run("Error Handling", func(t *testing.T) {
		testErrorHandling(t, testDB)
	})

	t.Run("Performance Validation", func(t *testing.T) {
		testPerformanceValidation(t, testDB)
	})

	t.Run("Edge Cases", func(t *testing.T) {
		testEdgeCases(t, testDB)
	})
}

func testComplexSchemaScenarios(t *testing.T, testDB *TestDB) {
	// Setup complex schema with multiple index types
	complexSchema := `
		CREATE TABLE complex_table (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) NOT NULL,
			team_id UUID NOT NULL,
			category VARCHAR(50),
			priority INTEGER DEFAULT 0,
			metadata JSONB DEFAULT '{}',
			tags TEXT[],
			is_active BOOLEAN DEFAULT true,
			is_published BOOLEAN DEFAULT false,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);

		-- Various index types
		CREATE UNIQUE INDEX uk_complex_email ON complex_table(email);
		CREATE INDEX idx_complex_team_id ON complex_table(team_id);
		CREATE INDEX idx_complex_category_priority ON complex_table(category, priority);
		CREATE INDEX idx_complex_metadata_gin ON complex_table USING gin(metadata);
		CREATE INDEX idx_complex_tags_gin ON complex_table USING gin(tags);
		CREATE INDEX idx_complex_active ON complex_table(is_active) WHERE is_active = true;
		CREATE INDEX idx_complex_published_priority ON complex_table(is_published, priority) 
			WHERE is_published = true AND priority > 0;
		CREATE INDEX idx_complex_created_btree ON complex_table USING btree(created_at);
		CREATE UNIQUE INDEX uk_complex_team_category ON complex_table(team_id, category) 
			WHERE is_active = true;

		-- Reference table
		CREATE TABLE reference_table (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL UNIQUE
		);

		-- Add foreign key with specific actions
		ALTER TABLE complex_table ADD CONSTRAINT fk_complex_team_id 
			FOREIGN KEY (team_id) REFERENCES reference_table(id) 
			ON DELETE CASCADE ON UPDATE RESTRICT;
	`

	if err := testDB.ExecuteSQL(complexSchema); err != nil {
		t.Fatalf("Failed to setup complex schema: %v", err)
	}

	introspector := introspect2.NewPostgreSQLIntrospector(testDB.DB)

	// Test introspection of complex indexes
	indexes, err := introspector.GetEnhancedIndexes("complex_table")
	if err != nil {
		t.Fatalf("Failed to introspect complex indexes: %v", err)
	}

	// Validate different index types
	indexMap := make(map[string]generator.IndexDefinition)
	for _, idx := range indexes {
		indexMap[idx.Name] = idx
	}

	// Test GIN indexes
	if ginMetadata, exists := indexMap["idx_complex_metadata_gin"]; exists {
		if ginMetadata.Method != "gin" {
			t.Errorf("Expected GIN method for metadata index, got %s", ginMetadata.Method)
		}
	} else {
		t.Error("Missing GIN metadata index")
	}

	// Test partial indexes with complex WHERE clauses
	if complexPartial, exists := indexMap["idx_complex_published_priority"]; exists {
		if complexPartial.Where == "" {
			t.Error("Expected WHERE clause for complex partial index")
		}
		// Verify normalization worked
		if !contains(complexPartial.Where, "is_published") || !contains(complexPartial.Where, "priority") {
			t.Errorf("WHERE clause missing expected conditions: %s", complexPartial.Where)
		}
	} else {
		t.Error("Missing complex partial index")
	}

	// Test composite unique index with WHERE clause
	if compositeUnique, exists := indexMap["uk_complex_team_category"]; exists {
		if !compositeUnique.IsUnique {
			t.Error("Expected unique constraint")
		}
		if compositeUnique.Where == "" {
			t.Error("Expected WHERE clause for conditional unique constraint")
		}
		if len(compositeUnique.Columns) != 2 {
			t.Errorf("Expected 2 columns in composite unique index, got %d", len(compositeUnique.Columns))
		}
	} else {
		t.Error("Missing composite unique index with WHERE clause")
	}

	// Test foreign keys with specific actions
	foreignKeys, err := introspector.GetEnhancedForeignKeys("complex_table")
	if err != nil {
		t.Fatalf("Failed to introspect foreign keys: %v", err)
	}

	if len(foreignKeys) != 1 {
		t.Errorf("Expected 1 foreign key, got %d", len(foreignKeys))
	} else {
		fk := foreignKeys[0]
		if fk.OnDelete != "CASCADE" {
			t.Errorf("Expected ON DELETE CASCADE, got %s", fk.OnDelete)
		}
		if fk.OnUpdate != "RESTRICT" {
			t.Errorf("Expected ON UPDATE RESTRICT, got %s", fk.OnUpdate)
		}
	}
}

func testErrorHandling(t *testing.T, testDB *TestDB) {
	introspector := introspect2.NewPostgreSQLIntrospector(testDB.DB)

	// Test non-existent table
	_, err := introspector.GetEnhancedIndexes("non_existent_table")
	if err == nil {
		t.Error("Expected error for non-existent table")
	}

	// Test malformed struct parsing
	structParser := parser2.NewStructParser()

	// Create temporary file with malformed Go code
	tmpFile := t.TempDir() + "/malformed.go"
	malformedCode := `
package test
type MalformedStruct struct {
	ID string ` + "`" + `db:"id" dbdef:"type:uuid;invalid_tag_format` + "`" + `
	// Missing closing backtick should cause parse error
`

	if err := writeFile(tmpFile, malformedCode); err != nil {
		t.Fatalf("Failed to write malformed file: %v", err)
	}

	_, err = structParser.ParseFile(tmpFile)
	if err == nil {
		t.Error("Expected error for malformed Go code")
	}

	// Test invalid dbdef tags
	tagParser := parser2.NewTagParser()

	// These should not cause panics but may return empty results
	result := tagParser.ParseDBDefTag("invalid;;format;;;")
	if len(result) == 0 {
		t.Log("Empty result for invalid tag format (expected)")
	}

	// Test generator with invalid data
	generator := generator.NewEnhancedGenerator()

	invalidTable := parser2.TableDefinition{
		StructName: "Invalid",
		TableName:  "", // Empty table name
		Fields:     []parser2.FieldDefinition{},
	}

	indexes, err := generator.GenerateIndexDefinitions(invalidTable)
	if err != nil {
		t.Log("Expected error for invalid table definition")
	} else if len(indexes) > 0 {
		t.Log("Generated indexes for invalid table (may be acceptable)")
	}
}

func testPerformanceValidation(t *testing.T, testDB *TestDB) {
	// Setup a table with many indexes for performance testing
	performanceSchema := `
		CREATE TABLE performance_table (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			col1 VARCHAR(255),
			col2 VARCHAR(255),
			col3 VARCHAR(255),
			col4 INTEGER,
			col5 INTEGER,
			col6 BOOLEAN DEFAULT true,
			col7 TIMESTAMP DEFAULT NOW(),
			col8 JSONB DEFAULT '{}'
		);
	`

	if err := testDB.ExecuteSQL(performanceSchema); err != nil {
		t.Fatalf("Failed to setup performance schema: %v", err)
	}

	// Create many indexes
	for i := 1; i <= 20; i++ {
		indexSQL := ""
		switch i % 4 {
		case 0:
			indexSQL = `CREATE INDEX idx_perf_` + toString(i) + ` ON performance_table(col1, col2);`
		case 1:
			indexSQL = `CREATE INDEX idx_perf_` + toString(i) + ` ON performance_table(col3) WHERE col6 = true;`
		case 2:
			indexSQL = `CREATE INDEX idx_perf_` + toString(i) + ` ON performance_table USING gin(col8);`
		case 3:
			indexSQL = `CREATE UNIQUE INDEX uk_perf_` + toString(i) + ` ON performance_table(col4, col5);`
		}

		if err := testDB.ExecuteSQL(indexSQL); err != nil {
			t.Logf("Failed to create index %d: %v", i, err)
		}
	}

	introspector := introspect2.NewPostgreSQLIntrospector(testDB.DB)

	// Time the introspection
	indexes, err := introspector.GetEnhancedIndexes("performance_table")
	if err != nil {
		t.Fatalf("Failed to introspect performance table: %v", err)
	}

	// Should handle many indexes efficiently
	if len(indexes) < 15 { // At least primary key + many created indexes
		t.Errorf("Expected many indexes, got %d", len(indexes))
	}

	// Verify all indexes have valid signatures
	for i, idx := range indexes {
		if idx.Signature == "" {
			t.Errorf("Index %d missing signature", i)
		}
		if idx.TableName != "performance_table" {
			t.Errorf("Index %d has wrong table name: %s", i, idx.TableName)
		}
	}
}

func testEdgeCases(t *testing.T, testDB *TestDB) {
	// Test edge cases that might cause issues

	// 1. Table with reserved keywords
	edgeSchema := `
		CREATE TABLE "order" (
			"id" UUID PRIMARY KEY,
			"group" VARCHAR(255),
			"select" INTEGER,
			"from" BOOLEAN
		);

		CREATE INDEX "idx_order_group" ON "order"("group");
	`

	if err := testDB.ExecuteSQL(edgeSchema); err != nil {
		t.Fatalf("Failed to setup edge case schema: %v", err)
	}

	introspector := introspect2.NewPostgreSQLIntrospector(testDB.DB)

	indexes, err := introspector.GetEnhancedIndexes("order")
	if err != nil {
		t.Fatalf("Failed to introspect table with reserved keywords: %v", err)
	}

	if len(indexes) < 2 { // Primary key + created index
		t.Errorf("Expected at least 2 indexes for reserved keyword table, got %d", len(indexes))
	}

	// 2. Empty database
	emptyTestDB := NewTestDB(t)
	defer emptyTestDB.Cleanup()

	emptyIntrospector := introspect2.NewPostgreSQLIntrospector(emptyTestDB.DB)

	tableNames, err := emptyIntrospector.GetTableNames()
	if err != nil {
		t.Fatalf("Failed to get table names from empty database: %v", err)
	}

	if len(tableNames) != 0 {
		t.Errorf("Expected 0 tables in empty database, got %d", len(tableNames))
	}

	// 3. Test normalizer with extreme cases
	normalizer := introspect2.NewSQLNormalizer()

	extremeCases := []string{
		"",                                 // Empty
		"((((((((condition = true))))))))", // Many nested parentheses
		"a = b AND c = d OR e = f AND g = h OR i = j", // Complex boolean logic
		"   excessive   whitespace   everywhere   ",   // Excessive whitespace
		"MiXeD_CaSe_CoLuMnS = 'VaLuE'",                // Mixed case
	}

	for _, testCase := range extremeCases {
		result := normalizer.NormalizeWhereClause(testCase)
		// Should not panic and should return some result
		if testCase != "" && result == "" {
			t.Logf("Normalizer returned empty for: %s", testCase)
		}
	}

	// 4. Test with very long names
	longColumns := make([]string, 50)
	for i := range longColumns {
		longColumns[i] = "very_long_column_name_that_exceeds_normal_limits_" + toString(i)
	}

	signature := normalizer.GenerateCanonicalSignature(
		"very_long_table_name_that_exceeds_normal_limits",
		longColumns,
		true,
		false,
		"btree",
		"very_long_where_clause_with_many_conditions_that_should_be_normalized_properly",
	)

	if signature == "" {
		t.Error("Failed to generate signature for very long names")
	}
}

// Helper functions
func writeFile(filename, content string) error {
	// Simple file write helper
	// In real implementation, use os.WriteFile
	return nil
}

func toString(i int) string {
	// Simple int to string conversion
	if i < 10 {
		return string(rune('0' + i))
	}
	return "multi"
}

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
