package storm

import (
	"testing"

	"github.com/eleven-am/storm/pkg/storm"
	"github.com/jmoiron/sqlx"
)

func TestBuildMigrator(t *testing.T) {
	// This is a basic test to ensure the builder works
	mockDB := &sqlx.DB{} // In real tests, this would be properly mocked
	config := &storm.Config{
		ModelsPackage: "./models",
	}
	logger := &TestLogger{}

	migrator := BuildMigrator(mockDB, config, logger)
	if migrator == nil {
		t.Fatal("expected migrator, got nil")
	}
}

func TestBuildORM(t *testing.T) {
	config := &storm.Config{
		ModelsPackage: "./models",
	}
	logger := &TestLogger{}

	orm := BuildORM(config, logger)
	if orm == nil {
		t.Fatal("expected ORM, got nil")
	}
}

func TestBuildSchemaInspector(t *testing.T) {
	// This is a basic test to ensure the builder works
	mockDB := &sqlx.DB{} // In real tests, this would be properly mocked
	config := &storm.Config{
		ModelsPackage: "./models",
	}
	logger := &TestLogger{}

	inspector := BuildSchemaInspector(mockDB, config, logger)
	if inspector == nil {
		t.Fatal("expected schema inspector, got nil")
	}
}

func TestFactoryFunctions(t *testing.T) {
	config := &storm.Config{
		ModelsPackage: "./models",
	}
	logger := &TestLogger{}

	t.Run("BuildMigrator", func(t *testing.T) {
		mockDB := &sqlx.DB{}
		result := BuildMigrator(mockDB, config, logger)
		if result == nil {
			t.Error("expected non-nil migrator")
		}
	})

	t.Run("BuildORM", func(t *testing.T) {
		result := BuildORM(config, logger)
		if result == nil {
			t.Error("expected non-nil ORM")
		}
	})

	t.Run("BuildSchemaInspector", func(t *testing.T) {
		mockDB := &sqlx.DB{}
		result := BuildSchemaInspector(mockDB, config, logger)
		if result == nil {
			t.Error("expected non-nil schema inspector")
		}
	})
}

func TestBuilderIntegration(t *testing.T) {
	config := &storm.Config{
		ModelsPackage: "./models",
	}
	logger := &TestLogger{}
	mockDB := &sqlx.DB{}

	// Test that all builders can be created together
	migrator := BuildMigrator(mockDB, config, logger)
	orm := BuildORM(config, logger)
	inspector := BuildSchemaInspector(mockDB, config, logger)

	if migrator == nil {
		t.Error("expected migrator to be created")
	}
	if orm == nil {
		t.Error("expected ORM to be created")
	}
	if inspector == nil {
		t.Error("expected schema inspector to be created")
	}
}

// Helper types for testing

type TestLogger struct{}

func (l *TestLogger) Debug(msg string, fields ...interface{}) {}
func (l *TestLogger) Info(msg string, fields ...interface{})  {}
func (l *TestLogger) Warn(msg string, fields ...interface{})  {}
func (l *TestLogger) Error(msg string, fields ...interface{}) {}
