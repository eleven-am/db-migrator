# Schema Definition Guide

Storm uses `dbdef` tags to define your database schema directly in your Go structs. This guide covers all available options.

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

## Basic Structure

Every model struct can have two types of tags:
- `dbdef` - Defines the database schema
- `db` - Maps struct fields to database columns
- `orm` - Defines relationships (covered in [Relationships Guide](relationships.md))

```go
type Model struct {
    // Table-level configuration
    _ struct{} `dbdef:"table:table_name;option1;option2"`
    
    // Field definitions
    FieldName Type `db:"column_name" dbdef:"type:db_type;constraint1;constraint2"`
}
```

## Table Configuration

Table configuration is defined on an anonymous struct field:

```go
type User struct {
    _ struct{} `dbdef:"table:users;comment:User accounts table"`
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
_ struct{} `dbdef:"table:products;index:idx_category,category_id;index:idx_sku,sku;unique:uk_sku,sku"`
```

## Field Types

Storm supports all PostgreSQL data types:

### Numeric Types

```go
// Integers
SmallInt  int16   `db:"small_int" dbdef:"type:smallint"`
Integer   int32   `db:"integer" dbdef:"type:integer"`
BigInt    int64   `db:"big_int" dbdef:"type:bigint"`

// Serial (auto-increment)
ID        int64   `db:"id" dbdef:"type:bigserial;primary_key"`

// Floating point
Real      float32 `db:"real" dbdef:"type:real"`
Double    float64 `db:"double" dbdef:"type:double precision"`

// Precise numeric
Decimal   string  `db:"price" dbdef:"type:decimal(10,2)"`
Numeric   string  `db:"amount" dbdef:"type:numeric(15,4)"`
```

### Text Types

```go
// Variable length
Name      string  `db:"name" dbdef:"type:varchar(100)"`
Email     string  `db:"email" dbdef:"type:varchar(255)"`

// Fixed length
Code      string  `db:"code" dbdef:"type:char(10)"`

// Unlimited length
Content   string  `db:"content" dbdef:"type:text"`

// PostgreSQL specific
Tsvector  string  `db:"search_vector" dbdef:"type:tsvector"`
```

### Date/Time Types

```go
// Time types
Date      time.Time `db:"date" dbdef:"type:date"`
Time      time.Time `db:"time" dbdef:"type:time"`
Timestamp time.Time `db:"timestamp" dbdef:"type:timestamp"`
Timestamptz time.Time `db:"created_at" dbdef:"type:timestamptz"`

// Intervals
Duration  string   `db:"duration" dbdef:"type:interval"`
```

### Boolean Type

```go
IsActive  bool     `db:"is_active" dbdef:"type:boolean;not_null;default:true"`
```

### UUID Type

```go
ID        string   `db:"id" dbdef:"type:uuid;primary_key;default:gen_random_uuid()"`
```

### JSON Types

```go
// JSON (text-based)
Metadata  string   `db:"metadata" dbdef:"type:json"`

// JSONB (binary, preferred)
Settings  string   `db:"settings" dbdef:"type:jsonb"`

// With Go types
Data      MyStruct `db:"data" dbdef:"type:jsonb"`
```

### Array Types

```go
Tags      []string `db:"tags" dbdef:"type:text[]"`
Numbers   []int    `db:"numbers" dbdef:"type:integer[]"`
```

### Special Types

```go
// Network addresses
IPAddress string   `db:"ip_address" dbdef:"type:inet"`
MacAddr   string   `db:"mac_address" dbdef:"type:macaddr"`

// Geometric types
Point     string   `db:"location" dbdef:"type:point"`
Box       string   `db:"area" dbdef:"type:box"`

// Money
Price     string   `db:"price" dbdef:"type:money"`
```

## Constraints

### Primary Key

```go
ID int64 `db:"id" dbdef:"type:bigserial;primary_key"`

// Composite primary key
type UserRole struct {
    _ struct{} `dbdef:"table:user_roles"`
    UserID string `db:"user_id" dbdef:"type:uuid;primary_key"`
    RoleID string `db:"role_id" dbdef:"type:uuid;primary_key"`
}
```

### Not Null

```go
Email string `db:"email" dbdef:"type:varchar(255);not_null"`
```

### Unique

```go
// Field-level unique
Email string `db:"email" dbdef:"type:varchar(255);unique"`

// Table-level unique (composite)
_ struct{} `dbdef:"table:users;unique:uk_email_tenant,email,tenant_id"`
```

### Check Constraints

```go
// Field-level check
Age int `db:"age" dbdef:"type:integer;check:age >= 0"`

// Table-level check
_ struct{} `dbdef:"table:orders;check:ck_valid_dates,start_date < end_date"`
```

## Indexes

### Simple Index

```go
_ struct{} `dbdef:"table:users;index:idx_email,email"`
```

### Composite Index

```go
_ struct{} `dbdef:"table:posts;index:idx_user_created,user_id,created_at"`
```

### Multiple Indexes

```go
_ struct{} `dbdef:"table:products;index:idx_category,category_id;index:idx_price,price"`
```

### Index Types (PostgreSQL Specific)

```go
// B-tree (default)
_ struct{} `dbdef:"table:users;index:idx_email,email"`

// For full-text search
_ struct{} `dbdef:"table:posts;index:idx_search,search_vector USING gin"`

// For JSONB
_ struct{} `dbdef:"table:events;index:idx_metadata,metadata USING gin"`
```

## Foreign Keys

### Basic Foreign Key

```go
UserID string `db:"user_id" dbdef:"type:uuid;not_null;foreign_key:users.id"`
```

### With Cascade Options

```go
// Cascade delete
UserID string `db:"user_id" dbdef:"type:uuid;foreign_key:users.id;on_delete:CASCADE"`

// Set null on delete
CategoryID *string `db:"category_id" dbdef:"type:uuid;foreign_key:categories.id;on_delete:SET NULL"`

// Restrict (default)
UserID string `db:"user_id" dbdef:"type:uuid;foreign_key:users.id;on_delete:RESTRICT"`

// Update cascade
UserID string `db:"user_id" dbdef:"type:uuid;foreign_key:users.id;on_update:CASCADE"`
```

### Composite Foreign Keys

```go
type OrderItem struct {
    _ struct{} `dbdef:"table:order_items;foreign_key:fk_order,order_id,order_number REFERENCES orders(id,number)"`
    OrderID     string `db:"order_id" dbdef:"type:uuid"`
    OrderNumber int    `db:"order_number" dbdef:"type:integer"`
}
```

## Defaults

### Static Defaults

```go
IsActive bool      `db:"is_active" dbdef:"type:boolean;default:true"`
Status   string    `db:"status" dbdef:"type:varchar(20);default:'pending'"`
Priority int       `db:"priority" dbdef:"type:integer;default:0"`
```

### Function Defaults

```go
ID        string    `db:"id" dbdef:"type:uuid;default:gen_random_uuid()"`
CreatedAt time.Time `db:"created_at" dbdef:"type:timestamptz;default:now()"`
UpdatedAt time.Time `db:"updated_at" dbdef:"type:timestamptz;default:current_timestamp"`
```

### Complex Defaults

```go
// With timezone
CreatedAt time.Time `db:"created_at" dbdef:"type:timestamptz;default:now() at time zone 'utc'"`

// Calculated default
Code string `db:"code" dbdef:"type:varchar(20);default:upper(substring(name from 1 for 3))"`
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
    Price    string `db:"price" dbdef:"type:text"`
    Quantity string `db:"quantity" dbdef:"type:text"`
}

// Good - using appropriate types
type Product struct {
    Price    string `db:"price" dbdef:"type:decimal(10,2)"`
    Quantity int    `db:"quantity" dbdef:"type:integer"`
}
```

### 2. Always Use Constraints

```go
// Bad - no constraints
type User struct {
    Email string `db:"email" dbdef:"type:varchar(255)"`
}

// Good - proper constraints
type User struct {
    Email string `db:"email" dbdef:"type:varchar(255);not_null;unique"`
}
```

### 3. Index Foreign Keys

```go
type Post struct {
    _ struct{} `dbdef:"table:posts;index:idx_user,user_id"`
    UserID string `db:"user_id" dbdef:"type:uuid;not_null;foreign_key:users.id"`
}
```

### 4. Use Timestamps

```go
type BaseModel struct {
    CreatedAt time.Time  `db:"created_at" dbdef:"type:timestamptz;not_null;default:now()"`
    UpdatedAt time.Time  `db:"updated_at" dbdef:"type:timestamptz;not_null;default:now()"`
}
```

### 5. Document with Comments

```go
type User struct {
    _ struct{} `dbdef:"table:users;comment:User accounts for the application"`
    
    ID    string `db:"id" dbdef:"type:uuid;primary_key;default:gen_random_uuid();comment:Unique user identifier"`
    Email string `db:"email" dbdef:"type:varchar(255);not_null;unique;comment:User email address for login"`
}
```

## Advanced Examples

### Multi-tenant Schema

```go
type TenantModel struct {
    _ struct{} `dbdef:"table:posts;unique:uk_tenant_slug,tenant_id,slug"`
    
    TenantID string `db:"tenant_id" dbdef:"type:uuid;not_null"`
    // ... other fields
}
```

### Audit Fields

```go
type AuditModel struct {
    CreatedBy string     `db:"created_by" dbdef:"type:uuid;not_null;foreign_key:users.id"`
    CreatedAt time.Time  `db:"created_at" dbdef:"type:timestamptz;not_null;default:now()"`
    UpdatedBy string     `db:"updated_by" dbdef:"type:uuid;not_null;foreign_key:users.id"`
    UpdatedAt time.Time  `db:"updated_at" dbdef:"type:timestamptz;not_null;default:now()"`
    DeletedBy *string    `db:"deleted_by" dbdef:"type:uuid;foreign_key:users.id"`
    DeletedAt *time.Time `db:"deleted_at" dbdef:"type:timestamptz"`
}
```

### Complex Constraints

```go
type Booking struct {
    _ struct{} `dbdef:"table:bookings;check:ck_valid_dates,check_in < check_out;check:ck_positive_price,total_price > 0"`
    
    CheckIn    time.Time `db:"check_in" dbdef:"type:date;not_null"`
    CheckOut   time.Time `db:"check_out" dbdef:"type:date;not_null"`
    TotalPrice string    `db:"total_price" dbdef:"type:decimal(10,2);not_null"`
}
```

## Next Steps

- [ORM Guide](orm-guide.md) - Learn about the ORM tags
- [Migrations Guide](migrations.md) - Managing schema changes
- [Relationships](relationships.md) - Defining relationships between models