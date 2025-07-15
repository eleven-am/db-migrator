package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/eleven-am/storm/pkg/storm"
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
Uses Storm's migration engine for schema comparison and migration generation.`,
	RunE: runMigrate,
}

func init() {
	// Database flags - these will override config file values
	migrateCmd.Flags().StringVar(&dbHost, "host", "localhost", "Database host")
	migrateCmd.Flags().StringVar(&dbPort, "port", "5432", "Database port")
	migrateCmd.Flags().StringVar(&dbUser, "user", "", "Database user")
	migrateCmd.Flags().StringVar(&dbPassword, "password", "", "Database password")
	migrateCmd.Flags().StringVar(&dbName, "dbname", "", "Database name")
	migrateCmd.Flags().StringVar(&dbSSLMode, "sslmode", "disable", "SSL mode (disable, require, verify-ca, verify-full)")

	migrateCmd.Flags().StringVar(&outputDir, "output", "", "Output directory for migration files")
	migrateCmd.Flags().StringVar(&packagePath, "package", "", "Path to package containing models")
	migrateCmd.Flags().StringVar(&migrationName, "name", "", "Migration name (optional)")
	migrateCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print migration without creating files")
	migrateCmd.Flags().BoolVar(&createDBIfNotExists, "create-if-not-exists", false, "Create the database if it does not exist")
	migrateCmd.Flags().BoolVar(&allowDestructive, "allow-destructive", false, "Allow potentially destructive operations")
	migrateCmd.Flags().BoolVar(&pushToDB, "push", false, "Execute the generated SQL directly on the database")
}

func runMigrate(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Apply config file values as defaults
	if stormConfig != nil {
		// Use config values if flags weren't specified
		if outputDir == "" && stormConfig.Migrations.Directory != "" {
			outputDir = stormConfig.Migrations.Directory
		}
		if packagePath == "" && stormConfig.Models.Package != "" {
			packagePath = stormConfig.Models.Package
		}
	}

	// Set final defaults if still empty
	if outputDir == "" {
		outputDir = "./migrations"
	}
	if packagePath == "" {
		packagePath = "./models"
	}

	// Build database URL - use global databaseURL which may come from config
	var dsn string
	if databaseURL != "" {
		dsn = databaseURL
	} else if dbUser != "" && dbName != "" {
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", 
			dbUser, dbPassword, dbHost, dbPort, dbName, dbSSLMode)
	} else {
		return fmt.Errorf("database connection required: use --url flag, individual connection flags, or specify in storm.yaml")
	}

	if verbose {
		cmd.Printf("Using database URL: %s\n", dsn)
		cmd.Printf("Models package: %s\n", packagePath)
		cmd.Printf("Output directory: %s\n", outputDir)
	}

	fmt.Println("Initializing Storm migration engine...")

	// Create Storm client
	config := storm.NewConfig()
	config.DatabaseURL = dsn
	config.ModelsPackage = packagePath
	config.MigrationsDir = outputDir
	config.Debug = debug

	stormClient, err := storm.NewWithConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create Storm client: %w", err)
	}
	defer stormClient.Close()

	// Test connection
	if err := stormClient.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	fmt.Println("Generating migration...")

	// Generate migration
	opts := storm.MigrateOptions{
		PackagePath: packagePath,
		OutputDir:   outputDir,
		DryRun:      dryRun,
	}

	if err := stormClient.Migrate(ctx, opts); err != nil {
		return fmt.Errorf("failed to generate migration: %w", err)
	}

	if dryRun {
		fmt.Println("Migration generated (dry run)")
	} else {
		fmt.Println("Migration generated successfully")
	}

	return nil
}