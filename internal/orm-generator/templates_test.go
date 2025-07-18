package orm_generator

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetadataTemplate(t *testing.T) {
	// Test that the metadata template constant exists and is not empty
	assert.NotEmpty(t, metadataTemplate)
	assert.Contains(t, metadataTemplate, "{{ .Model.Name }}")
	assert.Contains(t, metadataTemplate, "{{ .Package }}")
}

func TestColumnTemplate(t *testing.T) {
	// Test that the column template constant exists and is not empty
	assert.NotEmpty(t, columnTemplate)
	assert.Contains(t, columnTemplate, "Column")
}

func TestRepositoryTemplate(t *testing.T) {
	// Test that the repository template constant exists and is not empty
	assert.NotEmpty(t, repositoryTemplate)
	assert.Contains(t, repositoryTemplate, "Repository")
}

func TestTemplateHelperFunctions(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
		function func(string) string
	}{
		{
			name:     "toSnakeCase",
			input:    "TestUserName",
			expected: "test_user_name",
			function: toSnakeCase,
		},
		{
			name:     "toCamelCase",
			input:    "test_user_name",
			expected: "testUserName",
			function: toCamelCase,
		},
		{
			name:     "toPascalCase",
			input:    "test_user_name",
			expected: "TestUserName",
			function: toPascalCase,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.function(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestTemplateValidation(t *testing.T) {
	// Test that template constants are properly formatted
	templates := []string{metadataTemplate, columnTemplate, repositoryTemplate}

	for _, template := range templates {
		assert.NotEmpty(t, template)
		// Basic validation that templates have proper structure
		assert.Contains(t, template, "{{")
		assert.Contains(t, template, "}}")
	}
}

// Test helper functions (don't redefine existing functions)
func testToTitle(s string) string {
	return strings.Title(s)
}
