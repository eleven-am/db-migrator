package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/eleven-am/storm/pkg/storm-orm"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Example usage of the new Action system

// Define column references (these would typically be generated)
var UserColumns = struct {
	ID        orm.Column[int]
	Name      orm.StringColumn
	Email     orm.StringColumn
	Age       orm.NumericColumn[int]
	IsActive  orm.BoolColumn
	Tags      orm.ArrayColumn[string]
	Metadata  orm.JSONBColumn
	UpdatedAt orm.TimeColumn
}{
	ID:        orm.Column[int]{Name: "id", Table: "users"},
	Name:      orm.StringColumn{Column: orm.Column[string]{Name: "name", Table: "users"}},
	Email:     orm.StringColumn{Column: orm.Column[string]{Name: "email", Table: "users"}},
	Age:       orm.NumericColumn[int]{ComparableColumn: orm.ComparableColumn[int]{Column: orm.Column[int]{Name: "age", Table: "users"}}},
	IsActive:  orm.BoolColumn{Column: orm.Column[bool]{Name: "is_active", Table: "users"}},
	Tags:      orm.ArrayColumn[string]{Column: orm.Column[[]string]{Name: "tags", Table: "users"}},
	Metadata:  orm.JSONBColumn{Column: orm.Column[interface{}]{Name: "metadata", Table: "users"}},
	UpdatedAt: orm.TimeColumn{ComparableColumn: orm.ComparableColumn[time.Time]{Column: orm.Column[time.Time]{Name: "updated_at", Table: "users"}}},
}

func exampleUsage() {
	// Pretend we have a database connection and repository
	var db *sqlx.DB                 // initialized elsewhere
	var users *orm.Repository[User] // initialized elsewhere

	ctx := context.Background()

	// Type-safe Action-based updates
	rowsUpdated, err := users.Query(ctx).
		Where(UserColumns.IsActive.IsTrue()).
		Update(
			UserColumns.Name.Set("Updated Name"),
			UserColumns.UpdatedAt.SetNow(),
		)

	// More complex example with various action types
	rowsUpdated, err = users.Query(ctx).
		Where(UserColumns.Age.Gte(18)).
		Update(
			UserColumns.Name.Upper(),                            // Convert name to uppercase
			UserColumns.Age.Increment(1),                        // Increment age by 1
			UserColumns.Tags.Append("verified"),                 // Add "verified" to tags array
			UserColumns.Metadata.SetPath("last_updated", "now"), // Set JSONB path
			UserColumns.UpdatedAt.SetNow(),                      // Set timestamp to now
		)

	// Even more advanced actions
	rowsUpdated, err = users.Query(ctx).
		Where(UserColumns.Email.Like("%@example.com")).
		Update(
			UserColumns.Email.Lower(),        // Normalize email to lowercase
			UserColumns.Name.Prepend("Mr. "), // Add prefix to name
			UserColumns.Tags.Remove("temp"),  // Remove temporary tag
			UserColumns.Metadata.Merge(map[string]interface{}{ // Merge JSON data
				"source":  "migration",
				"version": "2.0",
			}),
		)

	fmt.Printf("Updated %d rows\n", rowsUpdated)
	if err != nil {
		log.Printf("Error: %v", err)
	}
}

// Benefits of the Action system:
// 1. Type safety - compile-time checking of column types and operations
// 2. IDE autocomplete - better developer experience
// 3. Cleaner, more readable code
// 4. Support for complex database operations (arrays, JSON, functions)
// 5. Prevents common mistakes like typos in column names
// 6. Supports advanced PostgreSQL features naturally
