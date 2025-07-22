# Storm Logging System

Storm now includes a comprehensive logging system that provides detailed visibility into operations while maintaining clean output for normal usage.

## Quick Start

### Using Verbose Mode

```bash
# Default mode - only warnings and errors
storm migrate

# Debug mode - includes info messages  
storm migrate --debug

# Verbose mode - includes all debug messages
storm migrate --verbose
```

## Log Levels

- **Debug**: Detailed technical information (only with `--verbose`)
- **Info**: High-level operation status (with `--debug` or `--verbose`)
- **Warn**: Warnings and recoverable issues (always shown)
- **Error**: Errors that need attention (always shown)

## Example Output

### Default Mode (Warnings/Errors only)
```
[15:04:05] WARN Unknown Go type 'CustomType', defaulting to TEXT component=schema
[15:04:05] ERROR Failed to execute DDL: syntax error at line 5 component=atlas
```

### Debug Mode (`--debug`)
```
[15:04:05] INFO Initializing Storm migration engine... component=cli
[15:04:05] INFO Generating migration... component=cli
[15:04:05] WARN Unknown Go type 'CustomType', defaulting to TEXT component=schema
[15:04:05] INFO Migration files generated successfully component=cli
```

### Verbose Mode (`--verbose`)
```
[15:04:05] DEBUG Using database URL: postgres://user:pass@localhost/db component=cli
[15:04:05] DEBUG Models package: ./models component=cli
[15:04:05] INFO Initializing Storm migration engine... component=cli
[15:04:05] DEBUG Pinging database to verify connection... component=cli
[15:04:05] DEBUG Starting schema generation for 5 tables component=sql
[15:04:05] DEBUG Processing table users with 8 columns component=sql
[15:04:05] DEBUG Processing unique constraint definition: uq_users_email component=schema
[15:04:05] DEBUG Generated UNIQUE constraint: CONSTRAINT uq_users_email UNIQUE (email) component=sql
[15:04:05] INFO Migration files generated successfully component=cli
```

## Component-Specific Logging

Different components log with their own context:

- `component=cli` - CLI operations
- `component=schema` - Schema generation
- `component=sql` - SQL generation
- `component=atlas` - Atlas migration operations
- `component=db` - Database operations
- `component=migration` - Migration processing
- `component=orm` - ORM generation
- `component=parser` - Code parsing

## Benefits

1. **Progressive Detail**: Choose how much information you want to see
2. **Debugging Made Easy**: Verbose mode shows exactly what Storm is doing
3. **Clean Default Output**: Normal usage isn't cluttered with debug info
4. **Contextual Information**: Each log entry shows which component generated it
5. **Colored Output**: Different colors for different levels make logs easy to scan

## Use Cases

### Debugging Migration Issues
```bash
storm migrate --verbose
# Shows detailed SQL generation, constraint processing, and Atlas operations
```

### Monitoring Production Migrations
```bash
storm migrate --debug
# Shows high-level progress without overwhelming detail
```

### Normal Development
```bash
storm migrate
# Clean output with only warnings and errors
```

## Implementation Details

The logging system is implemented in `internal/logger` and provides:

- Global logger instance
- Component-specific loggers
- Structured logging with fields
- Progress indicators for long operations
- Automatic integration with CLI flags

See `internal/logger/README.md` for technical details and API documentation.