# Go Database Migrator

[![Go Version](https://img.shields.io/badge/go-1.19+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Test Coverage](https://img.shields.io/badge/coverage-95%25-brightgreen.svg)](TEST_DOCUMENTATION.md)

A production-ready, struct-driven PostgreSQL database migration tool powered by **Stripe's pg-schema-diff**. Generate accurate, safe database migrations by comparing your Go structs with your actual database schema.

## Features

🚀 **Struct-Driven Migrations** - Define your schema in Go structs using `dbdef` tags  
🔥 **Stripe-Powered Engine** - Uses Stripe's battle-tested pg-schema-diff for constraint normalization  
🛡️ **Safety First** - Identifies destructive operations and requires explicit approval  
🔧 **PostgreSQL Native** - Built specifically for PostgreSQL with full feature support  
⚡ **Zero False Positives** - Eliminates CHECK constraint syntax comparison issues  
🏗️ **Zero-Downtime** - Leverages PostgreSQL's online migration capabilities  

## Why db-migrator?

### Comparison with Alternatives

| Feature | db-migrator | golang-migrate | Atlas | GORM AutoMigrate |
|---------|------------|----------------|-------|------------------|
| Struct-driven | ✅ | ❌ | ❌ | ✅ |
| Stripe pg-schema-diff | ✅ | ❌ | ❌ | ❌ |
| Down migrations | ✅ | ✅ | ✅ | ❌ |
| Safety checks | ✅ | ❌ | ✅ | ❌ |
| Zero false positives | ✅ | ❌ | ❌ | ❌ |
| Zero-downtime migrations | ✅ | ❌ | ✅ | ❌ |

## Quick Start

### Installation

```bash
go install github.com/eleven-am/db-migrator@latest
```

### Basic Usage

1. **Define your models** with `dbdef` tags:

```go
package models

import "time"

type User struct {
    // Table-level configuration
    _ struct{} `dbdef:"table:users;index:idx_users_team_id,team_id"`
    
    ID        string    `db:"id" dbdef:"type:cuid;primary_key;default:gen_cuid()"`
    Email     string    `db:"email" dbdef:"type:varchar(255);not_null;unique"`
    TeamID    string    `db:"team_id" dbdef:"type:cuid;not_null;foreign_key:teams.id"`
    IsActive  bool      `db:"is_active" dbdef:"type:boolean;not_null;default:true"`
    CreatedAt time.Time `db:"created_at" dbdef:"type:timestamptz;not_null;default:now()"`
}
```

2. **Generate migrations**:

```bash
db-migrator migrate \
  --url="postgres://user:pass@localhost/mydb" \
  --package="./internal/models" \
  --output="./migrations"
```

3. **Choose your workflow**:
   - **File-based**: Generate migration files for review and manual application
   - **Direct execution**: Use `--push` to apply changes immediately  
   - **Preview mode**: Use `--dry-run` to see changes without applying

## Architecture

### How Tag Parsing Works

The `db-migrator` leverages Go's powerful `reflect` package to inspect your struct definitions at runtime. When you specify the `--package` flag, the tool:

1.  **Scans Go Files**: It reads all `.go` files within the specified package path.
2.  **Identifies Structs**: It identifies `struct` types that are intended to represent database tables.
3.  **Parses `dbdef` Tags**: For each field in these structs, and for the struct itself (for table-level tags), it reads and parses the `dbdef` string tag. These tags are simple key-value pairs (e.g., `type:varchar(255)`, `primary_key`) separated by semicolons.
4.  **Builds Internal Schema**: The parsed information is then used to construct an in-memory representation of your *desired* database schema, which is later compared against the *actual* database schema.

This approach allows you to define your database schema directly alongside your Go models, ensuring consistency and reducing the need for separate schema definition files.

### Smart Struct-Driven Migration Strategy

The tool implements a "Smart Struct-Driven Migration" approach:

1. **Parse** Go structs with `dbdef` tags to understand desired schema
2. **Introspect** current PostgreSQL database schema  
3. **Compare** using normalized, signature-based matching
4. **Generate** safe SQL migrations with rollback support

### Signature-Based Comparison

Instead of comparing by names (which can differ), we use semantic signatures:

```
// These are considered identical:
STRUCT: idx_users_email     -> table:users|cols:email|unique:true|method:btree
DATABASE: users_email_key   -> table:users|cols:email|unique:true|method:btree
```

This eliminates false positives from naming differences while ensuring semantic accuracy.

## DBDef Tag Syntax

### Comprehensive `dbdef` Example

Here's an example combining various field and table-level attributes:

```go
package models

import "time"

type Product struct {
    // Table-level configuration: custom table name, composite unique constraint, and a partial index
    _ struct{} `dbdef:"table:store_products;unique:uk_product_sku_vendor,sku,vendor_id;index:idx_active_products,status where:status='active'"`
    
    ID          string                 `db:"id" dbdef:"type:uuid;primary_key;default:gen_random_uuid()"`
    Name        string                 `db:"name" dbdef:"type:varchar(255);not_null"`
    SKU         string                 `db:"sku" dbdef:"type:varchar(100);not_null"`
    VendorID    string                 `db:"vendor_id" dbdef:"type:uuid;not_null;foreign_key:vendors.id;on_delete:CASCADE"`
    Price       float64                `db:"price" dbdef:"type:numeric(10,2);not_null;default:0.00"`
    Description *string                `db:"description" dbdef:"type:text"` // Nullable field
    Status      string                 `db:"status" dbdef:"type:varchar(50);not_null;default:'draft'"`
    Metadata    map[string]interface{} `db:"metadata" dbdef:"type:jsonb;default:'{}'"`
    CreatedAt   time.Time              `db:"created_at" dbdef:"type:timestamptz;not_null;default:now()"`
    UpdatedAt   time.Time              `db:"updated_at" dbdef:"type:timestamptz;not_null;default:now()"`
}

// Vendor struct for foreign key reference
type Vendor struct {
    _ struct{} `dbdef:"table:vendors"`
    ID   string `db:"id" dbdef:"type:uuid;primary_key;default:gen_random_uuid()"`
    Name string `db:"name" dbdef:"type:varchar(255);not_null;unique"`
}
```

### Field-Level Tags

```go
type Example struct {
    ID       string `db:"id" dbdef:"type:cuid;primary_key;default:gen_cuid()"`
    Email    string `db:"email" dbdef:"type:varchar(255);not_null;unique"`
    TeamID   string `db:"team_id" dbdef:"type:cuid;foreign_key:teams.id;on_delete:CASCADE"`
    IsActive bool   `db:"is_active" dbdef:"type:boolean;default:true"`
}
```

**Supported field attributes:**
- `type:` - PostgreSQL data type (varchar, integer, boolean, jsonb, etc.)
- `primary_key` - Mark as primary key
- `unique` - Add unique constraint  
- `not_null` - NOT NULL constraint
- `default:` - Default value
- `foreign_key:` - Foreign key reference (table.column)
- `on_delete:`, `on_update:` - FK actions (CASCADE, RESTRICT, SET NULL)

### Table-Level Tags

```go
type Example struct {
    _ struct{} `dbdef:"table:examples;index:idx_name,column1,column2;unique:uk_name,col1,col2;index:idx_partial,status where:status='active'"`
    // ... fields
}
```

**Supported table attributes:**
- `table:` - Override table name (defaults to snake_case plural)
- `index:` - Create regular index (`name,col1,col2`)
- `unique:` - Create unique constraint (`name,col1,col2`)
- `where:` - Add WHERE clause for partial indexes

## Workflow Modes

The db-migrator supports three distinct workflow modes to fit different development needs:

### 1. File-Based Workflow (Default)
Generate migration files for review and version control:

```bash
db-migrator migrate --url="postgres://localhost/mydb" --package="./models"
```

**Benefits:**
- Migration files are version controlled
- Enables code review of schema changes  
- Compatible with existing migration tools
- Safe rollback with down migrations

### 2. Direct Execution Workflow
Apply schema changes immediately with `--push`:

```bash
db-migrator migrate --url="postgres://localhost/mydb" --package="./models" --push
```

**Benefits:**
- Instant schema updates during development
- Single command for generate + apply
- No intermediate files to manage
- Perfect for rapid iteration

### 3. Preview Workflow
Review changes without applying them using `--dry-run`:

```bash
db-migrator migrate --url="postgres://localhost/mydb" --package="./models" --dry-run
```

**Benefits:**
- Understand what changes will be made
- Validate schema differences
- Debug struct definitions
- Safe exploration of schema changes

## Troubleshooting

### Common Issues and Solutions

- **`pq: password authentication failed for user "..."`**:
  - **Cause**: Incorrect database username or password.
  - **Solution**: Double-check your `--user` and `--password` flags, or the credentials in your `--url`. Ensure the database user has the necessary permissions.

- **`dial tcp ...: connect: connection refused`**:
  - **Cause**: The PostgreSQL database is not running or is not accessible at the specified host and port.
  - **Solution**: Verify that your PostgreSQL server is running and listening on the correct port (`--port`). Check firewall rules if connecting remotely.

- **`ERROR: relation "..." does not exist`**:
  - **Cause**: Your Go struct defines a foreign key or references a table that does not exist in the target database.
  - **Solution**: Ensure all referenced tables exist. If you're creating a new schema, run an initial migration to create base tables.

- **`Failed to parse directory: ... no Go files found`**:
  - **Cause**: The `--package` path does not contain any `.go` files or the path is incorrect.
  - **Solution**: Verify the `--package` flag points to a directory containing your Go model structs.

- **`Migration generated but not applied`**:
  - **Cause**: You ran `db-migrator migrate` without the `--push` flag.
  - **Solution**: The default behavior is to generate `.up.sql` and `.down.sql` files for manual review. To apply directly, use `--push`. To see the SQL without applying, use `--dry-run`.

- **`Destructive operation detected: DROP TABLE ...`**:
  - **Cause**: The tool detected a schema change that would result in data loss (e.g., dropping a table or column).
  - **Solution**: By default, `db-migrator` prevents destructive operations. If you intend to perform such an operation, use the `--allow-destructive` flag. **Use with caution!**

## CLI Commands

### migrate

Generate database migrations by comparing Go structs with database schema.

```bash
db-migrator migrate [flags]
```

**Connection Options:**
```bash
--url string              # Full connection URL
--host string             # Database host (default "localhost")  
--port string             # Database port (default "5432")
--user string             # Database user
--password string         # Database password
--dbname string           # Database name
--sslmode string          # SSL mode (default "disable")
```

**Migration Options:**
```bash
--package string          # Go package path (default "./internal/db")
--output string           # Migration output directory (default "./migrations")
--name string             # Migration name prefix
--dry-run                 # Print changes without creating files
--push                    # Execute SQL directly on database
--allow-destructive       # Allow DROP operations
--create-if-not-exists    # Create database if missing
```

### Examples

```bash
# Basic migration
db-migrator migrate --url="postgres://localhost/mydb" --package="./models"

# With custom output directory
db-migrator migrate \
  --host="localhost" \
  --user="postgres" \
  --dbname="myapp" \
  --package="./internal/models" \
  --output="./db/migrations"

# Dry run to preview changes  
db-migrator migrate --url="postgres://localhost/mydb" --package="./models" --dry-run

# Allow destructive operations
db-migrator migrate --url="postgres://localhost/mydb" --package="./models" --allow-destructive

# Execute migration directly on database
db-migrator migrate --url="postgres://localhost/mydb" --package="./models" --push

# Generate and execute in one command
db-migrator migrate --url="postgres://localhost/mydb" --package="./models" --push --allow-destructive
```

## Common Patterns

### Soft Deletes

Add soft delete functionality with partial indexes:

```go
type Model struct {
    DeletedAt *time.Time `db:"deleted_at" dbdef:"type:timestamptz"`
    _ struct{} `dbdef:"index:idx_not_deleted,deleted_at where:deleted_at IS NULL"`
}
```

### Audit Fields

Track who created and modified records:

```go  
type Auditable struct {
    CreatedBy string    `db:"created_by" dbdef:"type:uuid;not_null;foreign_key:users.id"`
    UpdatedBy string    `db:"updated_by" dbdef:"type:uuid;not_null;foreign_key:users.id"`
    CreatedAt time.Time `db:"created_at" dbdef:"type:timestamptz;not_null;default:now()"`
    UpdatedAt time.Time `db:"updated_at" dbdef:"type:timestamptz;not_null;default:now()"`
}
```

### Multi-tenant Patterns

Implement row-level security with tenant isolation:

```go
type TenantModel struct {
    _ struct{} `dbdef:"table:products;index:idx_tenant_active,tenant_id,is_active"`
    
    TenantID string `db:"tenant_id" dbdef:"type:uuid;not_null;foreign_key:tenants.id"`
    // Composite unique constraint across tenant
    _ struct{} `dbdef:"unique:uk_tenant_sku,tenant_id,sku"`
}
```

### Versioning

Track record versions for audit trails:

```go
type Versioned struct {
    Version   int       `db:"version" dbdef:"type:integer;not_null;default:1"`
    UpdatedAt time.Time `db:"updated_at" dbdef:"type:timestamptz;not_null;default:now()"`
    _ struct{} `dbdef:"index:idx_version,entity_id,version"`
}
```

## Advanced Features

### Partial Indexes

Create conditional indexes with WHERE clauses:

```go
type Order struct {
    _ struct{} `dbdef:"table:orders;index:idx_active_orders,status where:status='active'"`
    
    Status string `db:"status" dbdef:"type:varchar(50);not_null"`
}
```

### Composite Indexes

Create multi-column indexes:

```go
type AuditLog struct {
    _ struct{} `dbdef:"table:audit_logs;index:idx_entity,entity_type,entity_id;unique:uk_audit,user_id,action,created_at"`
    
    EntityType string `db:"entity_type" dbdef:"type:varchar(50);not_null"`
    EntityID   string `db:"entity_id" dbdef:"type:cuid;not_null"`
}
```

### Foreign Key Actions

Specify CASCADE, RESTRICT, or SET NULL behavior:

```go
type Project struct {
    TeamID string `db:"team_id" dbdef:"type:cuid;foreign_key:teams.id;on_delete:CASCADE;on_update:RESTRICT"`
    OwnerID *string `db:"owner_id" dbdef:"type:cuid;foreign_key:users.id;on_delete:SET NULL"`
}
```

### JSONB Support

Full support for PostgreSQL JSONB columns:

```go
type Config struct {
    Metadata JSONField[map[string]interface{}] `db:"metadata" dbdef:"type:jsonb;default:'{}'"`
    Settings JSONField[AppSettings]            `db:"settings" dbdef:"type:jsonb;not_null"`
}
```

## Safety & Validation

### Destructive Operation Detection

The tool automatically identifies potentially dangerous operations:

- ❌ **DROP CONSTRAINT** (unique, foreign key)
- ❌ **DROP INDEX** (unique indexes)  
- ✅ **CREATE INDEX** (always safe)
- ✅ **CREATE CONSTRAINT** (always safe)
- ✅ **DROP INDEX** (non-unique regular indexes)

### Migration Safety Levels

1. **Safe Mode** (default) - Only generates safe operations
2. **Destructive Mode** (`--allow-destructive`) - Includes DROP operations
3. **Dry Run Mode** (`--dry-run`) - Shows changes without creating files

### Generated Migration Files

```
migrations/
├── 20240101120000_schema_update.up.sql    # Forward migration
└── 20240101120000_schema_update.down.sql  # Rollback migration  
```

## Integration Examples

### With sqlx and squirrel

db-migrator works seamlessly with popular Go database libraries:

```go
// Define your schema with db-migrator tags
type User struct {
    _ struct{} `dbdef:"table:users;index:idx_email,email"`
    
    ID        string    `db:"id" dbdef:"type:uuid;primary_key;default:gen_random_uuid()"`
    Email     string    `db:"email" dbdef:"type:varchar(255);not_null;unique"`
    TeamID    string    `db:"team_id" dbdef:"type:uuid;not_null;foreign_key:teams.id"`
    CreatedAt time.Time `db:"created_at" dbdef:"type:timestamptz;not_null;default:now()"`
}

// Use with sqlx - the db tags work perfectly
var user User
err := db.Get(&user, "SELECT * FROM users WHERE id = $1", userID)

// Use with squirrel - clean query building
query, args, _ := sq.Select("*").
    From("users").
    Where(sq.Eq{"team_id": teamID}).
    OrderBy("created_at DESC").
    ToSql()
```

### With golang-migrate

```bash
# Generate migrations
db-migrator migrate --url="postgres://localhost/mydb" --package="./models"

# Apply with golang-migrate  
migrate -path ./migrations -database "postgres://localhost/mydb" up
```

### With Atlas

```bash
# Generate schema from structs
db-migrator migrate --dry-run --package="./models" > schema.sql

# Use with Atlas
atlas migrate diff --to "file://schema.sql"
```

### In CI/CD Pipeline

```yaml
name: Database Migration Check
on: [push, pull_request]

jobs:
  migration-check:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.19'
      
      - name: Install db-migrator
        run: go install ./tools/db-migrator
      
      - name: Check for schema changes
        run: |
          db-migrator migrate \
            --url="postgres://postgres:postgres@localhost/postgres" \
            --package="./internal/models" \
            --dry-run
```

## Performance

### Benchmarks

The tool is optimized for performance with real-world databases:

```
BenchmarkIntrospection-8          100 req/sec    50ms per operation
BenchmarkNormalization-8        10000 req/sec     0.1ms per operation  
BenchmarkComparison-8            1000 req/sec     5ms per operation
```

### Scalability

Tested with:
- ✅ 100+ tables
- ✅ 1000+ indexes  
- ✅ Complex schema hierarchies
- ✅ Large WHERE clauses

## Testing

Comprehensive test suite with 95%+ coverage:

```bash
# Run all tests
make test

# Unit tests only (no database required)
make test-unit

# Integration tests (requires PostgreSQL)
make test-integration  

# Generate coverage report
make test-coverage
```

See [TEST_DOCUMENTATION.md](TEST_DOCUMENTATION.md) for detailed testing information.

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests for your changes
4. Ensure all tests pass (`make test`)
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

### Development Setup

```bash
# Clone repository
git clone https://github.com/eleven-am/db-migrator.git
cd db-migrator

# Install dependencies
go mod download

# Start PostgreSQL (for integration tests)
docker run --name postgres-test -e POSTGRES_PASSWORD=postgres -p 5432:5432 -d postgres:15

# Run tests
make test
```

## Current Limitations

- **PostgreSQL only** - MySQL support is on the roadmap
- **No stored procedures/functions** - Only tables, indexes, and constraints are supported
- **No custom types** - PostgreSQL domains and custom types not yet supported
- **Forward-only migrations** - No automatic rollback verification
- **No check constraints** - Check constraints parsing not implemented
- **No triggers** - Database triggers are not managed

## Roadmap

- [ ] MySQL support
- [ ] Schema validation rules
- [ ] Migration rollback verification
- [ ] Parallel migration execution
- [ ] Check constraints support
- [ ] Custom types (domains)
- [ ] Stored procedures/functions

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- 📖 **Documentation**: [GitHub Wiki](https://github.com/eleven-am/db-migrator/wiki)
- 🐛 **Bug Reports**: [GitHub Issues](https://github.com/eleven-am/db-migrator/issues)
- 💬 **Discussions**: [GitHub Discussions](https://github.com/eleven-am/db-migrator/discussions)
- 📧 **Email**: roy@theossaibrothers.com

## Acknowledgments

- PostgreSQL team for excellent database features
- Go community for robust tooling
- [pg_query_go](https://github.com/pganalyze/pg_query_go) for SQL parsing
- All contributors and testers

---

Built with ❤️ by [Roy OSSAI](https://github.com/eleven-am) for the Go and PostgreSQL communities.