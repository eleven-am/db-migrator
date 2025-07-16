package cli

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestNewRootCommand(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "storm_root_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldCwd)
	os.Chdir(tempDir)

	t.Run("creates root command", func(t *testing.T) {
		cmd := NewRootCommand()
		if cmd == nil {
			t.Fatal("NewRootCommand returned nil")
		}

		if cmd.Use != "storm" {
			t.Errorf("expected Use to be 'storm', got %s", cmd.Use)
		}

		if cmd.Short != "Storm - Unified Database Toolkit" {
			t.Errorf("expected Short to be 'Storm - Unified Database Toolkit', got %s", cmd.Short)
		}

		if cmd.Version == "" {
			t.Error("expected Version to be set")
		}
	})

	t.Run("has expected subcommands", func(t *testing.T) {
		cmd := NewRootCommand()

		expectedCommands := []string{
			"init",
			"migrate",
			"create",
			"generate",
			"verify",
			"introspect",
			"version",
			"orm",
		}

		for _, expectedCmd := range expectedCommands {
			found := false
			for _, subCmd := range cmd.Commands() {
				if subCmd.Name() == expectedCmd {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected command %s not found", expectedCmd)
			}
		}
	})

	t.Run("has expected flags", func(t *testing.T) {
		cmd := NewRootCommand()

		expectedFlags := []string{
			"config",
			"url",
			"debug",
			"verbose",
		}

		for _, expectedFlag := range expectedFlags {
			flag := cmd.PersistentFlags().Lookup(expectedFlag)
			if flag == nil {
				t.Errorf("expected flag %s not found", expectedFlag)
			}
		}
	})

	t.Run("persistent pre-run with valid config", func(t *testing.T) {
		// Create a valid config file
		configContent := `version: "1.0"
project: "test-project"
database:
  url: "postgres://localhost:5432/test"
schema:
  strict_mode: true
`
		configFile := filepath.Join(tempDir, "valid.yaml")
		err := ioutil.WriteFile(configFile, []byte(configContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		cmd := NewRootCommand()
		cmd.SetArgs([]string{"--config", configFile, "--verbose", "version"})

		// Capture output
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		// Execute the command
		err = cmd.Execute()
		if err != nil {
			t.Fatalf("command execution failed: %v", err)
		}

		// Verify config was loaded
		if stormConfig == nil {
			t.Error("expected stormConfig to be loaded")
		} else {
			if stormConfig.Project != "test-project" {
				t.Errorf("expected project test-project, got %s", stormConfig.Project)
			}
		}
	})

	t.Run("persistent pre-run with invalid config", func(t *testing.T) {
		// Create an invalid config file
		configContent := `invalid: yaml: content:
  - bad
    - format
`
		configFile := filepath.Join(tempDir, "invalid.yaml")
		err := ioutil.WriteFile(configFile, []byte(configContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		cmd := NewRootCommand()
		cmd.SetArgs([]string{"--config", configFile, "--verbose", "version"})

		// Capture output
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		// Execute the command
		err = cmd.Execute()
		if err != nil {
			t.Fatalf("command execution failed: %v", err)
		}

		// Check that warning was displayed
		output := buf.String()
		if !contains(output, "Warning: Failed to load config file") {
			t.Error("expected warning about failed config loading")
		}
	})

	t.Run("persistent pre-run with non-existent config", func(t *testing.T) {
		cmd := NewRootCommand()
		cmd.SetArgs([]string{"--config", "/non/existent/config.yaml", "--verbose", "version"})

		// Capture output
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		// Execute the command
		err = cmd.Execute()
		if err != nil {
			t.Fatalf("command execution failed: %v", err)
		}

		// Check that warning was displayed
		output := buf.String()
		if !contains(output, "Warning: Failed to load config file") {
			t.Error("expected warning about failed config loading")
		}
	})

	t.Run("database URL override", func(t *testing.T) {
		// Create a config with database URL
		configContent := `version: "1.0"
project: "test-project"
database:
  url: "postgres://localhost:5432/config"
`
		configFile := filepath.Join(tempDir, "url_test.yaml")
		err := ioutil.WriteFile(configFile, []byte(configContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		cmd := NewRootCommand()
		cmd.SetArgs([]string{"--config", configFile, "--url", "postgres://localhost:5432/override", "version"})

		// Execute the command
		err = cmd.Execute()
		if err != nil {
			t.Fatalf("command execution failed: %v", err)
		}

		// Verify URL was overridden
		if databaseURL != "postgres://localhost:5432/override" {
			t.Errorf("expected database URL to be overridden, got %s", databaseURL)
		}
	})

	t.Run("database URL from config", func(t *testing.T) {
		// Reset global variables
		databaseURL = ""
		stormConfig = nil

		// Create a config with database URL
		configContent := `version: "1.0"
project: "test-project"
database:
  url: "postgres://localhost:5432/fromconfig"
`
		configFile := filepath.Join(tempDir, "url_config.yaml")
		err := ioutil.WriteFile(configFile, []byte(configContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		cmd := NewRootCommand()
		cmd.SetArgs([]string{"--config", configFile, "version"})

		// Execute the command
		err = cmd.Execute()
		if err != nil {
			t.Fatalf("command execution failed: %v", err)
		}

		// Verify URL was set from config
		if databaseURL != "postgres://localhost:5432/fromconfig" {
			t.Errorf("expected database URL from config, got %s", databaseURL)
		}
	})

	t.Run("debug and verbose flags", func(t *testing.T) {
		// Reset global variables
		debug = false
		verbose = false

		cmd := NewRootCommand()
		cmd.SetArgs([]string{"--debug", "--verbose", "version"})

		// Execute the command
		err = cmd.Execute()
		if err != nil {
			t.Fatalf("command execution failed: %v", err)
		}

		// Verify flags were set
		if !debug {
			t.Error("expected debug flag to be set")
		}
		if !verbose {
			t.Error("expected verbose flag to be set")
		}
	})
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && containsHelper(s, substr)
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
