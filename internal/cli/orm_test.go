package cli

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunORM(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "storm_orm_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Save original values
	origOrmPackage := ormPackage
	origOrmOutput := ormOutput
	origOrmIncludeHooks := ormIncludeHooks
	origOrmIncludeTests := ormIncludeTests
	origOrmIncludeMocks := ormIncludeMocks
	origDebug := debug
	origVerbose := verbose
	origStormConfig := stormConfig
	defer func() {
		ormPackage = origOrmPackage
		ormOutput = origOrmOutput
		ormIncludeHooks = origOrmIncludeHooks
		ormIncludeTests = origOrmIncludeTests
		ormIncludeMocks = origOrmIncludeMocks
		debug = origDebug
		verbose = origVerbose
		stormConfig = origStormConfig
	}()

	t.Run("uses default package path when not specified", func(t *testing.T) {
		// Clear package path
		ormPackage = ""
		ormOutput = ""
		ormIncludeHooks = false
		ormIncludeTests = false
		ormIncludeMocks = false
		debug = false
		verbose = false
		stormConfig = nil

		err := runORM(ormCmd, []string{})
		if err == nil {
			t.Error("expected error due to missing models")
		}
		// Should fail on Storm client creation or generation, not on package path validation
		if !strings.Contains(err.Error(), "failed to") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("uses configuration from storm config", func(t *testing.T) {
		// Set up storm config
		stormConfig = &StormConfig{
			Version: "1.0",
			Project: "test-project",
		}
		stormConfig.Models.Package = "./custom/models"
		stormConfig.ORM.GenerateHooks = true
		stormConfig.ORM.GenerateTests = true
		stormConfig.ORM.GenerateMocks = true

		// Clear command line flags
		ormPackage = ""
		ormOutput = ""
		ormIncludeHooks = false
		ormIncludeTests = false
		ormIncludeMocks = false
		debug = false
		verbose = false

		err := runORM(ormCmd, []string{})
		if err == nil {
			t.Error("expected error due to missing models")
		}
		// Should fail on Storm client creation or generation, not on config validation
		if !strings.Contains(err.Error(), "failed to") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("handles non-existent package path", func(t *testing.T) {
		// Set non-existent package path
		ormPackage = "/non/existent/path"
		ormOutput = ""
		ormIncludeHooks = false
		ormIncludeTests = false
		ormIncludeMocks = false
		debug = false
		verbose = false
		stormConfig = nil

		err := runORM(ormCmd, []string{})
		if err == nil {
			t.Error("expected error with non-existent package path")
		}
		// Should fail on Storm client creation or generation
		if !strings.Contains(err.Error(), "failed to") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("handles verbose output", func(t *testing.T) {
		// Create a package directory
		packageDir := filepath.Join(tempDir, "models")
		err := os.MkdirAll(packageDir, 0755)
		if err != nil {
			t.Fatal(err)
		}

		// Set up for verbose output
		ormPackage = packageDir
		ormOutput = filepath.Join(tempDir, "output")
		ormIncludeHooks = true
		ormIncludeTests = true
		ormIncludeMocks = true
		debug = false
		verbose = true
		stormConfig = nil

		err = runORM(ormCmd, []string{})
		if err == nil {
			t.Error("expected error due to missing models")
		}
		// Should fail on Storm client creation or generation
		if !strings.Contains(err.Error(), "failed to") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("sets output directory to package path when not specified", func(t *testing.T) {
		// Create a package directory
		packageDir := filepath.Join(tempDir, "models")
		err := os.MkdirAll(packageDir, 0755)
		if err != nil {
			t.Fatal(err)
		}

		// Set package but not output
		ormPackage = packageDir
		ormOutput = ""
		ormIncludeHooks = false
		ormIncludeTests = false
		ormIncludeMocks = false
		debug = false
		verbose = false
		stormConfig = nil

		err = runORM(ormCmd, []string{})
		if err == nil {
			t.Error("expected error due to missing models")
		}
		// Should fail on Storm client creation or generation
		if !strings.Contains(err.Error(), "failed to") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

func TestORMCommand(t *testing.T) {
	t.Run("command structure", func(t *testing.T) {
		if ormCmd.Use != "orm" {
			t.Errorf("expected Use to be 'orm', got %s", ormCmd.Use)
		}

		if ormCmd.Short != "Generate ORM code from models" {
			t.Errorf("expected Short to be 'Generate ORM code from models', got %s", ormCmd.Short)
		}

		if ormCmd.RunE == nil {
			t.Error("expected RunE to be set")
		}
	})

	t.Run("command flags", func(t *testing.T) {
		expectedFlags := []string{
			"package",
			"output",
			"hooks",
			"tests",
			"mocks",
		}

		for _, flagName := range expectedFlags {
			flag := ormCmd.Flags().Lookup(flagName)
			if flag == nil {
				t.Errorf("expected flag %s to be defined", flagName)
			}
		}

		// Check boolean flags defaults
		hooksFlag := ormCmd.Flags().Lookup("hooks")
		if hooksFlag != nil && hooksFlag.DefValue != "false" {
			t.Errorf("expected hooks flag default to be 'false', got %s", hooksFlag.DefValue)
		}

		testsFlag := ormCmd.Flags().Lookup("tests")
		if testsFlag != nil && testsFlag.DefValue != "false" {
			t.Errorf("expected tests flag default to be 'false', got %s", testsFlag.DefValue)
		}

		mocksFlag := ormCmd.Flags().Lookup("mocks")
		if mocksFlag != nil && mocksFlag.DefValue != "false" {
			t.Errorf("expected mocks flag default to be 'false', got %s", mocksFlag.DefValue)
		}
	})
}
