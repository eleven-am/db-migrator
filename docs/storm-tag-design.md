# Storm Tag Design: Unified Tag Architecture

## Overview

This document outlines the design for unifying Storm ORM's multiple tags (`db`, `dbdef`, `orm`) into a single, comprehensive `storm` tag. This unified approach will provide a cleaner API, better validation, and improved extensibility.

## Current State vs. Proposed

### Current Multi-Tag System
```go
type User struct {
    _ struct{} `storm:"table:users;index:idx_users_email,email;unique:uk_users_email,email"`
    
    ID        string    `db:"id" storm:"type:uuid;primary_key;default:gen_random_uuid()"`
    Email     string    `db:"email" storm:"type:varchar(255);not_null;unique"`
    Password  string    `db:"password_hash" storm:"type:varchar(255);not_null"`
    IsActive  bool      `db:"is_active" storm:"type:boolean;not_null;default:true"`
    CreatedAt time.Time `db:"created_at" storm:"type:timestamptz;not_null;default:now()"`
    
    // Relationships
    Todos      []Todo     `storm:"relation:has_many:Todo;foreign_key:user_id"`
    Categories []Category `storm:"relation:has_many:Category;foreign_key:user_id"`
    Tags       []Tag      `storm:"relation:has_many_through:Tag;join_table:user_tags;source_fk:user_id;target_fk:tag_id"`
}
```

### Proposed Unified `storm` Tag
```go
type User struct {
    _ struct{} `storm:"table:users;index:idx_users_email,email;unique:uk_users_email,email"`
    
    ID        string    `storm:"column:id;type:uuid;primary_key;default:gen_random_uuid()"`
    Email     string    `storm:"column:email;type:varchar(255);not_null;unique"`
    Password  string    `storm:"column:password_hash;type:varchar(255);not_null"`
    IsActive  bool      `storm:"column:is_active;type:boolean;not_null;default:true"`
    CreatedAt time.Time `storm:"column:created_at;type:timestamptz;not_null;default:now()"`
    
    // Relationships
    Todos      []Todo     `storm:"relation:has_many:Todo;foreign_key:user_id"`
    Categories []Category `storm:"relation:has_many:Category;foreign_key:user_id"`
    Tags       []Tag      `storm:"relation:has_many_through:Tag;join_table:user_tags;source_fk:user_id;target_fk:tag_id"`
}
```

## Syntax Design

### Core Syntax Rules
- **Separator**: Semicolon (`;`) separates different attributes
- **Assignment**: Colon (`:`) separates attribute name from value
- **Lists**: Comma (`,`) separates multiple values where applicable
- **Flags**: Attributes without values are treated as boolean flags

### Attribute Categories

#### 1. Column Definition
```go
// Basic column mapping
`storm:"column:database_column_name"`

// With PostgreSQL type
`storm:"column:id;type:uuid"`

// With constraints
`storm:"column:email;type:varchar(255);not_null;unique"`

// With default value
`storm:"column:created_at;type:timestamptz;not_null;default:now()"`

// With check constraint
`storm:"column:age;type:integer;check:age >= 0"`
```

#### 2. Primary Keys and Constraints
```go
// Primary key
`storm:"column:id;type:uuid;primary_key;default:gen_random_uuid()"`

// Unique constraint
`storm:"column:email;type:varchar(255);not_null;unique"`

// Multiple constraints
`storm:"column:status;type:varchar(20);not_null;default:pending;check:status IN ('pending', 'active', 'inactive')"`
```

#### 3. Foreign Keys
```go
// Basic foreign key
`storm:"column:user_id;type:uuid;foreign_key:users.id"`

// With referential actions
`storm:"column:user_id;type:uuid;foreign_key:users.id;on_delete:CASCADE;on_update:CASCADE"`

// With constraint name
`storm:"column:category_id;type:uuid;foreign_key:categories.id;constraint:fk_todo_category"`
```

#### 4. Relationships
```go
// One-to-many
`storm:"relation:has_many:Todo;foreign_key:user_id"`

// Many-to-one
`storm:"relation:belongs_to:User;foreign_key:user_id"`

// One-to-one
`storm:"relation:has_one:Profile;foreign_key:user_id"`

// Many-to-many through join table
`storm:"relation:has_many_through:Tag;join_table:user_tags;source_fk:user_id;target_fk:tag_id"`

// With additional options
`storm:"relation:belongs_to:User;foreign_key:user_id;target_key:id;preload:true"`
```

#### 5. Table-Level Definitions (on `_ struct{}`)
```go
// Table name
`storm:"table:custom_table_name"`

// With indexes
`storm:"table:users;index:idx_users_email,email;index:idx_users_created,created_at"`

// With unique constraints
`storm:"table:users;unique:uk_users_email,email;unique:uk_users_username,username"`

// Combined
`storm:"table:users;index:idx_users_email,email;unique:uk_users_email,email;index:idx_users_active,is_active where:(is_active = true)"`
```

#### 6. Special Cases
```go
// Exclude from database operations (equivalent to db:"-")
`storm:"ignore"`

// JSON column with custom type
`storm:"column:metadata;type:jsonb;default:'{}'"`

// Array column
`storm:"column:tags;type:text[];default:'{}'"`

// Enum column
`storm:"column:status;type:user_status;enum:pending,active,inactive;default:pending"`
```

## Attribute Reference

### Column Attributes
| Attribute | Description | Example |
|-----------|-------------|---------|
| `column` | Database column name | `column:user_id` |
| `type` | PostgreSQL data type | `type:uuid`, `type:varchar(255)` |
| `primary_key` | Mark as primary key | `primary_key` |
| `not_null` | NOT NULL constraint | `not_null` |
| `unique` | UNIQUE constraint | `unique` |
| `default` | Default value | `default:now()`, `default:'pending'` |
| `check` | CHECK constraint | `check:age >= 0` |
| `foreign_key` | Foreign key reference | `foreign_key:users.id` |
| `on_delete` | ON DELETE action | `on_delete:CASCADE` |
| `on_update` | ON UPDATE action | `on_update:SET NULL` |
| `constraint` | Custom constraint name | `constraint:fk_user_team` |
| `prev` | Previous column name (for migrations) | `prev:old_column_name` |
| `enum` | Enum values | `enum:pending,active,inactive` |
| `array_type` | Array element type | `array_type:varchar(50)` |

### Relationship Attributes
| Attribute | Description | Example |
|-----------|-------------|---------|
| `relation` | Relationship type | `relation:has_many:Todo` |
| `foreign_key` | Foreign key column | `foreign_key:user_id` |
| `source_key` | Source key column | `source_key:id` |
| `target_key` | Target key column | `target_key:user_id` |
| `join_table` | Join table name | `join_table:user_tags` |
| `source_fk` | Source foreign key in join table | `source_fk:user_id` |
| `target_fk` | Target foreign key in join table | `target_fk:tag_id` |
| `preload` | Auto-preload relationship | `preload:true` |
| `cascade` | Cascade operations | `cascade:delete` |

### Table Attributes (on `_ struct{}`)
| Attribute | Description | Example |
|-----------|-------------|---------|
| `table` | Table name | `table:custom_users` |
| `index` | Create index | `index:idx_name,column1,column2` |
| `unique` | Unique constraint | `unique:uk_name,column1,column2` |
| `check` | Table-level check constraint | `check:start_date < end_date` |
| `partition` | Partitioning strategy | `partition:range:created_at` |

### Special Attributes
| Attribute | Description | Example |
|-----------|-------------|---------|
| `ignore` | Exclude from database operations | `ignore` |
| `json` | JSON serialization name | `json:user_name` |
| `validate` | Custom validation rules | `validate:email,required` |
| `immutable` | Immutable field (create-only) | `immutable` |
| `computed` | Computed/derived field | `computed:full_name` |

## Complete Examples

### Basic Model
```go
type User struct {
    _ struct{} `storm:"table:users;index:idx_users_email,email;unique:uk_users_email,email"`
    
    ID        string    `storm:"column:id;type:uuid;primary_key;default:gen_random_uuid()"`
    Email     string    `storm:"column:email;type:varchar(255);not_null;unique"`
    Name      string    `storm:"column:name;type:varchar(100);not_null"`
    IsActive  bool      `storm:"column:is_active;type:boolean;not_null;default:true"`
    CreatedAt time.Time `storm:"column:created_at;type:timestamptz;not_null;default:now()"`
    UpdatedAt time.Time `storm:"column:updated_at;type:timestamptz;not_null;default:now()"`
}
```

### Model with Relationships
```go
type Todo struct {
    _ struct{} `storm:"table:todos;index:idx_todos_user,user_id;index:idx_todos_due,due_date where:(due_date IS NOT NULL)"`
    
    ID          string     `storm:"column:id;type:uuid;primary_key;default:gen_random_uuid()"`
    UserID      string     `storm:"column:user_id;type:uuid;not_null;foreign_key:users.id;on_delete:CASCADE"`
    CategoryID  *string    `storm:"column:category_id;type:uuid;foreign_key:categories.id;on_delete:SET NULL"`
    Title       string     `storm:"column:title;type:varchar(255);not_null"`
    Description *string    `storm:"column:description;type:text"`
    Status      string     `storm:"column:status;type:todo_status;enum:pending,in_progress,completed;default:pending"`
    Priority    int        `storm:"column:priority;type:integer;default:0;check:priority >= 0 AND priority <= 10"`
    DueDate     *time.Time `storm:"column:due_date;type:timestamptz"`
    CreatedAt   time.Time  `storm:"column:created_at;type:timestamptz;not_null;default:now()"`
    UpdatedAt   time.Time  `storm:"column:updated_at;type:timestamptz;not_null;default:now()"`
    
    // Relationships
    User      *User      `storm:"relation:belongs_to:User;foreign_key:user_id"`
    Category  *Category  `storm:"relation:belongs_to:Category;foreign_key:category_id"`
    Tags      []Tag      `storm:"relation:has_many_through:Tag;join_table:todo_tags;source_fk:todo_id;target_fk:tag_id"`
    Comments  []Comment  `storm:"relation:has_many:Comment;foreign_key:todo_id;cascade:delete"`
}
```

### Model with Advanced Features
```go
type Order struct {
    _ struct{} `storm:"table:orders;index:idx_orders_user_date,user_id,created_at;unique:uk_orders_number,order_number;check:total_amount >= 0"`
    
    ID           string          `storm:"column:id;type:uuid;primary_key;default:gen_random_uuid()"`
    UserID       string          `storm:"column:user_id;type:uuid;not_null;foreign_key:users.id"`
    OrderNumber  string          `storm:"column:order_number;type:varchar(50);not_null;unique"`
    Status       string          `storm:"column:status;type:order_status;enum:pending,processing,shipped,delivered,cancelled;default:pending"`
    TotalAmount  decimal.Decimal `storm:"column:total_amount;type:decimal(10,2);not_null;check:total_amount >= 0"`
    Currency     string          `storm:"column:currency;type:varchar(3);not_null;default:USD"`
    Metadata     map[string]any  `storm:"column:metadata;type:jsonb;default:'{}'"`
    Tags         []string        `storm:"column:tags;type:text[];default:'{}'"`
    ShippedAt    *time.Time      `storm:"column:shipped_at;type:timestamptz"`
    DeliveredAt  *time.Time      `storm:"column:delivered_at;type:timestamptz"`
    CreatedAt    time.Time       `storm:"column:created_at;type:timestamptz;not_null;default:now()"`
    UpdatedAt    time.Time       `storm:"column:updated_at;type:timestamptz;not_null;default:now()"`
    
    // Relationships
    User         *User        `storm:"relation:belongs_to:User;foreign_key:user_id"`
    OrderItems   []OrderItem  `storm:"relation:has_many:OrderItem;foreign_key:order_id;cascade:delete"`
    Payments     []Payment    `storm:"relation:has_many:Payment;foreign_key:order_id"`
    
    // Computed fields
    ItemCount    int          `storm:"ignore" json:"item_count"`
    IsShipped    bool         `storm:"computed:shipped_at IS NOT NULL" json:"is_shipped"`
}
```

## Migration Strategy

### Phase 1: Dual Support
- Implement `storm` tag parser alongside existing parsers
- Support both old and new syntax simultaneously
- Add deprecation warnings for old tags

### Phase 2: Migration Tools
- Provide automated migration tool to convert existing tags
- Generate migration scripts for codebases
- Update documentation and examples

### Phase 3: Gradual Deprecation
- New features only available in `storm` tag
- Increase deprecation warnings
- Provide clear migration timeline

### Phase 4: Full Migration
- Remove support for old tags
- Simplify codebase by removing legacy parsers
- Update all examples and documentation

## Implementation Architecture

### Parser Structure
```go
type StormTag struct {
    // Column definition
    Column      string
    Type        string
    Constraints []string
    Default     string
    Check       string
    
    // Foreign key
    ForeignKey  *ForeignKeyDef
    OnDelete    string
    OnUpdate    string
    
    // Relationship
    Relation    *RelationshipDef
    
    // Special flags
    Ignore      bool
    Computed    string
    Immutable   bool
    
    // Validation
    Validate    []string
}

type ForeignKeyDef struct {
    Table      string
    Column     string
    Constraint string
}

type RelationshipDef struct {
    Type       string  // has_many, belongs_to, etc.
    Target     string  // Target model name
    ForeignKey string
    SourceKey  string
    TargetKey  string
    JoinTable  string
    SourceFK   string
    TargetFK   string
    Preload    bool
    Cascade    []string
}
```

### Processing Pipeline
1. **Lexical Analysis**: Parse semicolon-separated attributes
2. **Syntactic Analysis**: Validate attribute syntax and structure
3. **Semantic Analysis**: Validate relationships and constraints
4. **Code Generation**: Generate appropriate ORM code
5. **Schema Generation**: Generate SQL DDL statements

## Benefits

### For Developers
- **Single Tag**: Only one tag to learn and use
- **Consistency**: Uniform syntax across all features
- **Discoverability**: All options available in one place
- **Validation**: Better error messages and validation

### For Maintainers
- **Simplicity**: Single parser to maintain
- **Extensibility**: Easy to add new features
- **Testing**: Unified testing strategy
- **Documentation**: Single source of truth

### For Tools
- **IDE Support**: Better autocomplete and validation
- **Linting**: Comprehensive syntax checking
- **Migration**: Automated conversion tools
- **Generation**: Simplified code generation

## Conclusion

The unified `storm` tag represents a significant improvement to the Storm ORM developer experience. By consolidating three separate tags into a single, comprehensive syntax, we create a more maintainable, extensible, and user-friendly API while preserving all existing functionality.

The migration strategy ensures backward compatibility during the transition period, while the new syntax provides a foundation for future enhancements and features.