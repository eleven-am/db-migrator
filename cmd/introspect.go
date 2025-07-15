package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/eleven-am/storm/internal/introspect"
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
	Short: "Inspect database schema and export in various formats",
	Long: `Inspect database schema to analyze structure, relationships, and export documentation.
	
This command provides read-only inspection of your database schema, including:
- Tables, columns, and their properties
- Foreign keys and relationships
- Indexes and constraints
- Views, functions, and sequences
- Database metadata and statistics

Export formats supported: json, yaml, markdown, sql, dot (GraphViz), go (structs)`,
	RunE: runIntrospect,
}

func init() {
	introspectCmd.Flags().StringVarP(&introspectDBURL, "database", "d", "", "Database connection URL (required)")
	introspectCmd.Flags().StringVarP(&introspectFormat, "format", "f", "markdown", "Export format: json, yaml, markdown, sql, dot, go")
	introspectCmd.Flags().StringVarP(&introspectOutput, "output", "o", "", "Output file (default: stdout)")
	introspectCmd.Flags().StringVarP(&introspectTable, "table", "t", "", "Inspect specific table only")
	introspectCmd.Flags().StringVarP(&introspectSchema, "schema", "s", "public", "Database schema to inspect")
	introspectCmd.Flags().StringVarP(&introspectPackage, "package", "p", "models", "Package name for Go struct generation")

	introspectCmd.MarkFlagRequired("database")
}

func runIntrospect(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	db, err := sql.Open("postgres", introspectDBURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	inspector := introspect.NewInspector(db, "postgres")

	if introspectFormat == "go" {

		schema, err := inspector.GetSchema(ctx)
		if err != nil {
			return fmt.Errorf("failed to inspect database: %w", err)
		}

		generator := introspect.NewStructGenerator(schema, introspectPackage)
		output, err := generator.GenerateStructs()
		if err != nil {
			return fmt.Errorf("failed to generate structs: %w", err)
		}

		if introspectOutput != "" {
			if err := os.WriteFile(introspectOutput, []byte(output), 0644); err != nil {
				return fmt.Errorf("failed to write output file: %w", err)
			}
			fmt.Printf("Go structs generated to %s\n", introspectOutput)
		} else {
			fmt.Print(output)
		}
		return nil
	}

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
		return fmt.Errorf("unsupported format: %s", introspectFormat)
	}

	var output []byte

	if introspectTable != "" {

		table, err := inspector.GetTable(ctx, introspectSchema, introspectTable)
		if err != nil {
			return fmt.Errorf("failed to inspect table: %w", err)
		}

		schema := &introspect.DatabaseSchema{
			Name:   introspectDBURL,
			Tables: map[string]*introspect.TableSchema{table.Name: table},
			Metadata: introspect.DatabaseMetadata{
				InspectedAt: time.Now(),
				TableCount:  1,
			},
		}

		output, err = inspector.ExportSchema(schema, format)
		if err != nil {
			return fmt.Errorf("failed to export schema: %w", err)
		}
	} else {

		schema, err := inspector.GetSchema(ctx)
		if err != nil {
			return fmt.Errorf("failed to inspect database: %w", err)
		}

		output, err = inspector.ExportSchema(schema, format)
		if err != nil {
			return fmt.Errorf("failed to export schema: %w", err)
		}
	}

	if introspectOutput != "" {
		if err := os.WriteFile(introspectOutput, output, 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Printf("Schema exported to %s\n", introspectOutput)
	} else {
		fmt.Print(string(output))
	}

	return nil
}
