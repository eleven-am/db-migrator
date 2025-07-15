# Configuration Guide

Storm supports multiple configuration methods to fit different deployment scenarios. This guide covers all configuration options and best practices.

## Configuration Methods

Storm configuration follows this priority order:
1. **Command-line flags** (highest priority)
2. **Environment variables**
3. **Configuration file** (`storm.yaml`)
4. **Default values** (lowest priority)

## Configuration File

### Creating a Configuration File

Initialize a new configuration file:

```bash
storm init
```

This creates `storm.yaml` with default settings.

### Configuration File Structure

```yaml
version: 1
project: my-project

database:
  driver: postgres
  url: postgres://user:password@localhost:5432/dbname?sslmode=disable
  max_connections: 25
  
models:
  package: ./models
  
migrations:
  directory: ./migrations
  table: schema_migrations
  auto_apply: false
  
orm:
  generate_hooks: true
  generate_tests: false
  generate_mocks: false
  
schema:
  strict_mode: true
  naming_convention: snake_case
```

### Configuration File Locations

Storm looks for configuration files in this order:
1. Path specified by `--config` flag
2. Path in `STORM_CONFIG` environment variable
3. `storm.yaml`
4. `storm.yml`
5. `.storm.yaml`
6. `.storm.yml`

## Configuration Options

### Database Configuration

```yaml
database:
  # Database driver (currently only postgres supported)
  driver: postgres
  
  # Connection URL (can include all connection parameters)
  url: postgres://user:password@host:port/dbname?sslmode=disable
  
  # Connection pool settings
  max_connections: 25
  max_idle_connections: 5
  connection_max_lifetime: 1h
```

#### Connection URL Format

```
postgres://[user[:password]@][host][:port][/dbname][?param1=value1&...]
```

Common parameters:
- `sslmode`: `disable`, `require`, `verify-ca`, `verify-full`
- `connect_timeout`: Connection timeout in seconds
- `application_name`: Application name for pg_stat_activity
- `search_path`: Schema search path

### Models Configuration

```yaml
models:
  # Path to package containing model definitions
  package: ./models
  
  # Alternative paths for multi-module projects
  # package: ../shared/models
  # package: github.com/myorg/myapp/models
```

### Migrations Configuration

```yaml
migrations:
  # Directory to store migration files
  directory: ./migrations
  
  # Table name for tracking applied migrations
  table: schema_migrations
  
  # Automatically apply migrations on startup
  auto_apply: false
  
  # Migration file naming
  file_format: "{{.Version}}_{{.Name}}.sql"
```

### ORM Configuration

```yaml
orm:
  # Generate lifecycle hooks (BeforeCreate, AfterUpdate, etc.)
  generate_hooks: true
  
  # Generate test files for repositories
  generate_tests: false
  
  # Generate mock implementations
  generate_mocks: false
  
  # Custom templates directory
  templates_dir: ./templates/orm
```

### Schema Configuration

```yaml
schema:
  # Strict mode enforces all constraints
  strict_mode: true
  
  # Naming convention for database objects
  # Options: snake_case, camelCase
  naming_convention: snake_case
  
  # Schema name (PostgreSQL)
  schema_name: public
  
  # Enable schema versioning
  versioning: true
```

## Environment Variables

All configuration options can be set via environment variables:

```bash
# Database settings
export STORM_DATABASE_URL="postgres://user:pass@localhost/mydb"
export STORM_DATABASE_DRIVER="postgres"
export STORM_DATABASE_MAX_CONNECTIONS="50"

# Models settings
export STORM_MODELS_PACKAGE="./internal/models"

# Migrations settings
export STORM_MIGRATIONS_DIR="./db/migrations"
export STORM_MIGRATIONS_TABLE="storm_migrations"
export STORM_AUTO_MIGRATE="true"

# ORM settings
export STORM_GENERATE_HOOKS="true"
export STORM_GENERATE_TESTS="true"
export STORM_GENERATE_MOCKS="false"

# Schema settings
export STORM_STRICT_MODE="true"
export STORM_NAMING_CONVENTION="snake_case"
```

## Command-Line Flags

### Global Flags

Available on all commands:

```bash
storm --config custom-storm.yaml migrate
storm --url "postgres://..." migrate
storm --debug migrate
storm --verbose migrate
```

### Command-Specific Flags

Override configuration for specific commands:

```bash
# Migration command
storm migrate \
  --package ./other/models \
  --output ./other/migrations \
  --dry-run

# ORM command
storm orm \
  --package ./models \
  --output ./generated \
  --hooks \
  --tests
```

## Multiple Environments

### Environment-Specific Files

Create separate configuration files for each environment:

```yaml
# storm.dev.yaml
database:
  url: postgres://localhost:5432/myapp_dev?sslmode=disable
  
# storm.prod.yaml
database:
  url: postgres://prod-host:5432/myapp?sslmode=require
  max_connections: 100
```

Use them with:

```bash
# Development
storm --config storm.dev.yaml migrate

# Production
storm --config storm.prod.yaml migrate
```

### Environment Variable Overrides

```yaml
# storm.yaml
database:
  url: ${DATABASE_URL:-postgres://localhost:5432/myapp_dev}
  max_connections: ${DB_MAX_CONNECTIONS:-25}
```

Note: Variable substitution requires external processing.

## Docker Configuration

### Using Environment Variables

```dockerfile
FROM golang:1.21-alpine

ENV STORM_DATABASE_URL=${DATABASE_URL}
ENV STORM_MODELS_PACKAGE=./models
ENV STORM_MIGRATIONS_DIR=./migrations

COPY . /app
WORKDIR /app

RUN go install github.com/eleven-am/storm/cmd/storm@latest
RUN storm migrate
```

### Docker Compose

```yaml
version: '3.8'

services:
  app:
    build: .
    environment:
      STORM_DATABASE_URL: postgres://postgres:password@db:5432/myapp
      STORM_AUTO_MIGRATE: "true"
    depends_on:
      - db

  db:
    image: postgres:15
    environment:
      POSTGRES_PASSWORD: password
      POSTGRES_DB: myapp
```

## CI/CD Configuration

### GitHub Actions

```yaml
name: Database Migrations

on:
  push:
    branches: [main]

jobs:
  migrate:
    runs-on: ubuntu-latest
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Install Storm
        run: go install github.com/eleven-am/storm/cmd/storm@latest
      
      - name: Run Migrations
        env:
          STORM_DATABASE_URL: ${{ secrets.DATABASE_URL }}
        run: |
          storm migrate --dry-run
          storm migrate --push
```

### GitLab CI

```yaml
migrate:
  stage: deploy
  script:
    - go install github.com/eleven-am/storm/cmd/storm@latest
    - storm migrate --push
  variables:
    STORM_DATABASE_URL: ${DATABASE_URL}
  only:
    - main
```

## Security Best Practices

### 1. Never Commit Secrets

```yaml
# storm.yaml - DON'T DO THIS
database:
  url: postgres://user:actualpassword@host:5432/db  # BAD!

# storm.yaml - DO THIS
database:
  url: ${DATABASE_URL}  # Good - use environment variable
```

### 2. Use Secret Management

```bash
# AWS Secrets Manager
export STORM_DATABASE_URL=$(aws secretsmanager get-secret-value \
  --secret-id prod/db/connection \
  --query SecretString --output text)

# Kubernetes Secrets
kubectl create secret generic storm-config \
  --from-literal=STORM_DATABASE_URL="postgres://..."
```

### 3. Restrict File Permissions

```bash
# Set appropriate permissions
chmod 600 storm.yaml
chmod 600 .env
```

### 4. Use SSL/TLS

```yaml
database:
  url: postgres://user:pass@host:5432/db?sslmode=require&sslcert=client.crt&sslkey=client.key
```

## Advanced Configuration

### Connection Pool Tuning

```yaml
database:
  # Maximum number of open connections
  max_connections: 100
  
  # Maximum number of idle connections
  max_idle_connections: 10
  
  # Maximum time a connection can be reused
  connection_max_lifetime: 30m
  
  # Maximum time to wait for a connection
  connection_timeout: 30s
```

### Multi-Database Setup

For applications using multiple databases:

```yaml
# storm.yaml
databases:
  primary:
    url: postgres://host1:5432/main_db
    models_package: ./models/primary
    
  analytics:
    url: postgres://host2:5432/analytics_db
    models_package: ./models/analytics
```

### Custom Migration Naming

```yaml
migrations:
  # Use timestamp-based versions
  version_format: "20060102150405"
  
  # Custom file naming
  file_format: "{{.Version}}_{{.Name}}.sql"
  
  # Separate up/down files
  split_files: true
  up_suffix: ".up.sql"
  down_suffix: ".down.sql"
```

## Troubleshooting

### Configuration Not Loading

1. Check file exists and is readable:
```bash
ls -la storm.yaml
```

2. Validate YAML syntax:
```bash
yamllint storm.yaml
```

3. Enable verbose mode:
```bash
storm --verbose migrate
```

### Connection Issues

Test connection independently:
```bash
psql "postgres://user:pass@host:5432/db?sslmode=disable"
```

### Environment Variable Issues

Print current configuration:
```bash
storm config show  # If implemented
```

Or check environment:
```bash
env | grep STORM_
```

## Best Practices

1. **Use Configuration Files for Defaults**
   - Keep common settings in storm.yaml
   - Override with environment variables for secrets

2. **Environment-Specific Overrides**
   - Use different config files for dev/staging/prod
   - Override with environment variables in CI/CD

3. **Version Control**
   - Commit storm.yaml with safe defaults
   - Never commit credentials or secrets
   - Use .gitignore for environment-specific files

4. **Documentation**
   - Document required environment variables
   - Provide example configuration files
   - Include setup instructions in README

## Next Steps

- [Getting Started](getting-started.md) - Initial setup
- [CLI Reference](cli-reference.md) - All commands and options
- [Migrations Guide](migrations.md) - Managing database changes