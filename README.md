# Storm - Type-Safe PostgreSQL ORM and Migration Tool for Go

[![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://golang.org)
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

1. **Define your models** in Go structs with `db` and `storm` tags
2. **Generate migrations** by comparing structs with your database
3. **Generate ORM code** with type-safe repositories and query builders
4. **Use the generated code** with full compile-time validation

```go
// 1. Define your model
type User struct {
    ID    string `db:"id" storm:"type:uuid;primary_key"`
    Email string `db:"email" storm:"type:varchar(255);not_null;unique"`
    Posts []Post `storm:"relation:has_many:Post;foreign_key:user_id"`
}

// 2. storm migrate  (generates SQL migrations)
// 3. storm orm      (generates ORM code)

// 4. Use type-safe operations
users, err := storm.Users.Query(ctx).
    Where(Users.Email.Like("%@company.com")).
    Include("Posts").  // Type-safe eager loading
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
- [Examples](#-comprehensive-examples)
- [Why Storm?](#why-storm)
- [Middleware System](#-middleware-system)
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
# users, err := storm.Users.Query(ctx).Find()
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

// User model with storm tags for schema definition
type User struct {
    _ struct{} `storm:"table:users;index:idx_users_email,email"`
    
    ID        string    `db:"id" storm:"type:uuid;primary_key;default:gen_random_uuid()"`
    Email     string    `db:"email" storm:"type:varchar(255);not_null;unique"`
    Name      string    `db:"name" storm:"type:varchar(100);not_null"`
    CreatedAt time.Time `db:"created_at" storm:"type:timestamptz;not_null;default:now()"`
    
    // ORM relationships
    Posts []Post `storm:"relation:has_many:Post;foreign_key:user_id"`
}

// Post model
type Post struct {
    _ struct{} `storm:"table:posts;index:idx_posts_user,user_id"`
    
    ID        string    `db:"id" storm:"type:uuid;primary_key;default:gen_random_uuid()"`
    UserID    string    `db:"user_id" storm:"type:uuid;not_null;foreign_key:users.id"`
    Title     string    `db:"title" storm:"type:varchar(255);not_null"`
    Content   string    `db:"content" storm:"type:text"`
    Published bool      `db:"published" storm:"type:boolean;not_null;default:false"`
    CreatedAt time.Time `db:"created_at" storm:"type:timestamptz;not_null;default:now()"`
    
    // ORM relationships
    User *User `storm:"relation:belongs_to:User;foreign_key:user_id"`
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
- **Type-safe authorization** with generated `Authorize` methods that use model-specific query types
- **Relationship methods** like `Include("Posts")`, `Include("Comments")` on query builders for type-safe eager loading
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
    users, err := storm.Users.Query(ctx).
        Where(models.Users.Email.Like("%@company.com")).
        OrderBy(models.Users.CreatedAt.Desc()).
        Limit(10).
        Find()
    
    // Complex conditions with And/Or/Not helpers
    activePosts, err := storm.Posts.Query(ctx).
        Where(storm.And(
            models.Posts.Published.Eq(true),
            models.Posts.CreatedAt.After(time.Now().AddDate(0, -1, 0)),
        )).
        OrderBy(models.Posts.CreatedAt.Desc()).
        Find()
    
    // Advanced filtering with multiple logical operators
    searchResults, err := storm.Posts.Query(ctx).
        Where(storm.And(
            storm.Or(
                models.Posts.Title.Like("%Go%"),
                models.Posts.Content.Like("%golang%"),
            ),
            models.Posts.Published.Eq(true),
            storm.Not(models.Posts.UserID.IsNull()),
        )).
        Find()
    
    // Complex user filtering example
    targetUsers, err := storm.Users.Query(ctx).
        Where(storm.And(
            models.Users.IsActive.Eq(true),
            storm.Or(
                models.Users.Email.Contains("gmail"),
                models.Users.Email.Contains("yahoo"),
            ),
            storm.Not(models.Users.TeamID.IsNull()),
        )).
        Find()
    
    // === RELATIONSHIPS ===
    
    // Type-safe eager loading with generated methods
    usersWithPosts, err := storm.Users.Query(ctx).
        Include("Posts").  // Type-safe relationship loading
        Find()
    
    // Chain multiple relationships
    authorWithEverything, err := storm.Users.Query(ctx).
        Where(models.Users.ID.Eq(authorID)).
        Include("Posts").
        Include("Comments").
        Include("Team").
        First()
    
    // Load specific relationship conditions (still available)
    usersWithRecentPosts, err := storm.Users.Query(ctx).
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
    rowsUpdated, err := storm.Posts.Query(ctx).
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
    totalUsers, err := storm.Users.Query(ctx).Count()
    
    activeUserCount, err := storm.Users.Query(ctx).
        Where(models.Users.IsActive.Eq(true)).
        Count()
    
    // Check existence
    hasAdminUser, err := storm.Users.Query(ctx).
        Where(models.Users.Email.Eq("admin@company.com")).
        Exists()
    
    // Time-based queries
    recentPosts, err := storm.Posts.Query(ctx).
        Where(models.Posts.CreatedAt.After(time.Now().AddDate(0, 0, -7))).
        OrderBy(models.Posts.CreatedAt.Desc()).
        Find()
    
    // Pagination
    page2Users, err := storm.Users.Query(ctx).
        OrderBy(models.Users.CreatedAt.Desc()).
        Offset(20).
        Limit(10).
        Find()
    
    // === AUTHORIZATION ===
    
    // Create authorized repository with type-safe query filtering
    authorizedUsers := storm.Users.Authorize(func(ctx context.Context, query *models.UserQuery) *models.UserQuery {
        // Extract tenant from context
        tenantID := ctx.Value("tenant_id").(string)
        return query.Where(models.Users.TenantID.Eq(tenantID))
    })
    
    // All queries through authorized repo will include tenant filter
    tenantUsers, err := authorizedUsers.Query(ctx).
        Where(models.Users.IsActive.Eq(true)).
        Find() // Automatically filtered by tenant
    
    // === JOINS & RELATIONSHIPS ===
    
    // Get users with their posts using type-safe Include methods
    usersWithPosts, err := storm.Users.Query(ctx).
        Where(models.Users.CreatedAt.After(time.Now().AddDate(0, -1, 0))).
        Include("Posts").  // Type-safe relationship loading
        OrderBy(models.Users.CreatedAt.Desc()).
        Find()
}
```

## Core Concepts

### 🏗️ Struct-Driven Development

Your Go structs define your database schema using `storm` tags. Storm ensures your database always matches your structs.

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
users, _ := storm.Users.Query(ctx).
    Where(models.Users.InvalidColumn.Eq("value")). // Compiler error!
    Find()

// ✅ Type mismatch caught at compile time
users, _ := storm.Users.Query(ctx).
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
- 🏷️ **[Schema Definition (storm tags)](docs/schema-definition.md)** - Complete tag reference
- 🔄 **[Migrations Guide](docs/migrations.md)** - Managing database changes (Coming Soon)
- 📊 **[ORM Guide](docs/orm-guide.md)** - Using the generated ORM
- 🔍 **[Query Builder](docs/query-builder.md)** - Building complex queries
- 🔌 **[Relationships](docs/relationships.md)** - Defining and using relationships (Coming Soon)
- ⚡ **[Performance Guide](docs/performance.md)** - Optimization tips (Coming Soon)
- 🔧 **[CLI Reference](docs/cli-reference.md)** - All commands and options

## 📚 Comprehensive Examples

### Real-World Use Cases

#### 📝 E-commerce Product Management

```go
// Product model with rich validation
type Product struct {
    _ struct{} `storm:"table:products;index:idx_products_category,category_id;index:idx_products_price,price"`
    
    ID          string          `db:"id" storm:"type:uuid;primary_key;default:gen_random_uuid()"`
    SKU         string          `db:"sku" storm:"type:varchar(50);not_null;unique"`
    Name        string          `db:"name" storm:"type:varchar(255);not_null"`
    Description string          `db:"description" storm:"type:text"`
    Price       decimal.Decimal `db:"price" storm:"type:decimal(10,2);not_null"`
    Stock       int             `db:"stock" storm:"type:integer;not_null;default:0"`
    CategoryID  string          `db:"category_id" storm:"type:uuid;not_null;foreign_key:categories.id"`
    IsActive    bool            `db:"is_active" storm:"type:boolean;not_null;default:true"`
    CreatedAt   time.Time       `db:"created_at" storm:"type:timestamptz;not_null;default:now()"`
    UpdatedAt   time.Time       `db:"updated_at" storm:"type:timestamptz;not_null;default:now()"`
    
    // Relationships
    Category *Category `storm:"relation:belongs_to:Category;foreign_key:category_id"`
    Reviews  []Review  `storm:"relation:has_many:Review;foreign_key:product_id"`
    Tags     []Tag     `storm:"relation:has_many_through:Tag;join_table:product_tags;source_fk:product_id;target_fk:tag_id"`
}

// Advanced product queries
func ProductExamples(storm *models.Storm, ctx context.Context) {
    // Find products in stock within price range
    inStockProducts, err := storm.Products.Query(ctx).
        Where(storm.And(
            models.Products.Stock.Gt(0),
            models.Products.Price.Between(decimal.NewFromFloat(10.0), decimal.NewFromFloat(100.0)),
            models.Products.IsActive.Eq(true),
        )).
        Include("Category", "Tags").
        OrderBy(models.Products.Price.Asc()).
        Find()
    
    // Search products by name or description with complex logic
    searchResults, err := storm.Products.Query(ctx).
        Where(storm.And(
            storm.Or(
                models.Products.Name.Like("%laptop%"),
                models.Products.Description.Like("%computer%"),
            ),
            models.Products.IsActive.Eq(true),
            storm.Not(models.Products.CategoryID.IsNull()),
        )).
        Find()
    
    // Bulk update prices with 10% discount
    discountedRows, err := storm.Products.Query(ctx).
        Where(models.Products.CategoryID.Eq("electronics-category-id")).
        UpdateMany(ctx, map[string]interface{}{
            "price": squirrel.Expr("price * 0.9"),
            "updated_at": time.Now(),
        })
    
    // Popular products with low stock alert
    lowStockProducts, err := storm.Products.Query(ctx).
        Where(storm.And(
            models.Products.IsActive.Eq(true),
            models.Products.Stock.Lt(10),
        )).
        Include("Category", "Reviews").
        OrderBy(models.Products.Stock.Asc()).
        Limit(20).
        Find()
}
```

#### 🎫 Event Management System

```go
// Event with complex scheduling
type Event struct {
    _ struct{} `storm:"table:events;index:idx_events_datetime,start_time,end_time;index:idx_events_venue,venue_id"`
    
    ID          string    `db:"id" storm:"type:uuid;primary_key;default:gen_random_uuid()"`
    Title       string    `db:"title" storm:"type:varchar(255);not_null"`
    StartTime   time.Time `db:"start_time" storm:"type:timestamptz;not_null"`
    EndTime     time.Time `db:"end_time" storm:"type:timestamptz;not_null"`
    VenueID     string    `db:"venue_id" storm:"type:uuid;not_null;foreign_key:venues.id"`
    MaxCapacity int       `db:"max_capacity" storm:"type:integer;not_null"`
    TicketPrice float64   `db:"ticket_price" storm:"type:decimal(8,2);not_null"`
    Status      string    `db:"status" storm:"type:varchar(50);not_null;default:'scheduled'"`
    
    // Relationships
    Venue        *Venue        `storm:"relation:belongs_to:Venue;foreign_key:venue_id"`
    Registrations []Registration `storm:"relation:has_many:Registration;foreign_key:event_id"`
    Speakers     []Speaker     `storm:"relation:has_many_through:Speaker;join_table:event_speakers;source_fk:event_id;target_fk:speaker_id"`
}

func EventExamples(storm *models.Storm, ctx context.Context) {
    // Find upcoming events with venue information
    upcomingEvents, err := storm.Events.Query(ctx).
        Where(storm.And(
            models.Events.StartTime.After(time.Now()),
            models.Events.Status.Eq("scheduled"),
        )).
        Include("Venue").                                // Type-safe venue loading
        IncludeWhere("Registrations",                  // Load specific registrations
            models.Registrations.Status.Eq("confirmed"),
        ).
        OrderBy(models.Events.StartTime.Asc()).
        Limit(10).
        Find()
    
    // Batch registration with transaction
    err = storm.WithTransaction(ctx, func(tx *models.Storm) error {
        event, err := tx.Events.FindByID(ctx, "event-id")
        if err != nil {
            return err
        }
        
        // Check capacity with complex conditions
        currentRegistrations, err := tx.Registrations.Query(ctx).
            Where(storm.And(
                models.Registrations.EventID.Eq(event.ID),
                models.Registrations.Status.Eq("confirmed"),
                storm.Not(models.Registrations.CancelledAt.IsNotNull()),
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
    // Get active users from last 30 days
    activeRecentUsers, err := storm.Users.Query(ctx).
        Where(storm.And(
            models.Users.CreatedAt.After(time.Now().AddDate(0, 0, -30)),
            models.Users.IsActive.Eq(true),
        )).
        OrderBy(models.Users.CreatedAt.Desc()).
        Find()
    
    // Count total vs active
    totalRecentUsers, err := storm.Users.Query(ctx).
        Where(models.Users.CreatedAt.After(time.Now().AddDate(0, 0, -30))).
        Count()
    
    // Get high-value orders from last quarter
    highValueOrders, err := storm.Orders.Query(ctx).
        Where(storm.And(
            models.Orders.CreatedAt.After(time.Now().AddDate(0, -3, 0)),
            models.Orders.Status.Eq("completed"),
            models.Orders.TotalAmount.Gt(decimal.NewFromInt(1000)),
        )).
        Include("Customer").                             // Type-safe customer loading
        Include("OrderItems").                           // Type-safe order items loading
        OrderBy(models.Orders.TotalAmount.Desc()).
        Limit(100).
        Find()
    
    // Get VIP customers with high lifetime value
    vipCustomers, err := storm.Users.Query(ctx).
        Include("Orders").                               // Type-safe order loading
        Where(models.Users.IsActive.Eq(true)).
        Find()
    
    // Filter for completed orders in memory or use IncludeWhere for conditional loading
    vipWithCompletedOrders, err := storm.Users.Query(ctx).
        IncludeWhere("Orders",                         // Load only completed orders
            models.Orders.Status.Eq("completed"),
        ).
        Where(models.Users.IsActive.Eq(true)).
        Find()
    
    // Simple filtering in memory for customers with 10+ orders
    var topCustomers []*models.User
    for _, customer := range vipCustomers {
        if len(customer.Orders) >= 10 {
            topCustomers = append(topCustomers, customer)
        }
    }
}
```

#### 🔐 Multi-tenant SaaS Application

```go
// Tenant-aware models
type Organization struct {
    _ struct{} `storm:"table:organizations;index:idx_orgs_subdomain,subdomain"`
    
    ID        string `db:"id" storm:"type:uuid;primary_key;default:gen_random_uuid()"`
    Name      string `db:"name" storm:"type:varchar(255);not_null"`
    Subdomain string `db:"subdomain" storm:"type:varchar(50);not_null;unique"`
    Plan      string `db:"plan" storm:"type:varchar(50);not_null;default:'free'"`
    IsActive  bool   `db:"is_active" storm:"type:boolean;not_null;default:true"`
    
    Users    []User    `storm:"relation:has_many:User;foreign_key:org_id"`
    Projects []Project `storm:"relation:has_many:Project;foreign_key:org_id"`
}

type User struct {
    _ struct{} `storm:"table:users;index:idx_users_org,org_id;index:idx_users_email,email"`
    
    ID    string `db:"id" storm:"type:uuid;primary_key;default:gen_random_uuid()"`
    OrgID string `db:"org_id" storm:"type:uuid;not_null;foreign_key:organizations.id"`
    Email string `db:"email" storm:"type:varchar(255);not_null"`
    Role  string `db:"role" storm:"type:varchar(50);not_null;default:'member'"`
    
    Organization *Organization `storm:"relation:belongs_to:Organization;foreign_key:org_id"`
}

func MultiTenantExamples(storm *models.Storm, ctx context.Context, orgID string, userID string, userRole string) {
    // Find active admin users for organization
    adminUsers, err := storm.Users.Query(ctx).
        Where(storm.And(
            models.Users.OrgID.Eq(orgID),
            storm.Or(
                models.Users.Role.Eq("admin"),
                models.Users.Role.Eq("owner"),
            ),
            models.Users.IsActive.Eq(true),
        )).
        Find()
    
    // Tenant-specific project counts
    totalProjects, err := storm.Projects.Query(ctx).
        Where(models.Projects.OrgID.Eq(orgID)).
        Count()
    
    activeProjects, err := storm.Projects.Query(ctx).
        Where(storm.And(
            models.Projects.OrgID.Eq(orgID),
            models.Projects.Status.Eq("active"),
        )).
        Count()
    
    // Get projects with their users
    projectsWithUsers, err := storm.Projects.Query(ctx).
        Where(models.Projects.OrgID.Eq(orgID)).
        Include("AssignedUsers").                        // Type-safe user loading
        Find()
    
    // Cross-tenant reporting (admin only)
    activeOrgs, err := storm.Organizations.Query(ctx).
        Where(models.Organizations.IsActive.Eq(true)).
        Include("Users").                                // Type-safe user loading
        Include("Projects").                             // Type-safe project loading
        OrderBy(models.Organizations.CreatedAt.Desc()).
        Find()
    
    // Build report from loaded data
    type TenantReport struct {
        Name         string
        Plan         string
        UserCount    int
        ProjectCount int
        LastActivity time.Time
    }
    
    var tenantReports []TenantReport
    for _, org := range activeOrgs {
        lastActivity := org.CreatedAt
        for _, project := range org.Projects {
            if project.CreatedAt.After(lastActivity) {
                lastActivity = project.CreatedAt
            }
        }
        
        tenantReports = append(tenantReports, TenantReport{
            Name:         org.Name,
            Plan:         org.Plan,
            UserCount:    len(org.Users),
            ProjectCount: len(org.Projects),
            LastActivity: lastActivity,
        })
    }
}
```

### Example Projects

Complete example applications are coming soon! We're working on:

- **Todo Application** - Full CRUD operations with relationships, user management, and categories
- **Blog System** - Multi-author blogging platform with comments, tags, and SEO features  
- **E-commerce Platform** - Complete online store with products, orders, inventory, and payment processing
- **Event Management** - Event scheduling, registration, and venue management system
- **Multi-tenant SaaS** - Organization-scoped data with user roles and subscription management

For now, refer to the comprehensive examples in the sections above and the [Getting Started Guide](docs/getting-started.md).

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

## 🔧 Middleware System

Storm provides a powerful middleware system that allows you to intercept and modify database operations. This is essential for production applications that need features like multi-tenancy, audit logging, soft deletes, authorization, and performance monitoring.

### How Middleware Works

Middleware functions wrap around database operations, giving you access to:
- **Operation context** (create, update, delete, query)
- **Query builders** before execution
- **Table and model information**
- **Custom metadata** for request tracking
- **Timing and performance data**

### Basic Middleware Structure

```go
repo.AddMiddleware(func(next QueryMiddlewareFunc) QueryMiddlewareFunc {
    return func(ctx *MiddlewareContext) error {
        // Before operation
        // Modify ctx.QueryBuilder, add metadata, validate, etc.
        
        err := next(ctx) // Execute the operation
        
        // After operation
        // Log results, handle errors, etc.
        
        return err
    }
})
```

### Production-Ready Examples

#### 🏢 Multi-Tenancy & Authorization

Storm provides type-safe authorization through generated repository methods:

```go
// Create authorized repository with type-safe filtering
authorizedUsers := storm.Users.Authorize(func(ctx context.Context, query *models.UserQuery) *models.UserQuery {
    tenantID := ctx.Value("tenant_id").(string)
    return query.Where(models.Users.TenantID.Eq(tenantID))
})

// All queries through authorized repo automatically include tenant filter
users, err := authorizedUsers.Query(ctx).
    Where(models.Users.IsActive.Eq(true)).
    OrderBy(models.Users.CreatedAt.Desc()).
    Find() // Automatically filtered by tenant

// Complex authorization with visibility rules
authorizedPosts := storm.Posts.Authorize(func(ctx context.Context, query *models.PostQuery) *models.PostQuery {
    user := ctx.Value("user").(*User)
    return query.Where(storm.And(
        models.Posts.TenantID.Eq(user.TenantID),
        storm.Or(
            models.Posts.Visibility.Eq("public"),
            models.Posts.TeamID.In(user.TeamIDs...),
            models.Posts.AuthorID.Eq(user.ID),
        ),
    ))
})

// Type-safe relationship loading works with authorization
visiblePosts, err := authorizedPosts.Query(ctx).
    Where(models.Posts.Published.Eq(true)).
    Include("Author").
    Include("Tags").
    Find()

// For create operations, explicitly set tenant
newUser := &User{
    TenantID: currentTenantID, // Explicit tenant assignment
    Email:    "user@example.com",
    Role:     "member",
}
err = storm.Users.Create(ctx, newUser)

// For updates, use authorized repository
rowsUpdated, err := authorizedUsers.Query(ctx).
    Where(models.Users.Role.Eq("trial")).
    UpdateMany(map[string]interface{}{
        "role": "expired",
        "updated_at": time.Now(),
    })
```

#### 🔐 Row-Level Security Patterns

```go
// Define reusable authorization helpers
type AuthFilters struct {
    UserID   string
    TenantID string
    Role     string
    TeamIDs  []string
}

// Create domain-specific query functions
func GetVisibleProjects(storm *models.Storm, ctx context.Context, auth AuthFilters) ([]Project, error) {
    baseQuery := storm.Projects.Query(ctx).
        Where(Projects.TenantID.Eq(auth.TenantID))
    
    switch auth.Role {
    case "admin":
        // Admins see all projects in tenant
        return baseQuery.Find()
    case "pm":
        // Project managers see their projects + public ones
        return baseQuery.Where(storm.Or(
            Projects.ManagerID.Eq(auth.UserID),
            Projects.Visibility.Eq("public"),
            Projects.TeamID.In(auth.TeamIDs...),
        )).Find()
    default:
        // Members only see projects they're assigned to
        return baseQuery.Where(storm.Or(
            Projects.OwnerID.Eq(auth.UserID),
            Projects.MemberIDs.Contains(auth.UserID),
            storm.And(
                Projects.Visibility.Eq("public"),
                Projects.TeamID.In(auth.TeamIDs...),
            ),
        )).Find()
    }
}

// Usage remains clean and explicit
projects, err := GetVisibleProjects(storm, ctx, authFilters)
```

#### 🗑️ Soft Delete System

Convert hard deletes to soft deletes and filter out deleted records:

```go
func AddSoftDeleteMiddleware(repo *Repository[T]) {
    repo.AddMiddleware(func(next QueryMiddlewareFunc) QueryMiddlewareFunc {
        return func(ctx *MiddlewareContext) error {
            switch ctx.Operation {
            case OpQuery:
                if sb, ok := ctx.QueryBuilder.(squirrel.SelectBuilder); ok {
                    // Automatically filter out soft-deleted records
                    ctx.QueryBuilder = sb.Where(squirrel.Eq{"deleted_at": nil})
                }
            case OpDelete:
                // Convert DELETE to UPDATE with deleted_at timestamp
                ctx.Operation = OpUpdate
                ctx.QueryBuilder = squirrel.Update(ctx.TableName).
                    Set("deleted_at", time.Now()).
                    Where(ctx.QueryBuilder.(squirrel.DeleteBuilder).WhereParts...)
            }
            return next(ctx)
        }
    })
}

// Usage
AddSoftDeleteMiddleware(storm.Users)
err := storm.Users.Delete(ctx, userID) // Sets deleted_at instead of removing
users, err := storm.Users.Query(ctx).Find() // Only returns non-deleted users
```

#### 🔍 Query Tracing & Debugging

Use middleware for query inspection and debugging:

```go
func AddQueryTracingMiddleware(repo *Repository[T], logger *log.Logger) {
    repo.AddMiddleware(func(next QueryMiddlewareFunc) QueryMiddlewareFunc {
        return func(ctx *MiddlewareContext) error {
            // Log the query before execution
            if ctx.Query != "" {
                logger.Debug("Executing query", map[string]interface{}{
                    "table":     ctx.TableName,
                    "operation": string(ctx.Operation),
                    "query":     ctx.Query,
                    "args":      ctx.Args,
                })
            }
            
            err := next(ctx)
            
            if err != nil {
                logger.Error("Query failed", map[string]interface{}{
                    "error": err.Error(),
                    "query": ctx.Query,
                })
            }
            
            return err
        }
    })
}

// Usage - helpful for development and debugging
if config.DebugMode {
    AddQueryTracingMiddleware(storm.Users, debugLogger)
}
```

#### 📊 Audit Logging

Automatically log all database operations:

```go
type AuditLogger struct {
    logger *log.Logger
    userID string
}

func (al *AuditLogger) AddAuditMiddleware(repo *Repository[T]) {
    repo.AddMiddleware(func(next QueryMiddlewareFunc) QueryMiddlewareFunc {
        return func(ctx *MiddlewareContext) error {
            // Capture start time
            startTime := time.Now()
            
            // Add user context
            ctx.Metadata["user_id"] = al.userID
            ctx.Metadata["operation_id"] = generateOperationID()
            
            // Execute operation
            err := next(ctx)
            
            // Log the operation
            duration := time.Since(startTime)
            logEntry := map[string]interface{}{
                "table":      ctx.TableName,
                "operation":  string(ctx.Operation),
                "user_id":    al.userID,
                "duration":   duration.Milliseconds(),
                "success":    err == nil,
                "timestamp":  time.Now().UTC(),
            }
            
            if err != nil {
                logEntry["error"] = err.Error()
                al.logger.Error("Database operation failed", logEntry)
            } else {
                al.logger.Info("Database operation completed", logEntry)
            }
            
            return err
        }
    })
}

// Usage
auditLogger := &AuditLogger{
    logger: myLogger,
    userID: getCurrentUserID(request),
}
auditLogger.AddAuditMiddleware(storm.Users)
```

#### ⚡ Performance Monitoring

Track query performance and detect slow operations:

```go
func AddPerformanceMiddleware(repo *Repository[T], slowThreshold time.Duration) {
    repo.AddMiddleware(func(next QueryMiddlewareFunc) QueryMiddlewareFunc {
        return func(ctx *MiddlewareContext) error {
            startTime := time.Now()
            
            err := next(ctx)
            
            duration := time.Since(startTime)
            if duration > slowThreshold {
                log.Warn("Slow query detected", map[string]interface{}{
                    "table":     ctx.TableName,
                    "operation": string(ctx.Operation),
                    "duration":  duration.Milliseconds(),
                    "query":     ctx.Query,
                })
            }
            
            // Send metrics to monitoring system
            metrics.Histogram("db.operation.duration",
                float64(duration.Milliseconds()),
                map[string]string{
                    "table":     ctx.TableName,
                    "operation": string(ctx.Operation),
                })
            
            return err
        }
    })
}

// Usage
AddPerformanceMiddleware(storm.Users, 100*time.Millisecond)
```

#### 🔄 Request Context Integration

Pass HTTP request context through to database operations:

```go
func AddRequestContextMiddleware(repo *Repository[T]) {
    repo.AddMiddleware(func(next QueryMiddlewareFunc) QueryMiddlewareFunc {
        return func(ctx *MiddlewareContext) error {
            // Extract request ID from context
            if requestID := ctx.Context.Value("request_id"); requestID != nil {
                ctx.Metadata["request_id"] = requestID
            }
            
            // Extract user information
            if user := ctx.Context.Value("user"); user != nil {
                ctx.Metadata["user"] = user
            }
            
            // Check for request cancellation
            select {
            case <-ctx.Context.Done():
                return ctx.Context.Err()
            default:
            }
            
            return next(ctx)
        }
    })
}

// Usage with HTTP handler
func UserHandler(w http.ResponseWriter, r *http.Request) {
    ctx := context.WithValue(r.Context(), "request_id", generateRequestID())
    ctx = context.WithValue(ctx, "user", getCurrentUser(r))
    
    AddRequestContextMiddleware(storm.Users)
    users, err := storm.Users.Query(ctx).Find()
    // Request context flows through middleware
}
```

### Middleware Best Practices

Use middleware for cross-cutting concerns that don't affect query logic:

```go
// ✅ GOOD: Middleware for operational concerns
repo.AddMiddleware(performanceMiddleware)  // Monitor slow queries
repo.AddMiddleware(auditMiddleware)        // Log all operations
repo.AddMiddleware(retryMiddleware)        // Retry on connection errors
repo.AddMiddleware(circuitBreakerMiddleware) // Prevent cascading failures

// ❌ AVOID: Using middleware for business logic
// Instead of hiding filters in middleware:
// repo.AddMiddleware(tenantFilterMiddleware)
// repo.AddMiddleware(authorizationMiddleware)

// ✅ BETTER: Make filtering explicit in queries
users, err := storm.Users.Query(ctx).
    Where(storm.And(
        Users.TenantID.Eq(tenantID),     // Explicit tenant scope
        Users.IsActive.Eq(true),
    )).
    Find()
```

### When to Use Middleware vs Query Methods

| Use Case | Middleware | Query Method |
|----------|------------|---------------|
| Multi-tenancy filtering | ❌ Hidden magic | ✅ Explicit `.Where()` |
| Authorization rules | ❌ Hard to test | ✅ Explicit filtering |
| Soft deletes | ✅ Transparent | ✅ Or use `.NotDeleted()` |
| Audit logging | ✅ Cross-cutting | ❌ Too verbose |
| Performance monitoring | ✅ Operational | ❌ Not business logic |
| Query retry | ✅ Infrastructure | ❌ Not domain concern |
| Request context | ✅ Pass-through | ❌ Automatic |

### Available Operation Types

```go
const (
    OpCreate     OperationType = "create"      // Single record insert
    OpCreateMany OperationType = "create_many" // Bulk insert
    OpUpdate     OperationType = "update"      // Single record update
    OpUpdateMany OperationType = "update_many" // Bulk update
    OpDelete     OperationType = "delete"      // Delete operation
    OpUpsert     OperationType = "upsert"      // Insert or update
    OpUpsertMany OperationType = "upsert_many" // Bulk upsert
    OpBulkUpdate OperationType = "bulk_update" // Bulk update with VALUES
    OpFind       OperationType = "find"        // Single record select
    OpQuery      OperationType = "query"       // Multi-record select
)
```

### Advanced Patterns

#### Conditional Middleware

Apply middleware only for specific conditions:

```go
func ConditionalMiddleware(condition func(*MiddlewareContext) bool, middleware QueryMiddleware) QueryMiddleware {
    return func(next QueryMiddlewareFunc) QueryMiddlewareFunc {
        return func(ctx *MiddlewareContext) error {
            if condition(ctx) {
                return middleware(next)(ctx)
            }
            return next(ctx)
        }
    }
}

// Usage: Only apply tenancy to specific tables
repo.AddMiddleware(ConditionalMiddleware(
    func(ctx *MiddlewareContext) bool {
        return ctx.TableName != "system_config" // Skip tenancy for system tables
    },
    tenancyMiddleware,
))
```

The middleware system makes Storm production-ready by providing the hooks needed for enterprise features while maintaining type safety and performance.

## Contributing

We welcome contributions! Contribution guidelines are coming soon.

## License

Storm is released under the MIT License. See [LICENSE](LICENSE) for details.

## Support

- 📖 [Documentation](docs/)
- 💬 [GitHub Discussions](https://github.com/eleven-am/storm/discussions) (Coming Soon)
- 🐛 [Issue Tracker](https://github.com/eleven-am/storm/issues) (Coming Soon)

---

Built with ❤️ by [Roy OSSAI](https://github.com/eleven-am) for the Go and PostgreSQL communities.