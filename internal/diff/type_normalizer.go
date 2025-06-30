package diff

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// NormalizePostgreSQLType normalizes PostgreSQL type names to a canonical form
// to avoid false positives when comparing types
func NormalizePostgreSQLType(typeName string) string {
	// Convert to lowercase for comparison
	normalized := strings.ToLower(strings.TrimSpace(typeName))
	
	
	// Common PostgreSQL type aliases
	typeAliases := map[string]string{
		"character varying": "varchar",
		"character":         "char",
		"int":               "integer",
		"int2":              "smallint",
		"int4":              "integer",
		"int8":              "bigint",
		"float4":            "real",
		"float8":            "double precision",
		"numeric":           "decimal",
		"boolean":           "bool",
		"timestamp with time zone":    "timestamptz",
		"timestamp without time zone": "timestamp",
		"time with time zone":         "timetz",
		"time without time zone":      "time",
	}
	
	// Apply alias mapping to the whole type
	// Sort keys by length (longest first) to match more specific types first
	var sortedKeys []string
	for k := range typeAliases {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Slice(sortedKeys, func(i, j int) bool {
		return len(sortedKeys[i]) > len(sortedKeys[j])
	})
	
	for _, original := range sortedKeys {
		if strings.HasPrefix(normalized, original) {
			normalized = typeAliases[original] + normalized[len(original):]
			break
		}
	}
	
	// Additional normalizations
	normalized = strings.ReplaceAll(normalized, "  ", " ")
	// Remove space before parenthesis
	normalized = regexp.MustCompile(`\s+\(`).ReplaceAllString(normalized, "(")
	
	
	return normalized
}

// NormalizeDefault normalizes default value expressions
func NormalizeDefault(defaultValue string) string {
	if defaultValue == "" {
		return ""
	}
	
	// Remove surrounding quotes and type casts that PostgreSQL adds
	normalized := strings.TrimSpace(defaultValue)
	
	// Handle interval formats
	// '30 seconds'::interval -> 30 seconds
	// '00:00:30'::interval -> 30 seconds
	if strings.Contains(normalized, "::interval") {
		normalized = normalizeIntervalDefault(normalized)
	}
	
	// Handle array defaults
	// '{}'::text[] -> {}
	// '{}'::integer[] -> {}
	// ARRAY[]::text[] -> {}
	if regexp.MustCompile(`^'?\{\}'?::\w+\[\]$`).MatchString(normalized) {
		return "{}"
	}
	if regexp.MustCompile(`^ARRAY\[\]::\w+\[\]$`).MatchString(normalized) {
		return "{}"
	}
	
	// Handle JSONB defaults
	// '{}'::jsonb -> {}
	// First check if it's already just {}
	if normalized == "{}" {
		return "{}"
	}
	// Handle various JSONB empty object formats
	if regexp.MustCompile(`^'?\{\}'?::jsonb?$`).MatchString(normalized) {
		return "{}"
	}
	
	// Remove type casts for common types
	typeCasts := []string{
		"::text",
		"::character varying",
		"::varchar",
		"::character",
		"::char",
		"::jsonb",
		"::json",
		"::integer",
		"::bigint",
		"::boolean",
		"::bool",
		"::timestamptz",
		"::timestamp with time zone",
		"::timestamp without time zone",
		"::interval",
		"::numeric",
		"::decimal",
		"::real",
		"::double precision",
	}
	
	// Sort by length descending to match longer casts first
	sort.Slice(typeCasts, func(i, j int) bool {
		return len(typeCasts[i]) > len(typeCasts[j])
	})
	
	for _, cast := range typeCasts {
		// Handle both ::type and ::type(n) formats
		castPattern := regexp.QuoteMeta(cast)
		// Add optional (n) or (n,m) for types with precision
		re := regexp.MustCompile(castPattern + `(?:\(\d+(?:,\d+)?\))?`)
		normalized = re.ReplaceAllString(normalized, "")
	}
	
	// Remove parentheses around simple values
	if strings.HasPrefix(normalized, "(") && strings.HasSuffix(normalized, ")") {
		inner := normalized[1 : len(normalized)-1]
		if !strings.Contains(inner, "(") { // Not a function call
			normalized = inner
		}
	}
	
	// Remove single quotes around string literals
	if strings.HasPrefix(normalized, "'") && strings.HasSuffix(normalized, "'") {
		normalized = normalized[1 : len(normalized)-1]
	}
	
	// Normalize common function calls
	replacements := map[string]string{
		"now()":              "now()",
		"CURRENT_TIMESTAMP":  "now()",
		"current_timestamp":  "now()",
		"gen_random_uuid()":  "gen_random_uuid()",
		"uuid_generate_v4()": "gen_random_uuid()",
		"gen_cuid()":        "gen_cuid()",
	}
	
	for old, new := range replacements {
		if strings.EqualFold(normalized, old) {
			return new
		}
	}
	
	return normalized
}

// normalizeIntervalDefault normalizes PostgreSQL interval default values
func normalizeIntervalDefault(value string) string {
	// Remove ::interval suffix
	value = strings.TrimSuffix(value, "::interval")
	
	// Remove quotes
	value = strings.Trim(value, "'\"")
	
	// Handle time format: '00:00:30' -> '30 seconds'
	matched, _ := regexp.MatchString(`^\d{2}:\d{2}:\d{2}$`, value)
	if matched {
		parts := strings.Split(value, ":")
		hours, _ := strconv.Atoi(parts[0])
		minutes, _ := strconv.Atoi(parts[1])
		seconds, _ := strconv.Atoi(parts[2])
		
		// Convert to the most readable format
		if hours == 0 && minutes == 0 {
			return fmt.Sprintf("%d seconds", seconds)
		} else if hours == 0 {
			return fmt.Sprintf("%d minutes %d seconds", minutes, seconds)
		} else {
			return fmt.Sprintf("%d hours %d minutes %d seconds", hours, minutes, seconds)
		}
	}
	
	// Already in readable format
	return value
}