package cmd

import (
	"fmt"
	"os"

	"github.com/eleven-am/storm/internal/orm-generator"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "db-migrator",
	Short: "Smart struct-driven database migration tool",
	Long: `A database migration tool that generates migrations from Go struct definitions.
	
It compares your current Go structs with the database schema and generates
safe migration files with automatic detection of renames and unsafe changes.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(verifyCmd)
	rootCmd.AddCommand(versionCmd)

	ormCLI := orm_generator.NewCLICommands()
	rootCmd.AddCommand(ormCLI.GetRootCommand())
}
