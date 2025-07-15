package cli

import (
	"github.com/eleven-am/storm/pkg/storm"
	"github.com/spf13/cobra"
)

// Global configuration variables
var (
	configFile  string
	stormConfig *StormConfig
	databaseURL string
	debug       bool
	verbose     bool
)

// NewRootCommand creates the root command for Storm CLI
func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "storm",
		Short: "Storm - Unified Database Toolkit",
		Long: `Storm is a unified database toolkit that combines schema management,
ORM generation, and database operations under a single, cohesive API.

Storm provides powerful tools for:
- Database migrations and schema management  
- ORM code generation from Go models
- Database schema introspection and analysis
- Modern CLI with rich output capabilities`,
		Version: storm.Version,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Load configuration file
			var err error
			stormConfig, err = LoadStormConfig(configFile)
			if err != nil {
				// Just log the error, don't fail - config file is optional
				if verbose {
					cmd.Printf("Warning: Failed to load config file: %v\n", err)
				}
			}
			
			// Apply config values if not overridden by flags
			if stormConfig != nil {
				// For database URL
				if databaseURL == "" && stormConfig.Database.URL != "" {
					databaseURL = stormConfig.Database.URL
				}
				
				// Debug mode can be set in config too
				if !debug && stormConfig.Schema.StrictMode {
					// You could add a debug field to config if needed
				}
			}
		},
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default: storm.yaml)")
	rootCmd.PersistentFlags().StringVar(&databaseURL, "url", "", "database connection URL")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug output")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "enable verbose output")

	// Add subcommands (adapted from original db-migrator)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(verifyCmd)
	rootCmd.AddCommand(introspectCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(ormCmd)

	return rootCmd
}