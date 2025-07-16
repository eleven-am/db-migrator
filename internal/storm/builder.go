package storm

import (
	"github.com/eleven-am/storm/pkg/storm"
	"github.com/jmoiron/sqlx"
)

func BuildMigrator(db *sqlx.DB, config *storm.Config, logger storm.Logger) storm.Migrator {
	return NewMigrator(db, config, logger)
}

func BuildORM(config *storm.Config, logger storm.Logger) storm.ORMGenerator {
	return NewORM(config, logger)
}

func BuildSchemaInspector(db *sqlx.DB, config *storm.Config, logger storm.Logger) storm.SchemaInspector {
	return NewSchemaInspector(db, config, logger)
}
