package migrator

import (
	"fmt"
	"regexp"
	"strings"
)

// MigrationReverser handles the reversal of migration statements
type MigrationReverser struct{}

func NewMigrationReverser() *MigrationReverser {
	return &MigrationReverser{}
}

func (mr *MigrationReverser) ReverseSQL(sql string) (string, error) {

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

		return "", nil
	default:

		return fmt.Sprintf("-- WARNING: Unable to automatically reverse the following statement:\n-- %s\n-- Please add manual reversal if needed", sql), nil
	}
}

func (mr *MigrationReverser) reverseCreateTable(sql string) (string, error) {

	re := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([^\s(]+)`)
	matches := re.FindStringSubmatch(sql)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not extract table name from: %s", sql)
	}

	tableName := matches[1]
	return fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", tableName), nil
}

func (mr *MigrationReverser) reverseDropTable(sql string) (string, error) {

	return "-- WARNING: Cannot reverse DROP TABLE without original schema. Backup required for restoration", nil
}

func (mr *MigrationReverser) reverseAlterTable(sql string) (string, error) {
	normalizedSQL := strings.ToUpper(sql)

	tableRe := regexp.MustCompile(`(?i)ALTER\s+TABLE\s+([^\s]+)`)
	tableMatches := tableRe.FindStringSubmatch(sql)
	if len(tableMatches) < 2 {
		return "", fmt.Errorf("could not extract table name from ALTER TABLE: %s", sql)
	}
	tableName := tableMatches[1]

	switch {
	case strings.Contains(normalizedSQL, "ADD COLUMN"):

		colRe := regexp.MustCompile(`(?i)ADD\s+COLUMN\s+([^\s]+)`)
		colMatches := colRe.FindStringSubmatch(sql)
		if len(colMatches) < 2 {
			return "", fmt.Errorf("could not extract column name from ADD COLUMN: %s", sql)
		}
		columnName := colMatches[1]
		return fmt.Sprintf("ALTER TABLE %s DROP COLUMN IF EXISTS %s", tableName, columnName), nil

	case strings.Contains(normalizedSQL, "DROP COLUMN"):

		return "-- WARNING: Cannot reverse DROP COLUMN without original column definition. Backup required for restoration", nil

	case strings.Contains(normalizedSQL, "ADD CONSTRAINT"):

		constraintRe := regexp.MustCompile(`(?i)ADD\s+CONSTRAINT\s+([^\s]+)`)
		constraintMatches := constraintRe.FindStringSubmatch(sql)
		if len(constraintMatches) < 2 {
			return "", fmt.Errorf("could not extract constraint name from ADD CONSTRAINT: %s", sql)
		}
		constraintName := constraintMatches[1]
		return fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s", tableName, constraintName), nil

	case strings.Contains(normalizedSQL, "DROP CONSTRAINT"):

		return "-- WARNING: Cannot reverse DROP CONSTRAINT without original constraint definition", nil

	case strings.Contains(normalizedSQL, "ALTER COLUMN"):

		return fmt.Sprintf("-- WARNING: Cannot automatically reverse ALTER COLUMN. Manual reversal required for:\n-- %s", sql), nil

	case strings.Contains(normalizedSQL, "RENAME"):

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

func (mr *MigrationReverser) reverseCreateIndex(sql string) (string, error) {

	re := regexp.MustCompile(`(?i)CREATE\s+(?:UNIQUE\s+)?INDEX\s+(?:CONCURRENTLY\s+)?(?:IF\s+NOT\s+EXISTS\s+)?([^\s]+)`)
	matches := re.FindStringSubmatch(sql)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not extract index name from: %s", sql)
	}

	indexName := matches[1]
	return fmt.Sprintf("DROP INDEX IF EXISTS %s", indexName), nil
}

func (mr *MigrationReverser) reverseDropIndex(sql string) (string, error) {
	return "-- WARNING: Cannot reverse DROP INDEX without original index definition", nil
}

func (mr *MigrationReverser) reverseCreateSequence(sql string) (string, error) {
	re := regexp.MustCompile(`(?i)CREATE\s+SEQUENCE\s+(?:IF\s+NOT\s+EXISTS\s+)?([^\s]+)`)
	matches := re.FindStringSubmatch(sql)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not extract sequence name from: %s", sql)
	}

	sequenceName := matches[1]
	return fmt.Sprintf("DROP SEQUENCE IF EXISTS %s CASCADE", sequenceName), nil
}

func (mr *MigrationReverser) reverseDropSequence(sql string) (string, error) {
	return "-- WARNING: Cannot reverse DROP SEQUENCE without original sequence definition", nil
}

func (mr *MigrationReverser) reverseCreateType(sql string) (string, error) {
	re := regexp.MustCompile(`(?i)CREATE\s+TYPE\s+([^\s]+)`)
	matches := re.FindStringSubmatch(sql)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not extract type name from: %s", sql)
	}

	typeName := matches[1]
	return fmt.Sprintf("DROP TYPE IF EXISTS %s CASCADE", typeName), nil
}

func (mr *MigrationReverser) reverseDropType(sql string) (string, error) {
	return "-- WARNING: Cannot reverse DROP TYPE without original type definition", nil
}

func (mr *MigrationReverser) reverseCreateFunction(sql string) (string, error) {

	re := regexp.MustCompile(`(?i)CREATE\s+(?:OR\s+REPLACE\s+)?FUNCTION\s+([^\s(]+)\s*\(([^)]*)\)`)
	matches := re.FindStringSubmatch(sql)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not extract function name from: %s", sql)
	}

	functionName := matches[1]

	return fmt.Sprintf("DROP FUNCTION IF EXISTS %s CASCADE", functionName), nil
}

func (mr *MigrationReverser) reverseDropFunction(sql string) (string, error) {
	return "-- WARNING: Cannot reverse DROP FUNCTION without original function definition", nil
}

func (mr *MigrationReverser) reverseCreateTrigger(sql string) (string, error) {

	re := regexp.MustCompile(`(?i)CREATE\s+TRIGGER\s+([^\s]+)\s+.*?\s+ON\s+([^\s]+)`)
	matches := re.FindStringSubmatch(sql)
	if len(matches) < 3 {
		return "", fmt.Errorf("could not extract trigger name and table from: %s", sql)
	}

	triggerName := matches[1]
	tableName := matches[2]
	return fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON %s", triggerName, tableName), nil
}

func (mr *MigrationReverser) reverseDropTrigger(sql string) (string, error) {
	return "-- WARNING: Cannot reverse DROP TRIGGER without original trigger definition", nil
}
