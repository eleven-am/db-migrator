package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/eleven-am/db-migrator/internal/introspect"

	"github.com/spf13/cobra"
)

var (
	tableName  string
	jsonOutput bool
)

var introspectCmd = &cobra.Command{
	Use:   "introspect",
	Short: "Introspect database schema",
	Long:  `Display current database schema information`,
	RunE:  runIntrospect,
}

func init() {
	introspectCmd.Flags().StringVar(&dbURL, "db", "", "Database connection URL (required)")
	introspectCmd.Flags().StringVar(&tableName, "table", "", "Specific table to introspect")
	introspectCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	introspectCmd.MarkFlagRequired("db")
}

func runIntrospect(cmd *cobra.Command, args []string) error {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	introspector := introspect.NewPostgreSQLIntrospector(db)

	if tableName != "" {
		table, err := introspector.GetTable(tableName)
		if err != nil {
			return fmt.Errorf("failed to introspect table %s: %w", tableName, err)
		}

		if jsonOutput {
			data, err := json.MarshalIndent(table, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal table: %w", err)
			}
			fmt.Println(string(data))
		} else {
			printTable(*table)
		}
	} else {
		tables, err := introspector.GetTables()
		if err != nil {
			return fmt.Errorf("failed to introspect database: %w", err)
		}

		if jsonOutput {
			data, err := json.MarshalIndent(tables, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal tables: %w", err)
			}
			fmt.Println(string(data))
		} else {
			fmt.Printf("Found %d tables:\n\n", len(tables))
			for _, table := range tables {
				printTable(table)
				fmt.Println()
			}
		}
	}

	return nil
}

func printTable(table introspect.Table) {
	fmt.Printf("Table: %s\n", table.Name)
	fmt.Println("Columns:")

	for _, col := range table.Columns {
		fmt.Printf("  - %s %s", col.Name, col.Type)

		attrs := []string{}
		if col.IsPrimaryKey {
			attrs = append(attrs, "PRIMARY KEY")
		}
		if col.IsUnique {
			attrs = append(attrs, "UNIQUE")
		}
		if !col.IsNullable {
			attrs = append(attrs, "NOT NULL")
		}
		if col.DefaultValue != nil {
			attrs = append(attrs, fmt.Sprintf("DEFAULT %s", *col.DefaultValue))
		}
		if col.ForeignKey != nil {
			attrs = append(attrs, fmt.Sprintf("REFERENCES %s(%s)",
				col.ForeignKey.ReferencedTable,
				col.ForeignKey.ReferencedColumn))
		}

		if len(attrs) > 0 {
			fmt.Printf(" [%s]", attrs[0])
			for i := 1; i < len(attrs); i++ {
				fmt.Printf(", %s", attrs[i])
			}
		}
		fmt.Println()
	}

	if len(table.Indexes) > 0 {
		fmt.Println("Indexes:")
		for _, idx := range table.Indexes {
			fmt.Printf("  - %s", idx.Name)
			if idx.IsUnique {
				fmt.Printf(" UNIQUE")
			}
			if idx.IsPrimary {
				fmt.Printf(" PRIMARY")
			}
			fmt.Printf(" (%s)\n", idx.Columns)
		}
	}

	if len(table.Constraints) > 0 {
		fmt.Println("Constraints:")
		for _, con := range table.Constraints {
			fmt.Printf("  - %s %s", con.Name, con.Type)
			if con.Definition != "" {
				fmt.Printf(" %s", con.Definition)
			}
			if len(con.Columns) > 0 {
				fmt.Printf(" (%s)", con.Columns)
			}
			fmt.Println()
		}
	}
}
