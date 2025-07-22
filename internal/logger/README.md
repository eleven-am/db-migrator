# Storm Logger

A comprehensive logging system for the Storm database toolkit with support for multiple log levels, structured logging, and progress indicators.

## Features

- **Multiple Log Levels**: Debug, Info, Warn, Error, Fatal
- **Colored Output**: Different colors for different log levels
- **Structured Logging**: Add fields to log entries for better context
- **Progress Indicators**: Show progress for long-running operations
- **Component-Specific Loggers**: Pre-configured loggers for different components

## Usage

### Basic Logging

```go
import "github.com/eleven-am/storm/internal/logger"

// Simple logging
logger.Debug("Starting operation")
logger.Info("Processing table: %s", tableName)
logger.Warn("Deprecated feature used")
logger.Error("Failed to connect: %v", err)

// With fields
logger.WithField("table", "users").Info("Processing table")
logger.WithFields(map[string]interface{}{
    "table": "users",
    "rows": 1000,
}).Debug("Table statistics")
```

### Component-Specific Logging

```go
// Use pre-configured component loggers
logger.Schema().Debug("Generating schema for table: %s", tableName)
logger.SQL().Info("Generated %d SQL statements", count)
logger.Migration().Warn("Destructive change detected")
logger.Atlas().Debug("Atlas operation completed")
```

### Progress Indicators

```go
// Show progress for long operations
logger.StartProgress("Generating migrations")
// ... do work ...
logger.UpdateProgress("Processing table users")
// ... do more work ...
logger.EndProgress(true) // true for success, false for failure
```

### Integration with CLI

The logger automatically integrates with the CLI flags:

- `--verbose`: Sets log level to Debug
- `--debug`: Sets log level to Info (less verbose than --verbose)
- Default: Warn level (only warnings and errors)

### Log Levels

1. **Debug**: Detailed information for debugging
2. **Info**: General informational messages
3. **Warn**: Warning messages for potentially problematic situations
4. **Error**: Error messages for failures that don't stop execution
5. **Fatal**: Critical errors that cause the program to exit

### Example Output

```
[15:04:05] DEBUG Starting schema generation component=schema
[15:04:05] INFO Processing table: users component=sql
[15:04:05] WARN Unknown Go type 'CustomType', defaulting to TEXT component=schema
[15:04:05] ERROR Failed to execute DDL: pq: syntax error component=atlas
```

## Best Practices

1. Use component-specific loggers for better context
2. Use Debug level for detailed technical information
3. Use Info level for high-level operation status
4. Use Warn for deprecations or recoverable issues
5. Use Error for failures that need attention
6. Add structured fields for better searchability
7. Keep log messages concise but informative