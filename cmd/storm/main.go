package main

import (
	"fmt"
	"os"

	"github.com/eleven-am/storm/internal/cli"
	"github.com/eleven-am/storm/pkg/storm"
	stormInternal "github.com/eleven-am/storm/internal/storm"
)

func main() {
	if err := Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func Execute() error {
	// Initialize Storm factories
	initStormFactories()

	// Create and execute CLI
	cmd := cli.NewRootCommand()
	return cmd.Execute()
}

func initStormFactories() {
	// Import and register actual implementations
	// This is done here to avoid circular dependencies
	storm.MigratorFactory = stormInternal.BuildMigrator
	storm.ORMFactory = stormInternal.BuildORM
	storm.SchemaInspectorFactory = stormInternal.BuildSchemaInspector
}