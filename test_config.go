package main

import (
	"os"
	"testing"
)

// TestMain sets up and tears down test environment
func TestMain(m *testing.M) {
	// Setup test environment
	setupTestEnvironment()
	
	// Run tests
	code := m.Run()
	
	// Cleanup test environment
	cleanupTestEnvironment()
	
	os.Exit(code)
}

func setupTestEnvironment() {
	// Set test environment variables
	os.Setenv("TEST_MODE", "true")
	os.Setenv("LOG_LEVEL", "debug")
	
	// Any other global test setup
}

func cleanupTestEnvironment() {
	// Cleanup any global test resources
	os.Unsetenv("TEST_MODE")
	os.Unsetenv("LOG_LEVEL")
}