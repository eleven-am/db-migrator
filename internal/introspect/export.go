package introspect

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ExportSchema exports the database schema in the specified format
func (i *Inspector) ExportSchema(schema *DatabaseSchema, format ExportFormat) ([]byte, error) {
	switch format {
	case ExportFormatJSON:
		return exportJSON(schema)
	case ExportFormatYAML:
		return exportYAML(schema)
	case ExportFormatMarkdown:
		return exportMarkdown(schema)
	case ExportFormatSQL:
		return exportSQL(schema)
	case ExportFormatDOT:
		return exportDOT(schema)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// exportJSON exports schema as JSON
func exportJSON(schema *DatabaseSchema) ([]byte, error) {
	return json.MarshalIndent(schema, "", "  ")
}

// exportYAML exports schema as YAML
func exportYAML(schema *DatabaseSchema) ([]byte, error) {
	return yaml.Marshal(schema)
}

// exportMarkdown exports schema as Markdown documentation
func exportMarkdown(schema *DatabaseSchema) ([]byte, error) {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# Database Schema: %s\n\n", schema.Name))
	b.WriteString(fmt.Sprintf("Generated on: %s\n\n", schema.Metadata.InspectedAt.Format("2006-01-02 15:04:05")))

	b.WriteString("## Database Information\n\n")
	b.WriteString(fmt.Sprintf("- **Version**: %s\n", schema.Metadata.Version))
	b.WriteString(fmt.Sprintf("- **Encoding**: %s\n", schema.Metadata.Encoding))
	b.WriteString(fmt.Sprintf("- **Collation**: %s\n", schema.Metadata.Collation))
	b.WriteString(fmt.Sprintf("- **Size**: %.2f MB\n", float64(schema.Metadata.Size)/(1024*1024)))
	b.WriteString(fmt.Sprintf("- **Tables**: %d\n", schema.Metadata.TableCount))
	b.WriteString(fmt.Sprintf("- **Indexes**: %d\n", schema.Metadata.IndexCount))
	b.WriteString(fmt.Sprintf("- **Constraints**: %d\n\n", schema.Metadata.ConstraintCount))

	b.WriteString("## Tables\n\n")
	for _, table := range sortedTables(schema.Tables) {
		b.WriteString(fmt.Sprintf("### %s\n\n", table.Name))
		if table.Comment != "" {
			b.WriteString(fmt.Sprintf("_%s_\n\n", table.Comment))
		}

		b.WriteString("#### Columns\n\n")
		b.WriteString("| Name | Type | Nullable | Default | Description |\n")
		b.WriteString("|------|------|----------|---------|-------------|\n")

		for _, col := range table.Columns {
			nullable := "NO"
			if col.IsNullable {
				nullable = "YES"
			}

			defaultVal := ""
			if col.DefaultValue != nil {
				defaultVal = *col.DefaultValue
			}

			b.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
				col.Name, col.DataType, nullable, defaultVal, col.Comment))
		}
		b.WriteString("\n")

		if table.PrimaryKey != nil {
			b.WriteString("#### Primary Key\n\n")
			b.WriteString(fmt.Sprintf("- **Name**: %s\n", table.PrimaryKey.Name))
			b.WriteString(fmt.Sprintf("- **Columns**: %s\n\n", strings.Join(table.PrimaryKey.Columns, ", ")))
		}

		b.WriteString("#### Foreign Keys\n\n")
		if len(table.ForeignKeys) > 0 {
			for _, fk := range table.ForeignKeys {
				b.WriteString(fmt.Sprintf("- **%s**: %s → %s.%s (%s)\n",
					fk.Name,
					strings.Join(fk.Columns, ", "),
					fk.ReferencedTable,
					strings.Join(fk.ReferencedColumns, ", "),
					fmt.Sprintf("ON DELETE %s, ON UPDATE %s", fk.OnDelete, fk.OnUpdate)))
			}
		} else {
			b.WriteString("None\n")
		}
		b.WriteString("\n")

		if len(table.Indexes) > 0 {
			b.WriteString("#### Indexes\n\n")
			for _, idx := range table.Indexes {
				unique := ""
				if idx.IsUnique {
					unique = " (UNIQUE)"
				}
				cols := make([]string, 0)
				for _, c := range idx.Columns {
					if c.Name != "" {
						cols = append(cols, c.Name)
					} else {
						cols = append(cols, c.Expression)
					}
				}
				b.WriteString(fmt.Sprintf("- **%s**%s: %s\n", idx.Name, unique, strings.Join(cols, ", ")))
			}
			b.WriteString("\n")
		}

		if len(table.Constraints) > 0 {
			b.WriteString("#### Constraints\n\n")
			for _, c := range table.Constraints {
				b.WriteString(fmt.Sprintf("- **%s** (%s): %s\n", c.Name, c.Type, c.Definition))
			}
			b.WriteString("\n")
		}
	}

	if len(schema.Enums) > 0 {
		b.WriteString("## Enum Types\n\n")
		for name, enum := range schema.Enums {

			enumName := name
			if dotIdx := strings.LastIndex(name, "."); dotIdx > 0 {
				enumName = name[dotIdx+1:]
			}
			b.WriteString(fmt.Sprintf("### %s\n\n", enumName))
			for _, val := range enum.Values {
				b.WriteString(fmt.Sprintf("- %s\n", val))
			}
			b.WriteString("\n")
		}
	}

	if len(schema.Views) > 0 {
		b.WriteString("## Views\n\n")
		for name, view := range schema.Views {
			b.WriteString(fmt.Sprintf("### %s\n\n", name))
			if view.Comment != "" {
				b.WriteString(fmt.Sprintf("_%s_\n\n", view.Comment))
			}
			b.WriteString("```sql\n")
			b.WriteString(view.Definition)
			b.WriteString("\n```\n\n")
		}
	}

	return []byte(b.String()), nil
}

// exportSQL exports schema as SQL DDL
func exportSQL(schema *DatabaseSchema) ([]byte, error) {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("-- Database: %s\n", schema.Name))
	b.WriteString(fmt.Sprintf("-- Generated on: %s\n\n", schema.Metadata.InspectedAt.Format("2006-01-02 15:04:05")))

	if len(schema.Enums) > 0 {
		b.WriteString("-- Enum Types\n")
		for name, enum := range schema.Enums {
			b.WriteString(fmt.Sprintf("CREATE TYPE %s AS ENUM (\n", name))
			for i, val := range enum.Values {
				b.WriteString(fmt.Sprintf("    '%s'", val))
				if i < len(enum.Values)-1 {
					b.WriteString(",")
				}
				b.WriteString("\n")
			}
			b.WriteString(");\n\n")
		}
	}

	for _, table := range sortedTables(schema.Tables) {
		b.WriteString(fmt.Sprintf("-- Table: %s\n", table.Name))
		b.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", table.Name))

		for i, col := range table.Columns {
			b.WriteString(fmt.Sprintf("    %s %s", col.Name, col.DataType))

			if !col.IsNullable {
				b.WriteString(" NOT NULL")
			}

			if col.DefaultValue != nil {
				b.WriteString(fmt.Sprintf(" DEFAULT %s", *col.DefaultValue))
			}

			if i < len(table.Columns)-1 || table.PrimaryKey != nil || len(table.Constraints) > 0 {
				b.WriteString(",")
			}
			b.WriteString("\n")
		}

		if table.PrimaryKey != nil {
			b.WriteString(fmt.Sprintf("    CONSTRAINT %s PRIMARY KEY (%s)",
				table.PrimaryKey.Name, strings.Join(table.PrimaryKey.Columns, ", ")))
			if len(table.Constraints) > 0 {
				b.WriteString(",")
			}
			b.WriteString("\n")
		}

		for i, c := range table.Constraints {
			if c.Type == "FOREIGN KEY" {
				continue
			}
			b.WriteString(fmt.Sprintf("    CONSTRAINT %s %s", c.Name, c.Definition))
			if i < len(table.Constraints)-1 {
				b.WriteString(",")
			}
			b.WriteString("\n")
		}

		b.WriteString(");\n\n")

		for _, fk := range table.ForeignKeys {
			b.WriteString(fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s)",
				table.Name, fk.Name, strings.Join(fk.Columns, ", "),
				fk.ReferencedTable, strings.Join(fk.ReferencedColumns, ", ")))
			if fk.OnDelete != "NO ACTION" {
				b.WriteString(fmt.Sprintf(" ON DELETE %s", fk.OnDelete))
			}
			if fk.OnUpdate != "NO ACTION" {
				b.WriteString(fmt.Sprintf(" ON UPDATE %s", fk.OnUpdate))
			}
			b.WriteString(";\n")
		}

		for _, idx := range table.Indexes {
			if idx.IsPrimary {
				continue
			}

			unique := ""
			if idx.IsUnique {
				unique = "UNIQUE "
			}

			cols := make([]string, 0)
			for _, c := range idx.Columns {
				if c.Name != "" {
					cols = append(cols, c.Name)
				} else {
					cols = append(cols, c.Expression)
				}
			}

			b.WriteString(fmt.Sprintf("CREATE %sINDEX %s ON %s (%s)",
				unique, idx.Name, table.Name, strings.Join(cols, ", ")))

			if idx.Where != "" {
				b.WriteString(fmt.Sprintf(" WHERE %s", idx.Where))
			}
			b.WriteString(";\n")
		}

		b.WriteString("\n")
	}

	return []byte(b.String()), nil
}

// exportDOT exports schema as GraphViz DOT format for visualization
func exportDOT(schema *DatabaseSchema) ([]byte, error) {
	var b strings.Builder

	b.WriteString("digraph DatabaseSchema {\n")
	b.WriteString("    rankdir=LR;\n")
	b.WriteString("    node [shape=box];\n\n")

	for _, table := range schema.Tables {
		b.WriteString(fmt.Sprintf("    %s [label=\"%s", table.Name, table.Name))
		b.WriteString("|{")

		colStrs := make([]string, 0)
		for _, col := range table.Columns {
			pk := ""
			for _, pkCol := range table.PrimaryKey.Columns {
				if col.Name == pkCol {
					pk = " (PK)"
					break
				}
			}
			colStrs = append(colStrs, fmt.Sprintf("%s: %s%s", col.Name, col.DataType, pk))
		}
		b.WriteString(strings.Join(colStrs, "\\l"))
		b.WriteString("}\" shape=record];\n")
	}
	b.WriteString("\n")

	for _, table := range schema.Tables {
		for _, fk := range table.ForeignKeys {
			b.WriteString(fmt.Sprintf("    %s -> %s [label=\"%s\"];\n",
				table.Name, fk.ReferencedTable, fk.Name))
		}
	}

	b.WriteString("}\n")

	return []byte(b.String()), nil
}

// Helper function to sort tables by name
func sortedTables(tables map[string]*TableSchema) []*TableSchema {
	var result []*TableSchema
	var names []string

	for name := range tables {
		names = append(names, name)
	}

	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			if names[i] > names[j] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}

	for _, name := range names {
		result = append(result, tables[name])
	}

	return result
}
