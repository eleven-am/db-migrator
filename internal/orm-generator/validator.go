package orm_generator

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/eleven-am/storm/internal/parser"
)

// ModelValidationError represents a model validation error
type ModelValidationError struct {
	Type    string
	Field   string
	Message string
}

func (e ModelValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s.%s: %s", e.Type, e.Field, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// ValidationResult contains validation results
type ValidationResult struct {
	Valid  bool
	Errors []ModelValidationError
}

// ModelValidator validates ORM models
type ModelValidator struct {
	tagParser *ORMTagParser
}

// NewModelValidator creates a new model validator
func NewModelValidator() *ModelValidator {
	return &ModelValidator{
		tagParser: NewORMTagParser(),
	}
}

// ValidateModel validates a single model struct
func (v *ModelValidator) ValidateModel(modelType reflect.Type) ValidationResult {
	result := ValidationResult{Valid: true}

	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	if modelType.Kind() != reflect.Struct {
		result.Valid = false
		result.Errors = append(result.Errors, ModelValidationError{
			Type:    modelType.Name(),
			Message: "must be a struct type",
		})
		return result
	}

	// Check for primary key
	if !v.hasPrimaryKey(modelType) {
		result.Valid = false
		result.Errors = append(result.Errors, ModelValidationError{
			Type:    modelType.Name(),
			Message: "must have at least one primary key field",
		})
	}

	// Validate fields
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		fieldErrors := v.validateField(modelType.Name(), field)
		result.Errors = append(result.Errors, fieldErrors...)
		if len(fieldErrors) > 0 {
			result.Valid = false
		}
	}

	return result
}

// ValidateModels validates multiple models and their relationships
func (v *ModelValidator) ValidateModels(models map[string]reflect.Type) ValidationResult {
	result := ValidationResult{Valid: true}

	// First validate each model individually
	for _, modelType := range models {
		modelResult := v.ValidateModel(modelType)
		if !modelResult.Valid {
			result.Valid = false
			result.Errors = append(result.Errors, modelResult.Errors...)
		}
	}

	// Then validate relationships between models
	relationshipErrors := v.validateRelationships(models)
	if len(relationshipErrors) > 0 {
		result.Valid = false
		result.Errors = append(result.Errors, relationshipErrors...)
	}

	return result
}

// hasPrimaryKey checks if model has a primary key field
func (v *ModelValidator) hasPrimaryKey(modelType reflect.Type) bool {
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		dbdefTag := field.Tag.Get("dbdef")
		if dbdefTag == "" {
			continue
		}

		if strings.Contains(dbdefTag, "primary_key") {
			return true
		}
	}
	return false
}

// validateField validates a single struct field
func (v *ModelValidator) validateField(typeName string, field reflect.StructField) []ModelValidationError {
	var errors []ModelValidationError

	// Check db tag
	dbTag := field.Tag.Get("db")
	if dbTag == "" && field.Name != "ID" {
		// Allow fields without db tag for embedded structs or special cases
		if field.Type.Kind() != reflect.Struct || field.Anonymous {
			return errors
		}
	}

	// Validate dbdef tag syntax
	dbdefTag := field.Tag.Get("dbdef")
	if dbdefTag != "" {
		if err := v.validateDbdefTag(dbdefTag); err != nil {
			errors = append(errors, ModelValidationError{
				Type:    typeName,
				Field:   field.Name,
				Message: fmt.Sprintf("invalid dbdef tag: %s", err),
			})
		}
	}

	// Validate ORM relationship tags
	ormTag := field.Tag.Get("orm")
	if ormTag != "" {
		if _, err := v.tagParser.ParseORMTag(ormTag); err != nil {
			errors = append(errors, ModelValidationError{
				Type:    typeName,
				Field:   field.Name,
				Message: fmt.Sprintf("invalid orm tag: %s", err),
			})
		}
	}

	// Check field type compatibility
	if err := v.validateFieldType(field.Type); err != nil {
		errors = append(errors, ModelValidationError{
			Type:    typeName,
			Field:   field.Name,
			Message: fmt.Sprintf("unsupported field type: %s", err),
		})
	}

	return errors
}

// validateDbdefTag validates dbdef tag syntax
func (v *ModelValidator) validateDbdefTag(tag string) error {
	parts := strings.Split(tag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Check for valid dbdef options
		switch {
		case part == "primary_key":
		case part == "unique":
		case part == "not_null":
		case part == "auto_increment":
		case strings.HasPrefix(part, "default:"):
		case strings.HasPrefix(part, "size:"):
		case strings.HasPrefix(part, "type:"):
		case strings.HasPrefix(part, "references:"):
		default:
			return fmt.Errorf("unknown dbdef option: %s", part)
		}
	}
	return nil
}

// validateFieldType checks if field type is supported
func (v *ModelValidator) validateFieldType(fieldType reflect.Type) error {
	// Handle pointers
	if fieldType.Kind() == reflect.Ptr {
		fieldType = fieldType.Elem()
	}

	switch fieldType.Kind() {
	case reflect.String:
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
	case reflect.Float32, reflect.Float64:
	case reflect.Bool:
	case reflect.Slice:
		// Check slice element type
		if fieldType.Elem().Kind() == reflect.Uint8 {
			// []byte is supported
			return nil
		}
		return fmt.Errorf("unsupported slice type: %s", fieldType)
	case reflect.Struct:
		// Check for supported struct types
		typeName := fieldType.String()
		switch typeName {
		case "time.Time":
		case "json.RawMessage":
		case "uuid.UUID":
		default:
			// Allow embedded structs
			if fieldType.NumField() > 0 {
				return nil
			}
			return fmt.Errorf("unsupported struct type: %s", typeName)
		}
	default:
		return fmt.Errorf("unsupported kind: %s", fieldType.Kind())
	}

	return nil
}

// validateRelationships validates relationships between models
func (v *ModelValidator) validateRelationships(models map[string]reflect.Type) []ModelValidationError {
	var errors []ModelValidationError

	for modelName, modelType := range models {
		for i := 0; i < modelType.NumField(); i++ {
			field := modelType.Field(i)
			ormTag := field.Tag.Get("orm")
			if ormTag == "" {
				continue
			}

			parsedTag, err := v.tagParser.ParseORMTag(ormTag)
			if err != nil {
				// Already caught in field validation
				continue
			}

			// Validate target model exists
			targetTableName := strings.ToLower(parsedTag.Target + "s") // Simple pluralization
			targetExists := false
			for _, targetType := range models {
				if strings.ToLower(deriveTableName(targetType.Name())) == targetTableName {
					targetExists = true
					break
				}
			}

			if !targetExists {
				errors = append(errors, ModelValidationError{
					Type:    modelName,
					Field:   field.Name,
					Message: fmt.Sprintf("relationship target '%s' not found in registered models", parsedTag.Target),
				})
			}

			// Validate foreign key field exists
			if parsedTag.ForeignKey != "" {
				if !v.hasField(modelType, parsedTag.ForeignKey) {
					errors = append(errors, ModelValidationError{
						Type:    modelName,
						Field:   field.Name,
						Message: fmt.Sprintf("foreign key field '%s' not found", parsedTag.ForeignKey),
					})
				}
			}

			// Validate through table for many-to-many relationships
			if parsedTag.Type == "has_many_through" && parsedTag.Through != "" {
				throughTableName := strings.ToLower(parsedTag.Through)
				throughExists := false
				for _, throughType := range models {
					if strings.ToLower(deriveTableName(throughType.Name())) == throughTableName {
						throughExists = true
						break
					}
				}

				if !throughExists {
					errors = append(errors, ModelValidationError{
						Type:    modelName,
						Field:   field.Name,
						Message: fmt.Sprintf("through table '%s' not found in registered models", parsedTag.Through),
					})
				}
			}
		}
	}

	return errors
}

// hasField checks if a struct has a field with the given name
func (v *ModelValidator) hasField(structType reflect.Type, fieldName string) bool {
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		dbTag := field.Tag.Get("db")
		if dbTag == fieldName || field.Name == fieldName {
			return true
		}
	}
	return false
}

// ValidateModelsFromDirectory auto-discovers and validates models using the migrator's proven parser
func ValidateModelsFromDirectory(packagePath string) ValidationResult {
	// Use the SAME parser that the migrator uses successfully
	structParser := parser.NewStructParser()
	tables, err := structParser.ParseDirectory(packagePath)
	if err != nil {
		return ValidationResult{
			Valid: false,
			Errors: []ModelValidationError{{
				Type:    "System",
				Message: fmt.Sprintf("failed to parse directory %s: %v", packagePath, err),
			}},
		}
	}

	if len(tables) == 0 {
		return ValidationResult{
			Valid: false,
			Errors: []ModelValidationError{{
				Type:    "System",
				Message: fmt.Sprintf("no database models found in %s", packagePath),
			}},
		}
	}

	// Validate table-level requirements beyond basic parsing
	result := ValidationResult{Valid: true}
	tableNames := make(map[string]string) // tableName -> structName for duplicate detection

	// Filter to only include structs that are actual database models (have explicit table definitions)
	var dbModels []parser.TableDefinition
	for _, table := range tables {
		// Check if the table has an explicit table definition (not just derived name)
		if _, hasExplicitTable := table.TableLevel["table"]; hasExplicitTable {
			dbModels = append(dbModels, table)
		}
		// Skip structs without explicit table definitions - they're utility structs, not database models
	}

	for _, table := range dbModels {

		// Validate table name format (should be snake_case)
		if !isValidTableName(table.TableName) {
			result.Valid = false
			result.Errors = append(result.Errors, ModelValidationError{
				Type:    table.StructName,
				Message: fmt.Sprintf("invalid table name '%s' - should be snake_case and plural", table.TableName),
			})
		}

		// Check for duplicate table names
		if existingStruct, exists := tableNames[table.TableName]; exists {
			result.Valid = false
			result.Errors = append(result.Errors, ModelValidationError{
				Type:    table.StructName,
				Message: fmt.Sprintf("duplicate table name '%s' - already used by struct %s", table.TableName, existingStruct),
			})
		} else {
			tableNames[table.TableName] = table.StructName
		}

		// Validate primary key exists
		hasPK := false
		for _, field := range table.Fields {
			if _, isPK := field.DBDef["primary_key"]; isPK {
				hasPK = true
				break
			}
		}
		if !hasPK {
			result.Valid = false
			result.Errors = append(result.Errors, ModelValidationError{
				Type:    table.StructName,
				Message: "no primary key defined - add `dbdef:\"primary_key\"` to at least one field",
			})
		}
	}

	if result.Valid {
		fmt.Printf("✓ Successfully discovered %d models with valid table definitions\n", len(dbModels))
		for _, table := range dbModels {
			fmt.Printf("  - %s (table:%s)\n", table.StructName, table.TableName)
		}
	}

	return result
}

// validateTableDefinition validates a parser.TableDefinition
func (v *ModelValidator) validateTableDefinition(table parser.TableDefinition) []ModelValidationError {
	var errors []ModelValidationError

	// Check for primary key
	hasPK := false
	for _, field := range table.Fields {
		if _, isPK := field.DBDef["primary_key"]; isPK {
			hasPK = true
			break
		}
	}

	if !hasPK {
		errors = append(errors, ModelValidationError{
			Type:    table.StructName,
			Message: "must have at least one primary key field",
		})
	}

	// Validate each field
	for _, field := range table.Fields {
		fieldErrors := v.validateTableField(table.StructName, field)
		errors = append(errors, fieldErrors...)
	}

	return errors
}

// validateTableField validates a field from parser.FieldDefinition
func (v *ModelValidator) validateTableField(typeName string, field parser.FieldDefinition) []ModelValidationError {
	var errors []ModelValidationError

	// Validate dbdef tag syntax
	if field.DBDefTag != "" {
		if err := v.validateDbdefTag(field.DBDefTag); err != nil {
			errors = append(errors, ModelValidationError{
				Type:    typeName,
				Field:   field.Name,
				Message: fmt.Sprintf("invalid dbdef tag: %s", err),
			})
		}
	}

	// Basic field type validation
	if field.Type == "" {
		errors = append(errors, ModelValidationError{
			Type:    typeName,
			Field:   field.Name,
			Message: "field has no type information",
		})
	}

	return errors
}

// deriveTableName derives table name from struct name
func deriveTableName(structName string) string {
	// Convert PascalCase to snake_case and pluralize
	var result strings.Builder

	for i, r := range structName {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r + 32) // Convert to lowercase
	}

	snake := result.String()

	// Simple pluralization
	if strings.HasSuffix(snake, "y") && !strings.HasSuffix(snake, "ey") {
		return snake[:len(snake)-1] + "ies"
	}
	if strings.HasSuffix(snake, "s") || strings.HasSuffix(snake, "sh") ||
		strings.HasSuffix(snake, "ch") || strings.HasSuffix(snake, "x") {
		return snake + "es"
	}
	return snake + "s"
}

// isValidTableName checks if table name follows valid conventions
func isValidTableName(tableName string) bool {
	if tableName == "" {
		return false
	}

	// Check that it's snake_case (only lowercase letters, digits, underscores)
	for _, r := range tableName {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}

	// Don't start or end with underscore
	if strings.HasPrefix(tableName, "_") || strings.HasSuffix(tableName, "_") {
		return false
	}

	// Don't allow consecutive underscores
	if strings.Contains(tableName, "__") {
		return false
	}

	return true
}
