package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/eleven-am/storm/pkg/storm"
	"github.com/spf13/cobra"
)

var (
	generatePackage string
	generateOutput  string
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate initial schema from Go structs",
	Long: `Generate initial SQL schema from Go struct definitions without requiring a database connection.
	
This is useful for creating the initial database schema when setting up a new project.`,
	RunE: runGenerate,
}

func init() {
	generateCmd.Flags().StringVar(&generatePackage, "package", "./models", "Path to package containing models")
	generateCmd.Flags().StringVar(&generateOutput, "output", "schema.sql", "Output file for schema SQL")
}

func runGenerate(cmd *cobra.Command, args []string) error {
	absPath, err := filepath.Abs(generatePackage)
	if err != nil {
		return fmt.Errorf("failed to resolve package path: %w", err)
	}

	fmt.Printf("Parsing structs from: %s\n", absPath)

	config := storm.NewConfig()
	config.ModelsPackage = absPath
	config.Debug = debug

	stormClient, err := storm.NewWithConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create Storm client: %w", err)
	}
	defer stormClient.Close()

	ctx := context.Background()

	schemaSQL, err := stormClient.Schema().ExportSQL(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate schema SQL: %w", err)
	}

	outputPath, err := filepath.Abs(generateOutput)
	if err != nil {
		return fmt.Errorf("failed to resolve output path: %w", err)
	}

	err = os.WriteFile(outputPath, []byte(schemaSQL), 0644)
	if err != nil {
		return fmt.Errorf("failed to write SQL file: %w", err)
	}

	fmt.Printf("Schema written to: %s\n", outputPath)
	return nil
}
