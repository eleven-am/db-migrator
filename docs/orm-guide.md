# ORM Guide

Storm's ORM generator creates type-safe database access code from your models. This guide covers all ORM features.

## Table of Contents

- [Generated Files](#generated-files)
- [Basic CRUD Operations](#basic-crud-operations)
- [Query Builder](#query-builder)
- [Relationships](#relationships)
- [Transactions](#transactions)
- [Hooks](#hooks)
- [Advanced Features](#advanced-features)

## Generated Files

When you run `storm orm`, it generates:

```
models/
├── storm.go           # Main Storm instance
├── columns.go         # Type-safe column references
├── *_repository.go    # Repository for each model
├── *_query.go         # Query builder for each model
└── relationships.go   # Relationship helpers
```

## Basic CRUD Operations

### Create

```go
// Single record
user := &models.User{
    Email:    "john@example.com",
    Username: "johndoe",
}
err := storm.Users.Create(ctx, user)
// user.ID is now populated

// Multiple records
users := []*models.User{
    {Email: "user1@example.com", Username: "user1"},
    {Email: "user2@example.com", Username: "user2"},
}
err := storm.Users.CreateBatch(ctx, users)
```

### Read

```go
// Find by primary key
user, err := storm.Users.Find(ctx, "user-id-123")

// Find with query
user, err := storm.Users.Query().
    Where(models.Users.Email.Eq("john@example.com")).
    First()

// Find multiple
users, err := storm.Users.Query().
    Where(models.Users.IsActive.Eq(true)).
    Find()

// Check existence
exists, err := storm.Users.Query().
    Where(models.Users.Email.Eq("john@example.com")).
    Exists()
```

### Update

```go
// Update single record
user.Username = "newusername"
err := storm.Users.Update(ctx, user)

// Update specific fields
err := storm.Users.UpdateFields(ctx, user.ID, map[string]interface{}{
    "username": "newusername",
    "updated_at": time.Now(),
})

// Update with query
affected, err := storm.Users.Query().
    Where(models.Users.IsActive.Eq(false)).
    Update(map[string]interface{}{
        "deleted_at": time.Now(),
    })
```

### Delete

```go
// Delete by primary key
err := storm.Users.Delete(ctx, "user-id-123")

// Delete by model
err := storm.Users.Delete(ctx, user.ID)

// Delete with query
affected, err := storm.Users.Query().
    Where(models.Users.CreatedAt.Lt(thirtyDaysAgo)).
    Delete()

// Soft delete (if your model has DeletedAt)
err := storm.Users.SoftDelete(ctx, user.ID)
```

## Query Builder

### Basic Queries

```go
// Simple where clause
users, err := storm.Users.Query().
    Where(models.Users.IsActive.Eq(true)).
    Find()

// Multiple conditions (AND)
users, err := storm.Users.Query().
    Where(models.Users.IsActive.Eq(true)).
    Where(models.Users.Role.Eq("admin")).
    Find()

// OR conditions
users, err := storm.Users.Query().
    Where(storm.Or(
        models.Users.Role.Eq("admin"),
        models.Users.Role.Eq("moderator"),
    )).
    Find()
```

### Column Operations

```go
// Comparison operators
Where(models.Users.Age.Gt(18))          // >
Where(models.Users.Age.Gte(18))         // >=
Where(models.Users.Age.Lt(65))          // <
Where(models.Users.Age.Lte(65))         // <=
Where(models.Users.Age.Eq(25))          // =
Where(models.Users.Age.NotEq(25))       // !=

// IN/NOT IN
Where(models.Users.Status.In("active", "pending"))
Where(models.Users.Status.NotIn("deleted", "banned"))

// NULL checks
Where(models.Users.DeletedAt.IsNull())
Where(models.Users.DeletedAt.IsNotNull())

// String operations
Where(models.Users.Email.Like("%@example.com"))
Where(models.Users.Name.ILike("john%"))      // Case insensitive
Where(models.Users.Name.StartsWith("John"))
Where(models.Users.Name.EndsWith("son"))
Where(models.Users.Name.Contains("oh"))

// Date/Time operations
Where(models.Users.CreatedAt.After(yesterday))
Where(models.Users.CreatedAt.Before(tomorrow))
Where(models.Users.CreatedAt.Between(start, end))
Where(models.Users.CreatedAt.Today())
Where(models.Users.CreatedAt.ThisWeek())
Where(models.Users.CreatedAt.ThisMonth())
Where(models.Users.CreatedAt.LastNDays(7))
```

### Complex Queries

```go
// Nested conditions
users, err := storm.Users.Query().
    Where(storm.And(
        models.Users.IsActive.Eq(true),
        storm.Or(
            models.Users.Role.Eq("admin"),
            storm.And(
                models.Users.Role.Eq("user"),
                models.Users.CreatedAt.After(lastWeek),
            ),
        ),
    )).
    Find()

// Using method chaining
users, err := storm.Users.Query().
    Where(
        models.Users.IsActive.Eq(true).And(
            models.Users.Role.Eq("admin").Or(
                models.Users.Email.EndsWith("@company.com"),
            ),
        ),
    ).
    Find()
```

### Ordering and Limiting

```go
// Order by
users, err := storm.Users.Query().
    OrderBy(models.Users.CreatedAt.Desc()).
    Find()

// Multiple order by
users, err := storm.Users.Query().
    OrderBy(models.Users.Role.Asc()).
    OrderBy(models.Users.CreatedAt.Desc()).
    Find()

// Limit and offset
users, err := storm.Users.Query().
    Limit(10).
    Offset(20).
    Find()

// First/Last
firstUser, err := storm.Users.Query().
    OrderBy(models.Users.CreatedAt.Asc()).
    First()

lastUser, err := storm.Users.Query().
    OrderBy(models.Users.CreatedAt.Asc()).
    Last()
```

### Aggregations

```go
// Count
count, err := storm.Users.Query().
    Where(models.Users.IsActive.Eq(true)).
    Count()

// Distinct count
count, err := storm.Users.Query().
    CountDistinct(models.Users.Email)

// Sum, Avg, Min, Max
total, err := storm.Orders.Query().
    Sum(models.Orders.TotalAmount)

average, err := storm.Orders.Query().
    Where(models.Orders.Status.Eq("completed")).
    Avg(models.Orders.TotalAmount)
```

### Selecting Specific Columns

```go
// Select specific columns
type UserEmail struct {
    ID    string
    Email string
}

var results []UserEmail
err := storm.Users.Query().
    Select("id", "email").
    Where(models.Users.IsActive.Eq(true)).
    Scan(&results)
```

## Relationships

### Loading Relationships

```go
// Load single relationship
users, err := storm.Users.Query().
    With("Posts").
    Find()

// Load multiple relationships
users, err := storm.Users.Query().
    With("Posts", "Comments").
    Find()

// Load nested relationships
users, err := storm.Users.Query().
    With("Posts.Comments").
    Find()

// Conditional relationship loading
users, err := storm.Users.Query().
    WithWhere("Posts", models.Posts.Published.Eq(true)).
    Find()
```

### Querying Through Relationships

```go
// Find users who have posts
users, err := storm.Users.Query().
    Join("Posts").
    Where(models.Posts.Published.Eq(true)).
    Distinct().
    Find()

// Complex relationship queries
users, err := storm.Users.Query().
    Join("Posts").
    Join("Posts.Comments").
    Where(models.Comments.CreatedAt.After(lastWeek)).
    Find()
```

## Transactions

### Basic Transactions

```go
err := storm.WithTransaction(ctx, func(tx *models.Storm) error {
    user := &models.User{
        Email: "john@example.com",
    }
    if err := tx.Users.Create(ctx, user); err != nil {
        return err // Rollback
    }
    
    post := &models.Post{
        UserID: user.ID,
        Title:  "First Post",
    }
    if err := tx.Posts.Create(ctx, post); err != nil {
        return err // Rollback
    }
    
    return nil // Commit
})
```

### Transaction Options

```go
import "github.com/eleven-am/storm/internal/orm"

opts := &orm.TransactionOptions{
    Isolation: sql.LevelSerializable,
    ReadOnly:  false,
}

err := storm.WithTransactionOptions(ctx, opts, func(tx *models.Storm) error {
    // Transaction code
    return nil
})
```

### Savepoints

```go
err := storm.WithTransaction(ctx, func(tx *models.Storm) error {
    // First operation
    user := &models.User{Email: "user1@example.com"}
    tx.Users.Create(ctx, user)
    
    // Nested transaction (savepoint)
    err := tx.WithTransaction(ctx, func(tx2 *models.Storm) error {
        // This can rollback without affecting the outer transaction
        return someRiskyOperation(tx2)
    })
    
    if err != nil {
        // Handle error but continue outer transaction
        log.Printf("Risky operation failed: %v", err)
    }
    
    // Continue with outer transaction
    return nil
})
```

## Hooks

If generated with `--hooks` flag:

### Available Hooks

```go
type UserHooks interface {
    BeforeCreate(ctx context.Context, user *User) error
    AfterCreate(ctx context.Context, user *User) error
    BeforeUpdate(ctx context.Context, user *User) error
    AfterUpdate(ctx context.Context, user *User) error
    BeforeDelete(ctx context.Context, user *User) error
    AfterDelete(ctx context.Context, user *User) error
}
```

### Implementing Hooks

```go
type MyUserHooks struct{}

func (h *MyUserHooks) BeforeCreate(ctx context.Context, user *models.User) error {
    // Validate or modify user before creation
    if user.Email == "" {
        return errors.New("email is required")
    }
    
    // Hash password
    user.Password = hashPassword(user.Password)
    return nil
}

func (h *MyUserHooks) AfterCreate(ctx context.Context, user *models.User) error {
    // Send welcome email
    go sendWelcomeEmail(user.Email)
    return nil
}

// Register hooks
storm.Users.RegisterHooks(&MyUserHooks{})
```

## Advanced Features

### Batch Operations

```go
// Batch insert
users := []*models.User{
    {Email: "user1@example.com"},
    {Email: "user2@example.com"},
    // ... hundreds more
}

// Insert in batches of 100
err := storm.Users.CreateBatch(ctx, users, orm.BatchSize(100))

// Batch update
err := storm.Users.Query().
    Where(models.Users.Role.Eq("user")).
    BatchUpdate(ctx, map[string]interface{}{
        "role": "member",
    }, orm.BatchSize(500))
```

### Locking

```go
// Row-level locking
user, err := storm.Users.Query().
    Where(models.Users.ID.Eq(userID)).
    ForUpdate().
    First()

// Skip locked rows
users, err := storm.Users.Query().
    Where(models.Users.Status.Eq("pending")).
    ForUpdate().
    SkipLocked().
    Limit(10).
    Find()
```

### Raw SQL

```go
// Raw query
var users []models.User
err := storm.Raw("SELECT * FROM users WHERE email LIKE $1", "%@example.com").
    Scan(&users)

// Raw exec
result, err := storm.Exec("UPDATE users SET last_login = NOW() WHERE id = $1", userID)
```

### Query Debugging

```go
// Print generated SQL
query := storm.Users.Query().
    Where(models.Users.IsActive.Eq(true)).
    Where(models.Users.Role.In("admin", "moderator"))

sql, args := query.ToSQL()
fmt.Printf("SQL: %s\nArgs: %v\n", sql, args)

// Enable query logging
storm.EnableQueryLogging(true)
```

### Performance Optimization

```go
// Preload relationships efficiently
users, err := storm.Users.Query().
    With("Posts", "Comments"). // Loads in 3 queries total
    Find()

// Use includes for single query (if supported)
users, err := storm.Users.Query().
    Include("Posts").
    Find()

// Select only needed columns
type UserSummary struct {
    ID       string
    Username string
    PostCount int
}

var summaries []UserSummary
err := storm.Raw(`
    SELECT u.id, u.username, COUNT(p.id) as post_count
    FROM users u
    LEFT JOIN posts p ON p.user_id = u.id
    GROUP BY u.id, u.username
`).Scan(&summaries)
```

## Best Practices

### 1. Use Context

Always pass context for cancellation:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

users, err := storm.Users.Query().
    Where(models.Users.IsActive.Eq(true)).
    FindContext(ctx)
```

### 2. Handle Errors Properly

```go
user, err := storm.Users.Find(ctx, userID)
if err != nil {
    if errors.Is(err, orm.ErrNotFound) {
        // Handle not found
        return nil, fmt.Errorf("user not found")
    }
    // Handle other errors
    return nil, fmt.Errorf("database error: %w", err)
}
```

### 3. Use Transactions for Multiple Operations

```go
// Bad - multiple operations without transaction
storm.Users.Create(ctx, user)
storm.Posts.Create(ctx, post) // Could fail, leaving orphaned user

// Good - atomic operations
storm.WithTransaction(ctx, func(tx *models.Storm) error {
    tx.Users.Create(ctx, user)
    tx.Posts.Create(ctx, post)
    return nil
})
```

### 4. Avoid N+1 Queries

```go
// Bad - N+1 queries
users, _ := storm.Users.Query().Find()
for _, user := range users {
    posts, _ := storm.Posts.Query().
        Where(models.Posts.UserID.Eq(user.ID)).
        Find()
    user.Posts = posts
}

// Good - eager loading
users, _ := storm.Users.Query().
    With("Posts").
    Find()
```

## Next Steps

- [Query Builder](query-builder.md) - Deep dive into queries
- [Relationships](relationships.md) - All about relationships
- [Performance Guide](performance.md) - Optimization tips