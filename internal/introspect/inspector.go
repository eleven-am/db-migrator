package introspect

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Inspector provides methods to inspect database schema
type Inspector struct {
	db     *sql.DB
	driver string
}

func NewInspector(db *sql.DB, driver string) *Inspector {
	return &Inspector{
		db:     db,
		driver: driver,
	}
}

func (i *Inspector) GetSchema(ctx context.Context) (*DatabaseSchema, error) {
	switch i.driver {
	case "postgres":
		return i.getPostgreSQLSchema(ctx)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", i.driver)
	}
}

func (i *Inspector) GetTable(ctx context.Context, schemaName, tableName string) (*TableSchema, error) {
	switch i.driver {
	case "postgres":
		return i.getPostgreSQLTable(ctx, schemaName, tableName)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", i.driver)
	}
}

func (i *Inspector) GetTables(ctx context.Context) ([]*TableSchema, error) {
	switch i.driver {
	case "postgres":
		return i.getPostgreSQLTables(ctx)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", i.driver)
	}
}

func (i *Inspector) GetDatabaseMetadata(ctx context.Context) (*DatabaseMetadata, error) {
	switch i.driver {
	case "postgres":
		return i.getPostgreSQLMetadata(ctx)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", i.driver)
	}
}

func (i *Inspector) GetEnums(ctx context.Context) (map[string]*EnumSchema, error) {
	switch i.driver {
	case "postgres":
		return i.getPostgreSQLEnums(ctx)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", i.driver)
	}
}

func (i *Inspector) GetFunctions(ctx context.Context) (map[string]*FunctionSchema, error) {
	switch i.driver {
	case "postgres":
		return i.getPostgreSQLFunctions(ctx)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", i.driver)
	}
}

func (i *Inspector) GetSequences(ctx context.Context) (map[string]*SequenceSchema, error) {
	switch i.driver {
	case "postgres":
		return i.getPostgreSQLSequences(ctx)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", i.driver)
	}
}

func (i *Inspector) GetViews(ctx context.Context) (map[string]*ViewSchema, error) {
	switch i.driver {
	case "postgres":
		return i.getPostgreSQLViews(ctx)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", i.driver)
	}
}

func (i *Inspector) GetTableStatistics(ctx context.Context, schemaName, tableName string) (*TableStatistics, error) {
	switch i.driver {
	case "postgres":
		return i.getPostgreSQLTableStatistics(ctx, schemaName, tableName)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", i.driver)
	}
}

// TableStatistics contains statistical information about a table
type TableStatistics struct {
	TableName      string
	RowCount       int64
	TotalSizeBytes int64
	DataSizeBytes  int64
	IndexSizeBytes int64
	ToastSizeBytes int64
	LastVacuum     *time.Time
	LastAutoVacuum *time.Time
	LastAnalyze    *time.Time
	DeadTuples     int64
	LiveTuples     int64
}
