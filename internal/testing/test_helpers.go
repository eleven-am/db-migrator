package testing

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// TestDB provides a test database connection
type TestDB struct {
	DB      *sql.DB
	DBName  string
	ConnStr string
	t       *testing.T
}

// NewTestDB creates a new test database
func NewTestDB(t *testing.T) *TestDB {
	t.Helper()

	baseConnStr := "postgres://localhost/postgres?sslmode=disable"
	db, err := sql.Open("postgres", baseConnStr)
	if err != nil {
		t.Fatalf("Failed to connect to postgres: %v", err)
	}

	dbName := fmt.Sprintf("test_migrator_%d", time.Now().UnixNano())

	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	if err != nil {
		db.Close()
		t.Fatalf("Failed to create test database: %v", err)
	}
	db.Close()

	testConnStr := fmt.Sprintf("postgres://localhost/%s?sslmode=disable", dbName)
	testDB, err := sql.Open("postgres", testConnStr)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	return &TestDB{
		DB:      testDB,
		DBName:  dbName,
		ConnStr: testConnStr,
		t:       t,
	}
}

// Cleanup drops the test database
func (tdb *TestDB) Cleanup() {
	tdb.DB.Close()

	baseConnStr := "postgres://localhost/postgres?sslmode=disable"
	db, err := sql.Open("postgres", baseConnStr)
	if err != nil {
		tdb.t.Logf("Failed to connect for cleanup: %v", err)
		return
	}
	defer db.Close()

	_, err = db.Exec(fmt.Sprintf(`
		SELECT pg_terminate_backend(pg_stat_activity.pid)
		FROM pg_stat_activity
		WHERE pg_stat_activity.datname = '%s'
		AND pid <> pg_backend_pid()
	`, tdb.DBName))
	if err != nil {
		tdb.t.Logf("Failed to terminate connections: %v", err)
	}

	_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", tdb.DBName))
	if err != nil {
		tdb.t.Logf("Failed to drop test database: %v", err)
	}
}

// ExecuteSQL executes SQL statements
func (tdb *TestDB) ExecuteSQL(sql string) error {
	statements := strings.Split(sql, ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := tdb.DB.Exec(stmt); err != nil {
			return fmt.Errorf("failed to execute SQL: %w\nStatement: %s", err, stmt)
		}
	}
	return nil
}

// TableExists checks if a table exists
func (tdb *TestDB) TableExists(tableName string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1 
			FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = $1
		)
	`
	err := tdb.DB.QueryRow(query, tableName).Scan(&exists)
	return exists, err
}

// ColumnExists checks if a column exists in a table
func (tdb *TestDB) ColumnExists(tableName, columnName string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1 
			FROM information_schema.columns 
			WHERE table_schema = 'public' 
			AND table_name = $1
			AND column_name = $2
		)
	`
	err := tdb.DB.QueryRow(query, tableName, columnName).Scan(&exists)
	return exists, err
}

// GetColumnType returns the data type of a column
func (tdb *TestDB) GetColumnType(tableName, columnName string) (string, error) {
	var dataType string
	query := `
		SELECT data_type 
		FROM information_schema.columns 
		WHERE table_schema = 'public' 
		AND table_name = $1
		AND column_name = $2
	`
	err := tdb.DB.QueryRow(query, tableName, columnName).Scan(&dataType)
	return dataType, err
}

// IndexExists checks if an index exists
func (tdb *TestDB) IndexExists(indexName string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1 
			FROM pg_indexes 
			WHERE schemaname = 'public' 
			AND indexname = $1
		)
	`
	err := tdb.DB.QueryRow(query, indexName).Scan(&exists)
	return exists, err
}

// ConstraintExists checks if a constraint exists
func (tdb *TestDB) ConstraintExists(tableName, constraintName string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1 
			FROM information_schema.table_constraints 
			WHERE table_schema = 'public' 
			AND table_name = $1
			AND constraint_name = $2
		)
	`
	err := tdb.DB.QueryRow(query, tableName, constraintName).Scan(&exists)
	return exists, err
}
