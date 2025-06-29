package testing

import (
	"github.com/eleven-am/db-migrator/internal/parser"
	"strings"
)

// CreateTestModel creates a test model for testing
func CreateTestModel(name string, fields ...parser.FieldDefinition) parser.TableDefinition {
	return parser.TableDefinition{
		StructName: name,
		TableName:  ToSnakeCase(name) + "s",
		Fields:     fields,
		TableLevel: make(map[string]string),
	}
}

// CreateTestField creates a test field
func CreateTestField(name, dbType string, tags ...string) parser.FieldDefinition {
	field := parser.FieldDefinition{
		Name:   name,
		DBName: ToSnakeCase(name),
		Type:   dbType,
		DBDef:  make(map[string]string),
	}

	// Set type in dbdef
	field.DBDef["type"] = dbType

	// Parse tags
	for _, tag := range tags {
		// Add tags as flags (empty value) in DBDef map
		field.DBDef[tag] = ""
	}

	return field
}

// Common test fixtures
var (
	// Simple User model
	UserModel = CreateTestModel("User",
		CreateTestField("ID", "uuid", "primary_key"),
		CreateTestField("Email", "varchar(255)", "not_null", "unique"),
		CreateTestField("Name", "varchar(255)", "not_null"),
		CreateTestField("CreatedAt", "timestamp", "not_null"),
	)

	// Team model with foreign key
	TeamModel = CreateTestModel("Team",
		CreateTestField("ID", "uuid", "primary_key"),
		CreateTestField("Name", "varchar(255)", "not_null"),
		CreateTestField("OwnerID", "uuid", "not_null"),
	)

	// Model with various column types
	ProductModel = CreateTestModel("Product",
		CreateTestField("ID", "serial", "primary_key", "auto_increment"),
		CreateTestField("Name", "varchar(255)", "not_null"),
		CreateTestField("Price", "decimal(10,2)", "not_null"),
		CreateTestField("InStock", "boolean"),
		CreateTestField("Description", "text"),
		CreateTestField("Metadata", "jsonb"),
	)
)

// ToSnakeCase converts CamelCase to snake_case
func ToSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}
