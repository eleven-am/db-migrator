package generator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/stripe/pg-schema-diff/pkg/diff"
)

// MigrationReverser handles the reversal of migration statements
type MigrationReverser struct{}

// NewMigrationReverser creates a new instance of MigrationReverser
func NewMigrationReverser() *MigrationReverser {
	return &MigrationReverser{}
}

// ReverseStatements takes a slice of statements and returns their reversal
func (mr *MigrationReverser) ReverseStatements(statements []diff.Statement) ([]string, error) {
	reversedStatements := make([]string, 0, len(statements))

	// Process statements in reverse order to handle dependencies
	for i := len(statements) - 1; i >= 0; i-- {
		reversed, err := mr.reverseStatement(statements[i])
		if err != nil {
			return nil, fmt.Errorf("failed to reverse statement %d: %w", i+1, err)
		}
		if reversed != "" {
			reversedStatements = append(reversedStatements, reversed)
		}
	}

	return reversedStatements, nil
}

// reverseStatement reverses a single statement
func (mr *MigrationReverser) reverseStatement(stmt diff.Statement) (string, error) {
	return mr.ReverseSQL(stmt.ToSQL())
}

// ReverseSQL reverses a SQL statement string (exported for testing)
func (mr *MigrationReverser) ReverseSQL(sql string) (string, error) {
	// Normalize SQL for easier parsing
	normalizedSQL := strings.TrimSpace(strings.ToUpper(sql))

	switch {
	case strings.HasPrefix(normalizedSQL, "CREATE TABLE"):
		return mr.reverseCreateTable(sql)
	case strings.HasPrefix(normalizedSQL, "DROP TABLE"):
		return mr.reverseDropTable(sql)
	case strings.HasPrefix(normalizedSQL, "ALTER TABLE"):
		return mr.reverseAlterTable(sql)
	case strings.HasPrefix(normalizedSQL, "CREATE INDEX"), strings.HasPrefix(normalizedSQL, "CREATE UNIQUE INDEX"):
		return mr.reverseCreateIndex(sql)
	case strings.HasPrefix(normalizedSQL, "DROP INDEX"):
		return mr.reverseDropIndex(sql)
	case strings.HasPrefix(normalizedSQL, "CREATE SEQUENCE"):
		return mr.reverseCreateSequence(sql)
	case strings.HasPrefix(normalizedSQL, "DROP SEQUENCE"):
		return mr.reverseDropSequence(sql)
	case strings.HasPrefix(normalizedSQL, "CREATE TYPE"):
		return mr.reverseCreateType(sql)
	case strings.HasPrefix(normalizedSQL, "DROP TYPE"):
		return mr.reverseDropType(sql)
	case strings.HasPrefix(normalizedSQL, "CREATE FUNCTION"), strings.HasPrefix(normalizedSQL, "CREATE OR REPLACE FUNCTION"):
		return mr.reverseCreateFunction(sql)
	case strings.HasPrefix(normalizedSQL, "DROP FUNCTION"):
		return mr.reverseDropFunction(sql)
	case strings.HasPrefix(normalizedSQL, "CREATE TRIGGER"):
		return mr.reverseCreateTrigger(sql)
	case strings.HasPrefix(normalizedSQL, "DROP TRIGGER"):
		return mr.reverseDropTrigger(sql)
	case strings.HasPrefix(normalizedSQL, "COMMENT ON"):
		// Comments don't need reversal
		return "", nil
	default:
		// For unknown statements, add a warning comment
		return fmt.Sprintf("-- WARNING: Unable to automatically reverse the following statement:\n-- %s\n-- Please add manual reversal if needed", sql), nil
	}
}

// reverseCreateTable generates DROP TABLE statement
func (mr *MigrationReverser) reverseCreateTable(sql string) (string, error) {
	// Extract table name from CREATE TABLE statement
	re := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([^\s(]+)`)
	matches := re.FindStringSubmatch(sql)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not extract table name from: %s", sql)
	}

	tableName := matches[1]
	return fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", tableName), nil
}

// reverseDropTable would need the original CREATE TABLE statement
func (mr *MigrationReverser) reverseDropTable(sql string) (string, error) {
	// Cannot reverse DROP TABLE without the original schema
	return "-- WARNING: Cannot reverse DROP TABLE without original schema. Backup required for restoration", nil
}

// reverseAlterTable handles various ALTER TABLE operations
func (mr *MigrationReverser) reverseAlterTable(sql string) (string, error) {
	normalizedSQL := strings.ToUpper(sql)

	// Extract table name
	tableRe := regexp.MustCompile(`(?i)ALTER\s+TABLE\s+([^\s]+)`)
	tableMatches := tableRe.FindStringSubmatch(sql)
	if len(tableMatches) < 2 {
		return "", fmt.Errorf("could not extract table name from ALTER TABLE: %s", sql)
	}
	tableName := tableMatches[1]

	switch {
	case strings.Contains(normalizedSQL, "ADD COLUMN"):
		// Extract column name
		colRe := regexp.MustCompile(`(?i)ADD\s+COLUMN\s+([^\s]+)`)
		colMatches := colRe.FindStringSubmatch(sql)
		if len(colMatches) < 2 {
			return "", fmt.Errorf("could not extract column name from ADD COLUMN: %s", sql)
		}
		columnName := colMatches[1]
		return fmt.Sprintf("ALTER TABLE %s DROP COLUMN IF EXISTS %s", tableName, columnName), nil

	case strings.Contains(normalizedSQL, "DROP COLUMN"):
		// Cannot reverse DROP COLUMN without original column definition
		return "-- WARNING: Cannot reverse DROP COLUMN without original column definition. Backup required for restoration", nil

	case strings.Contains(normalizedSQL, "ADD CONSTRAINT"):
		// Extract constraint name
		constraintRe := regexp.MustCompile(`(?i)ADD\s+CONSTRAINT\s+([^\s]+)`)
		constraintMatches := constraintRe.FindStringSubmatch(sql)
		if len(constraintMatches) < 2 {
			return "", fmt.Errorf("could not extract constraint name from ADD CONSTRAINT: %s", sql)
		}
		constraintName := constraintMatches[1]
		return fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s", tableName, constraintName), nil

	case strings.Contains(normalizedSQL, "DROP CONSTRAINT"):
		// Cannot reverse DROP CONSTRAINT without original constraint definition
		return "-- WARNING: Cannot reverse DROP CONSTRAINT without original constraint definition", nil

	case strings.Contains(normalizedSQL, "ALTER COLUMN"):
		// Complex to reverse without knowing previous state
		return fmt.Sprintf("-- WARNING: Cannot automatically reverse ALTER COLUMN. Manual reversal required for:\n-- %s", sql), nil

	case strings.Contains(normalizedSQL, "RENAME"):
		// Handle column/table renames
		if strings.Contains(normalizedSQL, "RENAME COLUMN") {
			renameRe := regexp.MustCompile(`(?i)RENAME\s+COLUMN\s+([^\s]+)\s+TO\s+([^\s]+)`)
			renameMatches := renameRe.FindStringSubmatch(sql)
			if len(renameMatches) < 3 {
				return "", fmt.Errorf("could not extract column names from RENAME COLUMN: %s", sql)
			}
			oldName := renameMatches[1]
			newName := renameMatches[2]
			return fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s", tableName, newName, oldName), nil
		} else if strings.Contains(normalizedSQL, "RENAME TO") {
			renameRe := regexp.MustCompile(`(?i)RENAME\s+TO\s+([^\s]+)`)
			renameMatches := renameRe.FindStringSubmatch(sql)
			if len(renameMatches) < 2 {
				return "", fmt.Errorf("could not extract new table name from RENAME TO: %s", sql)
			}
			newTableName := renameMatches[1]
			return fmt.Sprintf("ALTER TABLE %s RENAME TO %s", newTableName, tableName), nil
		}

	default:
		return fmt.Sprintf("-- WARNING: Unhandled ALTER TABLE operation:\n-- %s", sql), nil
	}

	return "", fmt.Errorf("unhandled ALTER TABLE case: %s", sql)
}

// reverseCreateIndex generates DROP INDEX statement
func (mr *MigrationReverser) reverseCreateIndex(sql string) (string, error) {
	// Extract index name - handle various CREATE INDEX patterns
	re := regexp.MustCompile(`(?i)CREATE\s+(?:UNIQUE\s+)?INDEX\s+(?:CONCURRENTLY\s+)?(?:IF\s+NOT\s+EXISTS\s+)?([^\s]+)`)
	matches := re.FindStringSubmatch(sql)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not extract index name from: %s", sql)
	}

	indexName := matches[1]
	return fmt.Sprintf("DROP INDEX IF EXISTS %s", indexName), nil
}

// reverseDropIndex would need the original CREATE INDEX statement
func (mr *MigrationReverser) reverseDropIndex(sql string) (string, error) {
	return "-- WARNING: Cannot reverse DROP INDEX without original index definition", nil
}

// reverseCreateSequence generates DROP SEQUENCE statement
func (mr *MigrationReverser) reverseCreateSequence(sql string) (string, error) {
	re := regexp.MustCompile(`(?i)CREATE\s+SEQUENCE\s+(?:IF\s+NOT\s+EXISTS\s+)?([^\s]+)`)
	matches := re.FindStringSubmatch(sql)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not extract sequence name from: %s", sql)
	}

	sequenceName := matches[1]
	return fmt.Sprintf("DROP SEQUENCE IF EXISTS %s CASCADE", sequenceName), nil
}

// reverseDropSequence would need the original CREATE SEQUENCE statement
func (mr *MigrationReverser) reverseDropSequence(sql string) (string, error) {
	return "-- WARNING: Cannot reverse DROP SEQUENCE without original sequence definition", nil
}

// reverseCreateType generates DROP TYPE statement
func (mr *MigrationReverser) reverseCreateType(sql string) (string, error) {
	re := regexp.MustCompile(`(?i)CREATE\s+TYPE\s+([^\s]+)`)
	matches := re.FindStringSubmatch(sql)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not extract type name from: %s", sql)
	}

	typeName := matches[1]
	return fmt.Sprintf("DROP TYPE IF EXISTS %s CASCADE", typeName), nil
}

// reverseDropType would need the original CREATE TYPE statement
func (mr *MigrationReverser) reverseDropType(sql string) (string, error) {
	return "-- WARNING: Cannot reverse DROP TYPE without original type definition", nil
}

// reverseCreateFunction generates DROP FUNCTION statement
func (mr *MigrationReverser) reverseCreateFunction(sql string) (string, error) {
	// Extract function name and parameters
	re := regexp.MustCompile(`(?i)CREATE\s+(?:OR\s+REPLACE\s+)?FUNCTION\s+([^\s(]+)\s*\(([^)]*)\)`)
	matches := re.FindStringSubmatch(sql)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not extract function name from: %s", sql)
	}

	functionName := matches[1]
	// For simplicity, dropping with CASCADE to handle dependencies
	return fmt.Sprintf("DROP FUNCTION IF EXISTS %s CASCADE", functionName), nil
}

// reverseDropFunction would need the original CREATE FUNCTION statement
func (mr *MigrationReverser) reverseDropFunction(sql string) (string, error) {
	return "-- WARNING: Cannot reverse DROP FUNCTION without original function definition", nil
}

// reverseCreateTrigger generates DROP TRIGGER statement
func (mr *MigrationReverser) reverseCreateTrigger(sql string) (string, error) {
	// Extract trigger name and table
	re := regexp.MustCompile(`(?i)CREATE\s+TRIGGER\s+([^\s]+)\s+.*?\s+ON\s+([^\s]+)`)
	matches := re.FindStringSubmatch(sql)
	if len(matches) < 3 {
		return "", fmt.Errorf("could not extract trigger name and table from: %s", sql)
	}

	triggerName := matches[1]
	tableName := matches[2]
	return fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON %s", triggerName, tableName), nil
}

// reverseDropTrigger would need the original CREATE TRIGGER statement
func (mr *MigrationReverser) reverseDropTrigger(sql string) (string, error) {
	return "-- WARNING: Cannot reverse DROP TRIGGER without original trigger definition", nil
}
