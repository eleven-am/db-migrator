package introspect

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	// pg_query "github.com/pganalyze/pg_query_go/v6" // Temporarily disabled due to compilation issue
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

	// TODO: Re-enable pg_query when compilation issue is resolved
	// Try to parse using pg_query for proper normalization
	// normalized, err := n.parseAndNormalizeWhere(whereClause)
	// if err != nil {
	//     // Fallback to simple string normalization if parsing fails
	//     return n.simpleNormalizeWhere(whereClause)
	// }
	// return normalized

	// For now, use simple normalization
	return n.simpleNormalizeWhere(whereClause)
}

// parseAndNormalizeWhere uses PostgreSQL's parser for accurate normalization
// TODO: Re-enable when pg_query compilation issue is resolved
func (n *SQLNormalizer) parseAndNormalizeWhere(whereClause string) (string, error) {
	/*
	// Wrap in a minimal SELECT to make it parseable
	query := fmt.Sprintf("SELECT 1 WHERE %s", whereClause)
	
	result, err := pg_query.Parse(query)
	if err != nil {
		return "", fmt.Errorf("failed to parse WHERE clause: %w", err)
	}

	// Extract the WHERE clause from the parsed tree and normalize it
	normalized, err := pg_query.Deparse(result)
	if err != nil {
		return "", fmt.Errorf("failed to deparse WHERE clause: %w", err)
	}

	// Extract just the WHERE portion and clean it up
	whereStart := strings.Index(strings.ToUpper(normalized), "WHERE")
	if whereStart == -1 {
		return "", fmt.Errorf("WHERE clause not found in normalized query")
	}

	normalizedWhere := strings.TrimSpace(normalized[whereStart+5:]) // Skip "WHERE"
	// Basic cleanup - just normalize boolean values and trim
	normalizedWhere = regexp.MustCompile(`(?i)\btrue\b`).ReplaceAllString(normalizedWhere, "true")
	normalizedWhere = regexp.MustCompile(`(?i)\bfalse\b`).ReplaceAllString(normalizedWhere, "false")
	normalizedWhere = strings.TrimSpace(normalizedWhere)
	
	return normalizedWhere, nil
	*/
	return "", fmt.Errorf("pg_query temporarily disabled due to compilation issue")
}

// simpleNormalizeWhere provides fallback normalization when parsing fails
func (n *SQLNormalizer) simpleNormalizeWhere(whereClause string) string {
	// Remove outer parentheses
	whereClause = strings.TrimSpace(whereClause)
	for strings.HasPrefix(whereClause, "(") && strings.HasSuffix(whereClause, ")") {
		inner := strings.TrimSpace(whereClause[1 : len(whereClause)-1])
		if n.isBalancedParentheses(inner) {
			whereClause = inner
		} else {
			break
		}
	}

	// Normalize whitespace
	whereClause = regexp.MustCompile(`\s+`).ReplaceAllString(whereClause, " ")
	whereClause = strings.TrimSpace(whereClause)

	// Normalize boolean values
	whereClause = regexp.MustCompile(`(?i)\btrue\b`).ReplaceAllString(whereClause, "true")
	whereClause = regexp.MustCompile(`(?i)\bfalse\b`).ReplaceAllString(whereClause, "false")

	// Normalize comparison operators (add spaces around them)
	whereClause = regexp.MustCompile(`([^<>=!])(=|<>|!=|<=|>=|<|>)([^<>=!])`).ReplaceAllString(whereClause, "$1 $2 $3")

	// Clean up extra spaces that might have been introduced
	whereClause = regexp.MustCompile(`\s+`).ReplaceAllString(whereClause, " ")
	whereClause = strings.TrimSpace(whereClause)

	return whereClause
}


// isBalancedParentheses checks if parentheses are balanced in a string
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

	// Create a copy to avoid modifying the original
	normalized := make([]string, len(columns))
	for i, col := range columns {
		// Trim whitespace and normalize case
		normalized[i] = strings.TrimSpace(strings.ToLower(col))
	}

	// Sort only if order doesn't matter (for unique constraints)
	// For indexes, order usually matters for query optimization
	if !preserveOrder {
		sort.Strings(normalized)
	}

	return normalized
}

// NormalizeIndexMethod normalizes index method names
func (n *SQLNormalizer) NormalizeIndexMethod(method string) string {
	method = strings.TrimSpace(strings.ToLower(method))
	
	// Default to btree if empty
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
	
	// Table name (normalized)
	parts = append(parts, "table:"+strings.ToLower(strings.TrimSpace(tableName)))
	
	// Columns (preserve order for indexes as it affects performance)
	normalizedCols := n.NormalizeColumnList(columns, true)
	parts = append(parts, "cols:"+strings.Join(normalizedCols, ","))
	
	// Properties
	if isPrimary {
		parts = append(parts, "primary:true")
	}
	if isUnique {
		parts = append(parts, "unique:true")
	}
	
	// Method
	normalizedMethod := n.NormalizeIndexMethod(method)
	parts = append(parts, "method:"+normalizedMethod)
	
	// WHERE clause (normalized)
	if whereClause != "" {
		normalizedWhere := n.NormalizeWhereClause(whereClause)
		if normalizedWhere != "" {
			parts = append(parts, "where:"+normalizedWhere)
		}
	}
	
	return strings.Join(parts, "|")
}