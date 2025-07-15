package storm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eleven-am/storm/internal/generator"
	"github.com/eleven-am/storm/internal/migrator"
	"github.com/eleven-am/storm/internal/parser"
	"github.com/eleven-am/storm/pkg/storm"
	"github.com/jmoiron/sqlx"
)

// MigratorImpl implements the storm.Migrator interface
type MigratorImpl struct {
	db     *sqlx.DB
	config *storm.Config
	logger storm.Logger
}

// NewMigrator creates a new migrator instance
func NewMigrator(db *sqlx.DB, config *storm.Config, logger storm.Logger) *MigratorImpl {
	return &MigratorImpl{
		db:     db,
		config: config,
		logger: logger,
	}
}

// Generate creates a new migration based on model differences
func (m *MigratorImpl) Generate(ctx context.Context, opts storm.MigrateOptions) (*storm.Migration, error) {
	m.logger.Info("Generating migration...", "package", opts.PackagePath)

	// Ensure migrations directory exists
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create migrations directory: %w", err)
	}

	// Get current schema from database
	currentSchema, err := m.getCurrentSchema(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current schema: %w", err)
	}

	// Get desired schema from models
	desiredSchema, err := m.getDesiredSchema(opts.PackagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get desired schema: %w", err)
	}

	// Generate migration from differences
	migration, err := m.generateMigration(currentSchema, desiredSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to generate migration: %w", err)
	}

	// Save migration to file
	if !opts.DryRun {
		if err := m.saveMigration(migration, opts.OutputDir); err != nil {
			return nil, fmt.Errorf("failed to save migration: %w", err)
		}
	}

	return migration, nil
}

// Apply executes a migration
func (m *MigratorImpl) Apply(ctx context.Context, migration *storm.Migration) error {
	m.logger.Info("Applying migration...", "name", migration.Name)

	// Create migrations table if it doesn't exist
	if err := m.createMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Check if migration already applied
	applied, err := m.isMigrationApplied(ctx, migration.Name)
	if err != nil {
		return fmt.Errorf("failed to check migration status: %w", err)
	}

	if applied {
		m.logger.Info("Migration already applied", "name", migration.Name)
		return nil
	}

	// Begin transaction
	tx, err := m.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute migration
	if err := m.executeMigration(ctx, tx, migration); err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	// Record migration
	if err := m.recordMigration(ctx, tx, migration); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	m.logger.Info("Migration applied successfully", "name", migration.Name)
	return nil
}

// Rollback reverts a migration
func (m *MigratorImpl) Rollback(ctx context.Context, migration *storm.Migration) error {
	m.logger.Info("Rolling back migration...", "name", migration.Name)

	// Check if migration is applied
	applied, err := m.isMigrationApplied(ctx, migration.Name)
	if err != nil {
		return fmt.Errorf("failed to check migration status: %w", err)
	}

	if !applied {
		m.logger.Info("Migration not applied", "name", migration.Name)
		return nil
	}

	// Begin transaction
	tx, err := m.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute rollback
	if err := m.executeRollback(ctx, tx, migration); err != nil {
		return fmt.Errorf("failed to execute rollback: %w", err)
	}

	// Remove migration record
	if err := m.removeMigrationRecord(ctx, tx, migration); err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit rollback: %w", err)
	}

	m.logger.Info("Migration rolled back successfully", "name", migration.Name)
	return nil
}

// Status returns the current migration status
func (m *MigratorImpl) Status(ctx context.Context) (*storm.MigrationStatus, error) {
	// Create migrations table if it doesn't exist
	if err := m.createMigrationsTable(ctx); err != nil {
		return nil, fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get applied migrations
	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Get pending migrations
	pending, err := m.getPendingMigrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending migrations: %w", err)
	}

	return &storm.MigrationStatus{
		Applied:   len(applied),
		Pending:   len(pending),
		Available: len(applied) + len(pending),
		Current:   "",
	}, nil
}

// History returns migration history
func (m *MigratorImpl) History(ctx context.Context) ([]*storm.MigrationRecord, error) {
	// Create migrations table if it doesn't exist
	if err := m.createMigrationsTable(ctx); err != nil {
		return nil, fmt.Errorf("failed to create migrations table: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT name, applied_at, checksum
		FROM %s
		ORDER BY applied_at DESC
	`, m.config.MigrationsTable)

	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query migration history: %w", err)
	}
	defer rows.Close()

	var records []*storm.MigrationRecord
	for rows.Next() {
		var record storm.MigrationRecord
		var name, checksum string
		if err := rows.Scan(&name, &record.AppliedAt, &checksum); err != nil {
			return nil, fmt.Errorf("failed to scan migration record: %w", err)
		}
		record.ID = name
		record.Version = name
		record.Success = true
		records = append(records, &record)
	}

	return records, nil
}

// Pending returns pending migrations
func (m *MigratorImpl) Pending(ctx context.Context) ([]*storm.Migration, error) {
	return m.getPendingMigrations(ctx)
}

// Helper methods

func (m *MigratorImpl) createMigrationsTable(ctx context.Context) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			name VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			checksum VARCHAR(64) NOT NULL
		)
	`, m.config.MigrationsTable)

	_, err := m.db.ExecContext(ctx, query)
	return err
}

func (m *MigratorImpl) isMigrationApplied(ctx context.Context, name string) (bool, error) {
	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM %s WHERE name = $1
	`, m.config.MigrationsTable)

	var count int
	err := m.db.GetContext(ctx, &count, query, name)
	return count > 0, err
}

func (m *MigratorImpl) getAppliedMigrations(ctx context.Context) ([]string, error) {
	query := fmt.Sprintf(`
		SELECT name FROM %s ORDER BY applied_at
	`, m.config.MigrationsTable)

	var names []string
	err := m.db.SelectContext(ctx, &names, query)
	return names, err
}

func (m *MigratorImpl) getPendingMigrations(ctx context.Context) ([]*storm.Migration, error) {
	// Get all migration files
	files, err := filepath.Glob(filepath.Join(m.config.MigrationsDir, "*.sql"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob migration files: %w", err)
	}

	// Get applied migrations
	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	appliedMap := make(map[string]bool)
	for _, name := range applied {
		appliedMap[name] = true
	}

	var pending []*storm.Migration
	for _, file := range files {
		name := filepath.Base(file)
		name = strings.TrimSuffix(name, ".sql")

		if !appliedMap[name] {
			migration, err := m.loadMigration(file)
			if err != nil {
				return nil, fmt.Errorf("failed to load migration %s: %w", name, err)
			}
			pending = append(pending, migration)
		}
	}

	return pending, nil
}

func (m *MigratorImpl) loadMigration(filename string) (*storm.Migration, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read migration file: %w", err)
	}

	name := filepath.Base(filename)
	name = strings.TrimSuffix(name, ".sql")

	// Split into up and down parts
	parts := strings.Split(string(content), "-- +migrate Down")
	up := strings.TrimSpace(parts[0])
	down := ""
	if len(parts) > 1 {
		down = strings.TrimSpace(parts[1])
	}

	// Remove up marker
	up = strings.TrimPrefix(up, "-- +migrate Up")
	up = strings.TrimSpace(up)

	return &storm.Migration{
		Name:      name,
		UpSQL:     up,
		DownSQL:   down,
		Checksum:  m.calculateChecksum(up),
		CreatedAt: time.Now(),
	}, nil
}

func (m *MigratorImpl) executeMigration(ctx context.Context, tx *sqlx.Tx, migration *storm.Migration) error {
	if migration.UpSQL == "" {
		return nil
	}

	// Split statements by semicolon
	statements := strings.Split(migration.UpSQL, ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to execute statement: %s: %w", stmt, err)
		}
	}

	return nil
}

func (m *MigratorImpl) executeRollback(ctx context.Context, tx *sqlx.Tx, migration *storm.Migration) error {
	if migration.DownSQL == "" {
		return fmt.Errorf("no rollback script available for migration %s", migration.Name)
	}

	// Split statements by semicolon
	statements := strings.Split(migration.DownSQL, ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to execute rollback statement: %s: %w", stmt, err)
		}
	}

	return nil
}

func (m *MigratorImpl) recordMigration(ctx context.Context, tx *sqlx.Tx, migration *storm.Migration) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (name, applied_at, checksum)
		VALUES ($1, $2, $3)
	`, m.config.MigrationsTable)

	_, err := tx.ExecContext(ctx, query, migration.Name, time.Now(), migration.Checksum)
	return err
}

func (m *MigratorImpl) removeMigrationRecord(ctx context.Context, tx *sqlx.Tx, migration *storm.Migration) error {
	query := fmt.Sprintf(`
		DELETE FROM %s WHERE name = $1
	`, m.config.MigrationsTable)

	_, err := tx.ExecContext(ctx, query, migration.Name)
	return err
}

func (m *MigratorImpl) getCurrentSchema(ctx context.Context) (*storm.Schema, error) {
	// Use the existing introspection functionality
	schemaInspector := NewSchemaInspector(m.db, m.config, m.logger)
	return schemaInspector.Inspect(ctx)
}

func (m *MigratorImpl) getDesiredSchema(packagePath string) (*storm.Schema, error) {
	// Use the existing parser and generator functionality
	structParser := NewStructParser()
	models, err := structParser.ParseDirectory(packagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse structs: %w", err)
	}

	schemaGenerator := NewSchemaGenerator()
	schema, err := schemaGenerator.GenerateSchema(models)
	if err != nil {
		return nil, fmt.Errorf("failed to generate schema: %w", err)
	}

	// Convert the generator schema to storm schema
	return m.convertGeneratorSchemaToStorm(schema), nil
}

func (m *MigratorImpl) generateMigration(current, desired *storm.Schema) (*storm.Migration, error) {
	// Use the existing Atlas migration functionality
	atlasMigrator := NewAtlasMigrator(m.config.DatabaseURL)

	// Create migration options
	opts := MigrationOptions{
		PackagePath:      m.config.ModelsPackage,
		OutputDir:        m.config.MigrationsDir,
		DryRun:           false,
		AllowDestructive: false,
		PushToDB:         false,
	}

	// Generate migration using Atlas
	ctx := context.Background()
	result, err := atlasMigrator.GenerateMigration(ctx, m.db.DB, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate migration: %w", err)
	}

	// Convert result to Storm migration
	timestamp := time.Now().Format("20060102150405")
	name := fmt.Sprintf("%s_auto_migration", timestamp)

	return &storm.Migration{
		Name:      name,
		UpSQL:     result.UpSQL,
		DownSQL:   result.DownSQL,
		Checksum:  m.calculateChecksum(result.UpSQL),
		CreatedAt: time.Now(),
	}, nil
}

func (m *MigratorImpl) saveMigration(migration *storm.Migration, outputDir string) error {
	filename := filepath.Join(outputDir, migration.Name+".sql")
	content := fmt.Sprintf("-- +migrate Up\n%s\n\n-- +migrate Down\n%s\n", migration.UpSQL, migration.DownSQL)

	return os.WriteFile(filename, []byte(content), 0644)
}

func (m *MigratorImpl) calculateChecksum(content string) string {
	// Simple checksum - in production, use proper hash like SHA256
	return fmt.Sprintf("%x", len(content))
}

// Helper functions to interface with existing db-migrator functionality

func NewStructParser() *parser.StructParser {
	return parser.NewStructParser()
}

func NewSchemaGenerator() *generator.SchemaGenerator {
	return generator.NewSchemaGenerator()
}

func NewAtlasMigrator(databaseURL string) *migrator.AtlasMigrator {
	config := migrator.NewDBConfig(databaseURL)
	return migrator.NewAtlasMigrator(config)
}

type MigrationOptions = migrator.MigrationOptions

func (m *MigratorImpl) convertGeneratorSchemaToStorm(genSchema *generator.DatabaseSchema) *storm.Schema {
	stormSchema := &storm.Schema{
		Tables: make(map[string]*storm.Table),
	}

	for tableName, table := range genSchema.Tables {
		stormTable := &storm.Table{
			Name:    table.Name,
			Columns: make(map[string]*storm.Column),
		}

		for _, col := range table.Columns {
			stormCol := &storm.Column{
				Name:     col.Name,
				Type:     col.Type,
				Nullable: col.IsNullable,
			}

			if col.DefaultValue != nil {
				stormCol.Default = *col.DefaultValue
			}

			stormTable.Columns[col.Name] = stormCol
		}

		stormSchema.Tables[tableName] = stormTable
	}

	return stormSchema
}
