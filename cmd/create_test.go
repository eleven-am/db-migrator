package cmd

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunCreate(t *testing.T) {
	// Create temp directory for output
	tempDir, err := ioutil.TempDir("", "test_create_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Save original output dir
	oldOutputDir := outputDir
	outputDir = tempDir
	defer func() { outputDir = oldOutputDir }()

	// Test creating migration files
	err = runCreate(createCmd, []string{"add_users_table"})
	if err != nil {
		t.Fatalf("runCreate failed: %v", err)
	}

	// Check that files were created
	files, err := ioutil.ReadDir(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(files))
	}

	// Check file names pattern
	var upFile, downFile string
	for _, file := range files {
		name := file.Name()
		if strings.HasSuffix(name, "_add_users_table.up.sql") {
			upFile = name
		} else if strings.HasSuffix(name, "_add_users_table.down.sql") {
			downFile = name
		}
	}

	if upFile == "" || downFile == "" {
		t.Fatal("Migration files not found with expected names")
	}

	// Check file contents
	upContent, err := ioutil.ReadFile(filepath.Join(tempDir, upFile))
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(upContent), "Migration: add_users_table") {
		t.Error("UP file doesn't contain expected migration name")
	}

	if !strings.Contains(string(upContent), "Created at:") {
		t.Error("UP file doesn't contain creation timestamp")
	}
}

func TestRunCreate_DirectoryCreation(t *testing.T) {
	// Create temp directory
	tempDir, err := ioutil.TempDir("", "test_create_dir_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Set output to non-existent subdirectory
	outputPath := filepath.Join(tempDir, "nested", "migrations")

	// Save original output dir
	oldOutputDir := outputDir
	outputDir = outputPath
	defer func() { outputDir = oldOutputDir }()

	// Run create command
	err = runCreate(createCmd, []string{"test_migration"})
	if err != nil {
		t.Fatalf("runCreate failed: %v", err)
	}

	// Check that directory was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Output directory was not created")
	}

	// Check that files exist
	files, err := ioutil.ReadDir(outputPath)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}
}

func TestRunCreate_TimestampFormat(t *testing.T) {
	// Create temp directory
	tempDir, err := ioutil.TempDir("", "test_timestamp_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Save original output dir
	oldOutputDir := outputDir
	outputDir = tempDir
	defer func() { outputDir = oldOutputDir }()

	// Record time before creation
	timeBefore := time.Now().UTC()

	// Create migration
	err = runCreate(createCmd, []string{"timestamp_test"})
	if err != nil {
		t.Fatalf("runCreate failed: %v", err)
	}

	// Check files
	files, err := ioutil.ReadDir(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Verify timestamp format (YYYYMMDDHHMMSS)
	for _, file := range files {
		name := file.Name()
		// Extract timestamp part (first 14 characters)
		if len(name) >= 14 {
			timestamp := name[:14]
			// Try to parse it
			_, err := time.Parse("20060102150405", timestamp)
			if err != nil {
				t.Errorf("Invalid timestamp format in filename %s: %v", name, err)
			}

			// Parse and check it's close to current time
			parsedTime, _ := time.Parse("20060102150405", timestamp)
			if parsedTime.Before(timeBefore.Add(-time.Minute)) || parsedTime.After(time.Now().UTC().Add(time.Minute)) {
				t.Error("Timestamp is not within expected range")
			}
		}
	}
}
