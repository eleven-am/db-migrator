package diff

import (
	"fmt"
	"strings"

	generator2 "github.com/eleven-am/db-migrator/internal/generator"
)

// MigrationSQLGenerator generates SQL statements for schema changes
type MigrationSQLGenerator struct {
	sqlGen *generator2.SQLGenerator
}

// NewMigrationSQLGenerator creates a new migration SQL generator
func NewMigrationSQLGenerator() *MigrationSQLGenerator {
	return &MigrationSQLGenerator{
		sqlGen: generator2.NewSQLGenerator(),
	}
}

// GenerateSQL generates SQL for a single change
func (g *MigrationSQLGenerator) GenerateSQL(change Change) (upSQL, downSQL string, err error) {
	switch change.Type {
	case ChangeTypeCreateTable:
		return g.generateCreateTable(change)
	case ChangeTypeDropTable:
		return g.generateDropTable(change)
	case ChangeTypeRenameTable:
		return g.generateRenameTable(change)
	case ChangeTypeAddColumn:
		return g.generateAddColumn(change)
	case ChangeTypeDropColumn:
		return g.generateDropColumn(change)
	case ChangeTypeAlterColumn:
		return g.generateAlterColumn(change)
	case ChangeTypeRenameColumn:
		return g.generateRenameColumn(change)
	case ChangeTypeCreateIndex:
		return g.generateCreateIndex(change)
	case ChangeTypeDropIndex:
		return g.generateDropIndex(change)
	case ChangeTypeAddConstraint:
		return g.generateAddConstraint(change)
	case ChangeTypeDropConstraint:
		return g.generateDropConstraint(change)
	default:
		return "", "", fmt.Errorf("unknown change type: %s", change.Type)
	}
}

// GenerateMigration generates complete UP and DOWN migrations for all changes
func (g *MigrationSQLGenerator) GenerateMigration(result *DiffResult) (upSQL, downSQL string, err error) {
	var upStatements []string
	var downStatements []string

	needsGenCuid := g.checkNeedsGenCuid(result.Changes)

	if needsGenCuid {
		upStatements = append(upStatements, g.generateGenCuidFunction())
		downStatements = append([]string{"DROP FUNCTION IF EXISTS gen_cuid();"}, downStatements...)
	}

	orderedChanges := g.orderChangesForUpMigration(result.Changes)

	for _, change := range orderedChanges {
		up, down, err := g.GenerateSQL(change)
		if err != nil {
			return "", "", fmt.Errorf("failed to generate SQL for change: %w", err)
		}

		if up != "" {
			upStatements = append(upStatements, up)
		}
		if down != "" {
			downStatements = append([]string{down}, downStatements...)
		}
	}

	if result.HasUnsafeChanges {
		upSQL = "-- WARNING: This migration contains potentially unsafe changes that could result in data loss\n"
		upSQL += "-- Please review carefully before applying\n\n"
	}

	upSQL += strings.Join(upStatements, "\n\n")
	downSQL = strings.Join(downStatements, "\n\n")

	return upSQL, downSQL, nil
}

// orderChangesForUpMigration orders changes for safe execution
func (g *MigrationSQLGenerator) orderChangesForUpMigration(changes []Change) []Change {
	ordered := make([]Change, 0, len(changes))

	groups := map[ChangeType][]Change{
		ChangeTypeCreateTable:    {},
		ChangeTypeAddColumn:      {},
		ChangeTypeCreateIndex:    {},
		ChangeTypeAddConstraint:  {},
		ChangeTypeAlterColumn:    {},
		ChangeTypeRenameColumn:   {},
		ChangeTypeRenameTable:    {},
		ChangeTypeDropConstraint: {},
		ChangeTypeDropIndex:      {},
		ChangeTypeDropColumn:     {},
		ChangeTypeDropTable:      {},
	}

	for _, change := range changes {
		groups[change.Type] = append(groups[change.Type], change)
	}

	if len(groups[ChangeTypeCreateTable]) > 0 {
		groups[ChangeTypeCreateTable] = g.sortTableCreationsByDependencies(groups[ChangeTypeCreateTable])
	}

	typeOrder := []ChangeType{
		ChangeTypeCreateTable,
		ChangeTypeAddColumn,
		ChangeTypeAlterColumn,
		ChangeTypeRenameColumn,
		ChangeTypeRenameTable,
		ChangeTypeCreateIndex,
		ChangeTypeAddConstraint,
		ChangeTypeDropConstraint,
		ChangeTypeDropIndex,
		ChangeTypeDropColumn,
		ChangeTypeDropTable,
	}

	for _, changeType := range typeOrder {
		ordered = append(ordered, groups[changeType]...)
	}

	return ordered
}

// generateCreateTable generates SQL for creating a table
func (g *MigrationSQLGenerator) generateCreateTable(change Change) (upSQL, downSQL string, err error) {
	table, ok := change.NewValue.(*generator2.SchemaTable)
	if !ok {
		return "", "", fmt.Errorf("invalid table value for CREATE TABLE")
	}

	upSQL = g.sqlGen.GenerateCreateTable(*table)
	downSQL = fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE;", table.Name)

	return upSQL, downSQL, nil
}

// generateDropTable generates SQL for dropping a table
func (g *MigrationSQLGenerator) generateDropTable(change Change) (upSQL, downSQL string, err error) {
	table, ok := change.OldValue.(*generator2.SchemaTable)
	if !ok {
		return "", "", fmt.Errorf("invalid table value for DROP TABLE")
	}

	if change.IsUnsafe {
		upSQL = fmt.Sprintf("-- WARNING: %s\n", change.SafetyNotes)
		upSQL += fmt.Sprintf("-- DROP TABLE %s CASCADE;", table.Name)
	} else {
		upSQL = fmt.Sprintf("DROP TABLE %s CASCADE;", table.Name)
	}

	// For down migration, we need to recreate the table
	downSQL = g.sqlGen.GenerateCreateTable(*table)

	return upSQL, downSQL, nil
}

// generateRenameTable generates SQL for renaming a table
func (g *MigrationSQLGenerator) generateRenameTable(change Change) (upSQL, downSQL string, err error) {
	oldName := change.TableName
	newName, ok := change.NewValue.(string)
	if !ok {
		return "", "", fmt.Errorf("invalid new name for RENAME TABLE")
	}

	upSQL = fmt.Sprintf("ALTER TABLE %s RENAME TO %s;", oldName, newName)
	downSQL = fmt.Sprintf("ALTER TABLE %s RENAME TO %s;", newName, oldName)

	return upSQL, downSQL, nil
}

// generateAddColumn generates SQL for adding a column
func (g *MigrationSQLGenerator) generateAddColumn(change Change) (upSQL, downSQL string, err error) {
	column, ok := change.NewValue.(*generator2.SchemaColumn)
	if !ok {
		return "", "", fmt.Errorf("invalid column value for ADD COLUMN")
	}

	upSQL = fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s;",
		change.TableName,
		g.generateColumnDefinition(column))

	downSQL = fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;",
		change.TableName,
		column.Name)

	return upSQL, downSQL, nil
}

// generateDropColumn generates SQL for dropping a column
func (g *MigrationSQLGenerator) generateDropColumn(change Change) (upSQL, downSQL string, err error) {
	column, ok := change.OldValue.(*generator2.SchemaColumn)
	if !ok {
		return "", "", fmt.Errorf("invalid column value for DROP COLUMN")
	}

	if change.IsUnsafe {
		upSQL = fmt.Sprintf("-- WARNING: %s\n", change.SafetyNotes)
		upSQL += fmt.Sprintf("-- ALTER TABLE %s DROP COLUMN %s;",
			change.TableName,
			column.Name)
	} else {
		upSQL = fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;",
			change.TableName,
			column.Name)
	}

	downSQL = fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s;",
		change.TableName,
		g.generateColumnDefinition(column))

	return upSQL, downSQL, nil
}

// generateAlterColumn generates SQL for altering a column
func (g *MigrationSQLGenerator) generateAlterColumn(change Change) (upSQL, downSQL string, err error) {
	oldColumn, ok := change.OldValue.(*generator2.SchemaColumn)
	if !ok {
		return "", "", fmt.Errorf("invalid old column value for ALTER COLUMN")
	}

	newColumn, ok := change.NewValue.(*generator2.SchemaColumn)
	if !ok {
		return "", "", fmt.Errorf("invalid new column value for ALTER COLUMN")
	}

	var upStatements []string
	var downStatements []string

	// Only generate type change if normalized types are different
	if NormalizePostgreSQLType(oldColumn.Type) != NormalizePostgreSQLType(newColumn.Type) {
		if change.IsUnsafe {
			upStatements = append(upStatements,
				fmt.Sprintf("-- WARNING: Type change from %s to %s may cause data loss", oldColumn.Type, newColumn.Type))
		}
		upStatements = append(upStatements,
			fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s USING %s::%s",
				change.TableName, newColumn.Name, newColumn.Type, newColumn.Name, newColumn.Type))
		downStatements = append(downStatements,
			fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s USING %s::%s",
				change.TableName, oldColumn.Name, oldColumn.Type, oldColumn.Name, oldColumn.Type))
	}

	if oldColumn.IsNullable != newColumn.IsNullable {
		if newColumn.IsNullable {
			upStatements = append(upStatements,
				fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL",
					change.TableName, newColumn.Name))
			downStatements = append(downStatements,
				fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL",
					change.TableName, oldColumn.Name))
		} else {
			if change.IsUnsafe {
				upStatements = append(upStatements,
					fmt.Sprintf("-- WARNING: Setting NOT NULL will fail if NULL values exist"))
			}
			upStatements = append(upStatements,
				fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL",
					change.TableName, newColumn.Name))
			downStatements = append(downStatements,
				fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL",
					change.TableName, oldColumn.Name))
		}
	}

	if !g.defaultsEqual(oldColumn.DefaultValue, newColumn.DefaultValue) {
		if newColumn.DefaultValue != nil {
			upStatements = append(upStatements,
				fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s",
					change.TableName, newColumn.Name, *newColumn.DefaultValue))
		} else {
			upStatements = append(upStatements,
				fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP DEFAULT",
					change.TableName, newColumn.Name))
		}

		if oldColumn.DefaultValue != nil {
			downStatements = append(downStatements,
				fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s",
					change.TableName, oldColumn.Name, *oldColumn.DefaultValue))
		} else {
			downStatements = append(downStatements,
				fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP DEFAULT",
					change.TableName, oldColumn.Name))
		}
	}

	upSQL = strings.Join(upStatements, ";\n")
	if upSQL != "" {
		upSQL += ";"
	}

	downSQL = strings.Join(downStatements, ";\n")
	if downSQL != "" {
		downSQL += ";"
	}

	return upSQL, downSQL, nil
}

// generateRenameColumn generates SQL for renaming a column
func (g *MigrationSQLGenerator) generateRenameColumn(change Change) (upSQL, downSQL string, err error) {
	oldName, ok := change.OldValue.(string)
	if !ok {
		return "", "", fmt.Errorf("invalid old name for RENAME COLUMN")
	}

	newName := change.ColumnName

	upSQL = fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s;",
		change.TableName, oldName, newName)
	downSQL = fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s;",
		change.TableName, newName, oldName)

	return upSQL, downSQL, nil
}

// generateCreateIndex generates SQL for creating an index
func (g *MigrationSQLGenerator) generateCreateIndex(change Change) (upSQL, downSQL string, err error) {
	index, ok := change.NewValue.(*generator2.SchemaIndex)
	if !ok {
		return "", "", fmt.Errorf("invalid index value for CREATE INDEX")
	}

	upSQL = g.sqlGen.GenerateIndexDDL(change.TableName, *index)
	downSQL = fmt.Sprintf("DROP INDEX IF EXISTS %s;", index.Name)

	return upSQL, downSQL, nil
}

// generateDropIndex generates SQL for dropping an index
func (g *MigrationSQLGenerator) generateDropIndex(change Change) (upSQL, downSQL string, err error) {
	index, ok := change.OldValue.(*generator2.SchemaIndex)
	if !ok {
		return "", "", fmt.Errorf("invalid index value for DROP INDEX")
	}

	upSQL = fmt.Sprintf("DROP INDEX IF EXISTS %s;", index.Name)
	downSQL = g.sqlGen.GenerateIndexDDL(change.TableName, *index)

	return upSQL, downSQL, nil
}

// generateAddConstraint generates SQL for adding a constraint
func (g *MigrationSQLGenerator) generateAddConstraint(change Change) (upSQL, downSQL string, err error) {
	constraint, ok := change.NewValue.(*generator2.SchemaConstraint)
	if !ok {
		return "", "", fmt.Errorf("invalid constraint value for ADD CONSTRAINT")
	}

	upSQL = fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s %s",
		change.TableName, constraint.Name, g.generateConstraintDefinition(constraint))

	if len(constraint.Columns) > 0 {
		upSQL += fmt.Sprintf(" (%s)", strings.Join(constraint.Columns, ", "))
	}
	upSQL += ";"

	downSQL = fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s;",
		change.TableName, constraint.Name)

	return upSQL, downSQL, nil
}

// generateDropConstraint generates SQL for dropping a constraint
func (g *MigrationSQLGenerator) generateDropConstraint(change Change) (upSQL, downSQL string, err error) {
	constraint, ok := change.OldValue.(*generator2.SchemaConstraint)
	if !ok {
		return "", "", fmt.Errorf("invalid constraint value for DROP CONSTRAINT")
	}

	upSQL = fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s;",
		change.TableName, constraint.Name)

	downSQL = fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s %s",
		change.TableName, constraint.Name, g.generateConstraintDefinition(constraint))

	if len(constraint.Columns) > 0 {
		downSQL += fmt.Sprintf(" (%s)", strings.Join(constraint.Columns, ", "))
	}
	downSQL += ";"

	return upSQL, downSQL, nil
}

// generateColumnDefinition generates a column definition for CREATE/ALTER TABLE
func (g *MigrationSQLGenerator) generateColumnDefinition(col *generator2.SchemaColumn) string {
	parts := []string{col.Name, col.Type}

	if !col.IsNullable {
		parts = append(parts, "NOT NULL")
	}

	if col.DefaultValue != nil {
		parts = append(parts, fmt.Sprintf("DEFAULT %s", *col.DefaultValue))
	}

	if col.IsUnique && !col.IsPrimaryKey {
		parts = append(parts, "UNIQUE")
	}

	if col.ForeignKey != nil {
		parts = append(parts, fmt.Sprintf("REFERENCES %s(%s)",
			col.ForeignKey.ReferencedTable, col.ForeignKey.ReferencedColumn))

		if col.ForeignKey.OnDelete != "" && col.ForeignKey.OnDelete != "NO ACTION" {
			parts = append(parts, fmt.Sprintf("ON DELETE %s", col.ForeignKey.OnDelete))
		}
		if col.ForeignKey.OnUpdate != "" && col.ForeignKey.OnUpdate != "NO ACTION" {
			parts = append(parts, fmt.Sprintf("ON UPDATE %s", col.ForeignKey.OnUpdate))
		}
	}

	if col.CheckConstraint != nil {
		parts = append(parts, fmt.Sprintf("CHECK (%s)", *col.CheckConstraint))
	}

	return strings.Join(parts, " ")
}

// generateConstraintDefinition generates constraint definition SQL
func (g *MigrationSQLGenerator) generateConstraintDefinition(con *generator2.SchemaConstraint) string {
	switch con.Type {
	case "PRIMARY KEY":
		return "PRIMARY KEY"
	case "UNIQUE":
		return "UNIQUE"
	case "CHECK":
		return fmt.Sprintf("CHECK (%s)", con.Definition)
	case "FOREIGN KEY":
		// Foreign keys are handled differently
		return con.Type
	default:
		return con.Type
	}
}

// defaultsEqual checks if two default values are equal
func (g *MigrationSQLGenerator) defaultsEqual(d1, d2 *string) bool {
	if (d1 == nil) != (d2 == nil) {
		return false
	}
	if d1 != nil && NormalizeDefault(*d1) != NormalizeDefault(*d2) {
		return false
	}
	return true
}

// checkNeedsGenCuid checks if any tables use gen_cuid() function
func (g *MigrationSQLGenerator) checkNeedsGenCuid(changes []Change) bool {
	for _, change := range changes {
		if change.Type == ChangeTypeCreateTable {
			table, ok := change.NewValue.(*generator2.SchemaTable)
			if ok {
				for _, col := range table.Columns {
					if col.DefaultValue != nil && strings.Contains(*col.DefaultValue, "gen_cuid()") {
						return true
					}
				}
			}
		}
	}
	return false
}

// generateGenCuidFunction generates the gen_cuid() function
func (g *MigrationSQLGenerator) generateGenCuidFunction() string {
	return `-- Create gen_cuid() function for CUID generation
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE OR REPLACE FUNCTION gen_cuid() RETURNS char(25) AS $$
DECLARE
    timestamp_ms bigint;
    counter int;
    random_part text;
    result text;
BEGIN
    -- Get current timestamp in milliseconds
    timestamp_ms := EXTRACT(EPOCH FROM clock_timestamp()) * 1000;
    
    -- Get a counter (simplified version, in production you'd want a sequence)
    counter := floor(random() * 16777215)::int; -- 24 bits
    
    -- Generate random part (2 blocks of 4 characters each)
    random_part := substr(md5(gen_random_bytes(16)), 1, 8);
    
    -- Construct CUID: c + timestamp + counter + random
    result := 'c' || lpad(to_hex(timestamp_ms), 12, '0') || 
              lpad(to_hex(counter), 6, '0') || 
              random_part;
    
    -- Ensure we return exactly 25 characters
    RETURN substr(result, 1, 25);
END;
$$ LANGUAGE plpgsql VOLATILE;`
}

// sortTableCreationsByDependencies sorts CREATE TABLE changes by foreign key dependencies
func (g *MigrationSQLGenerator) sortTableCreationsByDependencies(createTableChanges []Change) []Change {
	dependencies := make(map[string][]string)
	tableChanges := make(map[string]Change)

	for _, change := range createTableChanges {
		table, ok := change.NewValue.(*generator2.SchemaTable)
		if !ok {
			continue
		}

		tableChanges[table.Name] = change
		dependencies[table.Name] = []string{}

		for _, col := range table.Columns {
			if col.ForeignKey != nil {
				refTable := col.ForeignKey.ReferencedTable
				if _, exists := tableChanges[refTable]; exists && refTable != table.Name {
					dependencies[table.Name] = append(dependencies[table.Name], refTable)
				}
			}
		}
	}

	sorted := make([]Change, 0, len(createTableChanges))
	visited := make(map[string]bool)
	visiting := make(map[string]bool)

	var visit func(string) bool
	visit = func(tableName string) bool {
		if visited[tableName] {
			return true
		}
		if visiting[tableName] {
			return false
		}

		visiting[tableName] = true

		for _, dep := range dependencies[tableName] {
			visit(dep)
		}

		visiting[tableName] = false
		visited[tableName] = true

		if change, exists := tableChanges[tableName]; exists {
			sorted = append(sorted, change)
		}

		return true
	}

	for tableName := range tableChanges {
		visit(tableName)
	}

	return sorted
}
