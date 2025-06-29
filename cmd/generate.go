package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	generator2 "github.com/eleven-am/db-migrator/internal/generator"
	"github.com/eleven-am/db-migrator/internal/parser"

	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate initial schema from Go structs",
	Long: `Generate initial SQL schema from Go struct definitions without requiring a database connection.
	
This is useful for creating the initial database schema when setting up a new project.`,
	RunE: runGenerate,
}

var (
	generatePackage string
	generateOutput  string
)

func init() {
	generateCmd.Flags().StringVar(&generatePackage, "package", "./internal/db", "Path to package containing models")
	generateCmd.Flags().StringVar(&generateOutput, "output", "schema.sql", "Output file for schema SQL")
}

func runGenerate(cmd *cobra.Command, args []string) error {
	absPath, err := filepath.Abs(generatePackage)
	if err != nil {
		return fmt.Errorf("failed to resolve package path: %w", err)
	}

	fmt.Printf("Parsing structs from: %s\n", absPath)
	sp := parser.NewStructParser()
	structs, err := sp.ParseDirectory(absPath)
	if err != nil {
		return fmt.Errorf("failed to parse structs: %w", err)
	}

	if len(structs) == 0 {
		return fmt.Errorf("no structs found in %s", absPath)
	}

	fmt.Printf("Found %d structs with dbdef tags\n", len(structs))

	sg := generator2.NewSchemaGenerator()
	schema, err := sg.GenerateSchema(structs)
	if err != nil {
		return fmt.Errorf("failed to generate schema: %w", err)
	}

	sqlGen := generator2.NewSQLGenerator()
	sql := sqlGen.GenerateSchema(schema)

	outputPath, err := filepath.Abs(generateOutput)
	if err != nil {
		return fmt.Errorf("failed to resolve output path: %w", err)
	}

	err = os.WriteFile(outputPath, []byte(sql), 0644)
	if err != nil {
		return fmt.Errorf("failed to write SQL file: %w", err)
	}

	fmt.Printf("Schema written to: %s\n", outputPath)
	fmt.Printf("Generated schema for %d tables\n", len(schema.Tables))

	return nil
}
