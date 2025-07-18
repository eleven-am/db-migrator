# Query Builder Guide

Storm's query builder provides a type-safe, fluent API for constructing database queries. This guide covers all query building features in detail.

## Table of Contents

- [Basic Queries](#basic-queries)
- [Where Conditions](#where-conditions)
- [Logical Operators](#logical-operators)
- [Column Operations](#column-operations)
- [Ordering](#ordering)
- [Limiting Results](#limiting-results)
- [Joins](#joins)
- [Aggregations](#aggregations)
- [Subqueries](#subqueries)
- [Raw Queries](#raw-queries)
- [Query Debugging](#query-debugging)

## Basic Queries

### Creating a Query

Every repository has a `Query()` method that returns a new query builder:

```go
query := storm.Users.Query()
```

### Executing Queries

```go
// Find all matching records
users, err := query.Find()

// Find first matching record
user, err := query.First()

// Find last matching record
user, err := query.Last()

// Check if any records exist
exists, err := query.Exists()

// Count matching records
count, err := query.Count()
```

## Where Conditions

### Simple Conditions

```go
// Equality
query.Where(models.Users.Email.Eq("john@example.com"))

// Inequality
query.Where(models.Users.Status.NotEq("deleted"))

// Comparison
query.Where(models.Users.Age.Gt(18))     // Greater than
query.Where(models.Users.Age.Gte(18))    // Greater than or equal
query.Where(models.Users.Age.Lt(65))     // Less than
query.Where(models.Users.Age.Lte(65))    // Less than or equal
```

### Multiple Conditions

Multiple `Where` calls are combined with AND:

```go
users, err := storm.Users.Query().
    Where(models.Users.IsActive.Eq(true)).
    Where(models.Users.Role.Eq("admin")).
    Where(models.Users.CreatedAt.After(lastMonth)).
    Find()
// Generates: WHERE is_active = true AND role = 'admin' AND created_at > '...'
```

## Logical Operators

### AND Operator

```go
// Implicit AND (multiple Where calls)
query.Where(condition1).Where(condition2)

// Explicit AND
query.Where(storm.And(condition1, condition2, condition3))

// Nested AND
query.Where(storm.And(
    models.Users.IsActive.Eq(true),
    storm.Or(
        models.Users.Role.Eq("admin"),
        models.Users.Role.Eq("moderator"),
    ),
))
```

### OR Operator

```go
// Simple OR
query.Where(storm.Or(
    models.Users.Role.Eq("admin"),
    models.Users.Role.Eq("moderator"),
))

// Complex OR with different fields
query.Where(storm.Or(
    models.Users.Email.EndsWith("@company.com"),
    models.Users.Department.Eq("IT"),
    storm.And(
        models.Users.Role.Eq("contractor"),
        models.Users.AccessLevel.Gte(3),
    ),
))
```

### NOT Operator

```go
// Simple NOT
query.Where(storm.Not(models.Users.Status.Eq("inactive")))

// NOT with complex conditions
query.Where(storm.Not(
    storm.Or(
        models.Users.Status.Eq("deleted"),
        models.Users.Status.Eq("banned"),
    ),
))

// Using method chaining
query.Where(models.Users.Status.Eq("active").Not())
```

### Method Chaining

Conditions support method chaining for more readable queries:

```go
query.Where(
    models.Users.IsActive.Eq(true).
        And(models.Users.EmailVerified.Eq(true)).
        And(
            models.Users.Role.Eq("admin").
                Or(models.Users.Role.Eq("superadmin")),
        ),
)
```

## Column Operations

### IN and NOT IN

```go
// IN
query.Where(models.Users.Status.In("active", "pending", "verified"))

// NOT IN
query.Where(models.Users.Status.NotIn("deleted", "banned"))

// With slice
statuses := []string{"active", "pending"}
query.Where(models.Users.Status.In(statuses...))
```

### NULL Checks

```go
// IS NULL
query.Where(models.Users.DeletedAt.IsNull())

// IS NOT NULL
query.Where(models.Users.EmailVerifiedAt.IsNotNull())
```

### String Operations

```go
// LIKE (case sensitive)
query.Where(models.Users.Email.Like("%@example.com"))

// ILIKE (case insensitive - PostgreSQL)
query.Where(models.Users.Name.ILike("john%"))

// Convenience methods
query.Where(models.Users.Name.StartsWith("John"))    // LIKE 'John%'
query.Where(models.Users.Email.EndsWith("@gmail.com")) // LIKE '%@gmail.com'
query.Where(models.Users.Bio.Contains("developer"))   // LIKE '%developer%'

// Regular expressions (PostgreSQL)
query.Where(models.Users.Email.Regexp("^[a-z]+@example\\.com$"))
```

### Numeric Operations

```go
// Between
query.Where(models.Products.Price.Between(10.00, 100.00))

// Mathematical operations in queries
query.Where(models.Orders.Quantity.Gt(models.Orders.MinQuantity))
```

### Date/Time Operations

```go
// Basic comparisons
query.Where(models.Users.CreatedAt.After(someDate))
query.Where(models.Users.CreatedAt.Before(someDate))
query.Where(models.Users.CreatedAt.Between(startDate, endDate))

// Convenience methods
query.Where(models.Users.CreatedAt.Today())
query.Where(models.Users.CreatedAt.ThisWeek())
query.Where(models.Users.CreatedAt.ThisMonth())
query.Where(models.Users.LastLoginAt.LastNDays(7))

// Time-based conditions
query.Where(models.Users.CreatedAt.Since(time.Now().Add(-30*24*time.Hour)))
query.Where(models.Events.StartTime.Until(time.Now().Add(7*24*time.Hour)))
```

### JSON Operations (PostgreSQL)

```go
// JSONB containment
query.Where(models.Users.Metadata.JSONBContains(`{"role": "admin"}`))

// JSONB key existence
query.Where(models.Users.Settings.JSONBHasKey("notifications"))

// JSONB path queries
query.Where(models.Users.Profile.JSONBPath("address.city").Eq("New York"))
```

### Array Operations (PostgreSQL)

```go
// Array contains
query.Where(models.Posts.Tags.Contains("golang"))

// Array overlap
query.Where(models.Posts.Categories.Overlaps([]string{"tech", "programming"}))

// Array length
query.Where(models.Posts.Tags.Length().Gt(3))
```

## Ordering

### Single Column Ordering

```go
// Ascending order
query.OrderBy(models.Users.CreatedAt.Asc())

// Descending order
query.OrderBy(models.Users.CreatedAt.Desc())
```

### Multiple Column Ordering

```go
query.
    OrderBy(models.Users.Role.Asc()).
    OrderBy(models.Users.CreatedAt.Desc())
// ORDER BY role ASC, created_at DESC
```

### Dynamic Ordering

```go
func getUsers(sortBy string, sortDesc bool) ([]User, error) {
    query := storm.Users.Query()
    
    switch sortBy {
    case "email":
        if sortDesc {
            query = query.OrderBy(models.Users.Email.Desc())
        } else {
            query = query.OrderBy(models.Users.Email.Asc())
        }
    case "created":
        if sortDesc {
            query = query.OrderBy(models.Users.CreatedAt.Desc())
        } else {
            query = query.OrderBy(models.Users.CreatedAt.Asc())
        }
    }
    
    return query.Find()
}
```

## Limiting Results

### Limit and Offset

```go
// Get first 10 records
query.Limit(10)

// Get records 21-30 (page 3 with 10 per page)
query.Limit(10).Offset(20)

// Pagination helper
func paginate(query orm.Query, page, pageSize int) orm.Query {
    return query.
        Limit(pageSize).
        Offset((page - 1) * pageSize)
}
```

### First and Last

```go
// Get first record (adds LIMIT 1)
user, err := query.First()

// Get last record (reverses order and adds LIMIT 1)
user, err := query.OrderBy(models.Users.ID.Asc()).Last()
```

## Joins

### Inner Joins

```go
// Join related table
users, err := storm.Users.Query().
    Join("Posts").
    Where(models.Posts.Published.Eq(true)).
    Find()

// Multiple joins
users, err := storm.Users.Query().
    Join("Posts").
    Join("Posts.Comments").
    Where(models.Comments.Approved.Eq(true)).
    Find()
```

### Left Joins

```go
// Left join to include users without posts
users, err := storm.Users.Query().
    LeftJoin("Posts").
    Find()
```

### Join Conditions

```go
// Additional join conditions
users, err := storm.Users.Query().
    JoinOn("Posts", storm.And(
        models.Posts.UserID.Eq(models.Users.ID),
        models.Posts.Published.Eq(true),
    )).
    Find()
```

## Aggregations

### Count Operations

```go
// Simple count
count, err := storm.Users.Query().
    Where(models.Users.IsActive.Eq(true)).
    Count()

// Distinct count
uniqueEmails, err := storm.Users.Query().
    CountDistinct(models.Users.Email)

// Count with grouping
type RoleCount struct {
    Role  string
    Count int
}

var results []RoleCount
err := storm.Users.Query().
    Select("role", "COUNT(*) as count").
    GroupBy("role").
    Scan(&results)
```

### Mathematical Aggregations

```go
// Sum
total, err := storm.Orders.Query().
    Where(models.Orders.Status.Eq("completed")).
    Sum(models.Orders.TotalAmount)

// Average
avg, err := storm.Products.Query().
    Where(models.Products.InStock.Eq(true)).
    Avg(models.Products.Price)

// Min/Max
minPrice, err := storm.Products.Query().Min(models.Products.Price)
maxPrice, err := storm.Products.Query().Max(models.Products.Price)
```

### Group By and Having

```go
// Group by with having
type CategoryStats struct {
    CategoryID string
    TotalSales float64
    OrderCount int
}

var stats []CategoryStats
err := storm.Raw(`
    SELECT 
        category_id,
        SUM(total_amount) as total_sales,
        COUNT(*) as order_count
    FROM orders
    WHERE status = 'completed'
    GROUP BY category_id
    HAVING SUM(total_amount) > 1000
`).Scan(&stats)
```

## Subqueries

### IN Subqueries

```go
// Users who have made purchases
activeUserIDs := storm.Orders.Query().
    Where(models.Orders.CreatedAt.After(lastMonth)).
    Select("user_id").
    Distinct()

users, err := storm.Users.Query().
    Where(models.Users.ID.InSubquery(activeUserIDs)).
    Find()
```

### EXISTS Subqueries

```go
// Users with at least one published post
users, err := storm.Users.Query().
    WhereExists(
        storm.Posts.Query().
            Where(models.Posts.UserID.EqColumn(models.Users.ID)).
            Where(models.Posts.Published.Eq(true)),
    ).
    Find()
```

## Raw Queries

### Raw SQL Queries

```go
// Simple raw query
var users []models.User
err := storm.Raw("SELECT * FROM users WHERE created_at > $1", lastWeek).
    Scan(&users)

// With named parameters
err := storm.RawNamed(`
    SELECT * FROM users 
    WHERE role = :role 
    AND created_at > :date
`, map[string]interface{}{
    "role": "admin",
    "date": lastWeek,
}).Scan(&users)
```

### Partial Raw Conditions

```go
// Mix raw SQL with query builder
users, err := storm.Users.Query().
    Where(models.Users.IsActive.Eq(true)).
    WhereRaw("EXTRACT(YEAR FROM created_at) = ?", 2024).
    Find()
```

## Query Debugging

### SQL Generation

```go
// Get SQL without executing
query := storm.Users.Query().
    Where(models.Users.IsActive.Eq(true)).
    OrderBy(models.Users.CreatedAt.Desc()).
    Limit(10)

sql, args := query.ToSQL()
fmt.Printf("SQL: %s\n", sql)
fmt.Printf("Args: %v\n", args)
```

### Query Logging

```go
// Enable global query logging
storm.EnableQueryLogging(true)

// Log specific query
query.Debug().Find()
```

### Query Explanation

```go
// Get query execution plan
plan, err := query.Explain()
fmt.Println("Execution plan:", plan)

// Analyze query performance
analysis, err := query.Analyze()
fmt.Println("Query analysis:", analysis)
```

## Advanced Examples

### Dynamic Query Building

```go
type UserFilter struct {
    Status      *string
    Roles       []string
    AgeMin      *int
    AgeMax      *int
    SearchText  string
    OnlyActive  bool
}

func buildUserQuery(filter UserFilter) *orm.Query {
    query := storm.Users.Query()
    
    if filter.Status != nil {
        query = query.Where(models.Users.Status.Eq(*filter.Status))
    }
    
    if len(filter.Roles) > 0 {
        query = query.Where(models.Users.Role.In(filter.Roles...))
    }
    
    if filter.AgeMin != nil && filter.AgeMax != nil {
        query = query.Where(models.Users.Age.Between(*filter.AgeMin, *filter.AgeMax))
    }
    
    if filter.SearchText != "" {
        pattern := "%" + filter.SearchText + "%"
        query = query.Where(storm.Or(
            models.Users.Name.ILike(pattern),
            models.Users.Email.ILike(pattern),
            models.Users.Bio.ILike(pattern),
        ))
    }
    
    if filter.OnlyActive {
        query = query.Where(models.Users.IsActive.Eq(true))
    }
    
    return query
}
```

### Reusable Query Scopes

```go
// Define reusable query modifiers
func ActiveUsers(q *orm.Query) *orm.Query {
    return q.Where(models.Users.IsActive.Eq(true)).
        Where(models.Users.DeletedAt.IsNull())
}

func VerifiedUsers(q *orm.Query) *orm.Query {
    return q.Where(models.Users.EmailVerifiedAt.IsNotNull())
}

func RecentUsers(q *orm.Query, days int) *orm.Query {
    since := time.Now().AddDate(0, 0, -days)
    return q.Where(models.Users.CreatedAt.After(since))
}

// Use scopes
users, err := RecentUsers(
    VerifiedUsers(
        ActiveUsers(storm.Users.Query()),
    ), 
    30,
).Find()
```

### Complex Business Queries

```go
// Find high-value customers
type CustomerStats struct {
    UserID         string
    TotalPurchases float64
    OrderCount     int
    LastOrderDate  time.Time
}

var highValueCustomers []CustomerStats
err := storm.Raw(`
    WITH customer_orders AS (
        SELECT 
            user_id,
            SUM(total_amount) as total_purchases,
            COUNT(*) as order_count,
            MAX(created_at) as last_order_date
        FROM orders
        WHERE status = 'completed'
        AND created_at > NOW() - INTERVAL '1 year'
        GROUP BY user_id
    )
    SELECT 
        u.id as user_id,
        co.total_purchases,
        co.order_count,
        co.last_order_date
    FROM users u
    JOIN customer_orders co ON co.user_id = u.id
    WHERE co.total_purchases > 1000
    OR co.order_count > 10
    ORDER BY co.total_purchases DESC
`).Scan(&highValueCustomers)
```

## Performance Tips

1. **Use indexes**: Ensure columns in WHERE clauses are indexed
2. **Limit results**: Always use LIMIT when you don't need all records
3. **Select only needed columns**: Use Select() to avoid fetching unnecessary data
4. **Avoid N+1 queries**: Use joins or eager loading for relationships
5. **Use raw queries for complex operations**: The query builder is great, but sometimes raw SQL is clearer and more efficient

## Next Steps

- [Schema Definition](schema-definition.md) - Learn about relationships and schema
- [ORM Guide](orm-guide.md) - Full ORM features
- [Configuration Guide](configuration.md) - Configuration options