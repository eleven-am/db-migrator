package storm

import (
	"github.com/eleven-am/storm/pkg/storm"
	"github.com/jmoiron/sqlx"
)

func BuildMigrator(db *sqlx.DB, config *ststorm.Config, logger ststorm.Logger) ststorm.Migrator {
	return NewMigrator(db, config, logger)
}

func BuildORM(config *ststorm.Config, logger ststorm.Logger) ststorm.ORMGenerator {
	return NewORM(config, logger)
}

func BuildSchemaInspector(db *sqlx.DB, config *ststorm.Config, logger ststorm.Logger) ststorm.SchemaInspector {
	return NewSchemaInspector(db, config, logger)
}
