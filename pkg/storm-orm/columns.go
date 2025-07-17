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

func (c Column[T]) String() string {
	if c.Table != "" {
		return fmt.Sprintf("%s.%s", c.Table, c.Name)
	}
	return c.Name
}

func (c Column[T]) Eq(value T) Condition {
	return Condition{squirrel.Eq{c.String(): value}}
}

func (c Column[T]) NotEq(value T) Condition {
	return Condition{squirrel.NotEq{c.String(): value}}
}

func (c Column[T]) In(values ...T) Condition {
	interfaces := make([]interface{}, len(values))
	for i, v := range values {
		interfaces[i] = v
	}
	return Condition{squirrel.Eq{c.String(): interfaces}}
}

func (c Column[T]) NotIn(values ...T) Condition {
	interfaces := make([]interface{}, len(values))
	for i, v := range values {
		interfaces[i] = v
	}
	return Condition{squirrel.NotEq{c.String(): interfaces}}
}

func (c Column[T]) IsNull() Condition {
	return Condition{squirrel.Eq{c.String(): nil}}
}

func (c Column[T]) IsNotNull() Condition {
	return Condition{squirrel.NotEq{c.String(): nil}}
}

func (c Column[T]) Asc() string {
	return c.String() + " ASC"
}

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

func (c ComparableColumn[T]) Gt(value T) Condition {
	return Condition{squirrel.Gt{c.String(): value}}
}

func (c ComparableColumn[T]) Gte(value T) Condition {
	return Condition{squirrel.GtOrEq{c.String(): value}}
}

func (c ComparableColumn[T]) Lt(value T) Condition {
	return Condition{squirrel.Lt{c.String(): value}}
}

func (c ComparableColumn[T]) Lte(value T) Condition {
	return Condition{squirrel.LtOrEq{c.String(): value}}
}

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

func (c StringColumn) Like(pattern string) Condition {
	return Condition{squirrel.Like{c.String(): pattern}}
}

func (c StringColumn) ILike(pattern string) Condition {
	return Condition{squirrel.ILike{c.String(): pattern}}
}

func (c StringColumn) StartsWith(prefix string) Condition {
	return c.Like(prefix + "%")
}

func (c StringColumn) EndsWith(suffix string) Condition {
	return c.Like("%" + suffix)
}

func (c StringColumn) Contains(substring string) Condition {
	return c.Like("%" + substring + "%")
}

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

func (c TimeColumn) After(t time.Time) Condition {
	return c.Gt(t)
}

func (c TimeColumn) Before(t time.Time) Condition {
	return c.Lt(t)
}

func (c TimeColumn) Since(t time.Time) Condition {
	return c.Gte(t)
}

func (c TimeColumn) Until(t time.Time) Condition {
	return c.Lte(t)
}

func (c TimeColumn) Today() Condition {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	return c.Between(startOfDay, endOfDay)
}

func (c TimeColumn) ThisWeek() Condition {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	startOfWeek := now.AddDate(0, 0, -weekday+1)
	startOfWeek = time.Date(startOfWeek.Year(), startOfWeek.Month(), startOfWeek.Day(), 0, 0, 0, 0, startOfWeek.Location())
	endOfWeek := startOfWeek.AddDate(0, 0, 7)
	return c.Between(startOfWeek, endOfWeek)
}

func (c TimeColumn) ThisMonth() Condition {
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0)
	return c.Between(startOfMonth, endOfMonth)
}

func (c TimeColumn) LastNDays(days int) Condition {
	now := time.Now()
	start := now.AddDate(0, 0, -days)
	return c.Between(start, now)
}

// BoolColumn provides boolean-specific operations
type BoolColumn struct {
	Column[bool]
}

func (c BoolColumn) IsTrue() Condition {
	return c.Eq(true)
}

func (c BoolColumn) IsFalse() Condition {
	return c.Eq(false)
}

// ArrayColumn provides PostgreSQL array-specific operations
type ArrayColumn[T any] struct {
	Column[[]T]
}

func (c ArrayColumn[T]) Contains(value T) Condition {
	return Condition{squirrel.Expr(c.String()+" @> ARRAY[?]", value)}
}

func (c ArrayColumn[T]) ContainedBy(values []T) Condition {
	return Condition{squirrel.Expr(c.String()+" <@ ?", values)}
}

func (c ArrayColumn[T]) Overlaps(values []T) Condition {
	return Condition{squirrel.Expr(c.String()+" && ?", values)}
}

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

func (c ArrayColumn[T]) IsEmpty() Condition {
	return c.Length().Eq(0)
}

func (c ArrayColumn[T]) IsNotEmpty() Condition {
	return c.Length().Gt(0)
}

// JSONBColumn provides PostgreSQL JSONB-specific operations
type JSONBColumn struct {
	Column[interface{}]
}

func (c JSONBColumn) Path(path string) JSONBColumn {
	return JSONBColumn{
		Column: Column[interface{}]{
			Name:  fmt.Sprintf("(%s->'%s')", c.String(), path),
			Table: "",
		},
	}
}

func (c JSONBColumn) PathText(path string) StringColumn {
	return StringColumn{
		Column: Column[string]{
			Name:  fmt.Sprintf("(%s->>'%s')", c.String(), path),
			Table: "",
		},
	}
}

func (c JSONBColumn) Contains(value interface{}) Condition {
	return Condition{squirrel.Expr(c.String()+" @> ?", value)}
}

func (c JSONBColumn) ContainedBy(value interface{}) Condition {
	return Condition{squirrel.Expr(c.String()+" <@ ?", value)}
}

func (c JSONBColumn) HasKey(key string) Condition {
	return Condition{squirrel.Expr(c.String()+" ? ?", key)}
}

func (c JSONBColumn) HasAnyKey(keys []string) Condition {
	return Condition{squirrel.Expr(c.String()+" ?| ?", keys)}
}

func (c JSONBColumn) HasAllKeys(keys []string) Condition {
	return Condition{squirrel.Expr(c.String()+" ?& ?", keys)}
}

// Condition wraps squirrel conditions for type safety
type Condition struct {
	condition squirrel.Sqlizer
}

func (c Condition) And(other Condition) Condition {
	return Condition{squirrel.And{c.condition, other.condition}}
}

func (c Condition) Or(other Condition) Condition {
	return Condition{squirrel.Or{c.condition, other.condition}}
}

func (c Condition) Not() Condition {
	return Condition{squirrel.Expr("NOT (?)", c.condition)}
}

func (c Condition) ToSqlizer() squirrel.Sqlizer {
	return c.condition
}
