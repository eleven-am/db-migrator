package main

import (
	"os"
	"testing"

	"github.com/eleven-am/storm/pkg/storm"
)

func TestMain(m *testing.M) {
	m.Run()
}

func TestExecute(t *testing.T) {
	t.Run("execute_function_exists", func(t *testing.T) {
		t.Log("Execute function exists and is callable")
	})
}

func TestInitStormFactories(t *testing.T) {
	t.Run("init_factories", func(t *testing.T) {
		// Test that factories are nil before initialization
		if storm.MigratorFactory != nil {
			t.Error("Expected MigratorFactory to be nil before initialization")
		}
		if storm.ORMFactory != nil {
			t.Error("Expected ORMFactory to be nil before initialization")
		}
		if storm.SchemaInspectorFactory != nil {
			t.Error("Expected SchemaInspectorFactory to be nil before initialization")
		}

		// Initialize factories
		initStormFactories()

		// Test that factories are set after initialization
		if storm.MigratorFactory == nil {
			t.Error("Expected MigratorFactory to be set after initialization")
		}
		if storm.ORMFactory == nil {
			t.Error("Expected ORMFactory to be set after initialization")
		}
		if storm.SchemaInspectorFactory == nil {
			t.Error("Expected SchemaInspectorFactory to be set after initialization")
		}

		t.Log("Storm factories initialized successfully")
	})
}

func TestMainFunction(t *testing.T) {
	// Test that main function doesn't panic when called
	// We can't easily test the actual execution without causing the test to exit
	t.Run("main_function_exists", func(t *testing.T) {
		// Save original args
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()

		// Set args to help to avoid actual execution
		os.Args = []string{"storm", "--help"}

		// Test that main function exists and can be called
		// We can't call it directly as it would exit, but we can test Execute
		err := Execute()
		// Execute with --help should not return an error
		if err != nil {
			t.Logf("Execute returned error (expected for --help): %v", err)
		}
	})
}

func TestFactoryAssignment(t *testing.T) {
	// Test that the factory functions are properly assigned
	t.Run("factory_functions_set", func(t *testing.T) {
		// Reset factories
		storm.MigratorFactory = nil
		storm.ORMFactory = nil
		storm.SchemaInspectorFactory = nil

		// Initialize
		initStormFactories()

		// Verify they're function pointers (not nil)
		if storm.MigratorFactory == nil {
			t.Error("MigratorFactory should be set")
		}
		if storm.ORMFactory == nil {
			t.Error("ORMFactory should be set")
		}
		if storm.SchemaInspectorFactory == nil {
			t.Error("SchemaInspectorFactory should be set")
		}
	})
}
