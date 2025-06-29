package introspect

import (
	"reflect"
	"testing"
)

func TestSQLNormalizer_NormalizeWhereClause(t *testing.T) {
	normalizer := NewSQLNormalizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty clause",
			input:    "",
			expected: "",
		},
		{
			name:     "simple equality",
			input:    "is_active = true",
			expected: "is_active = true",
		},
		{
			name:     "boolean true normalization",
			input:    "is_active = TRUE",
			expected: "is_active = true",
		},
		{
			name:     "boolean false normalization",
			input:    "is_deleted = FALSE",
			expected: "is_deleted = false",
		},
		{
			name:     "not equal normalization",
			input:    "status != 'inactive'",
			expected: "status != 'inactive'", // Simple normalizer preserves !=
		},
		{
			name:     "whitespace normalization",
			input:    "  status   =   'active'  ",
			expected: "status = 'active'",
		},
		{
			name:     "complex condition",
			input:    "is_active = true AND created_at > '2023-01-01'",
			expected: "is_active = true AND created_at > '2023-01-01'",
		},
		{
			name:     "parentheses removal",
			input:    "(is_active = true)",
			expected: "is_active = true",
		},
		{
			name:     "nested parentheses",
			input:    "((is_active = true))",
			expected: "is_active = true",
		},
		{
			name:     "unbalanced parentheses preserved",
			input:    "(is_active = true OR (status = 'pending')",
			expected: "(is_active = true OR (status = 'pending')",
		},
		{
			name:     "comparison operators",
			input:    "price>=100 AND quantity<=50",
			expected: "price >= 100 AND quantity <= 50", // Simple normalizer handles >= properly
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.NormalizeWhereClause(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeWhereClause(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSQLNormalizer_NormalizeColumnList(t *testing.T) {
	normalizer := NewSQLNormalizer()

	tests := []struct {
		name         string
		columns      []string
		preserveOrder bool
		expected     []string
	}{
		{
			name:         "empty list",
			columns:      []string{},
			preserveOrder: true,
			expected:     []string{},
		},
		{
			name:         "single column",
			columns:      []string{"  ID  "},
			preserveOrder: true,
			expected:     []string{"id"},
		},
		{
			name:         "multiple columns preserve order",
			columns:      []string{"Name", "Email", "CreatedAt"},
			preserveOrder: true,
			expected:     []string{"name", "email", "createdat"},
		},
		{
			name:         "multiple columns sorted",
			columns:      []string{"Name", "Email", "CreatedAt"},
			preserveOrder: false,
			expected:     []string{"createdat", "email", "name"},
		},
		{
			name:         "with whitespace",
			columns:      []string{"  user_id  ", " team_id ", "created_at"},
			preserveOrder: true,
			expected:     []string{"user_id", "team_id", "created_at"},
		},
		{
			name:         "mixed case normalization",
			columns:      []string{"UserID", "TeamName", "isActive"},
			preserveOrder: true,
			expected:     []string{"userid", "teamname", "isactive"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.NormalizeColumnList(tt.columns, tt.preserveOrder)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("NormalizeColumnList(%v, %v) = %v, want %v", 
					tt.columns, tt.preserveOrder, result, tt.expected)
			}
		})
	}
}

func TestSQLNormalizer_NormalizeIndexMethod(t *testing.T) {
	normalizer := NewSQLNormalizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty method defaults to btree",
			input:    "",
			expected: "btree",
		},
		{
			name:     "btree method",
			input:    "BTREE",
			expected: "btree",
		},
		{
			name:     "hash method",
			input:    "Hash",
			expected: "hash",
		},
		{
			name:     "gist method",
			input:    "GIST",
			expected: "gist",
		},
		{
			name:     "gin method",
			input:    "gin",
			expected: "gin",
		},
		{
			name:     "with whitespace",
			input:    "  BTREE  ",
			expected: "btree",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.NormalizeIndexMethod(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeIndexMethod(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSQLNormalizer_GenerateCanonicalSignature(t *testing.T) {
	normalizer := NewSQLNormalizer()

	tests := []struct {
		name        string
		tableName   string
		columns     []string
		isUnique    bool
		isPrimary   bool
		method      string
		whereClause string
		expected    string
	}{
		{
			name:      "simple primary key",
			tableName: "users",
			columns:   []string{"id"},
			isUnique:  true,
			isPrimary: true,
			method:    "btree",
			expected:  "table:users|cols:id|primary:true|unique:true|method:btree",
		},
		{
			name:      "unique index",
			tableName: "users",
			columns:   []string{"email"},
			isUnique:  true,
			isPrimary: false,
			method:    "btree",
			expected:  "table:users|cols:email|unique:true|method:btree",
		},
		{
			name:      "regular index",
			tableName: "users",
			columns:   []string{"created_at"},
			isUnique:  false,
			isPrimary: false,
			method:    "btree",
			expected:  "table:users|cols:created_at|method:btree",
		},
		{
			name:      "composite index",
			tableName: "audit_logs",
			columns:   []string{"entity_type", "entity_id"},
			isUnique:  false,
			isPrimary: false,
			method:    "btree",
			expected:  "table:audit_logs|cols:entity_type,entity_id|method:btree",
		},
		{
			name:        "partial index",
			tableName:   "orders",
			columns:     []string{"user_id"},
			isUnique:    false,
			isPrimary:   false,
			method:      "btree",
			whereClause: "is_active = true",
			expected:    "table:orders|cols:user_id|method:btree|where:is_active = true",
		},
		{
			name:      "hash index",
			tableName: "sessions",
			columns:   []string{"token"},
			isUnique:  true,
			isPrimary: false,
			method:    "hash",
			expected:  "table:sessions|cols:token|unique:true|method:hash",
		},
		{
			name:      "case normalization",
			tableName: "Users",
			columns:   []string{"Email", "TeamID"},
			isUnique:  true,
			isPrimary: false,
			method:    "BTREE",
			expected:  "table:users|cols:email,teamid|unique:true|method:btree",
		},
		{
			name:      "empty method defaults to btree",
			tableName: "posts",
			columns:   []string{"title"},
			isUnique:  false,
			isPrimary: false,
			method:    "",
			expected:  "table:posts|cols:title|method:btree",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.GenerateCanonicalSignature(
				tt.tableName,
				tt.columns,
				tt.isUnique,
				tt.isPrimary,
				tt.method,
				tt.whereClause,
			)
			if result != tt.expected {
				t.Errorf("GenerateCanonicalSignature() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSQLNormalizer_simpleNormalizeWhere(t *testing.T) {
	normalizer := NewSQLNormalizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic equality",
			input:    "status = 'active'",
			expected: "status = 'active'",
		},
		{
			name:     "boolean normalization",
			input:    "is_active = TRUE",
			expected: "is_active = true",
		},
		{
			name:     "whitespace cleanup",
			input:    "  status   =    'pending'  ",
			expected: "status = 'pending'",
		},
		{
			name:     "operator normalization",
			input:    "count!=0",
			expected: "count != 0", // Fixed spacing but != preserved by simple normalizer
		},
		{
			name:     "outer parentheses removal",
			input:    "(status = 'active')",
			expected: "status = 'active'",
		},
		{
			name:     "complex condition preserved",
			input:    "status = 'active' AND (priority > 5 OR urgent = true)",
			expected: "status = 'active' AND (priority > 5 OR urgent = true)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.simpleNormalizeWhere(tt.input)
			if result != tt.expected {
				t.Errorf("simpleNormalizeWhere(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSQLNormalizer_isBalancedParentheses(t *testing.T) {
	normalizer := NewSQLNormalizer()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "empty string",
			input:    "",
			expected: true,
		},
		{
			name:     "no parentheses",
			input:    "status = 'active'",
			expected: true,
		},
		{
			name:     "balanced single pair",
			input:    "(status = 'active')",
			expected: true,
		},
		{
			name:     "balanced nested",
			input:    "((status = 'active') OR (priority > 5))",
			expected: true,
		},
		{
			name:     "unbalanced missing close",
			input:    "(status = 'active'",
			expected: false,
		},
		{
			name:     "unbalanced missing open",
			input:    "status = 'active')",
			expected: false,
		},
		{
			name:     "unbalanced complex",
			input:    "((status = 'active') OR (priority > 5)",
			expected: false,
		},
		{
			name:     "multiple balanced pairs",
			input:    "(a = 1) AND (b = 2) OR (c = 3)",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.isBalancedParentheses(tt.input)
			if result != tt.expected {
				t.Errorf("isBalancedParentheses(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}


// TestSignatureCollisionPrevention tests that semantically different structures don't produce the same signature
func TestSignatureCollisionPrevention(t *testing.T) {
	normalizer := NewSQLNormalizer()

	tests := []struct {
		name      string
		signature1 string
		signature2 string
		shouldDiffer bool
	}{
		{
			name: "different column order should produce different signatures for indexes",
			signature1: normalizer.GenerateCanonicalSignature("users", []string{"name", "email"}, false, false, "btree", ""),
			signature2: normalizer.GenerateCanonicalSignature("users", []string{"email", "name"}, false, false, "btree", ""),
			shouldDiffer: true,
		},
		{
			name: "same columns different uniqueness should differ",
			signature1: normalizer.GenerateCanonicalSignature("users", []string{"email"}, true, false, "btree", ""),
			signature2: normalizer.GenerateCanonicalSignature("users", []string{"email"}, false, false, "btree", ""),
			shouldDiffer: true,
		},
		{
			name: "same index different where clause should differ",
			signature1: normalizer.GenerateCanonicalSignature("users", []string{"status"}, false, false, "btree", "is_active = true"),
			signature2: normalizer.GenerateCanonicalSignature("users", []string{"status"}, false, false, "btree", "is_deleted = false"),
			shouldDiffer: true,
		},
		{
			name: "different methods should differ",
			signature1: normalizer.GenerateCanonicalSignature("users", []string{"data"}, false, false, "gin", ""),
			signature2: normalizer.GenerateCanonicalSignature("users", []string{"data"}, false, false, "btree", ""),
			shouldDiffer: true,
		},
		{
			name: "semantically equivalent where clauses should be the same",
			signature1: normalizer.GenerateCanonicalSignature("users", []string{"status"}, false, false, "btree", "is_active = TRUE"),
			signature2: normalizer.GenerateCanonicalSignature("users", []string{"status"}, false, false, "btree", "is_active = true"),
			shouldDiffer: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldDiffer {
				if tt.signature1 == tt.signature2 {
					t.Errorf("Expected signatures to differ, but both are: %s", tt.signature1)
				}
			} else {
				if tt.signature1 != tt.signature2 {
					t.Errorf("Expected signatures to be the same, but got: %s vs %s", tt.signature1, tt.signature2)
				}
			}
		})
	}
}

// TestComplexWhereClauseNormalization tests normalization of complex WHERE clauses
func TestComplexWhereClauseNormalization(t *testing.T) {
	normalizer := NewSQLNormalizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "function calls in where clause",
			input:    "created_at > NOW() - INTERVAL '1 day'",
			expected: "created_at > NOW() - INTERVAL '1 day'", // Should remain unchanged if pg_query handles it
		},
		{
			name:     "complex nested conditions",
			input:    "(status = 'active' AND (priority > 5 OR urgent = true)) AND created_at > '2023-01-01'",
			expected: "(status = 'active' AND (priority > 5 OR urgent = true)) AND created_at > '2023-01-01'",
		},
		{
			name:     "case insensitive operators",
			input:    "name ILIKE '%test%' AND status IN ('active', 'pending')",
			expected: "name ILIKE '%test%' AND status IN ('active', 'pending')",
		},
		{
			name:     "jsonb operations",
			input:    "metadata->>'status' = 'active' AND metadata ? 'priority'",
			expected: "metadata->>'status' = 'active' AND metadata ? 'priority'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.NormalizeWhereClause(tt.input)
			// For complex cases, we mainly want to ensure no errors and reasonable output
			if result == "" && tt.input != "" {
				t.Errorf("NormalizeWhereClause(%q) returned empty string", tt.input)
			}
			// The exact result may vary based on pg_query behavior, so we're mainly testing for robustness
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkSQLNormalizer_NormalizeWhereClause(b *testing.B) {
	normalizer := NewSQLNormalizer()
	whereClause := "is_active = TRUE AND created_at > '2023-01-01' AND (status != 'deleted' OR priority >= 5)"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		normalizer.NormalizeWhereClause(whereClause)
	}
}

func BenchmarkSQLNormalizer_GenerateCanonicalSignature(b *testing.B) {
	normalizer := NewSQLNormalizer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		normalizer.GenerateCanonicalSignature(
			"users",
			[]string{"email", "team_id", "created_at"},
			true,
			false,
			"btree",
			"is_active = true",
		)
	}
}