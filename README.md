# Storm - Unified Database Toolkit for Go

[![Go Version](https://img.shields.io/badge/go-1.23+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Documentation](https://img.shields.io/badge/docs-complete-brightgreen.svg)](docs/)

Storm is a unified database toolkit that provides a complete, type-safe bridge between Go applications and PostgreSQL databases. Define your schema once in Go structs, and Storm handles everything else - migrations, ORM code generation, and type-safe queries.

## 🌟 Key Features

- **🏗️ Struct-Driven Development** - Your Go structs are the single source of truth
- **🔒 100% Type-Safe** - All queries are validated at compile time
- **⚡ Zero Runtime Reflection** - Everything is generated code for maximum performance
- **🎯 Zero False Positives** - Advanced schema comparison eliminates noise
- **🛡️ Safety First** - Automatic detection of destructive changes
- **🔄 Complete Lifecycle** - From schema design to queries, all in one toolkit

## 📋 Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Core Concepts](#core-concepts)
- [Configuration](#configuration)
- [Documentation](#documentation)
- [Examples](#examples)
- [Contributing](#contributing)

## Installation

```bash
go install github.com/eleven-am/storm/cmd/storm@latest
```

Or add to your project:

```bash
go get github.com/eleven-am/storm
```

## Quick Start

### Starting from Existing Database (Database-First)

If you have an existing PostgreSQL database, Storm can generate a complete ORM from it:

```bash
# Generate complete ORM from your database
storm introspect --database="postgres://user:pass@localhost/mydb" --output=./models

# Use the generated ORM immediately
# import "./models"
# storm := models.NewStorm(db)
# users, err := storm.Users.Query().Find()
```

### Starting from Go Structs (Code-First)

#### 1. Initialize your project

```bash
storm init
```

This creates a `storm.yaml` configuration file with sensible defaults.

#### 2. Define your models

```go
package models

import "time"

// User model with dbdef tags for schema definition
type User struct {
    _ struct{} `dbdef:"table:users;index:idx_users_email,email"`
    
    ID        string    `db:"id" dbdef:"type:uuid;primary_key;default:gen_random_uuid()"`
    Email     string    `db:"email" dbdef:"type:varchar(255);not_null;unique"`
    Name      string    `db:"name" dbdef:"type:varchar(100);not_null"`
    CreatedAt time.Time `db:"created_at" dbdef:"type:timestamptz;not_null;default:now()"`
    
    // ORM relationships
    Posts []Post `db:"-" orm:"has_many:Post,foreign_key:user_id"`
}

// Post model
type Post struct {
    _ struct{} `dbdef:"table:posts;index:idx_posts_user,user_id"`
    
    ID        string    `db:"id" dbdef:"type:uuid;primary_key;default:gen_random_uuid()"`
    UserID    string    `db:"user_id" dbdef:"type:uuid;not_null;foreign_key:users.id"`
    Title     string    `db:"title" dbdef:"type:varchar(255);not_null"`
    Content   string    `db:"content" dbdef:"type:text"`
    Published bool      `db:"published" dbdef:"type:boolean;not_null;default:false"`
    CreatedAt time.Time `db:"created_at" dbdef:"type:timestamptz;not_null;default:now()"`
    
    // ORM relationships
    User *User `db:"-" orm:"belongs_to:User,foreign_key:user_id"`
}
```

#### 3. Generate migrations

```bash
# Generate migration by comparing structs with database
storm migrate

# Apply the migration
storm migrate --push
```

#### 4. Generate ORM code

```bash
storm orm
```

This generates:
- Repository interfaces and implementations
- Type-safe query builders
- Column constants
- Relationship loaders

#### 5. Use the generated ORM

```go
package main

import (
    "context"
    "log"
    
    "github.com/jmoiron/sqlx"
    _ "github.com/lib/pq"
    "myapp/models"
)

func main() {
    // Connect to database
    db, err := sqlx.Connect("postgres", "postgres://user:pass@localhost/mydb")
    if err != nil {
        log.Fatal(err)
    }
    
    // Create Storm instance
    storm := models.NewStorm(db)
    ctx := context.Background()
    
    // Create a user
    user := &models.User{
        Email: "john@example.com",
        Name:  "John Doe",
    }
    if err := storm.Users.Create(ctx, user); err != nil {
        log.Fatal(err)
    }
    
    // Query with type-safe builders
    users, err := storm.Users.Query().
        Where(models.Users.Email.Like("%@example.com")).
        OrderBy(models.Users.CreatedAt.Desc()).
        Limit(10).
        Find()
    
    // Complex queries with And/Or/Not
    posts, err := storm.Posts.Query().
        Where(storm.And(
            models.Posts.Published.Eq(true),
            storm.Or(
                models.Posts.Title.Like("%Go%"),
                models.Posts.Content.Like("%golang%"),
            ),
        )).
        OrderBy(models.Posts.CreatedAt.Desc()).
        Find()
    
    // Transactions
    err = storm.WithTransaction(ctx, func(tx *models.Storm) error {
        post := &models.Post{
            UserID: user.ID,
            Title:  "Hello Storm",
            Content: "Storm makes database operations type-safe!",
        }
        return tx.Posts.Create(ctx, post)
    })
}
```

## Core Concepts

### 🏗️ Struct-Driven Development

Your Go structs define your database schema using `dbdef` tags. Storm ensures your database always matches your structs.

### 🔄 Migration Engine

Storm's migration engine:
- Compares your structs with the current database schema
- Generates precise SQL migrations
- Detects destructive changes and requires confirmation
- Supports rollbacks with automatically generated down migrations

### ⚡ ORM Generator

The ORM generator creates:
- **Repositories**: CRUD operations for each model
- **Query Builders**: Chainable, type-safe query construction
- **Column Constants**: Compile-time checked column references
- **Relationships**: Automatic loading of related data

### 🔍 Type Safety

Every database operation is type-safe:
```go
// ✅ Compile-time error if column doesn't exist
users, _ := storm.Users.Query().
    Where(models.Users.InvalidColumn.Eq("value")). // Compiler error!
    Find()

// ✅ Type mismatch caught at compile time
users, _ := storm.Users.Query().
    Where(models.Users.Age.Eq("not a number")). // Compiler error!
    Find()
```

## Configuration

Storm can be configured via:
1. Configuration file (`storm.yaml`)
2. Command-line flags
3. Environment variables

Priority: CLI flags > Config file > Defaults

See [Configuration Guide](docs/configuration.md) for details.

## Documentation

- 📘 **[Getting Started Guide](docs/getting-started.md)** - Step-by-step tutorial
- 🏷️ **[Schema Definition (dbdef tags)](docs/schema-definition.md)** - Complete tag reference
- 🔄 **[Migrations Guide](docs/migrations.md)** - Managing database changes
- 📊 **[ORM Guide](docs/orm-guide.md)** - Using the generated ORM
- 🔍 **[Query Builder](docs/query-builder.md)** - Building complex queries
- 🔌 **[Relationships](docs/relationships.md)** - Defining and using relationships
- ⚡ **[Performance Guide](docs/performance.md)** - Optimization tips
- 🔧 **[CLI Reference](docs/cli-reference.md)** - All commands and options

## Examples

Check out the [examples](examples/) directory:
- [Todo Application](examples/todo/) - Complete CRUD with relationships
- [Blog System](examples/blog/) - Multi-tenant blog with comments
- [E-commerce](examples/ecommerce/) - Products, orders, and inventory

## Why Storm?

### Comparison with Alternatives

| Feature | Storm | GORM | sqlx | ent |
|---------|-------|------|------|-----|
| Type Safety | ✅ Compile-time | ⚠️ Runtime | ❌ | ✅ |
| Performance | ✅ No reflection | ❌ Heavy reflection | ✅ | ✅ |
| Migrations | ✅ Automatic | ⚠️ Basic | ❌ | ✅ |
| Relationships | ✅ Type-safe | ✅ Runtime | ❌ | ✅ |
| Learning Curve | ✅ Simple | ✅ Simple | ✅ | ❌ Complex |
| Database Support | 🔧 PostgreSQL | ✅ Multiple | ✅ Multiple | ✅ Multiple |

### When to Use Storm

Storm is perfect when you want:
- ✅ Complete type safety with no runtime surprises
- ✅ Single source of truth for your schema
- ✅ Maximum performance with no reflection
- ✅ PostgreSQL-specific features
- ✅ Simple, predictable ORM without magic

### When NOT to Use Storm

Consider alternatives if you need:
- ❌ Multiple database support (MySQL, SQLite, etc.)
- ❌ NoSQL databases
- ❌ Dynamic schemas that change at runtime

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

Storm is released under the MIT License. See [LICENSE](LICENSE) for details.

## Support

- 📖 [Documentation](docs/)
- 💬 [GitHub Discussions](https://github.com/eleven-am/storm/discussions)
- 🐛 [Issue Tracker](https://github.com/eleven-am/storm/issues)

---

Built with ❤️ by [Roy OSSAI](https://github.com/eleven-am) for the Go and PostgreSQL communities.