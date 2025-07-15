# Todo Application Example

This example demonstrates how to use Storm ORM with a complete todo application schema.

## Features Demonstrated

- **Model Definition**: Complete todo app schema with users, todos, categories, tags, and comments
- **Relationships**: All relationship types including belongs_to, has_many, and many-to-many
- **Type-safe Queries**: Using generated column constants and query builders
- **Transactions**: Atomic operations across multiple tables
- **CRUD Operations**: Create, Read, Update, Delete examples

## Schema Overview

```
┌─────────────┐
│    Users    │
├─────────────┤
│ id (UUID)   │
│ email       │◄────────┐
│ name        │         │
│ password    │         │
└─────────────┘         │
      │                 │
      │ has_many        │ belongs_to
      ▼                 │
┌─────────────┐         │
│    Todos    │         │
├─────────────┤         │
│ id (UUID)   │         │
│ user_id ────┼─────────┘
│ category_id │◄────────┐
│ title       │         │
│ status      │         │
│ priority    │         │
└─────────────┘         │
      │                 │
      │ has_many        │ belongs_to
      ▼                 │
┌─────────────┐         │
│  Comments   │    ┌────┴──────┐
├─────────────┤    │Categories │
│ id (UUID)   │    ├───────────┤
│ todo_id     │    │ id (UUID) │
│ user_id     │    │ user_id   │
│ content     │    │ name      │
└─────────────┘    │ color     │
                   └───────────┘

Many-to-Many Relationships:
- Todos ◄─► Tags (through todo_tags)
- Users ◄─► Tags (through user_tags)
```

## Running the Example

1. Create a PostgreSQL database:
```sql
CREATE DATABASE todo_db;
```

2. Generate the schema:
```bash
storm migrate --package ./examples/todo --url "postgres://user:pass@localhost:5432/todo_db?sslmode=disable"
```

3. Run the example:
```go
package main

import "github.com/eleven-am/storm/examples/todo"

func main() {
    todo.ExampleUsage()
}
```

## Generated Files

The Storm ORM generator creates:

- `storm.go` - Main Storm instance with all repositories
- `*_repository.go` - Repository for each model with CRUD operations
- `*_query.go` - Type-safe query builders for each model
- `columns.go` - Column constants for type-safe queries
- `relationships.go` - Relationship loading helpers

## Key Patterns

### Type-safe Queries
```go
// Instead of error-prone strings:
// db.Query("SELECT * FROM todos WHERE user_id = ? AND status = ?", userID, "pending")

// Use type-safe column references:
todos, err := storm.Todos.Query().
    Where(Todos.UserID.Eq(userID)).
    Where(Todos.Status.Eq("pending")).
    Find()
```

### Transactions
```go
err := storm.WithTransaction(ctx, func(txStorm *Storm) error {
    // All operations here are atomic
    todo := &Todo{...}
    if err := txStorm.Todos.Create(ctx, todo); err != nil {
        return err // Automatic rollback
    }
    
    comment := &Comment{TodoID: todo.ID, ...}
    return txStorm.Comments.Create(ctx, comment)
})
```

### Complex Queries

Storm provides powerful query building capabilities through the storm instance methods `And()`, `Or()`, and `Not()`, as well as method chaining on conditions.

#### Basic Logical Operations

```go
// AND - Multiple Where calls are ANDed together (implicit)
todos, err := storm.Todos.Query().
    Where(Todos.UserID.Eq(userID)).
    Where(Todos.Status.Eq("pending")).
    Where(Todos.Priority.Eq("high")).
    Find()

// Explicit AND - Use storm.And()
todos, err := storm.Todos.Query().
    Where(storm.And(
        Todos.UserID.Eq(userID),
        Todos.Status.Eq("pending"),
        Todos.Priority.Eq("high"),
    )).
    Find()

// OR - Use storm.Or()
todos, err := storm.Todos.Query().
    Where(Todos.UserID.Eq(userID)).
    Where(storm.Or(
        Todos.Priority.Eq("high"),
        Todos.Priority.Eq("urgent"),
    )).
    Find()

// NOT - Use storm.Not()
todos, err := storm.Todos.Query().
    Where(Todos.UserID.Eq(userID)).
    Where(storm.Not(Todos.Status.Eq("completed"))).
    Find()
```

#### Method Chaining on Conditions

Conditions also support method chaining for more fluent queries:

```go
// Using .Or() method on conditions
todos, err := storm.Todos.Query().
    Where(Todos.UserID.Eq(userID)).
    Where(
        Todos.Priority.Eq("high").Or(
            Todos.Priority.Eq("medium").And(
                Todos.DueDate.Lt(tomorrow),
            ),
        ),
    ).
    Find()

// Using .Not() method on conditions
todos, err := storm.Todos.Query().
    Where(Todos.UserID.Eq(userID)).
    Where(Todos.Status.Eq("completed").Not()).
    Find()
```

#### Complex Nested Conditions

```go
// (A AND B) OR (C AND D)
todos, err := storm.Todos.Query().
    Where(Todos.UserID.Eq(userID)).
    Where(storm.Or(
        storm.And(
            Todos.Priority.Eq("high"),
            Todos.DueDate.Lt(time.Now()),
        ),
        storm.And(
            Todos.Priority.Eq("urgent"),
            Todos.Status.Eq("in_progress"),
        ),
    )).
    Find()

// NOT (A OR B) - Neither completed nor cancelled
todos, err := storm.Todos.Query().
    Where(Todos.UserID.Eq(userID)).
    Where(storm.Not(
        storm.Or(
            Todos.Status.Eq("completed"),
            Todos.Status.Eq("cancelled"),
        ),
    )).
    Find()

// Complex business logic
todos, err := storm.Todos.Query().
    Where(storm.And(
        Todos.UserID.Eq(userID),
        storm.Not(Todos.Status.Eq("completed")),
        storm.Or(
            // Overdue
            storm.And(
                Todos.DueDate.Lt(time.Now()),
                Todos.DueDate.IsNotNull(),
            ),
            // High priority due soon
            storm.And(
                Todos.DueDate.Between(time.Now(), tomorrow),
                storm.Or(
                    Todos.Priority.Eq("high"),
                    Todos.Priority.Eq("urgent"),
                ),
            ),
        ),
    )).
    OrderBy(Todos.DueDate.Asc()).
    Find()
```

#### Column Operations

```go
// IN and NOT IN
todos, err := storm.Todos.Query().
    Where(Todos.Priority.In("high", "urgent")).
    Where(Todos.Status.NotIn("completed", "cancelled")).
    Find()

// NULL checks
todos, err := storm.Todos.Query().
    Where(Todos.CategoryID.IsNull()).      // No category
    Where(Todos.DueDate.IsNotNull()).      // Has due date
    Find()

// String operations
todos, err := storm.Todos.Query().
    Where(Todos.Title.Like("%meeting%")).           // Contains
    Where(Todos.Description.StartsWith("Review")). // Starts with
    Where(Todos.Title.EndsWith("urgent")).         // Ends with
    Find()

// Date/Time operations
todos, err := storm.Todos.Query().
    Where(Todos.DueDate.Between(startDate, endDate)).
    Where(Todos.CreatedAt.Gte(lastWeek)).
    Where(Todos.UpdatedAt.Lt(yesterday)).
    Find()

// Numeric comparisons
todos, err := storm.Todos.Query().
    Where(Todos.Priority.Gt(1)).   // Greater than
    Where(Todos.Priority.Lte(3)).  // Less than or equal
    Find()
```

#### Dynamic Query Building

```go
// Build queries dynamically based on filters
query := storm.Todos.Query().Where(Todos.UserID.Eq(userID))

if filter.Status != nil {
    query = query.Where(Todos.Status.Eq(*filter.Status))
}

if filter.SearchText != "" {
    query = query.Where(storm.Or(
        Todos.Title.Like("%"+filter.SearchText+"%"),
        Todos.Description.Like("%"+filter.SearchText+"%"),
    ))
}

if len(filter.CategoryIDs) > 0 {
    query = query.Where(Todos.CategoryID.In(filter.CategoryIDs...))
}

todos, err := query.OrderBy(Todos.CreatedAt.Desc()).Find()
```

#### Business Logic Examples

```go
// Find overdue or high-priority upcoming todos
todos, err := storm.Todos.Query().
    Where(Todos.UserID.Eq(userID)).
    Where(storm.And(
        // Not completed
        Todos.Status.NotEq("completed"),
        // Either overdue OR high priority due soon
        storm.Or(
            // Overdue
            storm.And(
                Todos.DueDate.Lt(time.Now()),
                Todos.DueDate.IsNotNull(),
            ),
            // High/Urgent priority due within 24 hours
            storm.And(
                Todos.DueDate.Between(time.Now(), time.Now().Add(24*time.Hour)),
                Todos.Priority.In("high", "urgent"),
            ),
        ),
    )).
    OrderBy(Todos.DueDate.Asc()).
    Find()
```

See `advanced_queries.go` for more comprehensive examples including:
- Complex nested conditions
- Dynamic query building
- Search parameter builders
- Business logic queries