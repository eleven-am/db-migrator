package main

import (
	"github.com/eleven-am/storm/internal/storm"
	stormPkg "github.com/eleven-am/storm/pkg/storm"
)

func init() {
	// Register implementation factories
	stormPkg.MigratorFactory = ststorm.BuildMigrator
	stormPkg.ORMFactory = ststorm.BuildORM
	stormPkg.SchemaInspectorFactory = ststorm.BuildSchemaInspector
}

// NewStorm creates a new Storm instance with all implementations registered
func NewStorm(databaseURL string, opts ...stormPkg.Option) (*stormPkg.Storm, error) {
	return stormPkg.New(databaseURL, opts...)
}

// NewStormWithConfig creates a new Storm instance with explicit configuration
func NewStormWithConfig(config *stormPkg.Config) (*stormPkg.Storm, error) {
	return stormPkg.NewWithConfig(config)
}
