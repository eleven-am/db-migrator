package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eleven-am/db-migrator/internal/generator"
	"github.com/eleven-am/db-migrator/internal/parser"
	_ "github.com/lib/pq" //
	"github.com/spf13/cobra"
	"github.com/stripe/pg-schema-diff/pkg/diff"
	"github.com/stripe/pg-schema-diff/pkg/tempdb"
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
	Long:  `Compare current Go structs with database schema and generate migration files using production-grade schema comparison`,
	RunE:  runMigrate,
}

func init() {
	// Database connection flags
	migrateCmd.Flags().StringVar(&dbURL, "url", "", "Database connection URL (e.g., postgres://user:pass@localhost:5432/dbname?sslmode=disable)")
	migrateCmd.Flags().StringVar(&dbHost, "host", "localhost", "Database host")
	migrateCmd.Flags().StringVar(&dbPort, "port", "5432", "Database port")
	migrateCmd.Flags().StringVar(&dbUser, "user", "", "Database user")
	migrateCmd.Flags().StringVar(&dbPassword, "password", "", "Database password")
	migrateCmd.Flags().StringVar(&dbName, "dbname", "", "Database name")
	migrateCmd.Flags().StringVar(&dbSSLMode, "sslmode", "disable", "SSL mode (disable, require, verify-ca, verify-full)")

	// Migration flags
	migrateCmd.Flags().StringVar(&outputDir, "output", "./migrations", "Output directory for migration files")
	migrateCmd.Flags().StringVar(&packagePath, "package", "./internal/db", "Path to package containing models")
	migrateCmd.Flags().StringVar(&migrationName, "name", "", "Migration name (optional)")
	migrateCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print migration without creating files")
	migrateCmd.Flags().BoolVar(&createDBIfNotExists, "create-if-not-exists", false, "Create the database if it does not exist")
	migrateCmd.Flags().BoolVar(&allowDestructive, "allow-destructive", false, "Allow potentially destructive operations")
	migrateCmd.Flags().BoolVar(&pushToDB, "push", false, "Execute the generated SQL directly on the database")
}

func runMigrate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Build DSN
	var dsn string
	if dbURL != "" {
		dsn = dbURL
	} else if dbUser != "" && dbName != "" {
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)
	} else {
		return fmt.Errorf("either --url or both --user and --dbname must be provided")
	}

	if createDBIfNotExists {
		if err := ensureDBExists(dsn); err != nil {
			return err
		}
	}

	fmt.Println("Initializing schema migration engine...")

	// Step 1: Parse Go structs and generate DDL SQL
	fmt.Println("Parsing Go structs...")
	structParser := parser.NewStructParser()
	models, err := structParser.ParseDirectory(packagePath)
	if err != nil {
		return fmt.Errorf("failed to parse structs: %w", err)
	}

	fmt.Printf("Found %d models in %s\n", len(models), packagePath)

	// Step 2: Generate DDL SQL from structs
	fmt.Println("Generating DDL SQL from Go structs...")
	schemaGen := generator.NewSchemaGenerator()
	schema, err := schemaGen.GenerateSchema(models)
	if err != nil {
		return fmt.Errorf("failed to generate schema from structs: %w", err)
	}

	// Convert schema to DDL SQL
	sqlGen := generator.NewSQLGenerator()
	ddlSQL := sqlGen.GenerateSchema(schema)

	fmt.Printf("Generated DDL SQL (%d tables)\n", len(schema.Tables))

	// Step 3: Connect to database
	fmt.Println("Connecting to database...")
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Test connection
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Step 4: Use pg-schema-diff to generate migration plan
	fmt.Println("Analyzing schema differences...")

	// Create temporary database factory for pg-schema-diff
	createConnPoolForDb := func(ctx context.Context, dbName string) (*sql.DB, error) {
		// Build DSN for the temp database by replacing the database name
		var tempDSN string
		if dbURL != "" {
			// Extract base URL and replace database name
			if idx := strings.LastIndex(dbURL, "/"); idx != -1 {
				if queryIdx := strings.Index(dbURL[idx:], "?"); queryIdx != -1 {
					// Has query parameters
					tempDSN = dbURL[:idx+1] + dbName + dbURL[idx+queryIdx:]
				} else {
					// No query parameters
					tempDSN = dbURL[:idx+1] + dbName
				}
			} else {
				return nil, fmt.Errorf("invalid database URL format")
			}
		} else {
			tempDSN = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
				dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)
		}
		return sql.Open("postgres", tempDSN)
	}

	tempDbFactory, err := tempdb.NewOnInstanceFactory(ctx, createConnPoolForDb,
		tempdb.WithRootDatabase("postgres"),
	)

	if err != nil {
		return fmt.Errorf("failed to create temp database factory: %w", err)
	}

	defer tempDbFactory.Close()

	plan, err := diff.Generate(ctx,
		diff.DBSchemaSource(db),
		diff.DDLSchemaSource([]string{ddlSQL}),
		diff.WithTempDbFactory(tempDbFactory),
		diff.WithDataPackNewTables(),
		diff.WithDoNotValidatePlan(),
	)

	if err != nil {
		return fmt.Errorf("failed to generate migration plan: %w", err)
	}

	if len(plan.Statements) == 0 {
		fmt.Println("No schema changes detected! Database is up to date.")
		return nil
	}

	fmt.Printf("Found %d migration statements:\n", len(plan.Statements))

	var upSQL strings.Builder
	var downSQL strings.Builder

	upSQL.WriteString("-- Migration UP generated by db-migrator\n")
	upSQL.WriteString("-- Generated at: " + time.Now().UTC().Format(time.RFC3339) + "\n\n")

	for i, stmt := range plan.Statements {
		upSQL.WriteString(fmt.Sprintf("-- Statement %d\n", i+1))
		upSQL.WriteString(stmt.ToSQL())
		upSQL.WriteString(";\n\n")
	}

	downSQL.WriteString("-- Migration DOWN generated by db-migrator\n")
	downSQL.WriteString("-- Generated at: " + time.Now().UTC().Format(time.RFC3339) + "\n\n")
	downSQL.WriteString("-- WARNING: Reverse migration may cause data loss!\n")
	downSQL.WriteString("-- Review carefully before executing.\n\n")

	reverser := generator.NewMigrationReverser()
	reversedStatements, err := reverser.ReverseStatements(plan.Statements)
	if err != nil {
		fmt.Printf("Warning: Failed to generate complete DOWN migration: %v\n", err)
		downSQL.WriteString(fmt.Sprintf("-- ERROR: Failed to generate complete reversal: %v\n", err))
		downSQL.WriteString("-- Manual reversal may be required\n\n")
	} else {
		for i, stmt := range reversedStatements {
			downSQL.WriteString(fmt.Sprintf("-- Reversal of statement %d\n", len(plan.Statements)-i))
			downSQL.WriteString(stmt)
			downSQL.WriteString(";\n\n")
		}
	}

	upSQLString := upSQL.String()
	downSQLString := downSQL.String()

	hasDestructive := containsDestructiveOperations(plan.Statements)
	if hasDestructive && !allowDestructive {
		fmt.Println("\nPOTENTIALLY DESTRUCTIVE OPERATIONS DETECTED:")
		for i, stmt := range plan.Statements {
			if isDestructiveOperation(stmt) {
				fmt.Printf("  - Statement %d: %s\n", i+1, summarizeStatement(stmt))
			}
		}
		fmt.Println("\nUse --allow-destructive to proceed with these changes.")
		fmt.Println("Review the changes carefully as they may cause data loss.")
		return nil
	}

	if dryRun {
		fmt.Println("\n=== UP Migration ===")
		fmt.Println(upSQLString)
		fmt.Println("\n=== DOWN Migration ===")
		fmt.Println(downSQLString)
		return nil
	}

	if pushToDB {
		fmt.Println("Executing migration on database...")
		for i, stmt := range plan.Statements {
			fmt.Printf("Executing statement %d/%d...\n", i+1, len(plan.Statements))
			if _, err := db.ExecContext(ctx, stmt.ToSQL()); err != nil {
				return fmt.Errorf("failed to execute statement %d: %s\nError: %w", i+1, stmt.ToSQL(), err)
			}
		}

		fmt.Printf("\nMigration executed successfully! Applied %d changes.\n", len(plan.Statements))
		return nil
	}

	timestamp := time.Now().UTC().Format("20060102150405")
	if migrationName == "" {
		migrationName = "schema_update"
	}

	baseName := fmt.Sprintf("%s_%s", timestamp, migrationName)
	upFile := filepath.Join(outputDir, fmt.Sprintf("%s.up.sql", baseName))
	downFile := filepath.Join(outputDir, fmt.Sprintf("%s.down.sql", baseName))

	if err = os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if err = os.WriteFile(upFile, []byte(upSQLString), 0644); err != nil {
		return fmt.Errorf("failed to write UP migration: %w", err)
	}

	if err = os.WriteFile(downFile, []byte(downSQLString), 0644); err != nil {
		return fmt.Errorf("failed to write DOWN migration: %w", err)
	}

	fmt.Printf("\nMigration files created:\n")
	fmt.Printf("  UP:   %s\n", upFile)
	fmt.Printf("  DOWN: %s\n", downFile)

	return nil
}

func containsDestructiveOperations(statements []diff.Statement) bool {
	for _, stmt := range statements {
		if isDestructiveOperation(stmt) {
			return true
		}
	}
	return false
}

func isDestructiveOperation(stmt diff.Statement) bool {
	sql := strings.ToUpper(stmt.ToSQL())
	return strings.Contains(sql, "DROP TABLE") ||
		strings.Contains(sql, "DROP COLUMN") ||
		strings.Contains(sql, "DROP INDEX") ||
		strings.Contains(sql, "DROP CONSTRAINT")
}

func summarizeStatement(stmt diff.Statement) string {
	sql := stmt.ToSQL()
	if len(sql) > 100 {
		return sql[:100] + "..."
	}
	return sql
}
