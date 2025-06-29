package cmd

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eleven-am/db-migrator/internal/generator"
	"github.com/eleven-am/db-migrator/internal/introspect"
	"github.com/eleven-am/db-migrator/internal/parser"

	_ "github.com/lib/pq" // PostgreSQL driver
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
	Long:  `Compare current Go structs with database schema and generate migration files`,
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
	migrateCmd.Flags().BoolVar(&allowDestructive, "allow-destructive", false, "Allow potentially destructive operations (DROP constraints, DROP indexes)")
	migrateCmd.Flags().BoolVar(&pushToDB, "push", false, "Execute the generated SQL directly on the database")
}

func runMigrate(cmd *cobra.Command, args []string) error {
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

	fmt.Println("Parsing Go structs...")
	structParser := parser.NewStructParser()
	models, err := structParser.ParseDirectory(packagePath)
	if err != nil {
		return fmt.Errorf("failed to parse structs: %w", err)
	}

	fmt.Printf("Found %d models in %s\n", len(models), packagePath)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	fmt.Println("Using signature-based comparison...")

	enhancedGen := generator.NewEnhancedGenerator()
	introspector := introspect.NewPostgreSQLIntrospector(db)

	var allStructIndexes []generator.IndexDefinition
	var allStructForeignKeys []generator.ForeignKeyDefinition

	for _, model := range models {
		indexes, err := enhancedGen.GenerateIndexDefinitions(model)
		if err != nil {
			return fmt.Errorf("failed to generate indexes for %s: %w", model.StructName, err)
		}
		allStructIndexes = append(allStructIndexes, indexes...)

		foreignKeys, err := enhancedGen.GenerateForeignKeyDefinitions(model)
		if err != nil {
			return fmt.Errorf("failed to generate foreign keys for %s: %w", model.StructName, err)
		}
		allStructForeignKeys = append(allStructForeignKeys, foreignKeys...)
	}

	var allDBIndexes []generator.IndexDefinition
	var allDBForeignKeys []generator.ForeignKeyDefinition

	for _, model := range models {
		dbIndexes, err := introspector.GetEnhancedIndexes(model.TableName)
		if err != nil {
			return fmt.Errorf("failed to get indexes for %s: %w", model.TableName, err)
		}

		for _, dbIdx := range dbIndexes {
			genIdx := generator.IndexDefinition{
				Name:       dbIdx.Name,
				TableName:  dbIdx.TableName,
				Columns:    dbIdx.Columns,
				IsUnique:   dbIdx.IsUnique,
				IsPrimary:  dbIdx.IsPrimary,
				Method:     dbIdx.Method,
				Where:      dbIdx.Where,
				Definition: dbIdx.Definition,
				Signature:  dbIdx.Signature,
			}
			allDBIndexes = append(allDBIndexes, genIdx)
		}

		dbForeignKeys, err := introspector.GetEnhancedForeignKeys(model.TableName)
		if err != nil {
			return fmt.Errorf("failed to get foreign keys for %s: %w", model.TableName, err)
		}

		for _, dbFK := range dbForeignKeys {
			genFK := generator.ForeignKeyDefinition{
				Name:              dbFK.Name,
				TableName:         dbFK.TableName,
				Columns:           dbFK.Columns,
				ReferencedTable:   dbFK.ReferencedTable,
				ReferencedColumns: dbFK.ReferencedColumns,
				OnDelete:          dbFK.OnDelete,
				OnUpdate:          dbFK.OnUpdate,
				Definition:        dbFK.Definition,
				Signature:         dbFK.Signature,
			}
			allDBForeignKeys = append(allDBForeignKeys, genFK)
		}
	}

	comparison, err := enhancedGen.CompareSchemas(allStructIndexes, allStructForeignKeys, allDBIndexes, allDBForeignKeys)
	if err != nil {
		return fmt.Errorf("failed to compare schemas: %w", err)
	}

	if !enhancedGen.IsSafeOperation(comparison) && !allowDestructive {
		fmt.Println("\nPOTENTIALLY DESTRUCTIVE OPERATIONS DETECTED:")

		if len(comparison.ForeignKeysToDrop) > 0 {
			fmt.Printf("  - %d foreign key(s) to drop (data integrity risk)\n", len(comparison.ForeignKeysToDrop))
		}

		uniqueDropCount := 0
		for _, idx := range comparison.IndexesToDrop {
			if idx.IsUnique || idx.IsPrimary {
				uniqueDropCount++
			}
		}
		if uniqueDropCount > 0 {
			fmt.Printf("  - %d unique/primary index(es) to drop (duplicate data risk)\n", uniqueDropCount)
		}

		fmt.Println("\nUse --allow-destructive to proceed with these changes.")
		fmt.Println("Review the changes carefully as they may cause data integrity issues.")
		return nil
	}

	upStatements, downStatements, err := enhancedGen.GenerateSafeSQL(comparison, allowDestructive)
	if err != nil {
		return fmt.Errorf("failed to generate SQL: %w", err)
	}

	var upSQL, downSQL string
	if len(upStatements) == 0 {
		upSQL = ""
		downSQL = ""
	} else {
		upSQL = strings.Join(upStatements, "\n\n")
		downSQL = strings.Join(downStatements, "\n\n")
	}

	fmt.Printf("\nSchema comparison summary:\n")
	fmt.Printf("  Indexes to create: %d\n", len(comparison.IndexesToCreate))
	fmt.Printf("  Indexes to drop: %d\n", len(comparison.IndexesToDrop))
	fmt.Printf("  Foreign keys to create: %d\n", len(comparison.ForeignKeysToCreate))
	fmt.Printf("  Foreign keys to drop: %d\n", len(comparison.ForeignKeysToDrop))

	if upSQL == "" && downSQL == "" {
		fmt.Println("No changes detected. Database is up to date!")
		return nil
	}

	if dryRun {
		fmt.Println("\n=== UP Migration ===")
		fmt.Println(upSQL)
		fmt.Println("\n=== DOWN Migration ===")
		fmt.Println(downSQL)
		return nil
	}

	if pushToDB {
		fmt.Println("\nExecuting migration on database...")

		for i, stmt := range upStatements {
			fmt.Printf("Executing statement %d/%d...\n", i+1, len(upStatements))
			if _, err := db.Exec(stmt); err != nil {
				return fmt.Errorf("failed to execute statement %d: %s\nError: %w", i+1, stmt, err)
			}
		}

		fmt.Printf("\nMigration executed successfully! Applied %d statements.\n", len(upStatements))
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

	if err = os.WriteFile(upFile, []byte(upSQL), 0644); err != nil {
		return fmt.Errorf("failed to write UP migration: %w", err)
	}

	if err = os.WriteFile(downFile, []byte(downSQL), 0644); err != nil {
		return fmt.Errorf("failed to write DOWN migration: %w", err)
	}

	fmt.Printf("\nMigration files created:\n")
	fmt.Printf("  UP:   %s\n", upFile)
	fmt.Printf("  DOWN: %s\n", downFile)

	return nil
}
