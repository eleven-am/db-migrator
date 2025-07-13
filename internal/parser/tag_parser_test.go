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
