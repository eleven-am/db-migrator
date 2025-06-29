package testing

import (
	"strings"
	"testing"
)

// AssertSQL provides SQL assertion helpers
type AssertSQL struct {
	t *testing.T
}

// NewAssertSQL creates a new SQL assertion helper
func NewAssertSQL(t *testing.T) *AssertSQL {
	return &AssertSQL{t: t}
}

// Contains asserts that the SQL contains a substring
func (a *AssertSQL) Contains(sql, expected string) {
	a.t.Helper()
	if !strings.Contains(sql, expected) {
		a.t.Errorf("SQL does not contain expected string\nExpected: %s\nActual SQL:\n%s", expected, sql)
	}
}

// NotContains asserts that the SQL does not contain a substring
func (a *AssertSQL) NotContains(sql, unexpected string) {
	a.t.Helper()
	if strings.Contains(sql, unexpected) {
		a.t.Errorf("SQL contains unexpected string\nUnexpected: %s\nActual SQL:\n%s", unexpected, sql)
	}
}

// HasStatement asserts that the SQL contains a specific statement type
func (a *AssertSQL) HasStatement(sql, stmtType string) {
	a.t.Helper()
	upperSQL := strings.ToUpper(sql)
	upperStmt := strings.ToUpper(stmtType)
	
	if !strings.Contains(upperSQL, upperStmt) {
		a.t.Errorf("SQL does not contain %s statement\nActual SQL:\n%s", stmtType, sql)
	}
}

// CountStatements counts occurrences of a statement type
func (a *AssertSQL) CountStatements(sql, stmtType string) int {
	upperSQL := strings.ToUpper(sql)
	upperStmt := strings.ToUpper(stmtType)
	return strings.Count(upperSQL, upperStmt)
}

// AssertStatementCount asserts the number of specific statements
func (a *AssertSQL) AssertStatementCount(sql, stmtType string, expected int) {
	a.t.Helper()
	actual := a.CountStatements(sql, stmtType)
	if actual != expected {
		a.t.Errorf("Expected %d %s statements, got %d\nActual SQL:\n%s", 
			expected, stmtType, actual, sql)
	}
}

// IsCommented asserts that a line is commented out
func (a *AssertSQL) IsCommented(sql, statement string) {
	a.t.Helper()
	lines := strings.Split(sql, "\n")
	found := false
	isCommented := false
	
	for _, line := range lines {
		if strings.Contains(line, statement) {
			found = true
			if strings.HasPrefix(strings.TrimSpace(line), "--") {
				isCommented = true
			}
			break
		}
	}
	
	if !found {
		a.t.Errorf("Statement not found in SQL: %s", statement)
	} else if !isCommented {
		a.t.Errorf("Statement is not commented out: %s", statement)
	}
}

// HasWarning asserts that the SQL contains a warning comment
func (a *AssertSQL) HasWarning(sql string) {
	a.t.Helper()
	if !strings.Contains(sql, "WARNING") {
		a.t.Error("SQL does not contain WARNING comment")
	}
}

// TableAssertions provides table-specific assertions
type TableAssertions struct {
	t   *testing.T
	tdb *TestDB
}

// NewTableAssertions creates table assertion helpers
func NewTableAssertions(t *testing.T, tdb *TestDB) *TableAssertions {
	return &TableAssertions{t: t, tdb: tdb}
}

// AssertTableExists asserts that a table exists
func (ta *TableAssertions) AssertTableExists(tableName string) {
	ta.t.Helper()
	exists, err := ta.tdb.TableExists(tableName)
	if err != nil {
		ta.t.Fatalf("Error checking table existence: %v", err)
	}
	if !exists {
		ta.t.Errorf("Table %s does not exist", tableName)
	}
}

// AssertTableNotExists asserts that a table does not exist
func (ta *TableAssertions) AssertTableNotExists(tableName string) {
	ta.t.Helper()
	exists, err := ta.tdb.TableExists(tableName)
	if err != nil {
		ta.t.Fatalf("Error checking table existence: %v", err)
	}
	if exists {
		ta.t.Errorf("Table %s exists but should not", tableName)
	}
}

// AssertColumnExists asserts that a column exists
func (ta *TableAssertions) AssertColumnExists(tableName, columnName string) {
	ta.t.Helper()
	exists, err := ta.tdb.ColumnExists(tableName, columnName)
	if err != nil {
		ta.t.Fatalf("Error checking column existence: %v", err)
	}
	if !exists {
		ta.t.Errorf("Column %s.%s does not exist", tableName, columnName)
	}
}

// AssertColumnType asserts that a column has the expected type
func (ta *TableAssertions) AssertColumnType(tableName, columnName, expectedType string) {
	ta.t.Helper()
	actualType, err := ta.tdb.GetColumnType(tableName, columnName)
	if err != nil {
		ta.t.Fatalf("Error getting column type: %v", err)
	}
	if actualType != expectedType {
		ta.t.Errorf("Column %s.%s has type %s, expected %s", 
			tableName, columnName, actualType, expectedType)
	}
}

// AssertIndexExists asserts that an index exists
func (ta *TableAssertions) AssertIndexExists(indexName string) {
	ta.t.Helper()
	exists, err := ta.tdb.IndexExists(indexName)
	if err != nil {
		ta.t.Fatalf("Error checking index existence: %v", err)
	}
	if !exists {
		ta.t.Errorf("Index %s does not exist", indexName)
	}
}

// AssertConstraintExists asserts that a constraint exists
func (ta *TableAssertions) AssertConstraintExists(tableName, constraintName string) {
	ta.t.Helper()
	exists, err := ta.tdb.ConstraintExists(tableName, constraintName)
	if err != nil {
		ta.t.Fatalf("Error checking constraint existence: %v", err)
	}
	if !exists {
		ta.t.Errorf("Constraint %s on table %s does not exist", constraintName, tableName)
	}
}