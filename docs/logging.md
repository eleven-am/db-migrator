# Storm Logging System

Storm includes a comprehensive logging system that provides detailed insights into the migration and code generation process.

## Usage

### Command Line Flags

Storm supports two logging-related flags:

- `--verbose`: Enables debug-level logging, showing detailed information about every operation
- `--debug`: Enables info-level logging, showing important operations without overwhelming detail

```bash
# Normal operation (warnings and errors only)
storm migrate

# Verbose mode (all debug information)
storm migrate --verbose

# Debug mode (info, warnings, and errors)
storm migrate --debug
```

## Log Levels

The logging system supports five levels:

1. **DEBUG**: Most detailed level, includes all internal operations
2. **INFO**: Important operations and status updates
3. **WARN**: Warning messages that don't prevent operation
4. **ERROR**: Error messages for failures
5. **SILENT**: No output except fatal errors

## Log Output Format

Logs are formatted with timestamps and color-coded levels:

```
[15:04:05] DEBUG Processing unique constraint definition: uk_user_email
[15:04:05] INFO Starting schema generation for 5 tables
[15:04:05] WARN Unknown Go type 'CustomType', defaulting to TEXT
[15:04:05] ERROR Failed to execute DDL: syntax error at line 42
```

## Component-Specific Logging

Different components log with their own context:

- `schema`: Schema generation operations
- `sql`: SQL DDL generation
- `atlas`: Atlas migration operations
- `migration`: General migration operations
- `db`: Database connection and operations
- `orm`: ORM code generation
- `parser`: Struct and tag parsing
- `cli`: CLI command processing

## Examples

### Verbose Migration Output

```bash
$ storm migrate --verbose
[15:04:05] DEBUG component=cli Loaded config from storm.yaml
[15:04:05] DEBUG component=cli Using database URL from config: postgres://localhost/mydb
[15:04:05] DEBUG component=schema Processing unique constraint definition: uk_user_email
[15:04:05] DEBUG component=sql Starting schema generation for 5 tables
[15:04:05] DEBUG component=sql Processing table users with 8 columns
[15:04:05] DEBUG component=sql Generated UNIQUE constraint: CONSTRAINT uk_user_email UNIQUE (email)
[15:04:05] DEBUG component=atlas DDL uses CUID functions, creating them in temp database
[15:04:05] DEBUG component=atlas CUID functions created successfully
[15:04:05] INFO Migration completed successfully
```

### Normal Output

```bash
$ storm migrate
[15:04:05] INFO Migration completed successfully
```

### Error Output

```bash
$ storm migrate
[15:04:05] ERROR Failed to connect to database: connection refused
[15:04:05] ERROR Migration failed: unable to establish database connection
```

## Progress Indicators

For long-running operations, Storm shows progress indicators:

```
⏳ Generating schema...✅
⏳ Creating migration...✅
⏳ Applying migration...❌
```

## Programmatic Usage

The logging system can also be used programmatically:

```go
import "github.com/eleven-am/storm/internal/logger"

// Set global log level
logger.SetVerbose(true)

// Log with different levels
logger.Debug("Processing table %s", tableName)
logger.Info("Migration started")
logger.Warn("Column type unknown: %s", colType)
logger.Error("Failed to parse: %v", err)

// Component-specific logging
logger.Schema().Debug("Parsing struct tags")
logger.SQL().Info("Generating DDL")
logger.Atlas().Error("Migration failed: %v", err)

// Structured logging
logger.WithField("table", "users").
    WithField("columns", 8).
    Info("Processing table")

// Progress indicators
logger.StartProgress("Generating migration")
// ... do work ...
logger.EndProgress(true) // ✅ or ❌ based on success
```

## Configuration

The logging level can also be controlled through the storm.yaml configuration:

```yaml
# storm.yaml
logging:
  level: debug  # debug, info, warn, error, silent
  format: text  # currently only text is supported
```

## Best Practices

1. Use `--verbose` when debugging issues or wanting to understand what Storm is doing
2. Use `--debug` for normal development to see important operations
3. In production, use default settings to only see warnings and errors
4. Check logs when migrations fail to understand what went wrong
5. Component-specific logs help identify which part of the system has issues