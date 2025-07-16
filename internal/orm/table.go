package orm

// Table provides table-level operations and metadata
type Table struct {
	Name        string   `json:"name"`
	PrimaryKeys []string `json:"primary_keys"`
	Schema      string   `json:"schema,omitempty"`
}

func (t Table) FullName() string {
	if t.Schema != "" {
		return t.Schema + "." + t.Name
	}
	return t.Name
}

func (t Table) HasPrimaryKey(column string) bool {
	for _, pk := range t.PrimaryKeys {
		if pk == column {
			return true
		}
	}
	return false
}

func (t Table) IsCompositePrimaryKey() bool {
	return len(t.PrimaryKeys) > 1
}

func (t Table) GetPrimaryKeyColumns() []string {
	return t.PrimaryKeys
}
