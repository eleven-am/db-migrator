package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestToSnakeCase(t *testing.T) {
	parser := NewStructParser()
	
	tests := []struct {
		input    string
		expected string
	}{
		{"APIKey", "api_key"},
		{"UserID", "user_id"},
		{"HTTPRequest", "http_request"},
		{"XMLParser", "xml_parser"},
		{"IOReader", "io_reader"},
		{"TeamID42", "team_id42"},
		{"OAuth2Token", "oauth2_token"},
		{"SimpleCase", "simple_case"},
		{"lowercase", "lowercase"},
		{"UPPERCASE", "uppercase"},
		{"mixedUPPERCase", "mixed_upper_case"},
		{"APIKeyID", "api_key_id"},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parser.toSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("toSnakeCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDeriveTableName(t *testing.T) {
	parser := NewStructParser()
	
	tests := []struct {
		input    string
		expected string
	}{
		{"APIKey", "api_keys"},
		{"OAuthToken", "o_auth_tokens"},
		{"User", "users"},
		{"Team", "teams"},
		{"Category", "categories"},
		{"Process", "processes"},
		{"Index", "indexes"},
		{"Person", "persons"},
		{"Policy", "policies"},
		{"Analysis", "analyses"},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parser.deriveTableName(tt.input)
			if result != tt.expected {
				t.Errorf("deriveTableName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
func TestStructParser_ParseFile(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_model.go")
	
	testCode := `
package models

type User struct {
	ID        string    ` + "`" + `db:"id" dbdef:"type:uuid;primary_key;default:gen_random_uuid()"` + "`" + `
	Email     string    ` + "`" + `db:"email" dbdef:"type:varchar(255);not_null;unique"` + "`" + `
	Name      string    ` + "`" + `db:"name" dbdef:"type:varchar(100);not_null"` + "`" + `
	IsActive  bool      ` + "`" + `db:"is_active" dbdef:"type:boolean;default:true"` + "`" + `
	CreatedAt time.Time ` + "`" + `db:"created_at" dbdef:"type:timestamp;not_null;default:now()"` + "`" + `
}

type Team struct {
	ID      string ` + "`" + `db:"id" dbdef:"type:uuid;primary_key"` + "`" + `
	Name    string ` + "`" + `db:"name" dbdef:"type:varchar(255);not_null"` + "`" + `
	OwnerID string ` + "`" + `db:"owner_id" dbdef:"type:uuid;not_null;fk:users.id"` + "`" + `
}
`
	
	if err := os.WriteFile(testFile, []byte(testCode), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	parser := NewStructParser()
	tables, err := parser.ParseFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}
	
	// Verify we found both structs
	if len(tables) != 2 {
		t.Errorf("Expected 2 tables, got %d", len(tables))
	}
	
	// Find User table
	var userTable *TableDefinition
	for _, table := range tables {
		if table.StructName == "User" {
			userTable = &table
			break
		}
	}
	
	if userTable == nil {
		t.Fatal("User table not found")
	}
	
	// Test User table
	t.Run("User table", func(t *testing.T) {
		if userTable.TableName != "users" {
			t.Errorf("Expected table name 'users', got '%s'", userTable.TableName)
		}
		
		if len(userTable.Fields) != 5 {
			t.Errorf("Expected 5 fields, got %d", len(userTable.Fields))
		}
		
		// Check ID field
		idField := findField(userTable.Fields, "ID")
		if idField == nil {
			t.Fatal("ID field not found")
		}
		
		if idField.DBName != "id" {
			t.Errorf("Expected DB name 'id', got '%s'", idField.DBName)
		}
		
		if _, hasPK := idField.DBDef["primary_key"]; !hasPK {
			t.Error("ID field should be primary key")
		}
		
		if idField.DBDef["default"] != "gen_random_uuid()" {
			t.Errorf("Expected default 'gen_random_uuid()', got '%s'", idField.DBDef["default"])
		}
	})
	
	// Find Team table
	var teamTable *TableDefinition
	for _, table := range tables {
		if table.StructName == "Team" {
			teamTable = &table
			break
		}
	}
	
	if teamTable == nil {
		t.Fatal("Team table not found")
	}
	
	// Test Team table
	t.Run("Team table", func(t *testing.T) {
		if teamTable.TableName != "teams" {
			t.Errorf("Expected table name 'teams', got '%s'", teamTable.TableName)
		}
		
		// Check OwnerID field for foreign key
		ownerField := findField(teamTable.Fields, "OwnerID")
		if ownerField == nil {
			t.Fatal("OwnerID field not found")
		}
		
		fk := ownerField.DBDef["fk"]
		if fk == "" {
			t.Error("OwnerID should have foreign key")
		}
		
		if fk != "users.id" {
			t.Errorf("Expected foreign key 'users.id', got '%s'", fk)
		}
	})
}

func TestStructParser_ParseDirectory(t *testing.T) {
	// Create a temporary package directory
	tmpDir := t.TempDir()
	
	// Create multiple test files
	file1 := filepath.Join(tmpDir, "user.go")
	file2 := filepath.Join(tmpDir, "team.go")
	
	userCode := `
package models

type User struct {
	ID string ` + "`" + `db:"id" dbdef:"type:uuid;primary_key"` + "`" + `
}
`
	
	teamCode := `
package models

type Team struct {
	ID string ` + "`" + `db:"id" dbdef:"type:uuid;primary_key"` + "`" + `
}
`
	
	if err := os.WriteFile(file1, []byte(userCode), 0644); err != nil {
		t.Fatalf("Failed to write user file: %v", err)
	}
	
	if err := os.WriteFile(file2, []byte(teamCode), 0644); err != nil {
		t.Fatalf("Failed to write team file: %v", err)
	}
	
	parser := NewStructParser()
	tables, err := parser.ParseDirectory(tmpDir)
	if err != nil {
		t.Fatalf("Failed to parse directory: %v", err)
	}
	
	if len(tables) != 2 {
		t.Errorf("Expected 2 tables, got %d", len(tables))
	}
	
	// Verify both tables were found
	names := make(map[string]bool)
	for _, table := range tables {
		names[table.StructName] = true
	}
	
	if !names["User"] {
		t.Error("User table not found")
	}
	
	if !names["Team"] {
		t.Error("Team table not found")
	}
}

func findField(fields []FieldDefinition, name string) *FieldDefinition {
	for _, f := range fields {
		if f.Name == name {
			return &f
		}
	}
	return nil
}