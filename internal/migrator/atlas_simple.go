package migrator

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"
)

// GenerateAtlasSQL generates SQL from Atlas changes using the PlanChanges method
func GenerateAtlasSQL(ctx context.Context, driver migrate.Driver, changes []schema.Change) ([]string, error) {

	plan, err := driver.PlanChanges(ctx, "", changes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate plan: %w", err)
	}

	statements := make([]string, len(plan.Changes))
	for i, change := range plan.Changes {
		statements[i] = change.Cmd
		if change.Comment != "" {
			statements[i] = fmt.Sprintf("-- %s\n%s", change.Comment, change.Cmd)
		}
	}

	return statements, nil
}

// SimplifiedAtlasMigrator provides a simpler Atlas-based migration
type SimplifiedAtlasMigrator struct {
	config        *DBConfig
	tempDBManager *TempDBManager
}

// NewSimplifiedAtlasMigrator creates a new simplified Atlas migrator
func NewSimplifiedAtlasMigrator(config *DBConfig) *SimplifiedAtlasMigrator {
	return &SimplifiedAtlasMigrator{
		config:        config,
		tempDBManager: NewTempDBManager(config),
	}
}

// GenerateMigrationSimple generates migration using Atlas's Plan method
func (m *SimplifiedAtlasMigrator) GenerateMigrationSimple(ctx context.Context, sourceDB *sql.DB, targetDDL string) (upSQL []string, changes []schema.Change, err error) {

	sourceDriver, err := postgres.Open(sourceDB)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create source driver: %w", err)
	}

	currentRealm, err := sourceDriver.InspectRealm(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to inspect current schema: %w", err)
	}

	tempDBName := fmt.Sprintf("temp_atlas_%d", time.Now().Unix())
	tempDB, cleanup, err := m.tempDBManager.CreateTempDB(ctx, tempDBName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create temp database: %w", err)
	}
	defer cleanup()

	if _, err = tempDB.ExecContext(ctx, targetDDL); err != nil {
		return nil, nil, fmt.Errorf("failed to execute DDL in temp database: %w", err)
	}

	targetDriver, err := postgres.Open(tempDB)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create target driver: %w", err)
	}

	targetRealm, err := targetDriver.InspectRealm(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to inspect target schema: %w", err)
	}

	changes, err = sourceDriver.RealmDiff(currentRealm, targetRealm)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to calculate diff: %w", err)
	}

	if len(changes) == 0 {
		return []string{}, changes, nil
	}

	upSQL, err = GenerateAtlasSQL(ctx, sourceDriver, changes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate SQL: %w", err)
	}

	return upSQL, changes, nil
}

// IsDestructiveChange checks if a change is potentially destructive
func IsDestructiveChange(change schema.Change) bool {
	switch change.(type) {
	case *schema.DropTable, *schema.DropColumn, *schema.DropIndex, *schema.DropForeignKey:
		return true
	case *schema.ModifyTable:

		mod := change.(*schema.ModifyTable)
		for _, subChange := range mod.Changes {
			if IsDestructiveChange(subChange) {
				return true
			}
		}
	}
	return false
}

// DescribeChange returns a human-readable description of a change
func DescribeChange(change schema.Change) string {
	switch c := change.(type) {
	case *schema.AddTable:
		return fmt.Sprintf("Create table %s", c.T.Name)
	case *schema.DropTable:
		return fmt.Sprintf("Drop table %s", c.T.Name)
	case *schema.ModifyTable:
		return fmt.Sprintf("Modify table %s (%d changes)", c.T.Name, len(c.Changes))
	case *schema.AddColumn:
		return fmt.Sprintf("Add column %s", c.C.Name)
	case *schema.DropColumn:
		return fmt.Sprintf("Drop column %s", c.C.Name)
	case *schema.ModifyColumn:
		return fmt.Sprintf("Modify column %s", c.To.Name)
	case *schema.AddIndex:
		return fmt.Sprintf("Add index %s", c.I.Name)
	case *schema.DropIndex:
		return fmt.Sprintf("Drop index %s", c.I.Name)
	case *schema.AddForeignKey:
		return fmt.Sprintf("Add foreign key %s", c.F.Symbol)
	case *schema.DropForeignKey:
		return fmt.Sprintf("Drop foreign key %s", c.F.Symbol)
	default:
		return fmt.Sprintf("Change type %T", change)
	}
}

// CountDestructiveChanges counts destructive operations in a change list
func CountDestructiveChanges(changes []schema.Change) (count int, descriptions []string) {
	for _, change := range changes {
		if IsDestructiveChange(change) {
			count++
			descriptions = append(descriptions, DescribeChange(change))
		}
	}
	return count, descriptions
}
