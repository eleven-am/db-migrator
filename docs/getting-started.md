# Getting Started with Storm

This guide will walk you through setting up Storm in a new Go project from scratch.

## Prerequisites

- Go 1.21 or higher
- PostgreSQL 12 or higher
- Basic knowledge of Go and SQL

## Installation

Install the Storm CLI tool:

```bash
go install github.com/eleven-am/storm/cmd/storm@latest
```

Verify the installation:

```bash
storm version
```

## Step 1: Initialize Your Project

Create a new Go project:

```bash
mkdir myapp && cd myapp
go mod init myapp
```

Initialize Storm configuration:

```bash
storm init
```

This creates a `storm.yaml` file. Update the database connection:

```yaml
database:
  url: postgres://postgres:password@localhost:5432/myapp?sslmode=disable
```

## Step 2: Create Your Database

```bash
createdb myapp
```

Or using psql:

```sql
CREATE DATABASE myapp;
```

## Step 3: Define Your Models

Create a `models` directory and define your first model:

```go
// models/user.go
package models

import (
    "time"
)

type User struct {
    // Table configuration
    _ struct{} `dbdef:"table:users;index:idx_users_email,email"`
    
    // Fields
    ID        string    `db:"id" dbdef:"type:uuid;primary_key;default:gen_random_uuid()"`
    Email     string    `db:"email" dbdef:"type:varchar(255);not_null;unique"`
    Username  string    `db:"username" dbdef:"type:varchar(50);not_null;unique"`
    Password  string    `db:"password_hash" dbdef:"type:varchar(255);not_null"`
    IsActive  bool      `db:"is_active" dbdef:"type:boolean;not_null;default:true"`
    CreatedAt time.Time `db:"created_at" dbdef:"type:timestamptz;not_null;default:now()"`
    UpdatedAt time.Time `db:"updated_at" dbdef:"type:timestamptz;not_null;default:now()"`
}
```

## Step 4: Generate Your First Migration

Run the migration command:

```bash
storm migrate
```

This will:
1. Analyze your Go structs
2. Compare with the current database schema (empty in this case)
3. Generate a migration file in `./migrations/`

Review the generated migration:

```bash
cat migrations/001_*.sql
```

## Step 5: Apply the Migration

Apply the migration to your database:

```bash
storm migrate --push
```

Or manually:

```bash
psql -d myapp -f migrations/001_*.sql
```

## Step 6: Generate ORM Code

Generate the ORM code for your models:

```bash
storm orm
```

This creates several files in your models directory:
- `storm.go` - Main Storm instance
- `user_repository.go` - User CRUD operations
- `user_query.go` - Query builder for users
- `columns.go` - Type-safe column references

## Step 7: Use Storm in Your Application

Create a simple application:

```go
// main.go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/jmoiron/sqlx"
    _ "github.com/lib/pq"
    
    "myapp/models"
)

func main() {
    // Connect to database
    db, err := sqlx.Connect("postgres", "postgres://postgres:password@localhost:5432/myapp?sslmode=disable")
    if err != nil {
        log.Fatal("Failed to connect:", err)
    }
    defer db.Close()
    
    // Create Storm instance
    storm := models.NewStorm(db)
    ctx := context.Background()
    
    // Create a user
    user := &models.User{
        Email:    "john@example.com",
        Username: "johndoe",
        Password: "hashed_password_here",
    }
    
    if err := storm.Users.Create(ctx, user); err != nil {
        log.Fatal("Failed to create user:", err)
    }
    
    fmt.Printf("Created user with ID: %s\n", user.ID)
    
    // Query users
    users, err := storm.Users.Query().
        Where(models.Users.IsActive.Eq(true)).
        OrderBy(models.Users.CreatedAt.Desc()).
        Find()
    
    if err != nil {
        log.Fatal("Failed to query users:", err)
    }
    
    fmt.Printf("Found %d active users\n", len(users))
}
```

Run your application:

```bash
go run main.go
```

## Step 8: Add More Models

Let's add a Post model:

```go
// models/post.go
package models

import (
    "time"
)

type Post struct {
    _ struct{} `dbdef:"table:posts;index:idx_posts_user,user_id;index:idx_posts_published,published,created_at"`
    
    ID          string    `db:"id" dbdef:"type:uuid;primary_key;default:gen_random_uuid()"`
    UserID      string    `db:"user_id" dbdef:"type:uuid;not_null;foreign_key:users.id;on_delete:CASCADE"`
    Title       string    `db:"title" dbdef:"type:varchar(255);not_null"`
    Slug        string    `db:"slug" dbdef:"type:varchar(255);not_null;unique"`
    Content     string    `db:"content" dbdef:"type:text"`
    Published   bool      `db:"published" dbdef:"type:boolean;not_null;default:false"`
    PublishedAt *time.Time `db:"published_at" dbdef:"type:timestamptz"`
    CreatedAt   time.Time `db:"created_at" dbdef:"type:timestamptz;not_null;default:now()"`
    UpdatedAt   time.Time `db:"updated_at" dbdef:"type:timestamptz;not_null;default:now()"`
    
    // Relationships
    User *User `db:"-" orm:"belongs_to:User,foreign_key:user_id"`
}
```

Update the User model to include the relationship:

```go
// Add to User struct
Posts []Post `db:"-" orm:"has_many:Post,foreign_key:user_id"`
```

Generate a new migration:

```bash
storm migrate --name add_posts_table
storm migrate --push
```

Regenerate ORM code:

```bash
storm orm
```

## Step 9: Working with Relationships

```go
// Create a post
post := &models.Post{
    UserID:    user.ID,
    Title:     "My First Post",
    Slug:      "my-first-post",
    Content:   "Hello, Storm!",
    Published: true,
}

err = storm.Posts.Create(ctx, post)

// Query with relationships
posts, err := storm.Posts.Query().
    Where(models.Posts.Published.Eq(true)).
    With("User").  // Load the User relationship
    Find()

for _, post := range posts {
    fmt.Printf("Post: %s by %s\n", post.Title, post.User.Username)
}

// Query through relationships
userWithPosts, err := storm.Users.Query().
    Where(models.Users.ID.Eq(user.ID)).
    With("Posts").
    First()

fmt.Printf("User %s has %d posts\n", userWithPosts.Username, len(userWithPosts.Posts))
```

## Step 10: Transactions

```go
err = storm.WithTransaction(ctx, func(tx *models.Storm) error {
    // All operations in this function are atomic
    
    // Create a new post
    post := &models.Post{
        UserID:  user.ID,
        Title:   "Transactional Post",
        Slug:    "transactional-post",
        Content: "This post is created in a transaction",
    }
    
    if err := tx.Posts.Create(ctx, post); err != nil {
        return err // Will rollback
    }
    
    // Update user
    user.UpdatedAt = time.Now()
    if err := tx.Users.Update(ctx, user); err != nil {
        return err // Will rollback
    }
    
    return nil // Will commit
})
```

## Next Steps

Now that you have a basic understanding of Storm, explore:

- [Schema Definition Guide](schema-definition.md) - All dbdef tag options
- [ORM Guide](orm-guide.md) - Advanced ORM features
- [Query Builder](query-builder.md) - Complex queries
- [Relationships](relationships.md) - All relationship types
- [Migrations Guide](migrations.md) - Migration strategies

## Common Patterns

### Timestamps

Add timestamps to all your models:

```go
type BaseModel struct {
    CreatedAt time.Time `db:"created_at" dbdef:"type:timestamptz;not_null;default:now()"`
    UpdatedAt time.Time `db:"updated_at" dbdef:"type:timestamptz;not_null;default:now()"`
}

type User struct {
    BaseModel
    // ... other fields
}
```

### Soft Deletes

Implement soft deletes:

```go
type SoftDelete struct {
    DeletedAt *time.Time `db:"deleted_at" dbdef:"type:timestamptz;index"`
}

// Query only non-deleted records
users, err := storm.Users.Query().
    Where(models.Users.DeletedAt.IsNull()).
    Find()
```

### UUIDs vs Auto-increment IDs

Storm supports both patterns:

```go
// UUID (recommended)
ID string `db:"id" dbdef:"type:uuid;primary_key;default:gen_random_uuid()"`

// Auto-increment
ID int64 `db:"id" dbdef:"type:bigserial;primary_key"`
```

## Troubleshooting

### Migration Conflicts

If Storm detects conflicts:

```bash
storm migrate --allow-destructive
```

### ORM Generation Issues

Clear generated files and regenerate:

```bash
rm models/*_repository.go models/*_query.go models/columns.go models/storm.go
storm orm
```

### Connection Issues

Test your connection:

```bash
storm verify
```

## Summary

You've learned how to:
- ✅ Set up Storm in a new project
- ✅ Define models with schema tags
- ✅ Generate and apply migrations
- ✅ Generate type-safe ORM code
- ✅ Perform CRUD operations
- ✅ Build complex queries
- ✅ Work with relationships
- ✅ Use transactions

Continue to the next guide to learn about schema definition in detail.