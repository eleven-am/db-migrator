package storm

import (
	"github.com/eleven-am/storm/pkg/storm"
	"github.com/jmoiron/sqlx"
)

// BuildMigrator creates a new migrator implementation
func BuildMigrator(db *sqlx.DB, config *storm.Config, logger storm.Logger) storm.Migrator {
	return NewMigrator(db, config, logger)
}

// BuildORM creates a new ORM implementation
func BuildORM(config *storm.Config, logger storm.Logger) storm.ORMGenerator {
	return NewORM(config, logger)
}

// BuildSchemaInspector creates a new schema inspector implementation
func BuildSchemaInspector(db *sqlx.DB, config *storm.Config, logger storm.Logger) storm.SchemaInspector {
	return NewSchemaInspector(db, config, logger)
}
