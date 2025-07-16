package parser

import (
	"reflect"
	"testing"
)

func TestTagParser_ParseDBDefTag(t *testing.T) {
	parser := NewTagParser()

	tests := []struct {
		name     string
		tag      string
		expected map[string]string
	}{
		{
			name: "simple primary key",
			tag:  "type:uuid;primary_key",
			expected: map[string]string{
				"type":        "uuid",
				"primary_key": "",
			},
		},
		{
			name: "full field tags",
			tag:  "type:varchar(255);not_null;unique;default:'active'",
			expected: map[string]string{
				"type":     "varchar(255)",
				"not_null": "",
				"unique":   "",
				"default":  "'active'",
			},
		},
		{
			name: "foreign key",
			tag:  "type:uuid;fk:users.id;on_delete:CASCADE",
			expected: map[string]string{
				"type":      "uuid",
				"fk":        "users.id",
				"on_delete": "CASCADE",
			},
		},
		{
			name: "auto increment",
			tag:  "type:serial;primary_key;auto_increment",
			expected: map[string]string{
				"type":           "serial",
				"primary_key":    "",
				"auto_increment": "",
			},
		},
		{
			name: "with index",
			tag:  "type:varchar(100);index:idx_email",
			expected: map[string]string{
				"type":  "varchar(100)",
				"index": "idx_email",
			},
		},
		{
			name: "multiple indexes",
			tag:  "type:varchar(100);index:idx_email;index:idx_composite",
			expected: map[string]string{
				"type":  "varchar(100)",
				"index": "idx_email;idx_composite",
			},
		},
		{
			name: "with previous name hint",
			tag:  "type:varchar(100);prev:old_column_name",
			expected: map[string]string{
				"type": "varchar(100)",
				"prev": "old_column_name",
			},
		},
		{
			name: "boolean type",
			tag:  "type:boolean;default:true",
			expected: map[string]string{
				"type":    "boolean",
				"default": "true",
			},
		},
		{
			name: "jsonb type",
			tag:  "type:jsonb;default:'{}'",
			expected: map[string]string{
				"type":    "jsonb",
				"default": "'{}'",
			},
		},
		{
			name: "cuid type",
			tag:  "type:cuid;primary_key;default:gen_cuid()",
			expected: map[string]string{
				"type":        "cuid",
				"primary_key": "",
				"default":     "gen_cuid()",
			},
		},
		{
			name:     "empty tag",
			tag:      "",
			expected: map[string]string{},
		},
		{
			name: "complex default value",
			tag:  "type:timestamp;default:CURRENT_TIMESTAMP",
			expected: map[string]string{
				"type":    "timestamp",
				"default": "CURRENT_TIMESTAMP",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.ParseDBDefTag(tt.tag)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParseDBDefTag() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTagParser_HasFlag(t *testing.T) {
	parser := NewTagParser()

	tests := []struct {
		name     string
		tag      string
		flag     string
		expected bool
	}{
		{
			name:     "flag exists",
			tag:      "type:uuid;primary_key;not_null",
			flag:     "primary_key",
			expected: true,
		},
		{
			name:     "flag doesn't exist",
			tag:      "type:uuid;not_null",
			flag:     "primary_key",
			expected: false,
		},
		{
			name:     "empty tag",
			tag:      "",
			flag:     "primary_key",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := parser.ParseDBDefTag(tt.tag)
			result := parser.HasFlag(attrs, tt.flag)

			if result != tt.expected {
				t.Errorf("HasFlag() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTagParser_GetType(t *testing.T) {
	parser := NewTagParser()

	tests := []struct {
		name     string
		tag      string
		expected string
	}{
		{
			name:     "has type",
			tag:      "type:uuid;primary_key",
			expected: "uuid",
		},
		{
			name:     "no type",
			tag:      "primary_key;not_null",
			expected: "",
		},
		{
			name:     "empty tag",
			tag:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := parser.ParseDBDefTag(tt.tag)
			result := parser.GetType(attrs)

			if result != tt.expected {
				t.Errorf("GetType() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTagParser_GetDefault(t *testing.T) {
	parser := NewTagParser()

	tests := []struct {
		name     string
		tag      string
		expected string
	}{
		{
			name:     "has default",
			tag:      "type:varchar;default:'active'",
			expected: "'active'",
		},
		{
			name:     "no default",
			tag:      "type:varchar;not_null",
			expected: "",
		},
		{
			name:     "empty tag",
			tag:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := parser.ParseDBDefTag(tt.tag)
			result := parser.GetDefault(attrs)

			if result != tt.expected {
				t.Errorf("GetDefault() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTagParser_GetForeignKey(t *testing.T) {
	parser := NewTagParser()

	tests := []struct {
		name     string
		tag      string
		expected string
	}{
		{
			name:     "has foreign key",
			tag:      "type:uuid;fk:users.id",
			expected: "users.id",
		},
		{
			name:     "no foreign key",
			tag:      "type:uuid;not_null",
			expected: "",
		},
		{
			name:     "empty tag",
			tag:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := parser.ParseDBDefTag(tt.tag)
			result := parser.GetForeignKey(attrs)

			if result != tt.expected {
				t.Errorf("GetForeignKey() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTagParser_GetArrayType(t *testing.T) {
	parser := NewTagParser()

	tests := []struct {
		name     string
		tag      string
		expected string
	}{
		{
			name:     "has array type",
			tag:      "type:varchar[];array:varchar",
			expected: "varchar",
		},
		{
			name:     "no array type",
			tag:      "type:varchar;not_null",
			expected: "",
		},
		{
			name:     "empty tag",
			tag:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := parser.ParseDBDefTag(tt.tag)
			result := parser.GetArrayType(attrs)

			if result != tt.expected {
				t.Errorf("GetArrayType() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTagParser_GetEnum(t *testing.T) {
	parser := NewTagParser()

	tests := []struct {
		name     string
		tag      string
		expected []string
	}{
		{
			name:     "has enum",
			tag:      "type:enum;enum:active,inactive,pending",
			expected: []string{"active", "inactive", "pending"},
		},
		{
			name:     "no enum",
			tag:      "type:varchar;not_null",
			expected: nil,
		},
		{
			name:     "empty tag",
			tag:      "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := parser.ParseDBDefTag(tt.tag)
			result := parser.GetEnum(attrs)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("GetEnum() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTagParser_GetPrevName(t *testing.T) {
	parser := NewTagParser()

	tests := []struct {
		name     string
		tag      string
		expected string
	}{
		{
			name:     "has prev name",
			tag:      "type:varchar;prev:old_column_name",
			expected: "old_column_name",
		},
		{
			name:     "no prev name",
			tag:      "type:varchar;not_null",
			expected: "",
		},
		{
			name:     "empty tag",
			tag:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := parser.ParseDBDefTag(tt.tag)
			result := parser.GetPrevName(attrs)

			if result != tt.expected {
				t.Errorf("GetPrevName() = %v, want %v", result, tt.expected)
			}
		})
	}
}
