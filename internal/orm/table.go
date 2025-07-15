package orm

// Table provides table-level operations and metadata
type Table struct {
	Name        string   `json:"name"`
	PrimaryKeys []string `json:"primary_keys"`
	Schema      string   `json:"schema,omitempty"`
}

// FullName returns the full table name including schema if set
func (t Table) FullName() string {
	if t.Schema != "" {
		return t.Schema + "." + t.Name
	}
	return t.Name
}

// HasPrimaryKey checks if a column is a primary key
func (t Table) HasPrimaryKey(column string) bool {
	for _, pk := range t.PrimaryKeys {
		if pk == column {
			return true
		}
	}
	return false
}

// IsCompositePrimaryKey returns true if the table has composite primary keys
func (t Table) IsCompositePrimaryKey() bool {
	return len(t.PrimaryKeys) > 1
}

// GetPrimaryKeyColumns returns the primary key columns
func (t Table) GetPrimaryKeyColumns() []string {
	return t.PrimaryKeys
}
