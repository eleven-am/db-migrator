package orm_generator

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/eleven-am/storm/internal/parser"
)

// ORMTagParser handles parsing of ORM-specific tags for code generation
type ORMTagParser struct {
	// Cache for parsed tags
	tagCache map[string]*ParsedORMTag
}

// ParsedORMTag represents a parsed ORM tag
type ParsedORMTag struct {
	Type        string   // "belongs_to", "has_one", "has_many", "has_many_through"
	Target      string   // Target model/table name
	ForeignKey  string   // Foreign key column
	SourceKey   string   // Source key column (for has_many)
	TargetKey   string   // Target key column (for belongs_to)
	JoinTable   string   // Join table for has_many_through
	SourceFK    string   // Source FK in join table
	TargetFK    string   // Target FK in join table
	Conditions  []string // Additional conditions
	OrderBy     string   // Default ordering
	Dependent   string   // Dependent action (destroy, delete, nullify)
	Inverse     string   // Inverse relationship name
	Polymorphic string   // Polymorphic association
	Through     string   // Through association
	Validate    bool     // Whether to validate association
	Autosave    bool     // Whether to autosave association
	Counter     string   // Counter cache column
	Raw         string   // Raw tag value
}

// NewORMTagParser creates a new ORM tag parser
func NewORMTagParser() *ORMTagParser {
	return &ORMTagParser{
		tagCache: make(map[string]*ParsedORMTag),
	}
}

// ParseORMTag parses an ORM tag string into a structured format
func (p *ORMTagParser) ParseORMTag(tag string) (*ParsedORMTag, error) {
	if tag == "" {
		return nil, fmt.Errorf("empty ORM tag")
	}

	if cached, exists := p.tagCache[tag]; exists {
		return cached, nil
	}

	parsed := &ParsedORMTag{
		Raw:      tag,
		Validate: true,
	}

	parts := strings.Split(tag, ",")
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid ORM tag format")
	}

	mainPart := strings.TrimSpace(parts[0])
	if err := p.parseMainRelationship(mainPart, parsed); err != nil {
		return nil, fmt.Errorf("failed to parse main relationship: %w", err)
	}

	for i := 1; i < len(parts); i++ {
		part := strings.TrimSpace(parts[i])
		if part == "" {
			continue
		}

		if err := p.parseRelationshipOption(part, parsed); err != nil {
			return nil, fmt.Errorf("failed to parse option '%s': %w", part, err)
		}
	}

	if err := p.validateAndSetDefaults(parsed); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	p.tagCache[tag] = parsed
	return parsed, nil
}

// parseMainRelationship parses the main relationship definition
func (p *ORMTagParser) parseMainRelationship(main string, parsed *ParsedORMTag) error {
	parts := strings.Split(main, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid relationship format, expected 'type:target'")
	}

	relType := strings.TrimSpace(parts[0])
	target := strings.TrimSpace(parts[1])

	switch relType {
	case "belongs_to", "has_one", "has_many", "has_many_through":
		parsed.Type = relType
	default:
		return fmt.Errorf("invalid relationship type: %s", relType)
	}

	if target == "" {
		return fmt.Errorf("target model cannot be empty")
	}

	parsed.Target = target
	return nil
}

// parseRelationshipOption parses a relationship option
func (p *ORMTagParser) parseRelationshipOption(option string, parsed *ParsedORMTag) error {
	switch option {
	case "validate":
		parsed.Validate = true
		return nil
	case "no_validate":
		parsed.Validate = false
		return nil
	case "autosave":
		parsed.Autosave = true
		return nil
	case "no_autosave":
		parsed.Autosave = false
		return nil
	}

	if strings.Contains(option, ":") {
		parts := strings.SplitN(option, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid option format: %s", option)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "foreign_key":
			parsed.ForeignKey = value
		case "source_key":
			parsed.SourceKey = value
		case "target_key":
			parsed.TargetKey = value
		case "join_table":
			parsed.JoinTable = value
		case "source_fk":
			parsed.SourceFK = value
		case "target_fk":
			parsed.TargetFK = value
		case "order_by":
			parsed.OrderBy = value
		case "dependent":
			if !isValidDependentAction(value) {
				return fmt.Errorf("invalid dependent action: %s", value)
			}
			parsed.Dependent = value
		case "inverse":
			parsed.Inverse = value
		case "polymorphic":
			parsed.Polymorphic = value
		case "through":
			parsed.Through = value
		case "counter":
			parsed.Counter = value
		case "conditions":
			parsed.Conditions = strings.Split(value, ",")
		default:
			return fmt.Errorf("unknown option: %s", key)
		}
	} else {
		return fmt.Errorf("invalid option format: %s", option)
	}

	return nil
}

// validateAndSetDefaults validates the parsed tag and sets defaults
func (p *ORMTagParser) validateAndSetDefaults(parsed *ParsedORMTag) error {
	switch parsed.Type {
	case "belongs_to":
		if parsed.ForeignKey == "" {
			parsed.ForeignKey = toSnakeCase(parsed.Target) + "_id"
		}
		if parsed.TargetKey == "" {
			parsed.TargetKey = "id"
		}

	case "has_one", "has_many":
		if parsed.ForeignKey == "" {
			return fmt.Errorf("foreign_key is required for %s relationships", parsed.Type)
		}
		if parsed.SourceKey == "" {
			parsed.SourceKey = "id"
		}

	case "has_many_through":
		if parsed.JoinTable == "" {
			return fmt.Errorf("join_table is required for has_many_through relationships")
		}
		if parsed.SourceFK == "" {
			return fmt.Errorf("source_fk is required for has_many_through relationships")
		}
		if parsed.TargetFK == "" {
			return fmt.Errorf("target_fk is required for has_many_through relationships")
		}
		if parsed.SourceKey == "" {
			parsed.SourceKey = "id"
		}
		if parsed.TargetKey == "" {
			parsed.TargetKey = "id"
		}
	}

	return nil
}

// isValidDependentAction checks if a dependent action is valid
func isValidDependentAction(action string) bool {
	validActions := []string{"destroy", "delete", "nullify", "restrict"}
	for _, valid := range validActions {
		if action == valid {
			return true
		}
	}
	return false
}

// FieldMetadata represents metadata about a struct field for code generation
type FieldMetadata struct {
	Name         string            // Go field name
	Type         string            // Go type
	DBName       string            // Database column name
	DBType       string            // Database type
	IsPointer    bool              // Whether it's a pointer type
	IsArray      bool              // Whether it's an array/slice
	IsPrimaryKey bool              // Whether it's a primary key
	IsUnique     bool              // Whether it has unique constraint
	IsRequired   bool              // Whether it's required (not null)
	DefaultValue string            // Default value
	Tags         map[string]string // All struct tags
	DBDef        map[string]string // Parsed dbdef tags
	Relationship *ParsedORMTag     // Parsed ORM relationship tag
}

// ModelMetadata represents metadata about a model for code generation
type ModelMetadata struct {
	Name          string               // Struct name
	Package       string               // Package name
	TableName     string               // Database table name
	Fields        []FieldMetadata      // All fields
	Relationships []FieldMetadata      // Only relationship fields
	Columns       []FieldMetadata      // Only database columns
	PrimaryKeys   []string             // Primary key column names
	Indexes       []IndexMetadata      // Index definitions
	Constraints   []ConstraintMetadata // Constraint definitions
}

// IndexMetadata represents index metadata
type IndexMetadata struct {
	Name    string   // Index name
	Columns []string // Column names
	Unique  bool     // Whether it's a unique index
	Partial string   // Partial index condition
}

// ConstraintMetadata represents constraint metadata
type ConstraintMetadata struct {
	Name       string // Constraint name
	Type       string // Constraint type (CHECK, FOREIGN KEY, etc.)
	Definition string // Constraint definition
}

// ParseModelFromTable converts parser.TableDefinition to ModelMetadata with relationship parsing
func (p *ORMTagParser) ParseModelFromTable(table parser.TableDefinition) (*ModelMetadata, error) {
	metadata := &ModelMetadata{
		Name:          table.StructName,
		Package:       "",
		TableName:     table.TableName,
		Fields:        make([]FieldMetadata, 0),
		Relationships: make([]FieldMetadata, 0),
		Columns:       make([]FieldMetadata, 0),
		PrimaryKeys:   make([]string, 0),
		Indexes:       make([]IndexMetadata, 0),
		Constraints:   make([]ConstraintMetadata, 0),
	}

	for _, field := range table.Fields {
		fieldMeta, err := p.parseFieldFromAST(field)
		if err != nil {
			return nil, fmt.Errorf("failed to parse field %s: %w", field.Name, err)
		}

		metadata.Fields = append(metadata.Fields, fieldMeta)

		if fieldMeta.Relationship != nil {
			metadata.Relationships = append(metadata.Relationships, fieldMeta)
		} else {
			metadata.Columns = append(metadata.Columns, fieldMeta)

			if fieldMeta.IsPrimaryKey {
				metadata.PrimaryKeys = append(metadata.PrimaryKeys, fieldMeta.DBName)
			}
		}
	}

	return metadata, nil
}

// parseFieldFromAST parses a field from AST parser.FieldDefinition
func (p *ORMTagParser) parseFieldFromAST(field parser.FieldDefinition) (FieldMetadata, error) {
	fieldMeta := FieldMetadata{
		Name:      field.Name,
		Type:      field.Type,
		DBName:    field.DBName,
		IsPointer: field.IsPointer,
		IsArray:   field.IsArray,
		Tags:      make(map[string]string),
		DBDef:     field.DBDef,
	}

	fieldMeta.Tags["db"] = field.DBTag
	fieldMeta.Tags["dbdef"] = field.DBDefTag
	fieldMeta.Tags["json"] = field.JSONTag

	if _, isPK := field.DBDef["primary_key"]; isPK {
		fieldMeta.IsPrimaryKey = true
	}

	if _, isUnique := field.DBDef["unique"]; isUnique {
		fieldMeta.IsUnique = true
	}

	if defaultVal, hasDefault := field.DBDef["default"]; hasDefault {
		fieldMeta.DefaultValue = defaultVal
	}

	ormTag := field.ORMTag

	if ormTag != "" {
		parsedRel, err := p.ParseORMTag(ormTag)
		if err != nil {
			return fieldMeta, fmt.Errorf("invalid orm tag: %w", err)
		}

		fieldMeta.Relationship = parsedRel
		fieldMeta.Tags["orm"] = ormTag
	}

	return fieldMeta, nil
}

// parseField parses a single struct field (reflection-based, keep for backward compatibility)
func (p *ORMTagParser) parseField(field reflect.StructField) (FieldMetadata, error) {
	fieldMeta := FieldMetadata{
		Name:  field.Name,
		Type:  field.Type.String(),
		Tags:  make(map[string]string),
		DBDef: make(map[string]string),
	}

	fieldMeta.Tags["json"] = field.Tag.Get("json")
	fieldMeta.Tags["db"] = field.Tag.Get("db")
	fieldMeta.Tags["dbdef"] = field.Tag.Get("dbdef")
	fieldMeta.Tags["orm"] = field.Tag.Get("orm")

	fieldType := field.Type
	if fieldType.Kind() == reflect.Ptr {
		fieldMeta.IsPointer = true
		fieldType = fieldType.Elem()
	}

	if fieldType.Kind() == reflect.Slice || fieldType.Kind() == reflect.Array {
		fieldMeta.IsArray = true
	}

	if dbTag := field.Tag.Get("db"); dbTag != "" {
		if dbTag == "-" {
			fieldMeta.DBName = ""
		} else {
			fieldMeta.DBName = dbTag
		}
	} else {
		fieldMeta.DBName = toSnakeCase(field.Name)
	}

	if dbdefTag := field.Tag.Get("dbdef"); dbdefTag != "" {
		fieldMeta.DBDef = parseDBDefTag(dbdefTag)

		if _, exists := fieldMeta.DBDef["primary_key"]; exists {
			fieldMeta.IsPrimaryKey = true
		}
		if _, exists := fieldMeta.DBDef["unique"]; exists {
			fieldMeta.IsUnique = true
		}
		if _, exists := fieldMeta.DBDef["not_null"]; exists {
			fieldMeta.IsRequired = true
		}
		if defaultVal, exists := fieldMeta.DBDef["default"]; exists {
			fieldMeta.DefaultValue = defaultVal
		}
		if dbType, exists := fieldMeta.DBDef["type"]; exists {
			fieldMeta.DBType = dbType
		}
	}

	if ormTag := field.Tag.Get("orm"); ormTag != "" {
		relationship, err := p.ParseORMTag(ormTag)
		if err != nil {
			return fieldMeta, fmt.Errorf("failed to parse ORM tag: %w", err)
		}
		fieldMeta.Relationship = relationship
	}

	return fieldMeta, nil
}

// parseDBDefTag parses a dbdef tag into a map
func parseDBDefTag(tag string) map[string]string {
	result := make(map[string]string)

	parts := strings.Split(tag, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, ":") {
			kv := strings.SplitN(part, ":", 2)
			if len(kv) == 2 {
				key := strings.TrimSpace(kv[0])
				value := strings.TrimSpace(kv[1])
				result[key] = value
			}
		} else {
			result[part] = "true"
		}
	}

	return result
}

// toSnakeCase converts PascalCase to snake_case
func toSnakeCase(s string) string {
	var result strings.Builder

	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		if r >= 'A' && r <= 'Z' {
			result.WriteRune(r - 'A' + 'a')
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// toPascalCase converts snake_case to PascalCase
func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, "")
}

// toCamelCase converts snake_case to camelCase
func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	if len(parts) == 0 {
		return s
	}

	result := strings.ToLower(parts[0])
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			result += strings.ToUpper(parts[i][:1]) + strings.ToLower(parts[i][1:])
		}
	}
	return result
}

// pluralize provides simple pluralization
func pluralize(s string) string {
	if strings.HasSuffix(s, "y") && !strings.HasSuffix(s, "ey") {
		return s[:len(s)-1] + "ies"
	}
	if strings.HasSuffix(s, "s") || strings.HasSuffix(s, "sh") ||
		strings.HasSuffix(s, "ch") || strings.HasSuffix(s, "x") {
		return s + "es"
	}
	return s + "s"
}

// singularize provides simple singularization
func singularize(s string) string {
	if strings.HasSuffix(s, "ies") {
		return s[:len(s)-3] + "y"
	}
	if strings.HasSuffix(s, "es") && !strings.HasSuffix(s, "ses") {
		return s[:len(s)-2]
	}
	if strings.HasSuffix(s, "s") {
		return s[:len(s)-1]
	}
	return s
}
