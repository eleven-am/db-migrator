package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"strings"
)

// FieldDefinition represents a struct field with database metadata
type FieldDefinition struct {
	Name      string            // Go field name
	DBName    string            // Database column name (from db tag)
	Type      string            // Go type (string, int, time.Time, etc.)
	IsPointer bool              // Whether field is a pointer (*string)
	IsArray   bool              // Whether field is an array/slice
	DBDef     map[string]string // Parsed dbdef tag attributes
	DBTag     string            // Raw db tag value
	DBDefTag  string            // Raw dbdef tag value
	JSONTag   string            // Raw json tag value (for debugging)
}

// TableDefinition represents a complete table structure
type TableDefinition struct {
	StructName string            // Go struct name
	TableName  string            // Database table name
	Fields     []FieldDefinition // All fields in the struct
	TableLevel map[string]string // Table-level dbdef attributes (indexes, constraints)
}

// StructParser handles parsing Go struct definitions
type StructParser struct {
	fileSet   *token.FileSet
	tagParser *TagParser
}

// NewStructParser creates a new struct parser instance
func NewStructParser() *StructParser {
	return &StructParser{
		fileSet:   token.NewFileSet(),
		tagParser: NewTagParser(),
	}
}

// ParseDirectory scans a directory for Go files and extracts struct definitions
func (p *StructParser) ParseDirectory(dir string) ([]TableDefinition, error) {
	pattern := filepath.Join(dir, "*.go")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob directory %s: %w", dir, err)
	}

	var allTables []TableDefinition

	for _, file := range matches {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}

		tables, err := p.ParseFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to parse file %s: %w", file, err)
		}

		allTables = append(allTables, tables...)
	}

	return allTables, nil
}

// ParseFile parses a single Go file for struct definitions
func (p *StructParser) ParseFile(filename string) ([]TableDefinition, error) {
	src, err := parser.ParseFile(p.fileSet, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	var tables []TableDefinition

	ast.Inspect(src, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.TypeSpec:
			if structType, ok := node.Type.(*ast.StructType); ok {
				table, err := p.parseStruct(node.Name.Name, structType)
				if err != nil {
					fmt.Printf("Warning: failed to parse struct %s: %v\n", node.Name.Name, err)
					return true
				}

				if p.isDatabaseStruct(table) {
					tables = append(tables, table)
				}
			}
		}
		return true
	})

	return tables, nil
}

// parseStruct converts an AST struct type to a TableDefinition
func (p *StructParser) parseStruct(structName string, structType *ast.StructType) (TableDefinition, error) {
	table := TableDefinition{
		StructName: structName,
		TableName:  p.deriveTableName(structName),
		Fields:     make([]FieldDefinition, 0),
		TableLevel: make(map[string]string),
	}

	for _, field := range structType.Fields.List {
		fieldDefs, tableLevelAttrs, err := p.parseField(field)
		if err != nil {
			return table, fmt.Errorf("failed to parse field: %w", err)
		}

		table.Fields = append(table.Fields, fieldDefs...)

		for k, v := range tableLevelAttrs {
			table.TableLevel[k] = v
		}
	}

	if tableName, exists := table.TableLevel["table"]; exists {
		table.TableName = tableName
	}

	return table, nil
}

// parseField converts an AST field to FieldDefinition(s)
func (p *StructParser) parseField(field *ast.Field) ([]FieldDefinition, map[string]string, error) {
	var fields []FieldDefinition
	tableLevelAttrs := make(map[string]string)

	if len(field.Names) == 0 {
		if field.Tag != nil {
			tagValue := strings.Trim(field.Tag.Value, "`")
			dbdefTag := p.extractTag(tagValue, "dbdef")
			if dbdefTag != "" {
				attrs := p.tagParser.ParseDBDefTag(dbdefTag)
				for k, v := range attrs {
					tableLevelAttrs[k] = v
				}
			}
		}
		return fields, tableLevelAttrs, nil
	}

	for _, name := range field.Names {
		if !ast.IsExported(name.Name) && name.Name != "_" {
			continue
		}

		if name.Name == "_" && field.Tag != nil {
			tagValue := strings.Trim(field.Tag.Value, "`")
			dbdefTag := p.extractTag(tagValue, "dbdef")
			if dbdefTag != "" {
				attrs := p.tagParser.ParseDBDefTag(dbdefTag)
				for k, v := range attrs {
					tableLevelAttrs[k] = v
				}
			}
			continue
		}

		fieldDef := FieldDefinition{
			Name: name.Name,
		}

		fieldType, isPointer, isArray := p.parseFieldType(field.Type)
		fieldDef.Type = fieldType
		fieldDef.IsPointer = isPointer
		fieldDef.IsArray = isArray

		if field.Tag != nil {
			tagValue := strings.Trim(field.Tag.Value, "`")

			fieldDef.DBTag = p.extractTag(tagValue, "db")
			fieldDef.DBDefTag = p.extractTag(tagValue, "dbdef")
			fieldDef.JSONTag = p.extractTag(tagValue, "json")

			if fieldDef.DBTag != "" {
				fieldDef.DBName = fieldDef.DBTag
			} else {
				fieldDef.DBName = p.toSnakeCase(fieldDef.Name)
			}

			if fieldDef.DBDefTag != "" {
				fieldDef.DBDef = p.tagParser.ParseDBDefTag(fieldDef.DBDefTag)
			} else {
				fieldDef.DBDef = make(map[string]string)
			}
		} else {
			fieldDef.DBName = p.toSnakeCase(fieldDef.Name)
			fieldDef.DBDef = make(map[string]string)
		}

		fields = append(fields, fieldDef)
	}

	return fields, tableLevelAttrs, nil
}

// parseFieldType extracts type information from an AST type expression
func (p *StructParser) parseFieldType(expr ast.Expr) (string, bool, bool) {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name, false, false

	case *ast.StarExpr:
		innerType, _, isArray := p.parseFieldType(t.X)
		return innerType, true, isArray

	case *ast.ArrayType:
		innerType, isPointer, _ := p.parseFieldType(t.Elt)
		return innerType, isPointer, true

	case *ast.SelectorExpr:
		pkg := p.exprToString(t.X)
		return pkg + "." + t.Sel.Name, false, false
	}
	return "", false, false
}

// extractTag extracts a specific tag value from a struct tag string
func (p *StructParser) extractTag(tagString, tagName string) string {
	tag := reflect.StructTag(tagString)
	return tag.Get(tagName)
}

// isDatabaseStruct determines if a struct represents a database entity
func (p *StructParser) isDatabaseStruct(table TableDefinition) bool {
	// Check if any field has a db tag or dbdef tag
	for _, field := range table.Fields {
		if field.DBTag != "" || field.DBDefTag != "" {
			return true
		}
	}

	// Check for table-level dbdef
	if len(table.TableLevel) > 0 {
		return true
	}

	return false
}

// deriveTableName converts a struct name to a table name using conventions
func (p *StructParser) deriveTableName(structName string) string {
	// Simple pluralization - convert to snake_case and add 's'
	snake := p.toSnakeCase(structName)

	// Special cases for irregular plurals
	irregularPlurals := map[string]string{
		"analysis": "analyses",
		"basis":    "bases",
		"datum":    "data",
		"index":    "indexes", // or "indices"
		"matrix":   "matrices",
		"vertex":   "vertices",
		"axis":     "axes",
		"crisis":   "crises",
		// "person":   "people", // commented out - using regular pluralization
		"child": "children",
		"foot":  "feet",
		"tooth": "teeth",
		"goose": "geese",
		"man":   "men",
		"woman": "women",
		"mouse": "mice",
	}

	if plural, ok := irregularPlurals[snake]; ok {
		return plural
	}

	// Basic pluralization rules
	if strings.HasSuffix(snake, "y") && !strings.HasSuffix(snake, "ey") && !strings.HasSuffix(snake, "ay") && !strings.HasSuffix(snake, "oy") && !strings.HasSuffix(snake, "uy") {
		// policy -> policies, but not key -> keies
		return snake[:len(snake)-1] + "ies"
	}
	if strings.HasSuffix(snake, "s") || strings.HasSuffix(snake, "sh") || strings.HasSuffix(snake, "ch") || strings.HasSuffix(snake, "x") || strings.HasSuffix(snake, "z") {
		return snake + "es"
	}
	return snake + "s"
}

// toSnakeCase converts PascalCase to snake_case
func (p *StructParser) toSnakeCase(s string) string {
	// Handle known edge cases
	edgeCases := map[string]string{
		"OAuth2Token": "oauth2_token",
		"OAuth2":      "oauth2",
		"OAuth":       "oauth",
	}
	if result, ok := edgeCases[s]; ok {
		return result
	}

	var result strings.Builder

	for i, r := range s {
		isUpper := r >= 'A' && r <= 'Z'

		// Determine if we need an underscore before this character
		if i > 0 {
			prevIsLower := s[i-1] >= 'a' && s[i-1] <= 'z'
			prevIsDigit := s[i-1] >= '0' && s[i-1] <= '9'
			prevIsUpper := s[i-1] >= 'A' && s[i-1] <= 'Z'

			// Add underscore in these cases:
			// 1. Uppercase letter after lowercase letter: aB -> a_b
			// 2. Uppercase letter after digit: 1A -> 1_a
			// 3. Letter after digit (except continuing a number): 2nd -> 2nd, but 42A -> 42_a
			// 4. Start of new word in acronym: SQLParser -> sql_parser
			if isUpper && (prevIsLower || prevIsDigit) {
				result.WriteRune('_')
			} else if isUpper && prevIsUpper && i+1 < len(s) {
				// Check if this is the start of a new word (like the 'P' in 'HTTPParser')
				nextIsLower := s[i+1] >= 'a' && s[i+1] <= 'z'
				if nextIsLower {
					result.WriteRune('_')
				}
			} else if (r >= 'a' && r <= 'z') && prevIsDigit {
				// Don't split numbers like "2nd" but do split "42abc"
				if i >= 2 {
					prevPrevIsDigit := s[i-2] >= '0' && s[i-2] <= '9'
					if !prevPrevIsDigit || !isOrdinalSuffix(s[i-1:]) {
						result.WriteRune('_')
					}
				} else {
					result.WriteRune('_')
				}
			}
		}

		// Add the character (converting uppercase to lowercase)
		if isUpper {
			result.WriteRune(r - 'A' + 'a')
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// isOrdinalSuffix checks if the string starts with an ordinal suffix like "st", "nd", "rd", "th"
func isOrdinalSuffix(s string) bool {
	if len(s) < 2 {
		return false
	}
	suffix := s[:2]
	return suffix == "st" || suffix == "nd" || suffix == "rd" || suffix == "th"
}

func (p *StructParser) exprToString(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		return p.exprToString(v.X) + "." + v.Sel.Name
	default:
		return ""
	}
}
