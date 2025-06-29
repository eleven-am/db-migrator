# DB-Migrator Test Suite Documentation

## Overview

This document describes the comprehensive test suite for the db-migrator package. The test suite ensures reliability, performance, and correctness of the PostgreSQL database migration tool.

## Test Structure

### Test Categories

1. **Unit Tests** - Test individual components in isolation
2. **Integration Tests** - Test component interactions and database operations
3. **End-to-End Tests** - Test complete workflows
4. **Performance Tests** - Benchmark critical operations
5. **CLI Tests** - Test command-line interface

### Test Files

```
db-migrator/
├── parser/
│   ├── struct_parser_test.go       # Go struct parsing tests
│   └── tag_parser_test.go          # DBDef tag parsing tests
├── introspect/
│   └── sql_normalizer_test.go      # SQL normalization tests
├── generator/
│   ├── enhanced_generator_test.go  # Schema generation tests
│   └── sql_generator_test.go       # SQL generation tests
├── cmd/
│   └── migrate_test.go             # CLI command tests
├── testing/
│   ├── test_helpers.go             # Test utilities
│   ├── fixtures.go                 # Test data fixtures
│   └── comprehensive_test.go       # Edge case tests
├── postgres_test.go                # PostgreSQL integration tests
├── integration_test.go             # End-to-end workflow tests
└── test_config.go                  # Test configuration
```

## Test Coverage

### Parser Package (`parser/`)

**struct_parser_test.go**
- `TestToSnakeCase` - String case conversion (12 test cases)
- `TestDeriveTableName` - Table name pluralization (10 test cases)
- `TestStructParser_ParseFile` - Single file parsing with validation
- `TestStructParser_ParseDirectory` - Multi-file parsing

**tag_parser_test.go**
- `TestTagParser_ParseDBDefTag` - DBDef tag parsing (14 test cases)
  - Primary keys, foreign keys, unique constraints
  - Index definitions, defaults, type mappings
  - Complex scenarios like JSONB, CUID types

### Introspect Package (`introspect/`)

**sql_normalizer_test.go**
- `TestSQLNormalizer_NormalizeWhereClause` - WHERE clause normalization (11 test cases)
- `TestSQLNormalizer_NormalizeColumnList` - Column list processing (6 test cases)
- `TestSQLNormalizer_NormalizeIndexMethod` - Index method normalization (6 test cases)
- `TestSQLNormalizer_GenerateCanonicalSignature` - Signature generation (8 test cases)
- `TestSQLNormalizer_simpleNormalizeWhere` - Fallback normalization (6 test cases)
- `TestSQLNormalizer_isBalancedParentheses` - Parentheses validation (8 test cases)
- `TestSQLNormalizer_cleanNormalizedWhere` - Post-processing cleanup (6 test cases)
- Performance benchmarks for critical operations

### Generator Package (`generator/`)

**enhanced_generator_test.go**
- `TestEnhancedGenerator_GenerateIndexDefinitions` - Index generation (4 test cases)
  - Primary keys, field-level unique constraints
  - Table-level indexes, partial indexes
- `TestEnhancedGenerator_parseTableLevelIndex` - Index parsing (7 test cases)
  - Simple indexes, unique constraints, composite indexes
  - Partial indexes, malformed input handling
- `TestEnhancedGenerator_GenerateForeignKeyDefinitions` - FK generation (3 test cases)
- `TestEnhancedGenerator_CompareSchemas` - Schema comparison logic
- `TestEnhancedGenerator_IsSafeOperation` - Safety validation (5 test cases)
- `TestEnhancedGenerator_GenerateSafeSQL` - SQL generation (2 test cases)
- Performance benchmarks

**sql_generator_test.go**
- `TestSQLGenerator_GenerateCreateTable` - Table creation SQL (5 test cases)
- `TestSQLGenerator_GenerateIndexDDL` - Index creation SQL (4 test cases)
- `TestSQLGenerator_HasCUID` - CUID detection (3 test cases)
- `TestSQLGenerator_GenerateCreateDatabase` - Database creation SQL

### Integration Tests

**postgres_test.go**
- `TestPostgreSQLIntrospector_GetEnhancedIndexes` - Real database introspection
  - Complex index types (GIN, BTREE, partial, unique)
  - Index method detection, WHERE clause processing
- `TestPostgreSQLIntrospector_GetEnhancedForeignKeys` - FK introspection
  - Various FK actions (CASCADE, RESTRICT, SET NULL)
- `TestPostgreSQLIntrospector_GetTableNames` - Table discovery
- `TestPostgreSQLIntrospector_ErrorHandling` - Error conditions
- `TestPostgreSQLIntrospector_ComplexIndexes` - Advanced scenarios
- Performance benchmarks with real database operations

**integration_test.go**
- `TestEndToEndMigrationWorkflow` - Complete migration process
  - Struct parsing → schema generation → DB introspection → comparison → SQL generation
- `TestMigrationWithCompleteSchema` - Perfect schema match validation
- Tests realistic scenarios with actual Go structs and PostgreSQL

### CLI Tests

**cmd/migrate_test.go**
- `TestMigrateCommand` - Command execution (6 test cases)
  - Successful migrations, existing schema handling
  - Error conditions, invalid inputs
- `TestMigrateCommandFlags` - Flag validation (13 flags)
- `TestMigrateCommandValidation` - Input validation (4 test cases)
- `TestMigrateCommandOutput` - Output verification

### Comprehensive Edge Cases

**testing/comprehensive_test.go**
- `TestComprehensiveWorkflow` - Edge case validation
  - Complex schema scenarios with multiple index types
  - Error handling with malformed inputs
  - Performance validation with many indexes
  - Reserved keywords, empty databases, extreme cases

## Test Utilities

### Test Database (`testing/test_helpers.go`)

```go
type TestDB struct {
    DB       *sql.DB
    DBName   string
    ConnStr  string
}
```

**Features:**
- Automatic test database creation/cleanup
- SQL execution utilities
- Schema introspection helpers
- Existence checks for tables, columns, indexes, constraints

**Usage:**
```go
testDB := NewTestDB(t)
defer testDB.Cleanup()

// Execute schema setup
err := testDB.ExecuteSQL("CREATE TABLE users (...)")

// Check if table exists
exists, err := testDB.TableExists("users")
```

### Test Configuration

**Environment Variables:**
- `TEST_MODE=true` - Enables test-specific behavior
- `LOG_LEVEL=debug` - Verbose logging for debugging

**Test Modes:**
- Short mode (`go test -short`) - Unit tests only, no database required
- Full mode - All tests including integration tests

## Running Tests

### Quick Test Commands

```bash
# Unit tests only (no database required)
go test -short -v ./...

# All tests (requires PostgreSQL)
go test -v ./...

# Specific package
go test -v ./parser/...

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Benchmarks
go test -bench=. -benchmem ./...

# Race detection
go test -race ./...
```

### Makefile Targets

```bash
make test              # Run all tests
make test-unit         # Unit tests only
make test-integration  # Integration tests
make test-coverage     # Generate coverage report
make test-race         # Run with race detection
make test-bench        # Run benchmarks
make clean-test        # Clean test artifacts
```

## Test Data Patterns

### Realistic Test Models

Tests use realistic Go struct models that represent actual database schemas:

```go
type User struct {
    _        struct{} `dbdef:"table:users;index:idx_users_team_id,team_id"`
    
    ID        string    `db:"id" dbdef:"type:cuid;primary_key;default:gen_cuid()"`
    Email     string    `db:"email" dbdef:"type:varchar(255);not_null;unique"`
    TeamID    string    `db:"team_id" dbdef:"type:cuid;not_null;foreign_key:teams.id"`
    IsActive  bool      `db:"is_active" dbdef:"type:boolean;not_null;default:true"`
    CreatedAt time.Time `db:"created_at" dbdef:"type:timestamptz;not_null;default:now()"`
}
```

### Schema Scenarios

1. **Simple Schemas** - Basic tables with primary keys
2. **Complex Schemas** - Multiple index types, foreign keys, partial indexes
3. **Edge Cases** - Reserved keywords, empty schemas, malformed data
4. **Performance Scenarios** - Many tables/indexes for stress testing

## Validation Patterns

### Index Validation

```go
expectedIndexes := map[string]struct {
    isUnique  bool
    isPrimary bool
    columns   []string
    hasWhere  bool
}{
    "users_pkey":      {isUnique: true, isPrimary: true, columns: []string{"id"}},
    "users_email_key": {isUnique: true, isPrimary: false, columns: []string{"email"}},
}
```

### Signature Validation

Tests verify that schema signatures are generated correctly and consistently:

```go
expectedSignature := "table:users|cols:email|unique:true|method:btree"
if idx.Signature != expectedSignature {
    t.Errorf("Expected signature %s, got %s", expectedSignature, idx.Signature)
}
```

### SQL Validation

Generated SQL is validated for correctness:

```go
expectedSQL := "CREATE UNIQUE INDEX idx_users_email ON users (email);"
if !contains(upSQL[0], "CREATE UNIQUE INDEX") {
    t.Error("Missing CREATE INDEX statement")
}
```

## Performance Validation

### Benchmarks

Critical operations are benchmarked to ensure performance:

- `BenchmarkSQLNormalizer_NormalizeWhereClause`
- `BenchmarkSQLNormalizer_GenerateCanonicalSignature`
- `BenchmarkEnhancedGenerator_GenerateIndexDefinitions`
- `BenchmarkPostgreSQLIntrospector_GetEnhancedIndexes`

### Performance Scenarios

Tests validate performance with:
- Large numbers of indexes (20+ per table)
- Complex WHERE clauses
- Many columns in composite indexes
- Deep nested structures

## Error Handling Tests

### Expected Errors

Tests verify proper error handling for:
- Non-existent tables/databases
- Malformed Go code
- Invalid dbdef tags
- Connection failures
- Invalid SQL

### Error Message Validation

```go
_, err := introspector.GetEnhancedIndexes("non_existent_table")
if err == nil {
    t.Error("Expected error for non-existent table")
}
```

## Test Dependencies

### Required Services

Integration tests require:
- PostgreSQL server running locally
- Standard PostgreSQL extensions (uuid-ossp)

### Go Dependencies

Test-specific dependencies:
- `github.com/lib/pq` - PostgreSQL driver
- Standard Go testing package
- Temporary file/directory utilities

## Coverage Goals

### Target Coverage

- **Unit Tests**: >90% coverage for core logic
- **Integration Tests**: All major workflows covered
- **Error Paths**: All error conditions tested
- **Edge Cases**: Comprehensive edge case coverage

### Coverage Exclusions

- Debug print statements
- CLI help text generation
- Platform-specific code paths

## Continuous Integration

### CI Pipeline

Tests are designed to run in CI environments:

1. **Lint Check** - Code style validation
2. **Unit Tests** - Fast tests without external dependencies
3. **Integration Tests** - Full tests with PostgreSQL service
4. **Coverage Report** - Generate and validate coverage
5. **Benchmark Comparison** - Performance regression detection

### Environment Setup

CI environments need:
```yaml
services:
  postgres:
    image: postgres:15
    env:
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: test
```

## Test Maintenance

### Adding New Tests

When adding features:

1. **Unit Tests** - Test individual functions/methods
2. **Integration Tests** - Test with real database
3. **End-to-End Tests** - Test complete workflows
4. **Error Cases** - Test failure scenarios
5. **Performance Tests** - Benchmark critical paths

### Test Data Updates

When schema changes:

1. Update test fixtures in `testing/fixtures.go`
2. Update expected results in assertion helpers
3. Add new test cases for new functionality
4. Update documentation

This comprehensive test suite ensures the db-migrator package is production-ready and can be confidently published as a standalone package.