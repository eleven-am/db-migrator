# Storm - Type-Safe PostgreSQL ORM and Migration Tool for Go

[![Go Version](https://img.shields.io/badge/go-1.23+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Documentation](https://img.shields.io/badge/docs-complete-brightgreen.svg)](docs/)

**Storm** is a modern, type-safe ORM and database migration toolkit for Go and PostgreSQL. It eliminates runtime errors by generating compile-time validated database code from your Go structs, while providing intelligent schema migrations and a powerful query builder.

## What Storm Offers

**🔒 Complete Type Safety** - Every database operation is validated at compile time. No more runtime SQL errors from typos or type mismatches.

**⚡ Zero-Reflection Performance** - All ORM code is generated at build time. No runtime reflection means maximum performance.

**🏗️ Intelligent Migrations** - Automatically generate precise SQL migrations by comparing your Go structs with your database schema.

**🎯 Advanced Query Builder** - Chainable, type-safe queries with support for complex conditions, joins, and PostgreSQL-specific features.

**📊 Rich Relationship Support** - Define relationships in Go and get automatic eager/lazy loading with type safety.

**🛡️ Production Ready** - Built-in transaction support, connection pooling, and comprehensive error handling.

## 🚀 How Storm Works

Storm follows a simple workflow:

1. **Define your models** in Go structs with `dbdef` tags
2. **Generate migrations** by comparing structs with your database
3. **Generate ORM code** with type-safe repositories and query builders
4. **Use the generated code** with full compile-time validation

```go
// 1. Define your model
type User struct {
    ID    string `db:"id" dbdef:"type:uuid;primary_key"`
    Email string `db:"email" dbdef:"type:varchar(255);not_null;unique"`
    Posts []Post `db:"-" orm:"has_many:Post,foreign_key:user_id"`
}

// 2. storm migrate  (generates SQL migrations)
// 3. storm orm      (generates ORM code)

// 4. Use type-safe operations
users, err := storm.Users.Query().
    Where(Users.Email.Like("%@company.com")).
    Include("Posts").  // Eager load relationships
    OrderBy(Users.CreatedAt.Desc()).
    Find()
```

## 🌟 Key Features

- **🏗️ Struct-Driven Development** - Your Go structs are the single source of truth for schema
- **🔒 100% Type-Safe** - All queries, columns, and operations validated at compile time
- **⚡ Zero Runtime Reflection** - Everything is generated code for maximum performance
- **🎯 Smart Migrations** - Advanced schema comparison with zero false positives
- **🛡️ Safety First** - Automatic detection of destructive changes with confirmation prompts
- **🔄 Complete Toolkit** - Migrations, ORM generation, and querying in one unified tool
- **📊 Rich ORM Features** - CRUD operations, relationships, transactions, bulk operations
- **🔍 Database Introspection** - Generate complete ORM from existing PostgreSQL databases

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
- **Repository implementations** with full CRUD operations (Create, FindByID, Update, Delete, etc.)
- **Type-safe query builders** with method chaining and compile-time validation
- **Column constants** for all struct fields with appropriate column types
- **Relationship loaders** for automatic eager/lazy loading of related data
- **Transaction support** with automatic rollback on errors
- **Bulk operations** for high-performance batch processing (CreateMany, BulkUpdate, etc.)

#### 5. Use the generated ORM

```go
package main

import (
    "context"
    "time"
    
    "github.com/jmoiron/sqlx"
    _ "github.com/lib/pq"
    "myapp/models"
)

func main() {
    // Connect to database
    db, err := sqlx.Connect("postgres", "postgres://user:pass@localhost/mydb")
    if err != nil {
        panic(err)
    }
    
    // Create Storm instance with all repositories
    storm := models.NewStorm(db)
    ctx := context.Background()
    
    // === SINGLE RECORD OPERATIONS ===
    
    // Create a user
    user := &models.User{
        Email: "john@example.com",
        Name:  "John Doe",
    }
    err = storm.Users.Create(ctx, user)
    // user.ID is now populated from database
    
    // Find by ID
    foundUser, err := storm.Users.FindByID(ctx, user.ID)
    
    // Update
    foundUser.Name = "John Smith"
    err = storm.Users.Update(ctx, foundUser)
    
    // Delete
    err = storm.Users.Delete(ctx, user.ID)
    
    // === TYPE-SAFE QUERIES ===
    
    // Simple queries with type-safe column references
    users, err := storm.Users.Query().
        Where(models.Users.Email.Like("%@company.com")).
        OrderBy(models.Users.CreatedAt.Desc()).
        Limit(10).
        Find()
    
    // Complex conditions with And/Or
    activePosts, err := storm.Posts.Query().
        Where(models.Posts.Published.Eq(true).And(
            models.Posts.CreatedAt.After(time.Now().AddDate(0, -1, 0)),
        )).
        OrderBy(models.Posts.CreatedAt.Desc()).
        Find()
    
    // Advanced filtering with multiple conditions
    searchResults, err := storm.Posts.Query().
        Where(models.Posts.Title.Like("%Go%").Or(
            models.Posts.Content.Like("%golang%"),
        ).And(
            models.Posts.Published.Eq(true),
        )).
        Find()
    
    // === RELATIONSHIPS ===
    
    // Eager load relationships
    usersWithPosts, err := storm.Users.Query().
        Include("Posts").  // Load all posts for each user
        Find()
    
    // Load specific relationship conditions
    usersWithRecentPosts, err := storm.Users.Query().
        IncludeWhere("Posts", models.Posts.CreatedAt.After(time.Now().AddDate(0, 0, -7))).
        Find()
    
    // === BATCH OPERATIONS ===
    
    // Create multiple records
    newUsers := []*models.User{
        {Email: "user1@example.com", Name: "User One"},
        {Email: "user2@example.com", Name: "User Two"},
        {Email: "user3@example.com", Name: "User Three"},
    }
    err = storm.Users.CreateMany(ctx, newUsers)
    
    // Bulk update with conditions
    rowsUpdated, err := storm.Posts.Query().
        Where(models.Posts.Published.Eq(false)).
        UpdateMany(ctx, map[string]interface{}{
            "published": true,
            "updated_at": time.Now(),
        })
    
    // Upsert (insert or update on conflict)
    err = storm.Users.Upsert(ctx, &models.User{
        Email: "admin@company.com",
        Name:  "Admin User",
    }, orm.UpsertOptions{
        ConflictColumns: []string{"email"},
        UpdateColumns:   []string{"name", "updated_at"},
    })
    
    // === TRANSACTIONS ===
    
    // Automatic transaction with rollback on error
    err = storm.WithTransaction(ctx, func(tx *models.Storm) error {
        // Create user
        newUser := &models.User{Email: "txuser@example.com", Name: "TX User"}
        if err := tx.Users.Create(ctx, newUser); err != nil {
            return err // Will rollback
        }
        
        // Create post for user
        post := &models.Post{
            UserID:  newUser.ID,
            Title:   "My First Post",
            Content: "Content here...",
        }
        return tx.Posts.Create(ctx, post) // If this fails, user creation is rolled back
    })
    
    // === ADVANCED QUERIES ===
    
    // Aggregations and counting
    totalUsers, err := storm.Users.Query().Count()
    
    activeUserCount, err := storm.Users.Query().
        Where(models.Users.IsActive.Eq(true)).
        Count()
    
    // Check existence
    hasAdminUser, err := storm.Users.Query().
        Where(models.Users.Email.Eq("admin@company.com")).
        Exists()
    
    // Time-based queries
    recentPosts, err := storm.Posts.Query().
        Where(models.Posts.CreatedAt.After(time.Now().AddDate(0, 0, -7))).
        OrderBy(models.Posts.CreatedAt.Desc()).
        Find()
    
    // Pagination
    page2Users, err := storm.Users.Query().
        OrderBy(models.Users.CreatedAt.Desc()).
        Offset(20).
        Limit(10).
        Find()
    
    // === RAW QUERIES ===
    
    // When you need custom SQL
    customResults, err := storm.Users.Query().
        ExecuteRaw(`
            SELECT u.*, COUNT(p.id) as post_count 
            FROM users u 
            LEFT JOIN posts p ON u.id = p.user_id 
            WHERE u.created_at > $1 
            GROUP BY u.id 
            ORDER BY post_count DESC
        `, time.Now().AddDate(0, -1, 0))
}
```

## Core Concepts

### 🏗️ Struct-Driven Development

Your Go structs define your database schema using `dbdef` tags. Storm ensures your database always matches your structs.

### 🔄 Intelligent Migration Engine

Storm's migration system intelligently handles schema evolution:

- **Smart Schema Comparison** - Analyzes struct definitions vs. database state
- **Precise SQL Generation** - Creates exact DDL statements needed for changes
- **Safety Guards** - Detects destructive operations and requires explicit confirmation
- **Automatic Rollbacks** - Generates down migrations for every change
- **Zero False Positives** - Advanced diffing eliminates unnecessary migrations

### ⚡ Comprehensive ORM Generator

The ORM generator produces production-ready code:

- **Full Repository Pattern** - Complete CRUD with Create, FindByID, Update, Delete, CreateMany, BulkUpdate, Upsert operations
- **Type-Safe Query Builder** - Chainable queries with Where, OrderBy, Join, Include, Limit, and complex conditions
- **Column Type System** - StringColumn, NumericColumn, TimeColumn with specialized methods (Like, Between, After, etc.)
- **Relationship Management** - Belongs-to, has-many, has-one, many-to-many with automatic loading
- **Transaction Support** - Nested transactions with automatic rollback on errors
- **Performance Optimized** - Zero reflection, connection pooling, prepared statements

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

## 📚 Comprehensive Examples

### Real-World Use Cases

#### 📝 E-commerce Product Management

```go
// Product model with rich validation
type Product struct {
    _ struct{} `dbdef:"table:products;index:idx_products_category,category_id;index:idx_products_price,price"`
    
    ID          string          `db:"id" dbdef:"type:uuid;primary_key;default:gen_random_uuid()"`
    SKU         string          `db:"sku" dbdef:"type:varchar(50);not_null;unique"`
    Name        string          `db:"name" dbdef:"type:varchar(255);not_null"`
    Description string          `db:"description" dbdef:"type:text"`
    Price       decimal.Decimal `db:"price" dbdef:"type:decimal(10,2);not_null"`
    Stock       int             `db:"stock" dbdef:"type:integer;not_null;default:0"`
    CategoryID  string          `db:"category_id" dbdef:"type:uuid;not_null;foreign_key:categories.id"`
    IsActive    bool            `db:"is_active" dbdef:"type:boolean;not_null;default:true"`
    CreatedAt   time.Time       `db:"created_at" dbdef:"type:timestamptz;not_null;default:now()"`
    UpdatedAt   time.Time       `db:"updated_at" dbdef:"type:timestamptz;not_null;default:now()"`
    
    // Relationships
    Category *Category `db:"-" orm:"belongs_to:Category,foreign_key:category_id"`
    Reviews  []Review  `db:"-" orm:"has_many:Review,foreign_key:product_id"`
    Tags     []Tag     `db:"-" orm:"has_many_through:Tag,join_table:product_tags,source_fk:product_id,target_fk:tag_id"`
}

// Advanced product queries
func ProductExamples(storm *models.Storm, ctx context.Context) {
    // Find products in stock within price range
    inStockProducts, err := storm.Products.Query().
        Where(models.Products.Stock.Gt(0).And(
            models.Products.Price.Between(decimal.NewFromFloat(10.0), decimal.NewFromFloat(100.0)),
        )).
        Include("Category", "Tags").
        OrderBy(models.Products.Price.Asc()).
        Find()
    
    // Search products by name or description
    searchResults, err := storm.Products.Query().
        Where(models.Products.Name.Like("%laptop%").Or(
            models.Products.Description.Like("%computer%"),
        )).
        Where(models.Products.IsActive.Eq(true)).
        Find()
    
    // Bulk update prices with 10% discount
    discountedRows, err := storm.Products.Query().
        Where(models.Products.CategoryID.Eq("electronics-category-id")).
        UpdateMany(ctx, map[string]interface{}{
            "price": squirrel.Expr("price * 0.9"),
            "updated_at": time.Now(),
        })
    
    // Complex inventory report with custom SQL
    inventoryReport, err := storm.Products.Query().
        ExecuteRaw(`
            SELECT 
                p.name,
                p.sku,
                p.stock,
                p.price,
                c.name as category_name,
                COUNT(r.id) as review_count,
                AVG(r.rating) as avg_rating
            FROM products p
            LEFT JOIN categories c ON p.category_id = c.id
            LEFT JOIN reviews r ON p.id = r.product_id
            WHERE p.is_active = true
            GROUP BY p.id, c.name
            HAVING COUNT(r.id) > $1
            ORDER BY avg_rating DESC, p.stock DESC
        `, 5)
}
```

#### 🎫 Event Management System

```go
// Event with complex scheduling
type Event struct {
    _ struct{} `dbdef:"table:events;index:idx_events_datetime,start_time,end_time;index:idx_events_venue,venue_id"`
    
    ID          string    `db:"id" dbdef:"type:uuid;primary_key;default:gen_random_uuid()"`
    Title       string    `db:"title" dbdef:"type:varchar(255);not_null"`
    StartTime   time.Time `db:"start_time" dbdef:"type:timestamptz;not_null"`
    EndTime     time.Time `db:"end_time" dbdef:"type:timestamptz;not_null"`
    VenueID     string    `db:"venue_id" dbdef:"type:uuid;not_null;foreign_key:venues.id"`
    MaxCapacity int       `db:"max_capacity" dbdef:"type:integer;not_null"`
    TicketPrice float64   `db:"ticket_price" dbdef:"type:decimal(8,2);not_null"`
    Status      string    `db:"status" dbdef:"type:varchar(50);not_null;default:'scheduled'"`
    
    // Relationships
    Venue        *Venue        `db:"-" orm:"belongs_to:Venue,foreign_key:venue_id"`
    Registrations []Registration `db:"-" orm:"has_many:Registration,foreign_key:event_id"`
    Speakers     []Speaker     `db:"-" orm:"has_many_through:Speaker,join_table:event_speakers,source_fk:event_id,target_fk:speaker_id"`
}

func EventExamples(storm *models.Storm, ctx context.Context) {
    // Find upcoming events with available seats
    upcomingEvents, err := storm.Events.Query().
        Where(models.Events.StartTime.After(time.Now()).And(
            models.Events.Status.Eq("scheduled"),
        )).
        ExecuteRaw(`
            SELECT e.*, v.name as venue_name,
                   (e.max_capacity - COUNT(r.id)) as available_seats
            FROM events e
            LEFT JOIN venues v ON e.venue_id = v.id
            LEFT JOIN registrations r ON e.id = r.event_id AND r.status = 'confirmed'
            WHERE e.start_time > $1 AND e.status = 'scheduled'
            GROUP BY e.id, v.name
            HAVING (e.max_capacity - COUNT(r.id)) > 0
            ORDER BY e.start_time ASC
        `, time.Now())
    
    // Batch registration with transaction
    err = storm.WithTransaction(ctx, func(tx *models.Storm) error {
        event, err := tx.Events.FindByID(ctx, "event-id")
        if err != nil {
            return err
        }
        
        // Check capacity
        currentRegistrations, err := tx.Registrations.Query().
            Where(models.Registrations.EventID.Eq(event.ID).And(
                models.Registrations.Status.Eq("confirmed"),
            )).
            Count()
        if err != nil {
            return err
        }
        
        if currentRegistrations >= int64(event.MaxCapacity) {
            return errors.New("event is full")
        }
        
        // Create registration
        return tx.Registrations.Create(ctx, &models.Registration{
            EventID: event.ID,
            UserID:  "user-id",
            Status:  "confirmed",
        })
    })
}
```

#### 📊 Analytics and Reporting

```go
func AnalyticsExamples(storm *models.Storm, ctx context.Context) {
    // Daily user activity report
    dailyStats, err := storm.Users.Query().
        ExecuteRaw(`
            SELECT 
                DATE(created_at) as signup_date,
                COUNT(*) as new_users,
                COUNT(*) FILTER (WHERE is_active = true) as active_users
            FROM users 
            WHERE created_at >= $1
            GROUP BY DATE(created_at)
            ORDER BY signup_date DESC
        `, time.Now().AddDate(0, 0, -30))
    
    // Revenue by category with PostgreSQL window functions
    revenueReport, err := storm.Products.Query().
        ExecuteRaw(`
            WITH category_revenue AS (
                SELECT 
                    c.name as category,
                    SUM(oi.quantity * oi.price) as revenue,
                    COUNT(DISTINCT o.id) as order_count,
                    AVG(oi.quantity * oi.price) as avg_order_value
                FROM categories c
                LEFT JOIN products p ON c.id = p.category_id
                LEFT JOIN order_items oi ON p.id = oi.product_id
                LEFT JOIN orders o ON oi.order_id = o.id
                WHERE o.created_at >= $1
                GROUP BY c.id, c.name
            )
            SELECT *,
                   revenue / SUM(revenue) OVER() * 100 as revenue_percentage,
                   RANK() OVER(ORDER BY revenue DESC) as revenue_rank
            FROM category_revenue
            ORDER BY revenue DESC
        `, time.Now().AddDate(0, -3, 0))
    
    // Customer lifetime value calculation
    clvReport, err := storm.Users.Query().
        ExecuteRaw(`
            SELECT 
                u.id,
                u.email,
                COUNT(o.id) as total_orders,
                SUM(o.total_amount) as lifetime_value,
                AVG(o.total_amount) as avg_order_value,
                EXTRACT(DAYS FROM (MAX(o.created_at) - MIN(o.created_at))) as customer_lifespan_days
            FROM users u
            LEFT JOIN orders o ON u.id = o.user_id
            WHERE o.status = 'completed'
            GROUP BY u.id, u.email
            HAVING COUNT(o.id) > 0
            ORDER BY lifetime_value DESC
            LIMIT 100
        `)
}
```

#### 🔐 Multi-tenant SaaS Application

```go
// Tenant-aware models
type Organization struct {
    _ struct{} `dbdef:"table:organizations;index:idx_orgs_subdomain,subdomain"`
    
    ID        string `db:"id" dbdef:"type:uuid;primary_key;default:gen_random_uuid()"`
    Name      string `db:"name" dbdef:"type:varchar(255);not_null"`
    Subdomain string `db:"subdomain" dbdef:"type:varchar(50);not_null;unique"`
    Plan      string `db:"plan" dbdef:"type:varchar(50);not_null;default:'free'"`
    IsActive  bool   `db:"is_active" dbdef:"type:boolean;not_null;default:true"`
    
    Users    []User    `db:"-" orm:"has_many:User,foreign_key:org_id"`
    Projects []Project `db:"-" orm:"has_many:Project,foreign_key:org_id"`
}

type User struct {
    _ struct{} `dbdef:"table:users;index:idx_users_org,org_id;index:idx_users_email,email"`
    
    ID    string `db:"id" dbdef:"type:uuid;primary_key;default:gen_random_uuid()"`
    OrgID string `db:"org_id" dbdef:"type:uuid;not_null;foreign_key:organizations.id"`
    Email string `db:"email" dbdef:"type:varchar(255);not_null"`
    Role  string `db:"role" dbdef:"type:varchar(50);not_null;default:'member'"`
    
    Organization *Organization `db:"-" orm:"belongs_to:Organization,foreign_key:org_id"`
}

func MultiTenantExamples(storm *models.Storm, ctx context.Context, orgID string) {
    // All queries are automatically scoped to organization
    orgUsers, err := storm.Users.Query().
        Where(models.Users.OrgID.Eq(orgID)).
        Include("Organization").
        Find()
    
    // Tenant-specific analytics
    orgStats, err := storm.Projects.Query().
        ExecuteRaw(`
            SELECT 
                COUNT(*) as total_projects,
                COUNT(*) FILTER (WHERE status = 'active') as active_projects,
                COUNT(DISTINCT user_id) as active_users,
                AVG(EXTRACT(DAYS FROM (updated_at - created_at))) as avg_project_duration
            FROM projects 
            WHERE org_id = $1
        `, orgID)
    
    // Cross-tenant reporting (admin only)
    tenantComparison, err := storm.Organizations.Query().
        ExecuteRaw(`
            SELECT 
                o.name,
                o.plan,
                COUNT(u.id) as user_count,
                COUNT(p.id) as project_count,
                MAX(p.created_at) as last_activity
            FROM organizations o
            LEFT JOIN users u ON o.id = u.org_id
            LEFT JOIN projects p ON o.id = p.org_id
            WHERE o.is_active = true
            GROUP BY o.id, o.name, o.plan
            ORDER BY user_count DESC
        `)
}
```

### Example Projects

Check out the [examples](examples/) directory for complete applications:

- **[Todo Application](examples/todo/)** - Full CRUD operations with relationships, user management, and categories
- **[Blog System](examples/blog/)** - Multi-author blogging platform with comments, tags, and SEO features  
- **[E-commerce Platform](examples/ecommerce/)** - Complete online store with products, orders, inventory, and payment processing
- **[Event Management](examples/events/)** - Event scheduling, registration, and venue management system
- **[Multi-tenant SaaS](examples/saas/)** - Organization-scoped data with user roles and subscription management

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

**Perfect for:**
- ✅ **Type Safety First** projects where compile-time validation is critical
- ✅ **High Performance** applications that can't afford reflection overhead
- ✅ **PostgreSQL-centric** systems leveraging advanced PostgreSQL features
- ✅ **Schema-driven** development with structs as the source of truth
- ✅ **Team productivity** with auto-generated, documented code
- ✅ **Complex relationships** requiring type-safe eager/lazy loading
- ✅ **Production systems** needing robust migration management

### When NOT to Use Storm

Consider alternatives if you need:
- ❌ **Multi-database support** (MySQL, SQLite, Oracle, etc.)
- ❌ **NoSQL databases** (MongoDB, Redis, etc.)
- ❌ **Dynamic schemas** that change frequently at runtime
- ❌ **Legacy codebases** with existing ORM deeply integrated
- ❌ **Simple CRUD apps** where basic SQL might be sufficient

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