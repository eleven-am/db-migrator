package migrator

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"
)

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

func NewSimplifiedAtlasMigrator(config *DBConfig) *SimplifiedAtlasMigrator {
	return &SimplifiedAtlasMigrator{
		config:        config,
		tempDBManager: NewTempDBManager(config),
	}
}

func (m *SimplifiedAtlasMigrator) GenerateMigrationSimple(ctx context.Context, sourceDB *sql.DB, targetDDL string, createDBIfNotExists bool) (upSQL []string, changes []schema.Change, err error) {

	var currentRealm *schema.Realm

	// If CreateDBIfNotExists is true, assume empty database, skip inspection
	if createDBIfNotExists {
		currentRealm = &schema.Realm{
			Schemas: []*schema.Schema{
				{
					Name:   "public",
					Tables: []*schema.Table{},
				},
			},
		}
	} else {
		sourceDriver, err := postgres.Open(sourceDB)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create source driver: %w", err)
		}

		currentRealm, err = sourceDriver.InspectRealm(ctx, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to inspect current schema: %w", err)
		}
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

	// Use target driver for diff calculation when createDBIfNotExists is true
	var diffDriver migrate.Driver = targetDriver
	if !createDBIfNotExists {
		// For normal cases, we need to create a source driver for diff calculation
		sourceDriver, err := postgres.Open(sourceDB)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create source driver for diff: %w", err)
		}
		diffDriver = sourceDriver
	}

	changes, err = diffDriver.RealmDiff(currentRealm, targetRealm)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to calculate diff: %w", err)
	}

	if len(changes) == 0 {
		return []string{}, changes, nil
	}

	upSQL, err = GenerateAtlasSQL(ctx, diffDriver, changes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate SQL: %w", err)
	}

	return upSQL, changes, nil
}

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

func CountDestructiveChanges(changes []schema.Change) (count int, descriptions []string) {
	for _, change := range changes {
		if IsDestructiveChange(change) {
			count++
			descriptions = append(descriptions, DescribeChange(change))
		}
	}
	return count, descriptions
}

// isDBNotExistError checks if the error is due to database not existing
func isDBNotExistError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "does not exist") ||
		strings.Contains(errStr, "database") && strings.Contains(errStr, "not exist")
}
