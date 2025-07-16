# CLI Reference

Complete reference for all Storm CLI commands and options.

## Global Flags

These flags are available for all commands:

```bash
storm [global flags] <command> [command flags]
```

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--config` | `-c` | Path to configuration file | `storm.yaml` |
| `--url` | | Database connection URL | From config |
| `--debug` | | Enable debug output | `false` |
| `--verbose` | `-v` | Enable verbose output | `false` |
| `--help` | `-h` | Show help | |
| `--version` | | Show version | |

## Commands

### storm init

Initialize a new Storm configuration file.

```bash
storm init [flags]
```

**Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--project` | Project name | Current directory name |
| `--driver` | Database driver | `postgres` |
| `--force` | Overwrite existing config | `false` |

**Examples:**
```bash
# Create default configuration
storm init

# Create with custom project name
storm init --project myapp

# Overwrite existing configuration
storm init --force
```

### storm migrate

Generate database migrations by comparing Go structs with database schema.

```bash
storm migrate [flags]
```

**Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--package` | Path to models package | `./models` |
| `--output` | Output directory | `./migrations` |
| `--name` | Migration name | Auto-generated |
| `--dry-run` | Print SQL without creating files | `false` |
| `--push` | Apply migration to database | `false` |
| `--allow-destructive` | Allow destructive operations | `false` |
| `--create-if-not-exists` | Create database if missing | `false` |

**Database Connection Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--host` | Database host | `localhost` |
| `--port` | Database port | `5432` |
| `--user` | Database user | |
| `--password` | Database password | |
| `--dbname` | Database name | |
| `--sslmode` | SSL mode | `disable` |

**Examples:**
```bash
# Generate migration using config file
storm migrate

# Generate with custom name
storm migrate --name add_user_roles

# Generate and review without creating files
storm migrate --dry-run

# Generate and immediately apply
storm migrate --push

# Allow dropping columns/tables
storm migrate --allow-destructive

# Use specific database connection
storm migrate \
  --user postgres \
  --password secret \
  --dbname myapp \
  --host localhost
```

### storm orm

Generate ORM code from model definitions.

```bash
storm orm [flags]
```

**Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--package` | Path to models package | `./models` |
| `--output` | Output directory | Same as package |
| `--hooks` | Generate lifecycle hooks | `true` |
| `--tests` | Generate test files | `false` |
| `--mocks` | Generate mock implementations | `false` |

**Examples:**
```bash
# Generate ORM code with defaults
storm orm

# Generate with tests and mocks
storm orm --tests --mocks

# Generate to different directory
storm orm --output ./generated

# Skip hooks generation
storm orm --hooks=false
```

### storm create

Create various Storm-related files.

```bash
storm create <type> <name> [flags]
```

**Types:**
- `model` - Create a new model file
- `migration` - Create an empty migration file

**Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--package` | Package name | From config |
| `--output` | Output directory | From config |

**Examples:**
```bash
# Create a new model
storm create model user

# Create a migration file
storm create migration add_user_roles

# Create in specific package
storm create model product --package ./internal/models
```

### storm generate

Generate specific code components.

```bash
storm generate <component> [flags]
```

**Components:**
- `schema` - Generate SQL schema from models
- `docs` - Generate documentation
- `types` - Generate TypeScript types from models

**Examples:**
```bash
# Generate SQL schema
storm generate schema > schema.sql

# Generate TypeScript types
storm generate types --output ./frontend/types
```

### storm verify

Verify database connection and configuration.

```bash
storm verify [flags]
```

**Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--check-models` | Verify models can be parsed | `true` |
| `--check-db` | Verify database connection | `true` |

**Examples:**
```bash
# Verify everything
storm verify

# Only verify database connection
storm verify --check-models=false
```

### storm introspect

Generate complete Storm ORM code from existing database schema.

```bash
storm introspect [flags]
```

**Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--database` | Database connection URL (required) | |
| `--schema` | Database schema to inspect | `public` |
| `--table` | Generate ORM for specific table only | All tables |
| `--output` | Output directory for generated code | `./generated/<package>` |
| `--package` | Package name for generated code | `models` |

**Generated Files:**
- `models.go` - Go struct definitions with proper tags
- `columns.go` - Type-safe column constants
- `storm.go` - Central ORM access point
- `*_metadata.go` - Model metadata for zero-reflection ORM
- `*_repository.go` - Repository implementations with CRUD operations
- `*_query.go` - Type-safe query builders

**Examples:**
```bash
# Generate ORM from entire database
storm introspect --database="postgres://user:pass@localhost/mydb"

# Generate to specific directory
storm introspect --database="postgres://user:pass@localhost/mydb" \
  --output=./internal/models \
  --package=models

# Generate for specific table only
storm introspect --database="postgres://user:pass@localhost/mydb" \
  --table=users

# Use different schema
storm introspect --database="postgres://user:pass@localhost/mydb" \
  --schema=myschema

# Complete example with immediate usage
storm introspect --database="postgres://user:pass@localhost/mydb" \
  --output=./models \
  --package=models

# Then in your code:
# import "./models"
# storm := models.NewStorm(db)
# users, err := storm.Users.Query().Find()
```

### storm version

Show Storm version information.

```bash
storm version [flags]
```

**Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--json` | Output as JSON | `false` |

**Examples:**
```bash
# Show version
storm version

# JSON output for scripts
storm version --json
```

## Configuration Precedence

For all commands, configuration values are resolved in this order:

1. Command-line flags (highest priority)
2. Environment variables
3. Configuration file
4. Default values (lowest priority)

## Environment Variables

All flags can be set via environment variables with the `STORM_` prefix:

```bash
# Database URL
export STORM_DATABASE_URL="postgres://user:pass@localhost/mydb"

# Models package
export STORM_MODELS_PACKAGE="./internal/models"

# Enable debug mode
export STORM_DEBUG="true"
```

## Exit Codes

Storm uses standard exit codes:

- `0` - Success
- `1` - General error
- `2` - Configuration error
- `3` - Database connection error
- `4` - File system error
- `5` - Validation error

## Common Workflows

### Initial Setup

```bash
# 1. Initialize configuration
storm init

# 2. Define your models
# ... create model files ...

# 3. Generate initial migration
storm migrate --name initial_schema

# 4. Apply migration
storm migrate --push

# 5. Generate ORM code
storm orm
```

### Adding a New Model

```bash
# 1. Create model file
storm create model product

# 2. Edit the model file
# ... add fields ...

# 3. Generate migration
storm migrate --name add_products

# 4. Review and apply
storm migrate --push

# 5. Update ORM code
storm orm
```

### Working with Existing Database

```bash
# 1. Generate complete ORM from database
storm introspect --database="postgres://user:pass@localhost/mydb"

# 2. Review generated code in ./generated/models/
# ... adjust models.go if needed ...

# 3. Use the generated ORM immediately
# import "./generated/models"
# storm := models.NewStorm(db)

# 4. Future changes follow normal workflow
storm migrate
```

### CI/CD Pipeline

```bash
#!/bin/bash
# ci-migrate.sh

set -e

# Verify configuration
storm verify

# Generate migration (should be none if models match DB)
storm migrate --dry-run

# Run any pending migrations
storm migrate --push

# Regenerate ORM code
storm orm
```

## Debugging

### Enable Debug Output

```bash
# Via flag
storm --debug migrate

# Via environment
export STORM_DEBUG=true
storm migrate
```

### Verbose Mode

```bash
# Show detailed progress
storm --verbose migrate
```

### Dry Run

```bash
# See what would happen without making changes
storm migrate --dry-run
```

## Tips and Tricks

### 1. Alias Common Commands

```bash
# Add to ~/.bashrc or ~/.zshrc
alias sm='storm migrate'
alias smp='storm migrate --push'
alias so='storm orm'
```

### 2. Use Configuration Profiles

```bash
# Development
storm --config storm.dev.yaml migrate

# Production (read-only verification)
storm --config storm.prod.yaml migrate --dry-run
```

### 3. Script Complex Workflows

```bash
#!/bin/bash
# update-schema.sh

echo "Generating migration..."
storm migrate --name "$1"

echo "Review the migration file, then press Enter to apply..."
read

echo "Applying migration..."
storm migrate --push

echo "Regenerating ORM code..."
storm orm

echo "Done!"
```

### 4. Integration with Make

```makefile
# Makefile

.PHONY: migrate orm verify

migrate:
	storm migrate

migrate-push:
	storm migrate --push

orm:
	storm orm

verify:
	storm verify

setup: verify migrate-push orm
```

## Next Steps

- [Getting Started](getting-started.md) - Tutorial
- [Configuration Guide](configuration.md) - Configuration options
- [Migrations Guide](migrations.md) - Migration strategies