package migrator

import (
	"strings"
	"testing"
)

func TestEnsureDatabaseExists(t *testing.T) {
	t.Run("invalid DSN", func(t *testing.T) {
		err := EnsureDatabaseExists("invalid-dsn")
		if err == nil {
			t.Error("Expected error for invalid DSN")
		}
	})

	t.Run("connection error", func(t *testing.T) {
		err := EnsureDatabaseExists("postgres://invalid:invalid@nonexistent:5432/test")
		if err == nil {
			t.Error("Expected error for invalid connection")
		}
	})
}

func TestParseDSNForDB_DSNFormat(t *testing.T) {
	t.Run("DSN format", func(t *testing.T) {
		dsn := "host=localhost port=5432 user=test password=test dbname=testdb sslmode=disable"
		dbName, adminDSN, err := parseDSNForDB(dsn)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if dbName != "testdb" {
			t.Errorf("Expected dbName 'testdb', got '%s'", dbName)
		}

		if !strings.Contains(adminDSN, "dbname=postgres") {
			t.Errorf("Expected adminDSN to contain 'dbname=postgres', got '%s'", adminDSN)
		}
	})

	t.Run("DSN format missing dbname", func(t *testing.T) {
		dsn := "host=localhost port=5432 user=test password=test sslmode=disable"
		_, _, err := parseDSNForDB(dsn)

		if err == nil {
			t.Error("Expected error for missing dbname")
		}
	})
}

func TestGetDatabaseURL_EdgeCases(t *testing.T) {
	t.Run("empty sslmode defaults to disable", func(t *testing.T) {
		result := GetDatabaseURL("localhost", "5432", "user", "pass", "testdb", "")
		expected := "postgres://user:pass@localhost:5432/testdb?sslmode=disable"
		if result != expected {
			t.Errorf("GetDatabaseURL() = %q, want %q", result, expected)
		}
	})
}

func TestGetDatabaseDSN_EdgeCases(t *testing.T) {
	t.Run("empty sslmode defaults to disable", func(t *testing.T) {
		result := GetDatabaseDSN("localhost", "5432", "user", "pass", "testdb", "")
		expected := "host=localhost port=5432 user=user password=pass dbname=testdb sslmode=disable"
		if result != expected {
			t.Errorf("GetDatabaseDSN() = %q, want %q", result, expected)
		}
	})
}

func TestParseDSNForDB_EdgeCases(t *testing.T) {
	t.Run("URL with no query params", func(t *testing.T) {
		dsn := "postgres://user:pass@localhost:5432/testdb"
		dbName, adminDSN, err := parseDSNForDB(dsn)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if dbName != "testdb" {
			t.Errorf("Expected dbName 'testdb', got '%s'", dbName)
		}

		if adminDSN != "postgres://user:pass@localhost:5432/postgres" {
			t.Errorf("Expected adminDSN 'postgres://user:pass@localhost:5432/postgres', got '%s'", adminDSN)
		}
	})

	t.Run("URL format too short", func(t *testing.T) {
		dsn := "postgres://user"
		_, _, err := parseDSNForDB(dsn)

		if err == nil {
			t.Error("Expected error for invalid URL format")
		}
	})
}
