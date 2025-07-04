package parser

import (
	"fmt"
	"strings"
)

// TagParser handles parsing of dbdef struct tags
type TagParser struct{}

// NewTagParser creates a new tag parser instance
func NewTagParser() *TagParser {
	return &TagParser{}
}

// ParseDBDefTag parses a dbdef tag string into a map of attributes
// Format: "type:uuid;primary_key;default:gen_random_uuid();not_null"
// Returns: map[string]string{"type": "uuid", "primary_key": "", "default": "gen_random_uuid()", "not_null": ""}
func (p *TagParser) ParseDBDefTag(tagValue string) map[string]string {
	attributes := make(map[string]string)

	if tagValue == "" {
		return attributes
	}

	parts := strings.Split(tagValue, ";")

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

				if existing, exists := attributes[key]; exists {
					attributes[key] = existing + ";" + value
				} else {
					attributes[key] = value
				}
			}
		} else {
			attributes[part] = ""
		}
	}

	return attributes
}

// ValidateDBDefTag validates a dbdef tag for common errors
func (p *TagParser) ValidateDBDefTag(tagValue string) error {
	if tagValue == "" {
		return nil
	}

	attributes := p.ParseDBDefTag(tagValue)

	for key, value := range attributes {
		switch key {
		case "type":
			if err := p.validateType(value); err != nil {
				return fmt.Errorf("invalid type '%s': %w", value, err)
			}
		case "default":
			if err := p.validateDefault(value); err != nil {
				return fmt.Errorf("invalid default '%s': %w", value, err)
			}
		case "fk", "foreign_key":
			if err := p.validateForeignKey(value); err != nil {
				return fmt.Errorf("invalid foreign key '%s': %w", value, err)
			}
		case "check":
			if err := p.validateCheck(value); err != nil {
				return fmt.Errorf("invalid check constraint '%s': %w", value, err)
			}
		case "prev":
			if err := p.validatePrev(value); err != nil {
				return fmt.Errorf("invalid prev hint '%s': %w", value, err)
			}
		case "primary_key", "not_null", "unique", "auto_increment":
			if value != "" {
				return fmt.Errorf("flag attribute '%s' should not have a value", key)
			}
		default:
			fmt.Printf("Warning: unknown dbdef attribute '%s'\n", key)
		}
	}

	return nil
}

// validateType validates PostgreSQL column types
func (p *TagParser) validateType(typeValue string) error {
	if typeValue == "" {
		return fmt.Errorf("type cannot be empty")
	}

	validTypes := map[string]bool{
		"smallint": true, "integer": true, "bigint": true,
		"smallserial": true, "serial": true, "bigserial": true,

		"decimal": true, "numeric": true, "real": true, "double precision": true,

		"char": true, "varchar": true, "text": true,

		"timestamp": true, "timestamptz": true, "date": true, "time": true, "timetz": true,
		"interval": true,

		"boolean": true, "bool": true,

		"bytea": true,

		"json": true, "jsonb": true,

		"uuid":  true,
		"cuid":  true,
		"cuid2": true,

		"text[]": true, "integer[]": true, "uuid[]": true,

		"inet": true, "cidr": true, "macaddr": true,

		"point": true, "line": true, "lseg": true, "box": true, "path": true, "polygon": true, "circle": true,
	}

	baseType := typeValue
	if idx := strings.Index(typeValue, "("); idx != -1 {
		baseType = typeValue[:idx]
	}

	if !validTypes[strings.ToLower(baseType)] {
		return fmt.Errorf("unknown PostgreSQL type: %s", typeValue)
	}

	return nil
}

// validateDefault validates default value expressions
func (p *TagParser) validateDefault(defaultValue string) error {
	if defaultValue == "" {
		return fmt.Errorf("default value cannot be empty")
	}

	commonDefaults := []string{
		"now()", "current_timestamp", "current_date", "current_time",
		"gen_random_uuid()", "uuid_generate_v4()",
		"true", "false", "null",
	}

	lowerDefault := strings.ToLower(defaultValue)

	for _, common := range commonDefaults {
		if lowerDefault == common {
			return nil
		}
	}

	if (strings.HasPrefix(defaultValue, "'") && strings.HasSuffix(defaultValue, "'")) ||
		(strings.HasPrefix(defaultValue, "\"") && strings.HasSuffix(defaultValue, "\"")) {
		return nil
	}

	if strings.ContainsAny(defaultValue, "0123456789") &&
		!strings.ContainsAny(defaultValue, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		return nil
	}

	fmt.Printf("Warning: complex default expression '%s' - please verify manually\n", defaultValue)
	return nil
}

// validateForeignKey validates foreign key references
func (p *TagParser) validateForeignKey(fkValue string) error {
	if fkValue == "" {
		return fmt.Errorf("foreign key reference cannot be empty")
	}

	parts := strings.Split(fkValue, ".")
	if len(parts) != 2 {
		return fmt.Errorf("foreign key must be in format 'table.column', got: %s", fkValue)
	}

	tableName := strings.TrimSpace(parts[0])
	columnName := strings.TrimSpace(parts[1])

	if tableName == "" {
		return fmt.Errorf("table name cannot be empty in foreign key reference")
	}
	if columnName == "" {
		return fmt.Errorf("column name cannot be empty in foreign key reference")
	}

	return nil
}

// validateCheck validates check constraint expressions
func (p *TagParser) validateCheck(checkValue string) error {
	if checkValue == "" {
		return fmt.Errorf("check constraint cannot be empty")
	}

	return nil
}

// validatePrev validates previous column name hints for renames
func (p *TagParser) validatePrev(prevValue string) error {
	if prevValue == "" {
		return fmt.Errorf("prev hint cannot be empty")
	}

	if !isValidIdentifier(prevValue) {
		return fmt.Errorf("prev hint must be a valid identifier: %s", prevValue)
	}

	return nil
}

// isValidIdentifier checks if a string is a valid SQL identifier
func isValidIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}

	first := s[0]
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_') {
		return false
	}

	for i := 1; i < len(s); i++ {
		c := s[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}

	return true
}

// GetType extracts the PostgreSQL type from dbdef attributes
func (p *TagParser) GetType(attributes map[string]string) string {
	if typeVal, exists := attributes["type"]; exists {
		return typeVal
	}
	return ""
}

// HasFlag checks if a flag attribute is present
func (p *TagParser) HasFlag(attributes map[string]string, flag string) bool {
	_, exists := attributes[flag]
	return exists
}

// GetDefault extracts the default value from dbdef attributes
func (p *TagParser) GetDefault(attributes map[string]string) string {
	if defaultVal, exists := attributes["default"]; exists {
		return defaultVal
	}
	return ""
}

// GetForeignKey extracts the foreign key reference from dbdef attributes
func (p *TagParser) GetForeignKey(attributes map[string]string) string {
	if fkVal, exists := attributes["foreign_key"]; exists {
		return fkVal
	}
	if fkVal, exists := attributes["fk"]; exists {
		return fkVal
	}
	return ""
}

// GetPrevName extracts the previous column name for rename detection
func (p *TagParser) GetPrevName(attributes map[string]string) string {
	if prevVal, exists := attributes["prev"]; exists {
		return prevVal
	}
	return ""
}
