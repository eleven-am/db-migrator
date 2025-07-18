# Authorization with Storm ORM

Storm provides a powerful, type-safe authorization system that allows you to implement complex access control patterns while maintaining full type safety and composability. Authorization functions are applied at query creation time, ensuring that all database operations respect your security rules.

## Core Authorization Concepts

### 1. **Type-Safe Authorization Functions**

Authorization functions have the signature:
```go
func(ctx context.Context, query *ModelQuery) *ModelQuery
```

This means your authorization logic gets:
- Full access to the request context (user info, tenant ID, etc.)
- A type-safe query builder that you can modify
- Must return a modified query (enabling middleware-style chaining)

### 2. **Immutable Repository Pattern**

Each call to `Authorize()` returns a **new** repository instance - it doesn't modify the original. This means:
- You can create different authorized versions for different scenarios
- No side effects or shared state issues
- Perfect for dependency injection patterns

### 3. **Chainable Authorization**

Multiple authorization functions can be chained and are applied in order:
```go
authorizedRepo := baseRepo.
    Authorize(tenantFilter).
    Authorize(roleBasedFilter).
    Authorize(dataVisibilityFilter)
```

### 4. **Applied at Query Creation Time**

Authorization functions are called when you create a query (`repo.Query(ctx)`), not when you execute it. This means the filtering happens early in the pipeline.

## Generated Authorization Code

For each model, Storm generates type-safe authorization methods:

### Repository Authorization Method
```go
// Generated for each model (e.g., UserRepository)
func (r *UserRepository) Authorize(fn func(ctx context.Context, query *UserQuery) *UserQuery) *UserRepository {
    genericFn := func(ctx context.Context, query *storm.Query[User]) *storm.Query[User] {
        userQuery := &UserQuery{
            Query: query,
            repo:  r,
        }
        result := fn(ctx, userQuery)
        return result.Query
    }
    
    // Call the base Repository.Authorize with the converted function
    baseRepo := r.Repository.Authorize(genericFn)
    
    // Return a new UserRepository wrapping the authorized base repository
    return &UserRepository{
        Repository: baseRepo,
    }
}
```

### Type-Safe Query Wrappers
```go
// Generated UserQuery struct
type UserQuery struct {
    *storm.Query[User]
    repo *UserRepository
}

// All methods are wrapped to maintain type safety
func (q *UserQuery) Where(condition storm.Condition) *UserQuery {
    q.Query = q.Query.Where(condition)
    return q
}

func (q *UserQuery) OrderBy(expressions ...string) *UserQuery {
    q.Query = q.Query.OrderBy(expressions...)
    return q
}

// Generated relationship include methods
func (q *UserQuery) IncludePosts() *UserQuery {
    q.Query = q.Query.Include("Posts")
    return q
}
```

## Real-World Authorization Patterns

### 1. **Multi-Tenant Authorization**

```go
// Create tenant-scoped repository
func CreateTenantUserRepository(db *sqlx.DB, tenantID string) *UserRepository {
    baseRepo, _ := newUserRepository(db)
    
    return baseRepo.Authorize(func(ctx context.Context, query *UserQuery) *UserQuery {
        return query.Where(Users.TenantID.Eq(tenantID))
    })
}

// Usage
tenantUsers := CreateTenantUserRepository(db, "tenant-123")
users, err := tenantUsers.Query(ctx).
    Where(Users.IsActive.Eq(true)).
    IncludePosts().
    Find()
```

### 2. **Role-Based Authorization**

```go
func AuthorizeUsersByRole(repo *UserRepository, userRole string) *UserRepository {
    return repo.Authorize(func(ctx context.Context, query *UserQuery) *UserQuery {
        switch userRole {
        case "admin":
            // Admins see all users
            return query
        case "manager":
            // Managers see users in their teams
            user := ctx.Value("user").(AuthUser)
            return query.Where(Users.TeamID.In(user.ManagedTeams...))
        case "user":
            // Regular users see only themselves
            user := ctx.Value("user").(AuthUser)
            return query.Where(Users.ID.Eq(user.ID))
        default:
            // Unknown roles see nothing
            return query.Where(Users.ID.Eq("")) // Always false
        }
    })
}
```

### 3. **Context-Aware Authorization**

```go
func ContextAwareAuthorization(ctx context.Context, query *DocumentQuery) *DocumentQuery {
    user := ctx.Value("user").(User)
    
    // Check if user is accessing their own documents
    if userID := ctx.Value("target_user_id"); userID != nil {
        if userID.(string) == user.ID {
            return query.Where(Documents.OwnerID.Eq(user.ID))
        }
    }
    
    // Otherwise apply broader visibility rules
    return query.Where(And(
        Documents.TenantID.Eq(user.TenantID),
        Or(
            Documents.Visibility.Eq("public"),
            Documents.SharedWith.Contains(user.ID),
        ),
    ))
}
```

### 4. **Department-Based Authorization**

```go
func AuthorizePostsByDepartment(repo *PostRepository) *PostRepository {
    return repo.Authorize(func(ctx context.Context, query *PostQuery) *PostQuery {
        user := ctx.Value("user").(AuthUser)
        
        return query.Where(Or(
            // User's own posts
            Posts.AuthorID.Eq(user.ID),
            // Public posts
            Posts.Visibility.Eq("public"),
            // Department posts
            And(
                Posts.Visibility.Eq("department"),
                Posts.DepartmentID.Eq(user.DepartmentID),
            ),
            // Team posts
            And(
                Posts.Visibility.Eq("team"),
                Posts.TeamID.In(user.TeamIDs...),
            ),
        ))
    })
}
```

### 5. **Time-Based Authorization**

```go
func AuthorizeActiveContent(repo *PostRepository) *PostRepository {
    return repo.Authorize(func(ctx context.Context, query *PostQuery) *PostQuery {
        now := time.Now()
        return query.Where(And(
            Or(
                Posts.PublishAt.IsNull(),
                Posts.PublishAt.Before(now),
            ),
            Or(
                Posts.ExpireAt.IsNull(),
                Posts.ExpireAt.After(now),
            ),
            Posts.Status.Eq("published"),
        ))
    })
}
```

## Advanced Authorization Patterns

### 1. **Chained Authorization**

```go
func CreateSecureUserRepository(db *sqlx.DB) *UserRepository {
    baseRepo, _ := newUserRepository(db)
    
    return baseRepo.
        // First: Apply tenant filtering
        Authorize(func(ctx context.Context, query *UserQuery) *UserQuery {
            user := ctx.Value("user").(AuthUser)
            return query.Where(Users.TenantID.Eq(user.TenantID))
        }).
        // Then: Apply role-based filtering
        Authorize(func(ctx context.Context, query *UserQuery) *UserQuery {
            user := ctx.Value("user").(AuthUser)
            if user.Role == "admin" {
                return query
            }
            return query.Where(Users.TeamID.In(user.TeamIDs...))
        }).
        // Finally: Apply data visibility rules
        Authorize(func(ctx context.Context, query *UserQuery) *UserQuery {
            return query.Where(Or(
                Users.IsPublic.Eq(true),
                Users.VisibilityLevel.Gte(1),
            ))
        })
}
```

### 2. **Conditional Authorization**

```go
func ConditionalAuthorization(condition bool, authFunc func(ctx context.Context, query *UserQuery) *UserQuery) func(ctx context.Context, query *UserQuery) *UserQuery {
    return func(ctx context.Context, query *UserQuery) *UserQuery {
        if condition {
            return authFunc(ctx, query)
        }
        return query
    }
}

// Usage
authorizedRepo := baseRepo.Authorize(
    ConditionalAuthorization(
        user.Role != "admin",
        func(ctx context.Context, query *UserQuery) *UserQuery {
            user := ctx.Value("user").(AuthUser)
            return query.Where(Users.TenantID.Eq(user.TenantID))
        },
    ),
)
```

### 3. **Authorization with Logging**

```go
func LoggingAuthorization(logger *log.Logger) func(ctx context.Context, query *UserQuery) *UserQuery {
    return func(ctx context.Context, query *UserQuery) *UserQuery {
        user := ctx.Value("user").(AuthUser)
        
        logger.Info("Authorizing query", map[string]interface{}{
            "user_id": user.ID,
            "tenant_id": user.TenantID,
            "table": "users",
        })
        
        return query.Where(Users.TenantID.Eq(user.TenantID))
    }
}
```

### 4. **Reusable Authorization Helpers**

```go
// Define reusable authorization helpers
type AuthFilters struct {
    UserID   string
    TenantID string
    Role     string
    TeamIDs  []string
}

// Create domain-specific query functions
func GetVisibleProjects(storm *Storm, ctx context.Context, auth AuthFilters) ([]Project, error) {
    baseQuery := storm.Projects.Query(ctx).
        Where(Projects.TenantID.Eq(auth.TenantID))
    
    switch auth.Role {
    case "admin":
        // Admins see all projects in tenant
        return baseQuery.Find()
    case "pm":
        // Project managers see their projects + public ones
        return baseQuery.Where(Or(
            Projects.ManagerID.Eq(auth.UserID),
            Projects.Visibility.Eq("public"),
            Projects.TeamID.In(auth.TeamIDs...),
        )).Find()
    default:
        // Members only see projects they're assigned to
        return baseQuery.Where(Or(
            Projects.OwnerID.Eq(auth.UserID),
            Projects.MemberIDs.Contains(auth.UserID),
            And(
                Projects.Visibility.Eq("public"),
                Projects.TeamID.In(auth.TeamIDs...),
            ),
        )).Find()
    }
}

// Usage remains clean and explicit
projects, err := GetVisibleProjects(storm, ctx, authFilters)
```

## Usage Patterns

### In HTTP Handlers

```go
func GetUsers(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    user := ctx.Value("user").(AuthUser)
    
    // Create authorized repository for this request
    authorizedUsers := storm.Users.
        Authorize(func(ctx context.Context, query *UserQuery) *UserQuery {
            return query.Where(Users.TenantID.Eq(user.TenantID))
        }).
        Authorize(func(ctx context.Context, query *UserQuery) *UserQuery {
            if user.Role != "admin" {
                return query.Where(Users.TeamID.In(user.TeamIDs...))
            }
            return query
        })
    
    // Use the authorized repository
    users, err := authorizedUsers.Query(ctx).
        Where(Users.IsActive.Eq(true)).
        IncludePosts().
        OrderBy("created_at DESC").
        Find()
    
    // ... handle response
}
```

### In Service Layer

```go
type UserService struct {
    userRepo *UserRepository
}

func (s *UserService) GetUsersForRole(ctx context.Context, role string) ([]User, error) {
    authorizedRepo := s.userRepo.Authorize(func(ctx context.Context, query *UserQuery) *UserQuery {
        // Apply role-based filtering
        return applyRoleBasedFiltering(ctx, query, role)
    })
    
    return authorizedRepo.Query(ctx).
        Where(Users.IsActive.Eq(true)).
        Find()
}

func applyRoleBasedFiltering(ctx context.Context, query *UserQuery, role string) *UserQuery {
    user := ctx.Value("user").(AuthUser)
    
    switch role {
    case "admin":
        return query
    case "manager":
        return query.Where(Users.TeamID.In(user.ManagedTeams...))
    default:
        return query.Where(Users.ID.Eq(user.ID))
    }
}
```

### In Repository Factory

```go
type RepositoryFactory struct {
    db *sqlx.DB
}

func (f *RepositoryFactory) CreateAuthorizedUserRepo(ctx context.Context) *UserRepository {
    user := ctx.Value("user").(AuthUser)
    baseRepo, _ := newUserRepository(f.db)
    
    return baseRepo.
        Authorize(f.tenantFilter(user.TenantID)).
        Authorize(f.roleBasedFilter(user.Role)).
        Authorize(f.dataVisibilityFilter(user))
}

func (f *RepositoryFactory) tenantFilter(tenantID string) func(ctx context.Context, query *UserQuery) *UserQuery {
    return func(ctx context.Context, query *UserQuery) *UserQuery {
        return query.Where(Users.TenantID.Eq(tenantID))
    }
}
```

## Best Practices

### 1. **Make Authorization Explicit**
Don't hide authorization in middleware - make it clear in your code:

```go
// ✅ GOOD: Explicit authorization
authorizedUsers := storm.Users.Authorize(tenantFilter).Authorize(roleFilter)
users, err := authorizedUsers.Query(ctx).Find()

// ❌ AVOID: Hidden authorization in middleware
users, err := storm.Users.Query(ctx).Find() // Authorization happens invisibly
```

### 2. **Use Context for User Info**
Pass user data through context rather than global state:

```go
// ✅ GOOD: Context-based user info
func(ctx context.Context, query *UserQuery) *UserQuery {
    user := ctx.Value("user").(AuthUser)
    return query.Where(Users.TenantID.Eq(user.TenantID))
}

// ❌ AVOID: Global state
var currentUser AuthUser
func(ctx context.Context, query *UserQuery) *UserQuery {
    return query.Where(Users.TenantID.Eq(currentUser.TenantID))
}
```

### 3. **Compose Authorization Functions**
Build complex authorization by composing simpler functions:

```go
// ✅ GOOD: Composed authorization
repo := baseRepo.
    Authorize(tenantFilter).
    Authorize(roleFilter).
    Authorize(visibilityFilter)

// ❌ AVOID: Monolithic authorization
repo := baseRepo.Authorize(func(ctx context.Context, query *UserQuery) *UserQuery {
    // 50 lines of complex authorization logic
})
```

### 4. **Test Authorization Logic**
Authorization functions are pure functions that are easy to test:

```go
func TestTenantAuthorization(t *testing.T) {
    ctx := context.WithValue(context.Background(), "user", AuthUser{
        TenantID: "tenant-123",
    })
    
    query := &UserQuery{/* mock query */}
    result := tenantFilter(ctx, query)
    
    // Verify the query was modified correctly
    assert.Contains(t, result.String(), "tenant_id = 'tenant-123'")
}
```

### 5. **Create Reusable Helpers**
Build a library of common authorization patterns:

```go
// Authorization helpers
func TenantFilter(tenantID string) func(ctx context.Context, query *UserQuery) *UserQuery {
    return func(ctx context.Context, query *UserQuery) *UserQuery {
        return query.Where(Users.TenantID.Eq(tenantID))
    }
}

func RoleFilter(allowedRoles ...string) func(ctx context.Context, query *UserQuery) *UserQuery {
    return func(ctx context.Context, query *UserQuery) *UserQuery {
        user := ctx.Value("user").(AuthUser)
        if contains(allowedRoles, user.Role) {
            return query
        }
        return query.Where(Users.ID.Eq("")) // Always false
    }
}

// Usage
authorizedRepo := baseRepo.
    Authorize(TenantFilter("tenant-123")).
    Authorize(RoleFilter("admin", "manager"))
```

## Key Benefits

1. **Type Safety**: All authorization functions work with type-safe query builders
2. **Composability**: Multiple authorization functions can be chained
3. **Immutability**: Each `Authorize()` call returns a new repository instance
4. **Relationship Awareness**: Authorized queries work seamlessly with `Include*()` methods
5. **Context Awareness**: Authorization functions receive the full request context
6. **Performance**: Authorization happens at query building time, not execution time
7. **Testability**: Authorization functions are pure and easy to test
8. **Flexibility**: Support for complex authorization logic and conditional rules

## Migration from Other ORMs

### From GORM

```go
// GORM approach
db.Where("tenant_id = ?", tenantID).Find(&users)

// Storm approach
authorizedRepo := storm.Users.Authorize(func(ctx context.Context, query *UserQuery) *UserQuery {
    return query.Where(Users.TenantID.Eq(tenantID))
})
users, err := authorizedRepo.Query(ctx).Find()
```

### From Raw SQL

```go
// Raw SQL approach
query := "SELECT * FROM users WHERE tenant_id = $1 AND team_id IN ($2, $3)"
rows, err := db.Query(query, tenantID, team1, team2)

// Storm approach
users, err := storm.Users.
    Authorize(func(ctx context.Context, query *UserQuery) *UserQuery {
        return query.Where(And(
            Users.TenantID.Eq(tenantID),
            Users.TeamID.In(team1, team2),
        ))
    }).
    Query(ctx).
    Find()
```

Storm's authorization system provides the perfect balance of type safety, composability, and flexibility for implementing complex authorization logic while maintaining clean, testable code.