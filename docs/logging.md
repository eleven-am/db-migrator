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

## ORM Query Logging

Storm ORM supports optional query logging to help debug and monitor SQL queries executed by your application.

### Basic Usage

```go
// Without logging (default behavior)
storm := models.NewStorm(db)

// With simple built-in logger
logger := &storm.SimpleQueryLogger{}
storm := models.NewStorm(db, logger)
```

### QueryLogger Interface

Implement the `QueryLogger` interface to create custom loggers:

```go
type QueryLogger interface {
    LogQuery(query string, args []interface{}, duration time.Duration, err error)
}
```

### Built-in SimpleQueryLogger

Storm provides a basic logger that outputs to stdout:

```go
logger := &storm.SimpleQueryLogger{}
storm := models.NewStorm(db, logger)

// Output format:
// [SQL] [2.3ms] [SUCCESS] SELECT * FROM users WHERE id = $1 [123]
// [SQL] [1.1ms] [ERROR: no rows in result set] SELECT * FROM users WHERE id = $1 [999]
```

### Using with slog (Structured Logging)

For production applications, integrate with Go's structured logging:

```go
import "log/slog"

type SlogQueryLogger struct {
    logger *slog.Logger
}

func (s *SlogQueryLogger) LogQuery(query string, args []interface{}, duration time.Duration, err error) {
    attrs := []slog.Attr{
        slog.String("query", query),
        slog.Duration("duration", duration),
        slog.Any("args", args),
    }
    
    if err != nil {
        attrs = append(attrs, slog.String("error", err.Error()))
        s.logger.LogAttrs(nil, slog.LevelError, "SQL query failed", attrs...)
    } else {
        s.logger.LogAttrs(nil, slog.LevelInfo, "SQL query executed", attrs...)
    }
}

// Usage
jsonLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
queryLogger := &SlogQueryLogger{logger: jsonLogger}
storm := models.NewStorm(db, queryLogger)
```

### Custom Logger Examples

#### Performance Monitoring Logger
```go
type PerformanceLogger struct {
    slowQueryThreshold time.Duration
}

func (p *PerformanceLogger) LogQuery(query string, args []interface{}, duration time.Duration, err error) {
    if duration > p.slowQueryThreshold {
        log.Printf("SLOW QUERY [%v]: %s", duration, query)
    }
    if err != nil {
        log.Printf("QUERY ERROR [%v]: %s - %v", duration, query, err)
    }
}
```

#### Metrics Logger
```go
type MetricsLogger struct {
    queryCounter prometheus.Counter
    queryDuration prometheus.Histogram
}

func (m *MetricsLogger) LogQuery(query string, args []interface{}, duration time.Duration, err error) {
    m.queryCounter.Inc()
    m.queryDuration.Observe(duration.Seconds())
    
    if err != nil {
        // Record error metrics
    }
}
```

### Transaction Support

Query logging works seamlessly with transactions:

```go
storm := models.NewStorm(db, logger)

err := storm.WithTransaction(ctx, func(txStorm *Storm) error {
    // All queries within this transaction will be logged
    user, err := txStorm.Users().FindByID(ctx, 123)
    // ... more operations
    return nil
})
```

### Configuration Tips

1. **Development**: Use `SimpleQueryLogger` or slog with text handler for readable output
2. **Production**: Use structured logging (slog with JSON) for better monitoring
3. **Performance**: Implement sampling or filtering in custom loggers for high-traffic applications
4. **Security**: Be careful logging query arguments that might contain sensitive data

### Output Examples

**SimpleQueryLogger Output:**
```
[SQL] [1.2ms] [SUCCESS] SELECT id, name, email FROM users WHERE id = $1 [123]
[SQL] [0.8ms] [SUCCESS] INSERT INTO users (name, email) VALUES ($1, $2) [John Doe john@example.com]
[SQL] [2.1ms] [ERROR: duplicate key value violates unique constraint] INSERT INTO users (email) VALUES ($1) [john@example.com]
```

**Structured JSON Output:**
```json
{"time":"2024-01-15T10:30:45Z","level":"INFO","msg":"SQL query executed","query":"SELECT * FROM users WHERE id = $1","duration":"1.2ms","args":[123]}
{"time":"2024-01-15T10:30:46Z","level":"ERROR","msg":"SQL query failed","query":"INSERT INTO users (email) VALUES ($1)","duration":"0.9ms","args":["john@example.com"],"error":"duplicate key value"}
```

## Best Practices

1. Use `--verbose` when debugging issues or wanting to understand what Storm is doing
2. Use `--debug` for normal development to see important operations
3. In production, use default settings to only see warnings and errors
4. Check logs when migrations fail to understand what went wrong
5. Component-specific logs help identify which part of the system has issues
6. **Query Logging**: Enable query logging in development to debug SQL issues, use structured logging in production
7. **Performance**: Monitor query durations to identify slow queries and optimization opportunities