package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/eleven-am/storm/internal/introspect"
	orm_generator "github.com/eleven-am/storm/internal/orm-generator"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
)

var (
	introspectDBURL   string
	introspectFormat  string
	introspectOutput  string
	introspectTable   string
	introspectSchema  string
	introspectPackage string
)

var introspectCmd = &cobra.Command{
	Use:   "introspect",
	Short: "Generate complete Storm ORM code from existing database schema",
	Long: `Generate complete Storm ORM code by introspecting your database schema.
	
This command analyzes your existing database and generates:
- Go struct models with proper tags
- Type-safe column constants
- Repository implementations with CRUD operations
- Query builders for type-safe queries
- Relationship mappings from foreign keys
- Central Storm access point

The generated code provides a complete ORM layer ready for immediate use.

Example:
  storm introspect --database="postgres://user:pass@localhost/mydb" --output=./models --package=models`,
	RunE: runIntrospect,
}

func init() {
	introspectCmd.Flags().StringVarP(&introspectDBURL, "database", "d", "", "Database connection URL (required)")
	introspectCmd.Flags().StringVarP(&introspectOutput, "output", "o", "", "Output directory for generated code (default: ./generated/<package>)")
	introspectCmd.Flags().StringVarP(&introspectTable, "table", "t", "", "Generate ORM for specific table only")
	introspectCmd.Flags().StringVarP(&introspectSchema, "schema", "s", "public", "Database schema to inspect")
	introspectCmd.Flags().StringVarP(&introspectPackage, "package", "p", "models", "Package name for generated code")

	introspectCmd.Flags().StringVarP(&introspectFormat, "format", "f", "orm", "Export format (deprecated)")
	introspectCmd.Flags().MarkHidden("format")

	introspectCmd.MarkFlagRequired("database")
}

func runIntrospect(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	db, err := sqlx.Open("postgres", introspectDBURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	inspector := introspect.NewInspector(db.DB, "postgres")

	var schema *introspect.DatabaseSchema

	if introspectTable != "" {
		table, err := inspector.GetTable(ctx, introspectSchema, introspectTable)
		if err != nil {
			return fmt.Errorf("failed to inspect table: %w", err)
		}

		schema = &introspect.DatabaseSchema{
			Name:   introspectDBURL,
			Tables: map[string]*introspect.TableSchema{table.Name: table},
			Metadata: introspect.DatabaseMetadata{
				InspectedAt: time.Now(),
				TableCount:  1,
			},
		}
	} else {
		schema, err = inspector.GetSchema(ctx)
		if err != nil {
			return fmt.Errorf("failed to inspect database: %w", err)
		}
	}

	outputDir := introspectOutput
	if outputDir == "" {
		outputDir = filepath.Join("generated", introspectPackage)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	fmt.Printf("Generating models from database schema...\n")
	generator := introspect.NewStructGenerator(schema, introspectPackage)
	modelsContent, err := generator.GenerateStructs()
	if err != nil {
		return fmt.Errorf("failed to generate structs: %w", err)
	}

	modelsPath := filepath.Join(outputDir, "models.go")
	if err := os.WriteFile(modelsPath, []byte(modelsContent), 0644); err != nil {
		return fmt.Errorf("failed to write models file: %w", err)
	}
	fmt.Printf("  ✓ Generated models.go\n")

	fmt.Printf("Generating ORM code...\n")
	ormConfig := orm_generator.GenerationConfig{
		PackageName: introspectPackage,
		OutputDir:   outputDir,
	}
	ormGen := orm_generator.NewCodeGenerator(ormConfig)

	if err := ormGen.DiscoverModels(outputDir); err != nil {
		return fmt.Errorf("failed to discover models: %w", err)
	}

	if err := ormGen.GenerateAll(); err != nil {
		return fmt.Errorf("failed to generate ORM code: %w", err)
	}

	fmt.Printf("\n✅ Successfully generated Storm ORM code in %s\n", outputDir)
	fmt.Printf("\nGenerated files:\n")
	fmt.Printf("  - models.go          (struct definitions)\n")
	fmt.Printf("  - columns.go         (type-safe column constants)\n")
	fmt.Printf("  - ststorm.go           (main ORM entry point)\n")
	fmt.Printf("  - *_metadata.go      (model metadata)\n")
	fmt.Printf("  - *_repository.go    (repository implementations with query methods)\n")

	fmt.Printf("\nUsage example:\n")
	fmt.Printf("  import \"%s\"\n", introspectPackage)
	fmt.Printf("  \n")
	fmt.Printf("  storm := %s.NewStorm(db)\n", introspectPackage)
	fmt.Printf("  users, err := ststorm.Users.Query().Find()\n")

	if introspectFormat != "orm" && introspectFormat != "" {
		fmt.Printf("\nGenerating additional %s export...\n", introspectFormat)

		var format introspect.ExportFormat
		switch introspectFormat {
		case "json":
			format = introspect.ExportFormatJSON
		case "yaml":
			format = introspect.ExportFormatYAML
		case "markdown", "md":
			format = introspect.ExportFormatMarkdown
		case "sql":
			format = introspect.ExportFormatSQL
		case "dot", "graphviz":
			format = introspect.ExportFormatDOT
		default:
			fmt.Printf("Warning: unsupported format %s, skipping additional export\n", introspectFormat)
			return nil
		}

		output, err := inspector.ExportSchema(schema, format)
		if err != nil {
			fmt.Printf("Warning: failed to export %s format: %v\n", introspectFormat, err)
		} else {
			additionalPath := filepath.Join(outputDir, fmt.Sprintf("schema.%s", introspectFormat))
			if err := os.WriteFile(additionalPath, output, 0644); err != nil {
				fmt.Printf("Warning: failed to write %s file: %v\n", introspectFormat, err)
			} else {
				fmt.Printf("  ✓ Generated schema.%s\n", introspectFormat)
			}
		}
	}

	return nil
}
