package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/eleven-am/db-migrator/internal/migrator"
	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
)

var (
	// Database connection flags
	dbURL      string
	dbHost     string
	dbPort     string
	dbUser     string
	dbPassword string
	dbName     string
	dbSSLMode  string

	// Migration flags
	outputDir           string
	packagePath         string
	migrationName       string
	dryRun              bool
	createDBIfNotExists bool
	allowDestructive    bool
	pushToDB            bool
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Generate database migrations",
	Long: `Compare current Go structs with database schema and generate migration files.
Uses Atlas for schema comparison, which handles remote databases better than pg-schema-diff.`,
	RunE: runMigrate,
}

func init() {

	migrateCmd.Flags().StringVar(&dbURL, "url", "", "Database connection URL (e.g., postgres://user:pass@localhost:5432/dbname?sslmode=disable)")
	migrateCmd.Flags().StringVar(&dbHost, "host", "localhost", "Database host")
	migrateCmd.Flags().StringVar(&dbPort, "port", "5432", "Database port")
	migrateCmd.Flags().StringVar(&dbUser, "user", "", "Database user")
	migrateCmd.Flags().StringVar(&dbPassword, "password", "", "Database password")
	migrateCmd.Flags().StringVar(&dbName, "dbname", "", "Database name")
	migrateCmd.Flags().StringVar(&dbSSLMode, "sslmode", "disable", "SSL mode (disable, require, verify-ca, verify-full)")

	migrateCmd.Flags().StringVar(&outputDir, "output", "./migrations", "Output directory for migration files")
	migrateCmd.Flags().StringVar(&packagePath, "package", "./internal/db", "Path to package containing models")
	migrateCmd.Flags().StringVar(&migrationName, "name", "", "Migration name (optional)")
	migrateCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print migration without creating files")
	migrateCmd.Flags().BoolVar(&createDBIfNotExists, "create-if-not-exists", false, "Create the database if it does not exist")
	migrateCmd.Flags().BoolVar(&allowDestructive, "allow-destructive", false, "Allow potentially destructive operations")
	migrateCmd.Flags().BoolVar(&pushToDB, "push", false, "Execute the generated SQL directly on the database")
}

func runMigrate(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Build DSN
	var dsn string
	if dbURL != "" {
		dsn = dbURL
	} else if dbUser != "" && dbName != "" {
		if dbURL == "" {
			dsn = migrator.GetDatabaseURL(dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)
		} else {
			dsn = migrator.GetDatabaseDSN(dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)
		}
	} else {
		return fmt.Errorf("either --url or both --user and --dbname must be provided")
	}

	if createDBIfNotExists {
		if err := migrator.EnsureDatabaseExists(dsn); err != nil {
			return err
		}
	}

	fmt.Println("Initializing migrator migration engine...")

	fmt.Println("Connecting to database...")
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	db.SetConnMaxLifetime(10 * time.Minute)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	config := migrator.NewDBConfig(dsn)

	m := migrator.NewAtlasMigrator(config)

	opts := migrator.MigrationOptions{
		PackagePath:         packagePath,
		OutputDir:           outputDir,
		MigrationName:       migrationName,
		DryRun:              dryRun,
		AllowDestructive:    allowDestructive,
		PushToDB:            pushToDB,
		CreateDBIfNotExists: createDBIfNotExists,
	}

	result, err := m.GenerateMigration(ctx, db, opts)
	if err != nil {
		return fmt.Errorf("failed to generate migration: %w", err)
	}

	if result.HasDestructive && !allowDestructive && !dryRun {

		return nil
	}

	if dryRun || pushToDB {
		return nil
	}

	if len(result.Changes) > 0 && outputDir != "" {

	}

	return nil
}
