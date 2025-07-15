package cli

import (
	"context"
	"fmt"

	"github.com/eleven-am/storm/pkg/storm"
	"github.com/spf13/cobra"
)

var (
	ormPackage     string
	ormOutput      string
	ormIncludeHooks bool
	ormIncludeTests bool
	ormIncludeMocks bool
)

var ormCmd = &cobra.Command{
	Use:   "orm",
	Short: "Generate ORM code from models",
	Long: `Generate ORM code including repositories, queries, and utilities from Go model structs.
	
This command analyzes your Go struct definitions and generates:
- Repository interfaces and implementations
- Query builders and constants
- Lifecycle hooks (optional)
- Test files (optional)
- Mock implementations (optional)`,
	RunE: runORM,
}

func init() {
	ormCmd.Flags().StringVar(&ormPackage, "package", "", "Path to package containing models")
	ormCmd.Flags().StringVar(&ormOutput, "output", "", "Output directory for generated code (default: same as package)")
	ormCmd.Flags().BoolVar(&ormIncludeHooks, "hooks", false, "Generate lifecycle hooks")
	ormCmd.Flags().BoolVar(&ormIncludeTests, "tests", false, "Generate test files")
	ormCmd.Flags().BoolVar(&ormIncludeMocks, "mocks", false, "Generate mock implementations")
}

func runORM(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Apply config file values as defaults
	if stormConfig != nil {
		// Use config values if flags weren't specified
		if ormPackage == "" && stormConfig.Models.Package != "" {
			ormPackage = stormConfig.Models.Package
		}
		// Use ORM settings from config if not overridden
		if !cmd.Flags().Changed("hooks") && stormConfig.ORM.GenerateHooks {
			ormIncludeHooks = stormConfig.ORM.GenerateHooks
		}
		if !cmd.Flags().Changed("tests") && stormConfig.ORM.GenerateTests {
			ormIncludeTests = stormConfig.ORM.GenerateTests
		}
		if !cmd.Flags().Changed("mocks") && stormConfig.ORM.GenerateMocks {
			ormIncludeMocks = stormConfig.ORM.GenerateMocks
		}
	}

	// Set final defaults if still empty
	if ormPackage == "" {
		ormPackage = "./models"
	}
	if ormOutput == "" {
		ormOutput = ormPackage
	}

	if verbose {
		cmd.Printf("Models package: %s\n", ormPackage)
		cmd.Printf("Output directory: %s\n", ormOutput)
		cmd.Printf("Generate hooks: %v\n", ormIncludeHooks)
		cmd.Printf("Generate tests: %v\n", ormIncludeTests)
		cmd.Printf("Generate mocks: %v\n", ormIncludeMocks)
	}

	// Create Storm client (no database connection needed for ORM generation)
	config := storm.NewConfig()
	config.ModelsPackage = ormPackage
	config.Debug = debug
	config.DatabaseURL = "postgres://localhost/dummy" // Dummy URL for ORM generation

	stormClient, err := storm.NewWithConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create Storm client: %w", err)
	}
	defer stormClient.Close()

	fmt.Printf("Generating ORM code from models in %s\n", ormPackage)

	// Generate ORM code
	opts := storm.GenerateOptions{
		PackagePath:  ormPackage,
		OutputDir:    ormOutput,
		IncludeHooks: ormIncludeHooks,
		IncludeTests: ormIncludeTests,
		IncludeMocks: ormIncludeMocks,
	}

	if err := stormClient.Generate(ctx, opts); err != nil {
		return fmt.Errorf("failed to generate ORM code: %w", err)
	}

	fmt.Printf("ORM code generated successfully in %s\n", ormOutput)
	return nil
}