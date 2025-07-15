package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/eleven-am/storm/pkg/storm"
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

	// Create Storm client
	config := storm.NewConfig()
	config.DatabaseURL = introspectDBURL
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

	// Handle Go struct generation
	if introspectFormat == "go" {
		output, err := stormClient.Schema().ExportGo(ctx)
		if err != nil {
			return fmt.Errorf("failed to generate Go structs: %w", err)
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

	// Handle SQL export
	if introspectFormat == "sql" {
		output, err := stormClient.Schema().ExportSQL(ctx)
		if err != nil {
			return fmt.Errorf("failed to export SQL: %w", err)
		}

		if introspectOutput != "" {
			if err := os.WriteFile(introspectOutput, []byte(output), 0644); err != nil {
				return fmt.Errorf("failed to write output file: %w", err)
			}
			fmt.Printf("SQL schema exported to %s\n", introspectOutput)
		} else {
			fmt.Print(output)
		}
		return nil
	}

	// For other formats, get schema and inspect
	schema, err := stormClient.Introspect(ctx)
	if err != nil {
		return fmt.Errorf("failed to introspect database: %w", err)
	}

	// Simple markdown output for now
	if introspectFormat == "markdown" || introspectFormat == "md" {
		output := generateMarkdownOutput(schema)
		
		if introspectOutput != "" {
			if err := os.WriteFile(introspectOutput, []byte(output), 0644); err != nil {
				return fmt.Errorf("failed to write output file: %w", err)
			}
			fmt.Printf("Schema exported to %s\n", introspectOutput)
		} else {
			fmt.Print(output)
		}
		return nil
	}

	return fmt.Errorf("unsupported format: %s", introspectFormat)
}

func generateMarkdownOutput(schema *storm.Schema) string {
	output := "# Database Schema\n\n"
	output += fmt.Sprintf("Generated at: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	
	output += fmt.Sprintf("## Tables (%d)\n\n", len(schema.Tables))
	
	for tableName, table := range schema.Tables {
		output += fmt.Sprintf("### %s\n\n", tableName)
		output += "| Column | Type | Nullable | Default |\n"
		output += "|--------|------|----------|----------|\n"
		
		for columnName, column := range table.Columns {
			nullable := "NO"
			if column.Nullable {
				nullable = "YES"
			}
			output += fmt.Sprintf("| %s | %s | %s | %s |\n", 
				columnName, column.Type, nullable, column.Default)
		}
		
		output += "\n"
		
		// Add primary key info
		if table.PrimaryKey != nil {
			output += fmt.Sprintf("**Primary Key**: %s\n\n", table.PrimaryKey.Name)
		}
		
		// Add foreign keys
		if len(table.ForeignKeys) > 0 {
			output += "**Foreign Keys**:\n\n"
			for _, fk := range table.ForeignKeys {
				output += fmt.Sprintf("- %s → %s\n", fk.Name, fk.ForeignTable)
			}
			output += "\n"
		}
		
		// Add indexes
		if len(table.Indexes) > 0 {
			output += "**Indexes**:\n\n"
			for _, idx := range table.Indexes {
				unique := ""
				if idx.Unique {
					unique = " (UNIQUE)"
				}
				output += fmt.Sprintf("- %s%s\n", idx.Name, unique)
			}
			output += "\n"
		}
	}
	
	return output
}