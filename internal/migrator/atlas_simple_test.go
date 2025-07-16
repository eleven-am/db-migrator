package migrator

import (
	"testing"

	"ariga.io/atlas/sql/schema"
)

func TestNewSimplifiedAtlasMigrator(t *testing.T) {
	config := &DBConfig{
		URL: "postgres://test:test@localhost:5432/testdb",
	}

	migrator := NewSimplifiedAtlasMigrator(config)

	if migrator == nil {
		t.Fatal("Expected migrator to be created")
	}

	if migrator.config != config {
		t.Error("Expected config to be set")
	}

	if migrator.tempDBManager == nil {
		t.Error("Expected tempDBManager to be initialized")
	}
}

func TestIsDestructiveChange(t *testing.T) {
	tests := []struct {
		name     string
		change   schema.Change
		expected bool
	}{
		{
			name:     "DropTable is destructive",
			change:   &schema.DropTable{T: &schema.Table{Name: "users"}},
			expected: true,
		},
		{
			name:     "DropColumn is destructive",
			change:   &schema.DropColumn{C: &schema.Column{Name: "email"}},
			expected: true,
		},
		{
			name:     "DropIndex is destructive",
			change:   &schema.DropIndex{I: &schema.Index{Name: "idx_users_email"}},
			expected: true,
		},
		{
			name:     "DropForeignKey is destructive",
			change:   &schema.DropForeignKey{F: &schema.ForeignKey{Symbol: "fk_test"}},
			expected: true,
		},
		{
			name:     "AddTable is not destructive",
			change:   &schema.AddTable{T: &schema.Table{Name: "posts"}},
			expected: false,
		},
		{
			name:     "AddColumn is not destructive",
			change:   &schema.AddColumn{C: &schema.Column{Name: "title"}},
			expected: false,
		},
		{
			name:     "AddIndex is not destructive",
			change:   &schema.AddIndex{I: &schema.Index{Name: "idx_posts_title"}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDestructiveChange(tt.change)
			if result != tt.expected {
				t.Errorf("IsDestructiveChange() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestDescribeChange(t *testing.T) {
	tests := []struct {
		name     string
		change   schema.Change
		expected string
	}{
		{
			name:     "DropTable",
			change:   &schema.DropTable{T: &schema.Table{Name: "users"}},
			expected: "Drop table users",
		},
		{
			name:     "DropColumn",
			change:   &schema.DropColumn{C: &schema.Column{Name: "email"}},
			expected: "Drop column email",
		},
		{
			name:     "DropIndex",
			change:   &schema.DropIndex{I: &schema.Index{Name: "idx_users_email"}},
			expected: "Drop index idx_users_email",
		},
		{
			name:     "DropForeignKey",
			change:   &schema.DropForeignKey{F: &schema.ForeignKey{Symbol: "fk_test"}},
			expected: "Drop foreign key fk_test",
		},
		{
			name:     "AddTable",
			change:   &schema.AddTable{T: &schema.Table{Name: "posts"}},
			expected: "Create table posts",
		},
		{
			name:     "ModifyTable",
			change:   &schema.ModifyTable{T: &schema.Table{Name: "users"}},
			expected: "Modify table users (0 changes)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DescribeChange(tt.change)
			if result != tt.expected {
				t.Errorf("DescribeChange() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestCountDestructiveChanges(t *testing.T) {
	tests := []struct {
		name     string
		changes  []schema.Change
		expected int
	}{
		{
			name:     "No changes",
			changes:  []schema.Change{},
			expected: 0,
		},
		{
			name: "All destructive",
			changes: []schema.Change{
				&schema.DropTable{T: &schema.Table{Name: "users"}},
				&schema.DropColumn{C: &schema.Column{Name: "email"}},
				&schema.DropIndex{I: &schema.Index{Name: "idx_users_email"}},
			},
			expected: 3,
		},
		{
			name: "Mixed changes",
			changes: []schema.Change{
				&schema.AddTable{T: &schema.Table{Name: "posts"}},
				&schema.DropTable{T: &schema.Table{Name: "users"}},
				&schema.AddColumn{C: &schema.Column{Name: "title"}},
				&schema.DropColumn{C: &schema.Column{Name: "email"}},
				&schema.AddIndex{I: &schema.Index{Name: "idx_posts_title"}},
			},
			expected: 2,
		},
		{
			name: "No destructive changes",
			changes: []schema.Change{
				&schema.AddTable{T: &schema.Table{Name: "posts"}},
				&schema.AddColumn{C: &schema.Column{Name: "title"}},
				&schema.AddIndex{I: &schema.Index{Name: "idx_posts_title"}},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := CountDestructiveChanges(tt.changes)
			if result != tt.expected {
				t.Errorf("CountDestructiveChanges() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// Since the migrate.Driver interface is complex, we'll focus on testing the utility functions
// that don't require complex mocking. The main GenerateAtlasSQL function would require
// a full Atlas driver implementation which is beyond the scope of unit tests.

func TestIsDestructiveChange_ModifyTable(t *testing.T) {
	t.Run("ModifyTable with destructive changes", func(t *testing.T) {
		change := &schema.ModifyTable{
			T: &schema.Table{Name: "users"},
			Changes: []schema.Change{
				&schema.DropColumn{C: &schema.Column{Name: "email"}},
				&schema.AddColumn{C: &schema.Column{Name: "phone"}},
			},
		}

		result := IsDestructiveChange(change)
		if !result {
			t.Error("Expected ModifyTable with destructive changes to be destructive")
		}
	})

	t.Run("ModifyTable with non-destructive changes", func(t *testing.T) {
		change := &schema.ModifyTable{
			T: &schema.Table{Name: "users"},
			Changes: []schema.Change{
				&schema.AddColumn{C: &schema.Column{Name: "phone"}},
				&schema.AddIndex{I: &schema.Index{Name: "idx_phone"}},
			},
		}

		result := IsDestructiveChange(change)
		if result {
			t.Error("Expected ModifyTable with non-destructive changes to not be destructive")
		}
	})
}

func TestDescribeChange_AdditionalTypes(t *testing.T) {
	tests := []struct {
		name     string
		change   schema.Change
		expected string
	}{
		{
			name:     "AddColumn",
			change:   &schema.AddColumn{C: &schema.Column{Name: "phone"}},
			expected: "Add column phone",
		},
		{
			name:     "ModifyColumn",
			change:   &schema.ModifyColumn{To: &schema.Column{Name: "updated_email"}},
			expected: "Modify column updated_email",
		},
		{
			name:     "AddForeignKey",
			change:   &schema.AddForeignKey{F: &schema.ForeignKey{Symbol: "fk_user_profile"}},
			expected: "Add foreign key fk_user_profile",
		},
		{
			name:     "Unknown change type",
			change:   &schema.AddCheck{},
			expected: "Change type *schema.AddCheck",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DescribeChange(tt.change)
			if result != tt.expected {
				t.Errorf("DescribeChange() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
