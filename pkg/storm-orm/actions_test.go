package orm

import (
	"testing"
	"time"
)

func TestActions(t *testing.T) {
	// Test basic column actions
	nameCol := Column[string]{Name: "name", Table: "users"}
	ageCol := NumericColumn[int]{ComparableColumn: ComparableColumn[int]{Column: Column[int]{Name: "age", Table: "users"}}}
	updatedAtCol := TimeColumn{ComparableColumn: ComparableColumn[time.Time]{Column: Column[time.Time]{Name: "updated_at", Table: "users"}}}
	tagsCol := ArrayColumn[string]{Column: Column[[]string]{Name: "tags", Table: "users"}}

	tests := []struct {
		name           string
		action         Action
		expectedColumn string
		expectedExpr   string
		hasValue       bool
	}{
		{
			name:           "Column Set",
			action:         nameCol.Set("John"),
			expectedColumn: "users.name",
			expectedExpr:   "users.name = ?",
			hasValue:       true,
		},
		{
			name:           "Column SetNull",
			action:         nameCol.SetNull(),
			expectedColumn: "users.name",
			expectedExpr:   "users.name = NULL",
			hasValue:       false,
		},
		{
			name:           "Column SetDefault",
			action:         nameCol.SetDefault(),
			expectedColumn: "users.name",
			expectedExpr:   "users.name = DEFAULT",
			hasValue:       false,
		},
		{
			name:           "NumericColumn Increment",
			action:         ageCol.Increment(1),
			expectedColumn: "users.age",
			expectedExpr:   "users.age = users.age + ?",
			hasValue:       true,
		},
		{
			name:           "NumericColumn Decrement",
			action:         ageCol.Decrement(5),
			expectedColumn: "users.age",
			expectedExpr:   "users.age = users.age - ?",
			hasValue:       true,
		},
		{
			name:           "TimeColumn SetNow",
			action:         updatedAtCol.SetNow(),
			expectedColumn: "users.updated_at",
			expectedExpr:   "users.updated_at = NOW()",
			hasValue:       false,
		},
		{
			name:           "ArrayColumn Append",
			action:         tagsCol.Append("new-tag"),
			expectedColumn: "users.tags",
			expectedExpr:   "users.tags = array_append(users.tags, ?)",
			hasValue:       true,
		},
		{
			name:           "ArrayColumn Remove",
			action:         tagsCol.Remove("old-tag"),
			expectedColumn: "users.tags",
			expectedExpr:   "users.tags = array_remove(users.tags, ?)",
			hasValue:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.action.Column() != tt.expectedColumn {
				t.Errorf("Column() = %v, expected %v", tt.action.Column(), tt.expectedColumn)
			}
			if tt.action.Expression() != tt.expectedExpr {
				t.Errorf("Expression() = %v, expected %v", tt.action.Expression(), tt.expectedExpr)
			}
			if tt.hasValue && tt.action.Value() == nil {
				t.Errorf("Expected action to have a value but got nil")
			}
			if !tt.hasValue && tt.action.Value() != nil {
				t.Errorf("Expected action to have no value but got %v", tt.action.Value())
			}
		})
	}
}

func TestStringColumnActions(t *testing.T) {
	nameCol := StringColumn{Column: Column[string]{Name: "name", Table: "users"}}

	tests := []struct {
		name          string
		action        Action
		expectedExpr  string
		expectedValue interface{}
	}{
		{
			name:          "Concat",
			action:        nameCol.Concat(" Jr."),
			expectedExpr:  "users.name = users.name || ?",
			expectedValue: " Jr.",
		},
		{
			name:          "Prepend",
			action:        nameCol.Prepend("Mr. "),
			expectedExpr:  "users.name = ? || users.name",
			expectedValue: "Mr. ",
		},
		{
			name:          "Upper",
			action:        nameCol.Upper(),
			expectedExpr:  "users.name = UPPER(users.name)",
			expectedValue: nil,
		},
		{
			name:          "Lower",
			action:        nameCol.Lower(),
			expectedExpr:  "users.name = LOWER(users.name)",
			expectedValue: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.action.Expression() != tt.expectedExpr {
				t.Errorf("Expression() = %v, expected %v", tt.action.Expression(), tt.expectedExpr)
			}
			if tt.action.Value() != tt.expectedValue {
				t.Errorf("Value() = %v, expected %v", tt.action.Value(), tt.expectedValue)
			}
		})
	}
}

func TestJSONBColumnActions(t *testing.T) {
	metaCol := JSONBColumn{Column: Column[interface{}]{Name: "metadata", Table: "users"}}

	tests := []struct {
		name         string
		action       Action
		expectedExpr string
	}{
		{
			name:         "SetPath",
			action:       metaCol.SetPath("profile.name", "John"),
			expectedExpr: "users.metadata = jsonb_set(users.metadata, ?, ?)",
		},
		{
			name:         "RemovePath",
			action:       metaCol.RemovePath("temp_field"),
			expectedExpr: "users.metadata = users.metadata - ?",
		},
		{
			name:         "Merge",
			action:       metaCol.Merge(map[string]interface{}{"new_field": "value"}),
			expectedExpr: "users.metadata = users.metadata || ?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.action.Expression() != tt.expectedExpr {
				t.Errorf("Expression() = %v, expected %v", tt.action.Expression(), tt.expectedExpr)
			}
			if tt.action.Value() == nil {
				t.Errorf("Expected action to have a value but got nil")
			}
		})
	}
}
