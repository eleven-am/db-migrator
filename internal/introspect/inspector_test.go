package introspect

import (
	"context"
	"database/sql"
	"testing"
)

func TestNewInspector(t *testing.T) {
	// Mock database connection (would be nil in unit test)
	var db *sql.DB

	inspector := NewInspector(db, "postgres")

	if inspector == nil {
		t.Fatal("Expected inspector to be created")
	}

	if inspector.driver != "postgres" {
		t.Errorf("Expected driver to be 'postgres', got %s", inspector.driver)
	}
}

func TestInspector_UnsupportedDriver(t *testing.T) {
	var db *sql.DB
	inspector := NewInspector(db, "mysql")

	ctx := context.Background()

	_, err := inspector.GetSchema(ctx)
	if err == nil {
		t.Error("Expected error for unsupported driver")
	}
	if err.Error() != "unsupported database driver: mysql" {
		t.Errorf("Unexpected error message: %v", err)
	}

	_, err = inspector.GetTable(ctx, "public", "users")
	if err == nil {
		t.Error("Expected error for unsupported driver")
	}

	_, err = inspector.GetTables(ctx)
	if err == nil {
		t.Error("Expected error for unsupported driver")
	}

	_, err = inspector.GetDatabaseMetadata(ctx)
	if err == nil {
		t.Error("Expected error for unsupported driver")
	}

	_, err = inspector.GetEnums(ctx)
	if err == nil {
		t.Error("Expected error for unsupported driver")
	}

	_, err = inspector.GetFunctions(ctx)
	if err == nil {
		t.Error("Expected error for unsupported driver")
	}

	_, err = inspector.GetSequences(ctx)
	if err == nil {
		t.Error("Expected error for unsupported driver")
	}

	_, err = inspector.GetViews(ctx)
	if err == nil {
		t.Error("Expected error for unsupported driver")
	}

	_, err = inspector.GetTableStatistics(ctx, "public", "users")
	if err == nil {
		t.Error("Expected error for unsupported driver")
	}
}
