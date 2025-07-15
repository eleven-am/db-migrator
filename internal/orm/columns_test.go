package orm

import (
	"testing"
	"time"

	"github.com/Masterminds/squirrel"
)

func TestStringColumn(t *testing.T) {
	col := StringColumn{Column: Column[string]{Name: "name", Table: "users"}}

	tests := []struct {
		name     string
		method   func() Condition
		expected string
	}{
		{
			name:     "Eq",
			method:   func() Condition { return col.Eq("John") },
			expected: "users.name = ?",
		},
		{
			name:     "NotEq",
			method:   func() Condition { return col.NotEq("John") },
			expected: "users.name <> ?",
		},
		{
			name:     "Like",
			method:   func() Condition { return col.Like("%John%") },
			expected: "users.name LIKE ?",
		},
		{
			name:     "ILike",
			method:   func() Condition { return col.ILike("%john%") },
			expected: "users.name ILIKE ?",
		},
		{
			name:     "StartsWith",
			method:   func() Condition { return col.StartsWith("John") },
			expected: "users.name LIKE ?",
		},
		{
			name:     "EndsWith",
			method:   func() Condition { return col.EndsWith("Doe") },
			expected: "users.name LIKE ?",
		},
		{
			name:     "Contains",
			method:   func() Condition { return col.Contains("oh") },
			expected: "users.name LIKE ?",
		},
		{
			name:     "In",
			method:   func() Condition { return col.In("John", "Jane") },
			expected: "users.name IN (?,?)",
		},
		{
			name:     "IsNull",
			method:   func() Condition { return col.IsNull() },
			expected: "users.name IS NULL",
		},
		{
			name:     "IsNotNull",
			method:   func() Condition { return col.IsNotNull() },
			expected: "users.name IS NOT NULL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			condition := tt.method()
			sql, _, err := condition.ToSqlizer().ToSql()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sql != tt.expected {
				t.Errorf("expected SQL %q, got %q", tt.expected, sql)
			}
		})
	}
}

func TestNumericColumn(t *testing.T) {
	col := NumericColumn[int]{
		ComparableColumn: ComparableColumn[int]{
			Column: Column[int]{Name: "age", Table: "users"},
		},
	}

	tests := []struct {
		name     string
		method   func() Condition
		expected string
	}{
		{
			name:     "Gt",
			method:   func() Condition { return col.Gt(18) },
			expected: "users.age > ?",
		},
		{
			name:     "Gte",
			method:   func() Condition { return col.Gte(18) },
			expected: "users.age >= ?",
		},
		{
			name:     "Lt",
			method:   func() Condition { return col.Lt(65) },
			expected: "users.age < ?",
		},
		{
			name:     "Lte",
			method:   func() Condition { return col.Lte(65) },
			expected: "users.age <= ?",
		},
		{
			name:     "Between",
			method:   func() Condition { return col.Between(18, 65) },
			expected: "(users.age >= ? AND users.age <= ?)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			condition := tt.method()
			sql, _, err := condition.ToSqlizer().ToSql()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sql != tt.expected {
				t.Errorf("expected SQL %q, got %q", tt.expected, sql)
			}
		})
	}
}

func TestTimeColumn(t *testing.T) {
	col := TimeColumn{
		ComparableColumn: ComparableColumn[time.Time]{
			Column: Column[time.Time]{Name: "created_at", Table: "users"},
		},
	}

	now := time.Now()

	tests := []struct {
		name     string
		method   func() Condition
		expected string
	}{
		{
			name:     "After",
			method:   func() Condition { return col.After(now) },
			expected: "users.created_at > ?",
		},
		{
			name:     "Before",
			method:   func() Condition { return col.Before(now) },
			expected: "users.created_at < ?",
		},
		{
			name:     "Since",
			method:   func() Condition { return col.Since(now) },
			expected: "users.created_at >= ?",
		},
		{
			name:     "Until",
			method:   func() Condition { return col.Until(now) },
			expected: "users.created_at <= ?",
		},
		{
			name:     "Today",
			method:   func() Condition { return col.Today() },
			expected: "(users.created_at >= ? AND users.created_at <= ?)",
		},
		{
			name:     "ThisWeek",
			method:   func() Condition { return col.ThisWeek() },
			expected: "(users.created_at >= ? AND users.created_at <= ?)",
		},
		{
			name:     "ThisMonth",
			method:   func() Condition { return col.ThisMonth() },
			expected: "(users.created_at >= ? AND users.created_at <= ?)",
		},
		{
			name:     "LastNDays",
			method:   func() Condition { return col.LastNDays(7) },
			expected: "(users.created_at >= ? AND users.created_at <= ?)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			condition := tt.method()
			sql, _, err := condition.ToSqlizer().ToSql()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sql != tt.expected {
				t.Errorf("expected SQL %q, got %q", tt.expected, sql)
			}
		})
	}
}

func TestBoolColumn(t *testing.T) {
	col := BoolColumn{Column: Column[bool]{Name: "is_active", Table: "users"}}

	tests := []struct {
		name     string
		method   func() Condition
		expected string
	}{
		{
			name:     "IsTrue",
			method:   func() Condition { return col.IsTrue() },
			expected: "users.is_active = ?",
		},
		{
			name:     "IsFalse",
			method:   func() Condition { return col.IsFalse() },
			expected: "users.is_active = ?",
		},
		{
			name:     "Eq",
			method:   func() Condition { return col.Eq(true) },
			expected: "users.is_active = ?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			condition := tt.method()
			sql, _, err := condition.ToSqlizer().ToSql()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sql != tt.expected {
				t.Errorf("expected SQL %q, got %q", tt.expected, sql)
			}
		})
	}
}

func TestArrayColumn(t *testing.T) {
	col := ArrayColumn[string]{Column: Column[[]string]{Name: "tags", Table: "posts"}}

	tests := []struct {
		name     string
		method   func() Condition
		expected string
	}{
		{
			name:     "Contains",
			method:   func() Condition { return col.Contains("go") },
			expected: "posts.tags @> ARRAY[?]",
		},
		{
			name:     "ContainedBy",
			method:   func() Condition { return col.ContainedBy([]string{"go", "rust", "python"}) },
			expected: "posts.tags <@ ?",
		},
		{
			name:     "Overlaps",
			method:   func() Condition { return col.Overlaps([]string{"go", "rust"}) },
			expected: "posts.tags && ?",
		},
		{
			name:     "IsEmpty",
			method:   func() Condition { return col.IsEmpty() },
			expected: "array_length(posts.tags, 1) = ?",
		},
		{
			name:     "IsNotEmpty",
			method:   func() Condition { return col.IsNotEmpty() },
			expected: "array_length(posts.tags, 1) > ?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			condition := tt.method()
			sql, _, err := condition.ToSqlizer().ToSql()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sql != tt.expected {
				t.Errorf("expected SQL %q, got %q", tt.expected, sql)
			}
		})
	}
}

func TestJSONBColumn(t *testing.T) {
	col := JSONBColumn{Column: Column[interface{}]{Name: "metadata", Table: "users"}}

	tests := []struct {
		name     string
		method   func() Condition
		expected string
	}{
		{
			name:     "JSONBContains",
			method:   func() Condition { return col.JSONBContains(map[string]interface{}{"active": true}) },
			expected: "users.metadata @> ?",
		},
		{
			name:     "JSONBContainedBy",
			method:   func() Condition { return col.JSONBContainedBy(map[string]interface{}{"active": true, "role": "admin"}) },
			expected: "users.metadata <@ ?",
		},
		{
			name:     "JSONBHasKey",
			method:   func() Condition { return col.JSONBHasKey("active") },
			expected: "users.metadata ? ?",
		},
		{
			name:     "JSONBHasAnyKey",
			method:   func() Condition { return col.JSONBHasAnyKey([]string{"active", "role"}) },
			expected: "users.metadata ?| ?",
		},
		{
			name:     "JSONBHasAllKeys",
			method:   func() Condition { return col.JSONBHasAllKeys([]string{"active", "role"}) },
			expected: "users.metadata ?& ?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			condition := tt.method()
			sql, _, err := condition.ToSqlizer().ToSql()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sql != tt.expected {
				t.Errorf("expected SQL %q, got %q", tt.expected, sql)
			}
		})
	}
}

func TestConditionOperations(t *testing.T) {
	col1 := StringColumn{Column: Column[string]{Name: "name", Table: "users"}}
	col2 := NumericColumn[int]{
		ComparableColumn: ComparableColumn[int]{
			Column: Column[int]{Name: "age", Table: "users"},
		},
	}

	t.Run("And", func(t *testing.T) {
		condition := col1.Eq("John").And(col2.Gt(18))
		sql, _, err := condition.ToSqlizer().ToSql()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "(users.name = ? AND users.age > ?)"
		if sql != expected {
			t.Errorf("expected SQL %q, got %q", expected, sql)
		}
	})

	t.Run("Or", func(t *testing.T) {
		condition := col1.Eq("John").Or(col1.Eq("Jane"))
		sql, _, err := condition.ToSqlizer().ToSql()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "(users.name = ? OR users.name = ?)"
		if sql != expected {
			t.Errorf("expected SQL %q, got %q", expected, sql)
		}
	})

	t.Run("Not", func(t *testing.T) {
		condition := col1.Eq("John").Not()
		sql, _, err := condition.ToSqlizer().ToSql()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "NOT (users.name = ?)"
		if sql != expected {
			t.Errorf("expected SQL %q, got %q", expected, sql)
		}
	})
}

func TestColumnOrdering(t *testing.T) {
	col := StringColumn{Column: Column[string]{Name: "name", Table: "users"}}

	t.Run("Asc", func(t *testing.T) {
		result := col.Asc()
		expected := "users.name ASC"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("Desc", func(t *testing.T) {
		result := col.Desc()
		expected := "users.name DESC"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})
}

func TestColumnString(t *testing.T) {
	col := StringColumn{Column: Column[string]{Name: "email", Table: "users"}}
	expected := "users.email"
	if result := col.String(); result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// TestConditionSqlizer tests the Condition implementation
func TestConditionSqlizer(t *testing.T) {
	tests := []struct {
		name      string
		condition Condition
		expected  string
		args      []interface{}
	}{
		{
			name:      "simple condition",
			condition: Condition{condition: squirrel.Eq{"users.name": "John"}},
			expected:  "users.name = ?",
			args:      []interface{}{"John"},
		},
		{
			name:      "in condition",
			condition: Condition{condition: squirrel.Eq{"users.age": []int{18, 21, 25}}},
			expected:  "users.age IN (?,?,?)",
			args:      []interface{}{18, 21, 25},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args, err := tt.condition.ToSqlizer().ToSql()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sql != tt.expected {
				t.Errorf("expected SQL %q, got %q", tt.expected, sql)
			}
			if len(args) != len(tt.args) {
				t.Errorf("expected %d args, got %d", len(tt.args), len(args))
			}
		})
	}
}
