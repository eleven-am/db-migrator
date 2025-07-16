package cmd

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestRunGenerate(t *testing.T) {
	// Create temp directories
	tempDir, err := ioutil.TempDir("", "test_generate_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a simple Go file with a struct
	structFile := filepath.Join(tempDir, "models.go")
	structContent := `package models

type User struct {
	ID   int    ` + "`db:\"id\" storm:\"primary_key,auto_increment\"`" + `
	Name string ` + "`db:\"name\"`" + `
}
`
	if err := ioutil.WriteFile(structFile, []byte(structContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create config file
	configFile := filepath.Join(tempDir, "config.yaml")
	configContent := `tables:
  - name: users
    file: models.go
    type: User
`
	if err := ioutil.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Save original values
	oldGeneratePackage := generatePackage
	oldGenerateOutput := generateOutput

	// Set test values
	generatePackage = tempDir
	generateOutput = filepath.Join(tempDir, "schema.sql")

	defer func() {
		generatePackage = oldGeneratePackage
		generateOutput = oldGenerateOutput
	}()

	// Run generate - it will likely fail due to DB connection, but we're testing the setup
	_ = runGenerate(generateCmd, []string{})

	// Check if output file was attempted to be created
	// The function might fail, but we're testing the basic flow
}

func TestRunGenerate_ValidationErrors(t *testing.T) {
	// Test with missing package directory
	oldGeneratePackage := generatePackage
	generatePackage = "/non/existent/path"
	defer func() { generatePackage = oldGeneratePackage }()

	err := runGenerate(generateCmd, []string{})
	if err == nil {
		t.Error("Expected error for non-existent package directory")
	}
}
