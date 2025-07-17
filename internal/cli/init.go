package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	initProject string
	initDriver  string
	initForce   bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new Storm configuration file",
	Long: `Creates a ststorm.yaml configuration file with default settings.
This helps you get started with Storm by creating a template configuration
that you can customize for your project.`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringVar(&initProject, "project", "", "Project name")
	initCmd.Flags().StringVar(&initDriver, "driver", "postgres", "Database driver (postgres, mysql, sqlite)")
	initCmd.Flags().BoolVar(&initForce, "force", false, "Overwrite existing configuration file")
}

func runInit(cmd *cobra.Command, args []string) error {
	configPath := "ststorm.yaml"
	if _, err := os.Stat(configPath); err == nil && !initForce {
		return fmt.Errorf("ststorm.yaml already exists. Use --force to overwrite")
	}

	if initProject == "" {
		dir, err := os.Getwd()
		if err == nil {
			initProject = filepath.Base(dir)
		} else {
			initProject = "my-project"
		}
	}

	config := &StormConfig{
		Version: "1",
		Project: initProject,
	}

	config.Database.Driver = initDriver
	config.Database.URL = fmt.Sprintf("%s://user:password@localhost:5432/dbname?sslmode=disable", initDriver)
	config.Database.MaxConnections = 25

	config.Models.Package = "./models"

	config.Migrations.Directory = "./migrations"
	config.Migrations.Table = "schema_migrations"
	config.Migrations.AutoApply = false

	config.ORM.GenerateHooks = true
	config.ORM.GenerateTests = false
	config.ORM.GenerateMocks = false

	config.Schema.StrictMode = true
	config.Schema.NamingConvention = "snake_case"

	if err := SaveStormConfig(config, configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("Created ststorm.yaml configuration file\n")
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("1. Update the database URL in ststorm.yaml\n")
	fmt.Printf("2. Adjust the models package path if needed\n")
	fmt.Printf("3. Run 'storm migrate' to generate migrations\n")
	fmt.Printf("4. Run 'storm orm' to generate ORM code\n")

	return nil
}
