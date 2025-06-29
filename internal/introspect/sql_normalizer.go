package introspect

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v6"
)

// SQLNormalizer provides utilities for normalizing SQL expressions
type SQLNormalizer struct{}

// NewSQLNormalizer creates a new SQL normalizer
func NewSQLNormalizer() *SQLNormalizer {
	return &SQLNormalizer{}
}

// NormalizeWhereClause normalizes a WHERE clause predicate for signature comparison
func (n *SQLNormalizer) NormalizeWhereClause(whereClause string) string {
	if whereClause == "" {
		return ""
	}

	normalized, err := n.parseAndNormalizeWhere(whereClause)
	if err != nil {
		return n.simpleNormalizeWhere(whereClause)
	}

	return normalized
}

// parseAndNormalizeWhere uses PostgreSQL's parser for accurate normalization
func (n *SQLNormalizer) parseAndNormalizeWhere(whereClause string) (string, error) {
	query := fmt.Sprintf("SELECT 1 WHERE %s", whereClause)

	result, err := pg_query.Parse(query)
	if err != nil {
		return "", fmt.Errorf("failed to parse WHERE clause: %w", err)
	}

	normalized, err := pg_query.Deparse(result)
	if err != nil {
		return "", fmt.Errorf("failed to deparse WHERE clause: %w", err)
	}

	whereStart := strings.Index(strings.ToUpper(normalized), "WHERE")
	if whereStart == -1 {
		return "", fmt.Errorf("WHERE clause not found in normalized query")
	}

	normalizedWhere := strings.TrimSpace(normalized[whereStart+5:]) // Skip "WHERE"
	normalizedWhere = regexp.MustCompile(`(?i)\btrue\b`).ReplaceAllString(normalizedWhere, "true")
	normalizedWhere = regexp.MustCompile(`(?i)\bfalse\b`).ReplaceAllString(normalizedWhere, "false")
	normalizedWhere = strings.TrimSpace(normalizedWhere)

	return normalizedWhere, nil
}

// simpleNormalizeWhere provides fallback normalization when parsing fails
func (n *SQLNormalizer) simpleNormalizeWhere(whereClause string) string {
	whereClause = strings.TrimSpace(whereClause)
	for strings.HasPrefix(whereClause, "(") && strings.HasSuffix(whereClause, ")") {
		inner := strings.TrimSpace(whereClause[1 : len(whereClause)-1])
		if n.isBalancedParentheses(inner) {
			whereClause = inner
		} else {
			break
		}
	}

	whereClause = regexp.MustCompile(`\s+`).ReplaceAllString(whereClause, " ")
	whereClause = strings.TrimSpace(whereClause)

	whereClause = regexp.MustCompile(`(?i)\btrue\b`).ReplaceAllString(whereClause, "true")
	whereClause = regexp.MustCompile(`(?i)\bfalse\b`).ReplaceAllString(whereClause, "false")

	whereClause = regexp.MustCompile(`([^<>=!])(=|<>|!=|<=|>=|<|>)([^<>=!])`).ReplaceAllString(whereClause, "$1 $2 $3")

	whereClause = regexp.MustCompile(`\s+`).ReplaceAllString(whereClause, " ")
	whereClause = strings.TrimSpace(whereClause)

	return whereClause
}

func (n *SQLNormalizer) isBalancedParentheses(s string) bool {
	count := 0
	for _, r := range s {
		if r == '(' {
			count++
		} else if r == ')' {
			count--
			if count < 0 {
				return false
			}
		}
	}
	return count == 0
}

// NormalizeColumnList normalizes a list of column names for signature comparison
func (n *SQLNormalizer) NormalizeColumnList(columns []string, preserveOrder bool) []string {
	if len(columns) == 0 {
		return columns
	}

	normalized := make([]string, len(columns))
	for i, col := range columns {
		normalized[i] = strings.TrimSpace(strings.ToLower(col))
	}

	if !preserveOrder {
		sort.Strings(normalized)
	}

	return normalized
}

// NormalizeIndexMethod normalizes index method names
func (n *SQLNormalizer) NormalizeIndexMethod(method string) string {
	method = strings.TrimSpace(strings.ToLower(method))

	if method == "" {
		return "btree"
	}

	return method
}

// GenerateCanonicalSignature creates a normalized signature for index comparison
func (n *SQLNormalizer) GenerateCanonicalSignature(
	tableName string,
	columns []string,
	isUnique bool,
	isPrimary bool,
	method string,
	whereClause string,
) string {
	var parts []string

	parts = append(parts, "table:"+strings.ToLower(strings.TrimSpace(tableName)))

	normalizedCols := n.NormalizeColumnList(columns, true)
	parts = append(parts, "cols:"+strings.Join(normalizedCols, ","))

	if isPrimary {
		parts = append(parts, "primary:true")
	}
	if isUnique {
		parts = append(parts, "unique:true")
	}

	normalizedMethod := n.NormalizeIndexMethod(method)
	parts = append(parts, "method:"+normalizedMethod)

	if whereClause != "" {
		normalizedWhere := n.NormalizeWhereClause(whereClause)
		if normalizedWhere != "" {
			parts = append(parts, "where:"+normalizedWhere)
		}
	}

	return strings.Join(parts, "|")
}
