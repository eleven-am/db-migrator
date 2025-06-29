# DB Migrator Testing Framework

This testing framework provides comprehensive testing utilities for the database migration tool.

## Test Helpers

### TestDB
Provides a test database connection with automatic cleanup:

```go
func TestSomething(t *testing.T) {
    tdb := dbtest.NewTestDB(t)
    defer tdb.Cleanup()
    
    // Use tdb.DB for database operations
    // Use tdb.ExecuteSQL(sql) to run SQL
}
```

### Assertions

#### SQL Assertions
```go
sqlAssert := dbtest.NewAssertSQL(t)
sqlAssert.Contains(sql, "CREATE TABLE users")
sqlAssert.NotContains(sql, "DROP TABLE")
sqlAssert.HasWarning(sql)
sqlAssert.IsCommented(sql, "DROP TABLE users")
sqlAssert.AssertStatementCount(sql, "CREATE TABLE", 2)
```

#### Table Assertions
```go
tableAssert := dbtest.NewTableAssertions(t, tdb)
tableAssert.AssertTableExists("users")
tableAssert.AssertColumnExists("users", "email")
tableAssert.AssertColumnType("users", "email", "character varying")
tableAssert.AssertIndexExists("idx_users_email")
tableAssert.AssertConstraintExists("users", "users_pkey")
```

### Test Fixtures

Create test models easily:

```go
userModel := dbtest.CreateTestModel("User",
    dbtest.CreateTestField("ID", "uuid", "primary_key"),
    dbtest.CreateTestField("Email", "varchar(255)", "not_null", "unique"),
)
```

## Running Tests

### Unit Tests Only
```bash
make test
# or
make test-unit
```

### Integration Tests (requires PostgreSQL)
```bash
make test-integration
```

### All Tests
```bash
make test-all
```

### Coverage Report
```bash
make coverage
# Opens coverage.html in browser
```

### Specific Package Tests
```bash
make test-parser
make test-generator
make test-diff
make test-introspect
```

## Test Categories

### Unit Tests
- **Parser Tests**: Test struct parsing, tag parsing, table name generation
- **Generator Tests**: Test SQL generation, schema generation
- **Diff Tests**: Test schema comparison, change detection, safety analysis
- **Introspect Tests**: Test database introspection

### Integration Tests
- **Full Flow**: Test complete migration workflow from structs to database
- **Migration Ordering**: Test dependency resolution
- **Unsafe Changes**: Test detection and handling of destructive changes
- **Rollback**: Test DOWN migration generation and execution

### Snapshot Tests
- **SQL Output**: Verify generated SQL matches expected output
- **Migration Format**: Ensure consistent migration file format
- **Warning Comments**: Verify unsafe operations are properly commented

## Writing New Tests

### 1. Unit Test Example
```go
func TestMyFeature(t *testing.T) {
    // Setup
    parser := NewStructParser()
    
    // Execute
    result, err := parser.ParseFile("test.go")
    
    // Assert
    if err != nil {
        t.Fatalf("Unexpected error: %v", err)
    }
    if len(result) != 1 {
        t.Errorf("Expected 1 result, got %d", len(result))
    }
}
```

### 2. Integration Test Example
```go
func TestDatabaseMigration(t *testing.T) {
    // Create test database
    tdb := dbtest.NewTestDB(t)
    defer tdb.Cleanup()
    
    // Run migration
    if err := tdb.ExecuteSQL(migrationSQL); err != nil {
        t.Fatalf("Migration failed: %v", err)
    }
    
    // Verify results
    tableAssert := dbtest.NewTableAssertions(t, tdb)
    tableAssert.AssertTableExists("users")
}
```

### 3. Snapshot Test Example
```go
func TestSQLGeneration(t *testing.T) {
    // Generate SQL
    sql := generator.GenerateCreateTable(table)
    
    // Normalize for comparison
    sql = normalizeSQL(sql)
    expected = normalizeSQL(expectedSQL)
    
    // Compare
    if sql != expected {
        t.Errorf("SQL mismatch:\nGot:\n%s\n\nExpected:\n%s", sql, expected)
    }
}
```

## Best Practices

1. **Isolation**: Each test should create its own test database
2. **Cleanup**: Always defer cleanup of test resources
3. **Assertions**: Use provided assertion helpers for clarity
4. **Fixtures**: Use fixture helpers for consistent test data
5. **Categories**: Separate unit and integration tests
6. **Coverage**: Aim for >80% test coverage
7. **Speed**: Keep unit tests fast (<100ms each)
8. **Clarity**: Test names should describe what they test

## Troubleshooting

### PostgreSQL Connection Issues
```bash
# Ensure PostgreSQL is running
pg_isready

# Check connection
psql -h localhost -U postgres -c "SELECT 1"
```

### Test Database Cleanup
If test databases aren't cleaned up:
```sql
-- List test databases
SELECT datname FROM pg_database WHERE datname LIKE 'test_migrator_%';

-- Drop them
DROP DATABASE test_migrator_xxxx;
```