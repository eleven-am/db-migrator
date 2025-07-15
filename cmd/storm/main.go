package main

import (
	"fmt"
	"os"

	"github.com/eleven-am/storm/internal/cli"
	stormInternal "github.com/eleven-am/storm/internal/storm"
	"github.com/eleven-am/storm/pkg/storm"
)

func main() {
	if err := Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func Execute() error {
	initStormFactories()

	cmd := cli.NewRootCommand()
	return cmd.Execute()
}

func initStormFactories() {
	storm.MigratorFactory = stormInternal.BuildMigrator
	storm.ORMFactory = stormInternal.BuildORM
	storm.SchemaInspectorFactory = stormInternal.BuildSchemaInspector
}
