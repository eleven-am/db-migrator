package diff

import (
	"crypto/md5"
	"fmt"
	"sort"
	"strings"

	"github.com/eleven-am/db-migrator/internal/generator"
)

// ConstraintSignature represents a unique signature for a constraint
type ConstraintSignature struct {
	Type      string
	Columns   []string
	RefTable  string
	RefColumn string
	OnDelete  string
	OnUpdate  string
	CheckExpr string
}

// GenerateConstraintSignature creates a signature for constraint comparison
func GenerateConstraintSignature(constraint generator.SchemaConstraint) ConstraintSignature {
	sig := ConstraintSignature{
		Type: constraint.Type,
	}

	// Sort columns for consistent comparison
	columns := make([]string, len(constraint.Columns))
	copy(columns, constraint.Columns)
	sort.Strings(columns)
	sig.Columns = columns

	// For CHECK constraints, normalize the expression
	if constraint.Type == "CHECK" {
		sig.CheckExpr = NormalizeCheckExpression(constraint.Definition)
	}

	// For FOREIGN KEY constraints, extract reference info
	if constraint.Type == "FOREIGN KEY" && constraint.Definition != "" {
		// Parse FK definition to extract referenced table and column
		// This would need to be implemented based on the actual FK format
		sig.RefTable, sig.RefColumn, sig.OnDelete, sig.OnUpdate = parseFKDefinition(constraint.Definition)
	}

	return sig
}

// Hash returns a hash of the signature for quick comparison
func (s ConstraintSignature) Hash() string {
	parts := []string{s.Type}
	parts = append(parts, s.Columns...)

	if s.RefTable != "" {
		parts = append(parts, "REF:"+s.RefTable+"."+s.RefColumn)
		parts = append(parts, "DELETE:"+s.OnDelete)
		parts = append(parts, "UPDATE:"+s.OnUpdate)
	}

	if s.CheckExpr != "" {
		parts = append(parts, "CHECK:"+s.CheckExpr)
	}

	data := strings.Join(parts, "|")
	return fmt.Sprintf("%x", md5.Sum([]byte(data)))
}

// Equals compares two constraint signatures
func (s ConstraintSignature) Equals(other ConstraintSignature) bool {
	if s.Type != other.Type {
		return false
	}

	if len(s.Columns) != len(other.Columns) {
		return false
	}

	for i, col := range s.Columns {
		if col != other.Columns[i] {
			return false
		}
	}

	if s.Type == "FOREIGN KEY" {
		return s.RefTable == other.RefTable &&
			s.RefColumn == other.RefColumn &&
			s.OnDelete == other.OnDelete &&
			s.OnUpdate == other.OnUpdate
	}

	if s.Type == "CHECK" {
		return s.CheckExpr == other.CheckExpr
	}

	return true
}

// NormalizeCheckExpression normalizes a CHECK constraint expression
func NormalizeCheckExpression(expr string) string {
	// Remove extra whitespace
	expr = strings.TrimSpace(expr)
	expr = strings.ReplaceAll(expr, "  ", " ")

	// Remove outer parentheses if present
	if strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") {
		expr = expr[1 : len(expr)-1]
	}

	// Normalize spacing around operators
	expr = strings.ReplaceAll(expr, " = ", "=")
	expr = strings.ReplaceAll(expr, "= ", "=")
	expr = strings.ReplaceAll(expr, " =", "=")
	expr = strings.ReplaceAll(expr, " AND ", " AND ")
	expr = strings.ReplaceAll(expr, " OR ", " OR ")

	// Normalize boolean literals
	expr = strings.ReplaceAll(expr, "=true", "=true")
	expr = strings.ReplaceAll(expr, "=false", "=false")

	// Normalize type casts
	expr = strings.ReplaceAll(expr, "::text", "")
	expr = strings.ReplaceAll(expr, "::character varying", "")
	expr = strings.ReplaceAll(expr, "::varchar", "")
	expr = strings.ReplaceAll(expr, "::boolean", "")
	expr = strings.ReplaceAll(expr, "::bool", "")

	return expr
}

// parseFKDefinition extracts foreign key information from a definition string
func parseFKDefinition(definition string) (refTable, refColumn, onDelete, onUpdate string) {
	// Default actions
	onDelete = "NO ACTION"
	onUpdate = "NO ACTION"

	// PostgreSQL FK definition format: "FOREIGN KEY (column) REFERENCES table(column) ON DELETE action ON UPDATE action"
	// Parse REFERENCES table(column)
	if idx := strings.Index(definition, "REFERENCES "); idx != -1 {
		rest := definition[idx+11:] // Skip "REFERENCES "

		// Find the table name (ends at opening parenthesis)
		if parenIdx := strings.Index(rest, "("); parenIdx != -1 {
			refTable = strings.TrimSpace(rest[:parenIdx])

			// Find the column name (between parentheses)
			if closeParenIdx := strings.Index(rest[parenIdx:], ")"); closeParenIdx != -1 {
				refColumn = strings.TrimSpace(rest[parenIdx+1 : parenIdx+closeParenIdx])

				// Parse ON DELETE and ON UPDATE clauses
				remaining := rest[parenIdx+closeParenIdx+1:]

				// Parse ON DELETE
				if idx := strings.Index(remaining, "ON DELETE "); idx != -1 {
					actionStart := idx + 10
					actionEnd := strings.Index(remaining[actionStart:], " ON ")
					if actionEnd == -1 {
						onDelete = strings.TrimSpace(remaining[actionStart:])
					} else {
						onDelete = strings.TrimSpace(remaining[actionStart : actionStart+actionEnd])
					}
				}

				// Parse ON UPDATE
				if idx := strings.Index(remaining, "ON UPDATE "); idx != -1 {
					actionStart := idx + 10
					onUpdate = strings.TrimSpace(remaining[actionStart:])
				}
			}
		}
	}

	return
}

// GenerateIndexSignature creates a signature for index comparison
func GenerateIndexSignature(index generator.SchemaIndex) string {
	parts := []string{
		fmt.Sprintf("UNIQUE:%t", index.IsUnique),
		fmt.Sprintf("PRIMARY:%t", index.IsPrimary),
	}

	// Sort columns for consistent comparison
	columns := make([]string, len(index.Columns))
	copy(columns, index.Columns)
	sort.Strings(columns)
	parts = append(parts, "COLS:"+strings.Join(columns, ","))

	if index.Type != "" {
		parts = append(parts, "TYPE:"+index.Type)
	}

	if index.Where != "" {
		normalizedWhere := NormalizeCheckExpression(index.Where)
		parts = append(parts, "WHERE:"+normalizedWhere)
	}

	data := strings.Join(parts, "|")
	return fmt.Sprintf("%x", md5.Sum([]byte(data)))
}
