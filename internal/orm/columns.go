package orm

import (
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
)

// Column represents a type-safe database column reference
type Column[T any] struct {
	Name  string
	Table string
}

// String returns the full column reference for SQL
func (c Column[T]) String() string {
	if c.Table != "" {
		return fmt.Sprintf("%s.%s", c.Table, c.Name)
	}
	return c.Name
}

// Eq creates an equality condition
func (c Column[T]) Eq(value T) Condition {
	return Condition{squirrel.Eq{c.String(): value}}
}

// NotEq creates a not-equal condition
func (c Column[T]) NotEq(value T) Condition {
	return Condition{squirrel.NotEq{c.String(): value}}
}

// In creates an IN condition
func (c Column[T]) In(values ...T) Condition {
	interfaces := make([]interface{}, len(values))
	for i, v := range values {
		interfaces[i] = v
	}
	return Condition{squirrel.Eq{c.String(): interfaces}}
}

// NotIn creates a NOT IN condition
func (c Column[T]) NotIn(values ...T) Condition {
	interfaces := make([]interface{}, len(values))
	for i, v := range values {
		interfaces[i] = v
	}
	return Condition{squirrel.NotEq{c.String(): interfaces}}
}

// IsNull creates an IS NULL condition
func (c Column[T]) IsNull() Condition {
	return Condition{squirrel.Eq{c.String(): nil}}
}

// IsNotNull creates an IS NOT NULL condition
func (c Column[T]) IsNotNull() Condition {
	return Condition{squirrel.NotEq{c.String(): nil}}
}

// Asc creates an ascending order expression
func (c Column[T]) Asc() string {
	return c.String() + " ASC"
}

// Desc creates a descending order expression
func (c Column[T]) Desc() string {
	return c.String() + " DESC"
}

// ComparableColumn provides comparison operations for comparable types
type ComparableColumn[T Comparable] struct {
	Column[T]
}

// Comparable types that support comparison operators
type Comparable interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64 |
		~string |
		time.Time
}

// Gt creates a greater-than condition
func (c ComparableColumn[T]) Gt(value T) Condition {
	return Condition{squirrel.Gt{c.String(): value}}
}

// Gte creates a greater-than-or-equal condition
func (c ComparableColumn[T]) Gte(value T) Condition {
	return Condition{squirrel.GtOrEq{c.String(): value}}
}

// Lt creates a less-than condition
func (c ComparableColumn[T]) Lt(value T) Condition {
	return Condition{squirrel.Lt{c.String(): value}}
}

// Lte creates a less-than-or-equal condition
func (c ComparableColumn[T]) Lte(value T) Condition {
	return Condition{squirrel.LtOrEq{c.String(): value}}
}

// Between creates a BETWEEN condition
func (c ComparableColumn[T]) Between(min, max T) Condition {
	return Condition{squirrel.And{
		squirrel.GtOrEq{c.String(): min},
		squirrel.LtOrEq{c.String(): max},
	}}
}

// StringColumn provides string-specific operations
type StringColumn struct {
	Column[string]
}

// Like creates a LIKE condition
func (c StringColumn) Like(pattern string) Condition {
	return Condition{squirrel.Like{c.String(): pattern}}
}

// ILike creates a case-insensitive LIKE condition (PostgreSQL)
func (c StringColumn) ILike(pattern string) Condition {
	return Condition{squirrel.ILike{c.String(): pattern}}
}

// StartsWith creates a LIKE condition for prefix matching
func (c StringColumn) StartsWith(prefix string) Condition {
	return c.Like(prefix + "%")
}

// EndsWith creates a LIKE condition for suffix matching
func (c StringColumn) EndsWith(suffix string) Condition {
	return c.Like("%" + suffix)
}

// Contains creates a LIKE condition for substring matching
func (c StringColumn) Contains(substring string) Condition {
	return c.Like("%" + substring + "%")
}

// Regexp creates a regular expression condition (PostgreSQL)
func (c StringColumn) Regexp(pattern string) Condition {
	return Condition{squirrel.Expr(c.String()+" ~ ?", pattern)}
}

// NumericColumn provides numeric-specific operations
type NumericColumn[T Numeric] struct {
	ComparableColumn[T]
}

// Numeric types for mathematical operations
type Numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// TimeColumn provides time-specific operations
type TimeColumn struct {
	ComparableColumn[time.Time]
}

// After creates a condition for times after the given time
func (c TimeColumn) After(t time.Time) Condition {
	return c.Gt(t)
}

// Before creates a condition for times before the given time
func (c TimeColumn) Before(t time.Time) Condition {
	return c.Lt(t)
}

// Since creates a condition for times since (after or equal to) the given time
func (c TimeColumn) Since(t time.Time) Condition {
	return c.Gte(t)
}

// Until creates a condition for times until (before or equal to) the given time
func (c TimeColumn) Until(t time.Time) Condition {
	return c.Lte(t)
}

// Today creates a condition for times within today
func (c TimeColumn) Today() Condition {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	return c.Between(startOfDay, endOfDay)
}

// ThisWeek creates a condition for times within this week
func (c TimeColumn) ThisWeek() Condition {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 { // Sunday
		weekday = 7
	}
	startOfWeek := now.AddDate(0, 0, -weekday+1)
	startOfWeek = time.Date(startOfWeek.Year(), startOfWeek.Month(), startOfWeek.Day(), 0, 0, 0, 0, startOfWeek.Location())
	endOfWeek := startOfWeek.AddDate(0, 0, 7)
	return c.Between(startOfWeek, endOfWeek)
}

// ThisMonth creates a condition for times within this month
func (c TimeColumn) ThisMonth() Condition {
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0)
	return c.Between(startOfMonth, endOfMonth)
}

// LastNDays creates a condition for times within the last N days
func (c TimeColumn) LastNDays(days int) Condition {
	now := time.Now()
	start := now.AddDate(0, 0, -days)
	return c.Between(start, now)
}

// BoolColumn provides boolean-specific operations
type BoolColumn struct {
	Column[bool]
}

// IsTrue creates a condition for true values
func (c BoolColumn) IsTrue() Condition {
	return c.Eq(true)
}

// IsFalse creates a condition for false values
func (c BoolColumn) IsFalse() Condition {
	return c.Eq(false)
}

// ArrayColumn provides PostgreSQL array-specific operations
type ArrayColumn[T any] struct {
	Column[[]T]
}

// Contains creates a condition for arrays containing a value (PostgreSQL @> operator)
func (c ArrayColumn[T]) Contains(value T) Condition {
	return Condition{squirrel.Expr(c.String()+" @> ARRAY[?]", value)}
}

// ContainedBy creates a condition for arrays contained by another array (PostgreSQL <@ operator)
func (c ArrayColumn[T]) ContainedBy(values []T) Condition {
	return Condition{squirrel.Expr(c.String()+" <@ ?", values)}
}

// Overlaps creates a condition for arrays that overlap (PostgreSQL && operator)
func (c ArrayColumn[T]) Overlaps(values []T) Condition {
	return Condition{squirrel.Expr(c.String()+" && ?", values)}
}

// Length creates a condition based on array length
func (c ArrayColumn[T]) Length() NumericColumn[int] {
	return NumericColumn[int]{
		ComparableColumn: ComparableColumn[int]{
			Column: Column[int]{
				Name:  fmt.Sprintf("array_length(%s, 1)", c.String()),
				Table: "",
			},
		},
	}
}

// IsEmpty creates a condition for empty arrays
func (c ArrayColumn[T]) IsEmpty() Condition {
	return c.Length().Eq(0)
}

// IsNotEmpty creates a condition for non-empty arrays
func (c ArrayColumn[T]) IsNotEmpty() Condition {
	return c.Length().Gt(0)
}

// JSONBColumn provides PostgreSQL JSONB-specific operations
type JSONBColumn struct {
	Column[interface{}]
}

// JSONBPath creates a path expression for JSONB access
func (c JSONBColumn) JSONBPath(path string) JSONBColumn {
	return JSONBColumn{
		Column: Column[interface{}]{
			Name:  fmt.Sprintf("(%s->'%s')", c.String(), path),
			Table: "",
		},
	}
}

// JSONBPathText creates a path expression for JSONB text access
func (c JSONBColumn) JSONBPathText(path string) StringColumn {
	return StringColumn{
		Column: Column[string]{
			Name:  fmt.Sprintf("(%s->>'%s')", c.String(), path),
			Table: "",
		},
	}
}

// JSONBContains creates a condition for JSONB containment (@> operator)
func (c JSONBColumn) JSONBContains(value interface{}) Condition {
	return Condition{squirrel.Expr(c.String()+" @> ?", value)}
}

// JSONBContainedBy creates a condition for JSONB containment (<@ operator)
func (c JSONBColumn) JSONBContainedBy(value interface{}) Condition {
	return Condition{squirrel.Expr(c.String()+" <@ ?", value)}
}

// JSONBHasKey creates a condition for JSONB key existence (? operator)
func (c JSONBColumn) JSONBHasKey(key string) Condition {
	return Condition{squirrel.Expr(c.String()+" ? ?", key)}
}

// JSONBHasAnyKey creates a condition for JSONB any key existence (?| operator)
func (c JSONBColumn) JSONBHasAnyKey(keys []string) Condition {
	return Condition{squirrel.Expr(c.String()+" ?| ?", keys)}
}

// JSONBHasAllKeys creates a condition for JSONB all keys existence (?& operator)
func (c JSONBColumn) JSONBHasAllKeys(keys []string) Condition {
	return Condition{squirrel.Expr(c.String()+" ?& ?", keys)}
}

// Condition wraps squirrel conditions for type safety
type Condition struct {
	condition squirrel.Sqlizer
}

// And combines conditions with AND
func (c Condition) And(other Condition) Condition {
	return Condition{squirrel.And{c.condition, other.condition}}
}

// Or combines conditions with OR
func (c Condition) Or(other Condition) Condition {
	return Condition{squirrel.Or{c.condition, other.condition}}
}

// Not negates the condition
func (c Condition) Not() Condition {
	return Condition{squirrel.Expr("NOT (?)", c.condition)}
}

// ToSqlizer returns the underlying squirrel condition
func (c Condition) ToSqlizer() squirrel.Sqlizer {
	return c.condition
}
