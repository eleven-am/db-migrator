package main

import (
	"github.com/eleven-am/storm/internal/logger"
)

func main() {
	// Example 1: Basic logging at different levels
	logger.Debug("This is a debug message - only shown with --verbose")
	logger.Info("This is an info message - shown with --debug or --verbose")
	logger.Warn("This is a warning - always shown unless silent")
	logger.Error("This is an error - always shown")
	
	// Example 2: Component-specific logging
	logger.Schema().Debug("Processing table schema")
	logger.SQL().Info("Generated CREATE TABLE statement")
	logger.Migration().Warn("Found potentially destructive change")
	logger.Atlas().Error("Failed to connect to database")
	
	// Example 3: Structured logging with fields
	logger.WithField("table", "users").Info("Processing table")
	logger.WithFields(map[string]interface{}{
		"table": "users",
		"columns": 5,
		"constraints": 2,
	}).Debug("Table details")
	
	// Example 4: Progress indicators
	logger.StartProgress("Generating migrations")
	// Simulate work...
	logger.UpdateProgress("Processing table: users")
	// More work...
	logger.UpdateProgress("Processing table: posts")
	// Done!
	logger.EndProgress(true)
	
	// Example 5: Different verbosity levels
	// Run with: go run logging_demo.go (default - only warnings/errors)
	// Run with: go run logging_demo.go --debug (info level and above)
	// Run with: go run logging_demo.go --verbose (all messages)
}