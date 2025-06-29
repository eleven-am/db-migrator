package generator

import (
	"fmt"
)

// DependencySorter sorts tables based on their foreign key dependencies
type DependencySorter struct {
	tables map[string]SchemaTable
}

// NewDependencySorter creates a new dependency sorter
func NewDependencySorter(schema *DatabaseSchema) *DependencySorter {
	return &DependencySorter{
		tables: schema.Tables,
	}
}

// SortTables returns tables sorted by dependencies (tables with no dependencies first)
func (ds *DependencySorter) SortTables() ([]SchemaTable, error) {
	dependencies := make(map[string][]string)
	for tableName, table := range ds.tables {
		dependencies[tableName] = ds.getTableDependencies(table)
	}

	sorted := make([]SchemaTable, 0, len(ds.tables))
	visited := make(map[string]bool)
	visiting := make(map[string]bool)

	var visit func(string) error
	visit = func(tableName string) error {
		if visited[tableName] {
			return nil
		}
		if visiting[tableName] {
			return fmt.Errorf("circular dependency detected involving table %s", tableName)
		}

		visiting[tableName] = true

		for _, dep := range dependencies[tableName] {
			if dep != tableName {
				if err := visit(dep); err != nil {
					return err
				}
			}
		}

		visiting[tableName] = false
		visited[tableName] = true
		sorted = append(sorted, ds.tables[tableName])

		return nil
	}

	for tableName := range ds.tables {
		if err := visit(tableName); err != nil {
			return nil, err
		}
	}

	return sorted, nil
}

// getTableDependencies returns all tables that this table depends on
func (ds *DependencySorter) getTableDependencies(table SchemaTable) []string {
	deps := make(map[string]bool)

	for _, col := range table.Columns {
		if col.ForeignKey != nil {
			deps[col.ForeignKey.ReferencedTable] = true
		}
	}

	result := make([]string, 0, len(deps))
	for dep := range deps {
		result = append(result, dep)
	}

	return result
}
