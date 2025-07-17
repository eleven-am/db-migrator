package orm

import (
	"database/sql/driver"
	"fmt"
	"strings"
)

// StringArray handles PostgreSQL text[] arrays
type StringArray []string

// Scan implements the sql.Scanner interface for StringArray
func (sa *StringArray) Scan(value interface{}) error {
	if value == nil {
		*sa = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return sa.parseArray(string(v))
	case string:
		return sa.parseArray(v)
	default:
		return fmt.Errorf("cannot scan %T into StringArray", value)
	}
}

// Value implements the driver.Valuer interface for StringArray
func (sa StringArray) Value() (driver.Value, error) {
	if sa == nil {
		return nil, nil
	}

	if len(sa) == 0 {
		return "{}", nil
	}

	// Escape quotes and build PostgreSQL array literal
	var escaped []string
	for _, s := range sa {
		// Escape quotes by doubling them
		escaped = append(escaped, `"`+strings.ReplaceAll(s, `"`, `""`)+`"`)
	}

	return "{" + strings.Join(escaped, ",") + "}", nil
}

// parseArray parses a PostgreSQL array literal into a Go slice
func (sa *StringArray) parseArray(s string) error {
	if s == "" || s == "{}" {
		*sa = []string{}
		return nil
	}

	// Remove outer braces
	if !strings.HasPrefix(s, "{") || !strings.HasSuffix(s, "}") {
		return fmt.Errorf("invalid array format: %s", s)
	}

	content := s[1 : len(s)-1]
	if content == "" {
		*sa = []string{}
		return nil
	}

	// Parse array elements
	var result []string
	var current strings.Builder
	var inQuotes bool
	var i int

	for i < len(content) {
		char := content[i]

		switch char {
		case '"':
			if inQuotes {
				// Check if this is an escaped quote
				if i+1 < len(content) && content[i+1] == '"' {
					current.WriteByte('"')
					i += 2 // Skip both quotes
					continue
				}
				inQuotes = false
			} else {
				inQuotes = true
			}
		case ',':
			if !inQuotes {
				result = append(result, current.String())
				current.Reset()
				i++
				continue
			}
			current.WriteByte(char)
		default:
			current.WriteByte(char)
		}
		i++
	}

	// Add the last element
	if current.Len() > 0 || len(result) > 0 {
		result = append(result, current.String())
	}

	*sa = result
	return nil
}
