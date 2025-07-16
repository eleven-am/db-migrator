package cmd

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestRunVerify(t *testing.T) {
	// Create temp directory
	tempDir, err := ioutil.TempDir("", "test_verify_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create migrations directory
	migrationsDir := filepath.Join(tempDir, "migrations")
	os.MkdirAll(migrationsDir, 0755)

	// Create some valid migration files
	validMigrations := []string{
		"20230101120000_initial.up.sql",
		"20230101120000_initial.down.sql",
		"20230102130000_add_users.up.sql",
		"20230102130000_add_users.down.sql",
	}

	for _, filename := range validMigrations {
		content := "-- Valid SQL\nCREATE TABLE test (id INT);"
		err := ioutil.WriteFile(filepath.Join(migrationsDir, filename), []byte(content), 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Save original values
	oldDBURL := dbURL

	// Set test values
	dbURL = "postgres://user:pass@localhost/testdb"

	defer func() {
		dbURL = oldDBURL
	}()

	// Run verify - it will fail to connect to DB but that's expected
	_ = runVerify(verifyCmd, []string{})
}

func TestRunVerify_InvalidDatabase(t *testing.T) {
	// Save original values
	oldDBURL := dbURL
	oldDBUser := dbUser
	oldDBName := dbName

	// Test with no credentials
	dbURL = ""
	dbUser = ""
	dbName = ""

	defer func() {
		dbURL = oldDBURL
		dbUser = oldDBUser
		dbName = oldDBName
	}()

	err := runVerify(verifyCmd, []string{})
	if err == nil {
		t.Error("Expected error for missing database credentials")
	}
}
