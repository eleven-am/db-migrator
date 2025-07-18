# Schema Definition Guide

Storm uses `storm` tags to define your database schema directly in your Go structs. This guide covers all available options.

## Table of Contents

- [Basic Structure](#basic-structure)
- [Table Configuration](#table-configuration)
- [Field Types](#field-types)
- [Constraints](#constraints)
- [Indexes](#indexes)
- [Foreign Keys](#foreign-keys)
- [Defaults](#defaults)
- [Check Constraints](#check-constraints)
- [Complete Reference](#complete-reference)
- [Relationships](#relationships)

## Basic Structure

Every model struct currently uses two types of tags:
- `db` - Maps struct fields to database columns (for column naming)
- `storm` - Defines the database schema, constraints, and relationships

```go
type Model struct {
    // Table-level configuration
    _ struct{} `storm:"table:table_name;option1;option2"`
    
    // Field definitions
    FieldName Type `db:"column_name" storm:"type:db_type;constraint1;constraint2"`
}
```

## Table Configuration

Table configuration is defined on an anonymous struct field:

```go
type User struct {
    _ struct{} `storm:"table:users;comment:User accounts table"`
    // fields...
}
```

### Table Options

| Option | Description | Example |
|--------|-------------|---------|
| `table` | Table name (required) | `table:users` |
| `comment` | Table comment | `comment:Stores user accounts` |
| `index` | Create an index | `index:idx_email,email` |
| `unique` | Create unique constraint | `unique:uk_email,email` |
| `check` | Table-level check constraint | `check:ck_positive_age,age > 0` |

### Multiple Indexes Example

```go
_ struct{} `storm:"table:products;index:idx_category,category_id;index:idx_sku,sku;unique:uk_sku,sku"`
```

## Field Types

Storm supports all PostgreSQL data types:

### Numeric Types

```go
// Integers
SmallInt  int16   `db:"small_int" storm:"type:smallint"`
Integer   int32   `db:"integer" storm:"type:integer"`
BigInt    int64   `db:"big_int" storm:"type:bigint"`

// Serial (auto-increment)
ID        int64   `db:"id" storm:"type:bigserial;primary_key"`

// Floating point
Real      float32 `db:"real" storm:"type:real"`
Double    float64 `db:"double" storm:"type:double precision"`

// Precise numeric
Decimal   string  `db:"price" storm:"type:decimal(10,2)"`
Numeric   string  `db:"amount" storm:"type:numeric(15,4)"`
```

### Text Types

```go
// Variable length
Name      string  `db:"name" storm:"type:varchar(100)"`
Email     string  `db:"email" storm:"type:varchar(255)"`

// Fixed length
Code      string  `db:"code" storm:"type:char(10)"`

// Unlimited length
Content   string  `db:"content" storm:"type:text"`

// PostgreSQL specific
Tsvector  string  `db:"search_vector" storm:"type:tsvector"`
```

### Date/Time Types

```go
// Time types
Date      time.Time `db:"date" storm:"type:date"`
Time      time.Time `db:"time" storm:"type:time"`
Timestamp time.Time `db:"timestamp" storm:"type:timestamp"`
Timestamptz time.Time `db:"created_at" storm:"type:timestamptz"`

// Intervals
Duration  string   `db:"duration" storm:"type:interval"`
```

### Boolean Type

```go
IsActive  bool     `db:"is_active" storm:"type:boolean;not_null;default:true"`
```

### UUID Type

```go
ID        string   `db:"id" storm:"type:uuid;primary_key;default:gen_random_uuid()"`
```

### JSON Types

```go
// JSON (text-based)
Metadata  string   `db:"metadata" storm:"type:json"`

// JSONB (binary, preferred)
Settings  string   `db:"settings" storm:"type:jsonb"`

// With Go types
Data      MyStruct `db:"data" storm:"type:jsonb"`
```

### Array Types

```go
Tags      []string `db:"tags" storm:"type:text[]"`
Numbers   []int    `db:"numbers" storm:"type:integer[]"`
```

### Special Types

```go
// Network addresses
IPAddress string   `db:"ip_address" storm:"type:inet"`
MacAddr   string   `db:"mac_address" storm:"type:macaddr"`

// Geometric types
Point     string   `db:"location" storm:"type:point"`
Box       string   `db:"area" storm:"type:box"`

// Money
Price     string   `db:"price" storm:"type:money"`
```

## Constraints

### Primary Key

```go
ID int64 `db:"id" storm:"type:bigserial;primary_key"`

// Composite primary key
type UserRole struct {
    _ struct{} `storm:"table:user_roles"`
    UserID string `db:"user_id" storm:"type:uuid;primary_key"`
    RoleID string `db:"role_id" storm:"type:uuid;primary_key"`
}
```

### Not Null

```go
Email string `db:"email" storm:"type:varchar(255);not_null"`
```

### Unique

```go
// Field-level unique
Email string `db:"email" storm:"type:varchar(255);unique"`

// Table-level unique (composite)
_ struct{} `storm:"table:users;unique:uk_email_tenant,email,tenant_id"`
```

### Check Constraints

```go
// Field-level check
Age int `db:"age" storm:"type:integer;check:age >= 0"`

// Table-level check
_ struct{} `storm:"table:orders;check:ck_valid_dates,start_date < end_date"`
```

## Indexes

### Simple Index

```go
_ struct{} `storm:"table:users;index:idx_email,email"`
```

### Composite Index

```go
_ struct{} `storm:"table:posts;index:idx_user_created,user_id,created_at"`
```

### Multiple Indexes

```go
_ struct{} `storm:"table:products;index:idx_category,category_id;index:idx_price,price"`
```

### Index Types (PostgreSQL Specific)

```go
// B-tree (default)
_ struct{} `storm:"table:users;index:idx_email,email"`

// For full-text search
_ struct{} `storm:"table:posts;index:idx_search,search_vector USING gin"`

// For JSONB
_ struct{} `storm:"table:events;index:idx_metadata,metadata USING gin"`
```

## Foreign Keys

### Basic Foreign Key

```go
UserID string `db:"user_id" storm:"type:uuid;not_null;foreign_key:users.id"`
```

### With Cascade Options

```go
// Cascade delete
UserID string `db:"user_id" storm:"type:uuid;foreign_key:users.id;on_delete:CASCADE"`

// Set null on delete
CategoryID *string `db:"category_id" storm:"type:uuid;foreign_key:categories.id;on_delete:SET NULL"`

// Restrict (default)
UserID string `db:"user_id" storm:"type:uuid;foreign_key:users.id;on_delete:RESTRICT"`

// Update cascade
UserID string `db:"user_id" storm:"type:uuid;foreign_key:users.id;on_update:CASCADE"`
```

### Composite Foreign Keys

```go
type OrderItem struct {
    _ struct{} `storm:"table:order_items;foreign_key:fk_order,order_id,order_number REFERENCES orders(id,number)"`
    OrderID     string `db:"order_id" storm:"type:uuid"`
    OrderNumber int    `db:"order_number" storm:"type:integer"`
}
```

## Defaults

### Static Defaults

```go
IsActive bool      `db:"is_active" storm:"type:boolean;default:true"`
Status   string    `db:"status" storm:"type:varchar(20);default:'pending'"`
Priority int       `db:"priority" storm:"type:integer;default:0"`
```

### Function Defaults

```go
ID        string    `db:"id" storm:"type:uuid;default:gen_random_uuid()"`
CreatedAt time.Time `db:"created_at" storm:"type:timestamptz;default:now()"`
UpdatedAt time.Time `db:"updated_at" storm:"type:timestamptz;default:current_timestamp"`
```

### Complex Defaults

```go
// With timezone
CreatedAt time.Time `db:"created_at" storm:"type:timestamptz;default:now() at time zone 'utc'"`

// Calculated default
Code string `db:"code" storm:"type:varchar(20);default:upper(substring(name from 1 for 3))"`
```

## Complete Reference

### All Field-Level Options

| Option | Description | Example |
|--------|-------------|---------|
| `type` | PostgreSQL data type (required) | `type:varchar(255)` |
| `primary_key` | Mark as primary key | `primary_key` |
| `not_null` | Not null constraint | `not_null` |
| `unique` | Unique constraint | `unique` |
| `default` | Default value | `default:'pending'` |
| `foreign_key` | Foreign key reference | `foreign_key:users.id` |
| `on_delete` | FK delete action | `on_delete:CASCADE` |
| `on_update` | FK update action | `on_update:CASCADE` |
| `check` | Check constraint | `check:age >= 0` |
| `comment` | Column comment | `comment:User's email address` |

### All Table-Level Options

| Option | Description | Example |
|--------|-------------|---------|
| `table` | Table name (required) | `table:users` |
| `index` | Create index | `index:idx_name,column1,column2` |
| `unique` | Unique constraint | `unique:uk_name,column1,column2` |
| `check` | Check constraint | `check:ck_name,expression` |
| `foreign_key` | Composite FK | `foreign_key:fk_name,col1,col2 REFERENCES table(col1,col2)` |
| `comment` | Table comment | `comment:User accounts` |

## Best Practices

### 1. Use Appropriate Types

```go
// Bad - using string for everything
type Product struct {
    Price    string `db:"price" storm:"type:text"`
    Quantity string `db:"quantity" storm:"type:text"`
}

// Good - using appropriate types
type Product struct {
    Price    string `db:"price" storm:"type:decimal(10,2)"`
    Quantity int    `db:"quantity" storm:"type:integer"`
}
```

### 2. Always Use Constraints

```go
// Bad - no constraints
type User struct {
    Email string `db:"email" storm:"type:varchar(255)"`
}

// Good - proper constraints
type User struct {
    Email string `db:"email" storm:"type:varchar(255);not_null;unique"`
}
```

### 3. Index Foreign Keys

```go
type Post struct {
    _ struct{} `storm:"table:posts;index:idx_user,user_id"`
    UserID string `db:"user_id" storm:"type:uuid;not_null;foreign_key:users.id"`
}
```

### 4. Use Timestamps

```go
type BaseModel struct {
    CreatedAt time.Time  `db:"created_at" storm:"type:timestamptz;not_null;default:now()"`
    UpdatedAt time.Time  `db:"updated_at" storm:"type:timestamptz;not_null;default:now()"`
}
```

### 5. Document with Comments

```go
type User struct {
    _ struct{} `storm:"table:users;comment:User accounts for the application"`
    
    ID    string `db:"id" storm:"type:uuid;primary_key;default:gen_random_uuid();comment:Unique user identifier"`
    Email string `db:"email" storm:"type:varchar(255);not_null;unique;comment:User email address for login"`
}
```

## Advanced Examples

### Multi-tenant Schema

```go
type TenantModel struct {
    _ struct{} `storm:"table:posts;unique:uk_tenant_slug,tenant_id,slug"`
    
    TenantID string `db:"tenant_id" storm:"type:uuid;not_null"`
    // ... other fields
}
```

### Audit Fields

```go
type AuditModel struct {
    CreatedBy string     `db:"created_by" storm:"type:uuid;not_null;foreign_key:users.id"`
    CreatedAt time.Time  `db:"created_at" storm:"type:timestamptz;not_null;default:now()"`
    UpdatedBy string     `db:"updated_by" storm:"type:uuid;not_null;foreign_key:users.id"`
    UpdatedAt time.Time  `db:"updated_at" storm:"type:timestamptz;not_null;default:now()"`
    DeletedBy *string    `db:"deleted_by" storm:"type:uuid;foreign_key:users.id"`
    DeletedAt *time.Time `db:"deleted_at" storm:"type:timestamptz"`
}
```

### Complex Constraints

```go
type Booking struct {
    _ struct{} `storm:"table:bookings;check:ck_valid_dates,check_in < check_out;check:ck_positive_price,total_price > 0"`
    
    CheckIn    time.Time `db:"check_in" storm:"type:date;not_null"`
    CheckOut   time.Time `db:"check_out" storm:"type:date;not_null"`
    TotalPrice string    `db:"total_price" storm:"type:decimal(10,2);not_null"`
}
```

## Relationships

Storm supports defining relationships directly in your struct tags using the `relation:` prefix:

### Belongs To

```go
type Post struct {
    _ struct{} `storm:"table:posts"`
    
    ID     string `db:"id" storm:"type:uuid;primary_key;default:gen_random_uuid()"`
    UserID string `db:"user_id" storm:"type:uuid;not_null;foreign_key:users.id"`
    Title  string `db:"title" storm:"type:varchar(255);not_null"`
    
    // Relationship
    User *User `storm:"relation:belongs_to:User;foreign_key:user_id"`
}
```

### Has One

```go
type User struct {
    _ struct{} `storm:"table:users"`
    
    ID    string `db:"id" storm:"type:uuid;primary_key;default:gen_random_uuid()"`
    Email string `db:"email" storm:"type:varchar(255);not_null;unique"`
    
    // Relationship
    Profile *Profile `storm:"relation:has_one:Profile;foreign_key:user_id"`
}
```

### Has Many

```go
type User struct {
    _ struct{} `storm:"table:users"`
    
    ID    string `db:"id" storm:"type:uuid;primary_key;default:gen_random_uuid()"`
    Email string `db:"email" storm:"type:varchar(255);not_null;unique"`
    
    // Relationship
    Posts []Post `storm:"relation:has_many:Post;foreign_key:user_id"`
}
```

### Many to Many

```go
type User struct {
    _ struct{} `storm:"table:users"`
    
    ID    string `db:"id" storm:"type:uuid;primary_key;default:gen_random_uuid()"`
    Email string `db:"email" storm:"type:varchar(255);not_null;unique"`
    
    // Relationship
    Tags []Tag `storm:"relation:has_many_through:Tag;join_table:user_tags;source_fk:user_id;target_fk:tag_id"`
}
```

### Relationship Options

| Option | Description | Example |
|--------|-------------|---------|
| `relation` | Relationship type | `relation:belongs_to:User` |
| `foreign_key` | Foreign key field | `foreign_key:user_id` |
| `source_key` | Source key field | `source_key:id` |
| `target_key` | Target key field | `target_key:id` |
| `join_table` | Join table name | `join_table:user_tags` |
| `source_fk` | Source foreign key | `source_fk:user_id` |
| `target_fk` | Target foreign key | `target_fk:tag_id` |

## Next Steps

- [ORM Guide](orm-guide.md) - Learn about using the generated ORM
- [Query Builder](query-builder.md) - Building complex queries
- [Configuration Guide](configuration.md) - Configuration options