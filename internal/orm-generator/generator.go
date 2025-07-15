package orm_generator

import (
	"bytes"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/eleven-am/storm/internal/generator"
)

// CodeGenerator handles generation of type-safe ORM code
type CodeGenerator struct {
	tagParser   *ORMTagParser
	packageName string
	outputDir   string
	templates   map[string]*template.Template
	models      map[string]*ModelMetadata
}

// GenerationConfig configures code generation
type GenerationConfig struct {
	PackageName  string   // Package name for generated code
	OutputDir    string   // Output directory
	Models       []string // Model names to generate (empty = all)
	Features     []string // Features to generate (columns, repositories, etc.)
	TemplateDir  string   // Custom template directory
	FileHeader   string   // Custom file header
	IncludeTests bool     // Whether to generate tests
	IncludeDocs  bool     // Whether to generate documentation
}

// NewCodeGenerator creates a new code generator
func NewCodeGenerator(config GenerationConfig) *CodeGenerator {
	return &CodeGenerator{
		tagParser:   NewORMTagParser(),
		packageName: config.PackageName,
		outputDir:   config.OutputDir,
		templates:   make(map[string]*template.Template),
		models:      make(map[string]*ModelMetadata),
	}
}

// DiscoverModels auto-discovers models from a package directory using migrator's proven pipeline
func (g *CodeGenerator) DiscoverModels(packagePath string) error {
	// Auto-detect package name if not provided
	if g.packageName == "" {
		packageName, err := g.detectPackageName(packagePath)
		if err != nil {
			return fmt.Errorf("failed to detect package name: %w", err)
		}
		g.packageName = packageName
	}

	// Step 1: Use migrator's struct parser (same as migrate command)
	parser := structParser.NewStructParser()
	tables, err := parser.ParseDirectory(packagePath)
	if err != nil {
		return fmt.Errorf("failed to parse directory %s: %w", packagePath, err)
	}

	// Filter to only include structs that are actual database models (have explicit table definitions)
	var dbModels []structParser.TableDefinition
	for _, table := range tables {
		// Check if the table has an explicit table definition (not just derived name)
		if _, hasExplicitTable := table.TableLevel["table"]; hasExplicitTable {
			dbModels = append(dbModels, table)
		}
		// Skip structs without explicit table definitions - they're utility structs, not database models
	}

	// Step 2: Convert table definitions directly to our ModelMetadata
	for _, tableDef := range dbModels {
		metadata := g.convertTableDefinitionToModelMetadata(tableDef)
		g.models[metadata.Name] = metadata
	}

	return nil
}

// convertTableDefinitionToModelMetadata converts parser's TableDefinition to ModelMetadata
func (g *CodeGenerator) convertTableDefinitionToModelMetadata(tableDef structParser.TableDefinition) *ModelMetadata {
	metadata := &ModelMetadata{
		Name:          tableDef.StructName, // Use Go struct name, not table name
		TableName:     tableDef.TableName,  // Database table name
		Columns:       make([]FieldMetadata, 0, len(tableDef.Fields)),
		PrimaryKeys:   make([]string, 0),
		Indexes:       make([]IndexMetadata, 0),
		Relationships: make([]FieldMetadata, 0),
	}

	// Convert fields from parser format
	for _, field := range tableDef.Fields {
		fieldMeta := FieldMetadata{
			Name:   field.Name,   // Go field name
			DBName: field.DBName, // Database column name
			Type:   field.Type,   // Go type from parser
		}

		// Check if field is nullable (pointer type)
		fieldMeta.IsPointer = field.IsPointer
		fieldMeta.IsArray = field.IsArray

		// Parse ORM relationship tag if present
		if field.ORMTag != "" {
			parsedRel, err := g.tagParser.ParseORMTag(field.ORMTag)
			if err != nil {
				// Log error but continue processing
				fmt.Printf("Warning: failed to parse ORM tag for field %s.%s: %v\n", tableDef.StructName, field.Name, err)
			} else {
				fieldMeta.Relationship = parsedRel
				// This is a relationship field
				metadata.Relationships = append(metadata.Relationships, fieldMeta)
				continue // Don't add to columns
			}
		}

		// This is a database column field
		// Check for primary key
		if _, isPK := field.DBDef["primary_key"]; isPK {
			fieldMeta.IsPrimaryKey = true
			metadata.PrimaryKeys = append(metadata.PrimaryKeys, field.DBName)
		}

		// Check for unique constraint
		if _, isUnique := field.DBDef["unique"]; isUnique {
			fieldMeta.IsUnique = true
		}

		// Check for default value
		if defaultVal, hasDefault := field.DBDef["default"]; hasDefault {
			fieldMeta.DefaultValue = defaultVal
		}

		// Set database type from dbdef
		if dbType, hasType := field.DBDef["type"]; hasType {
			fieldMeta.DBType = dbType
		}

		// Only add to columns if it's not a relationship field
		metadata.Columns = append(metadata.Columns, fieldMeta)
	}

	return metadata
}

// detectPackageName extracts the package name from Go files in the directory
func (g *CodeGenerator) detectPackageName(packagePath string) (string, error) {
	// Parse just one file to get the package name
	pattern := filepath.Join(packagePath, "*.go")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to glob directory %s: %w", packagePath, err)
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no Go files found in directory %s", packagePath)
	}

	// Parse the first non-test file using Go's AST parser
	fileSet := token.NewFileSet()
	for _, file := range matches {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}

		src, err := parser.ParseFile(fileSet, file, nil, parser.ParseComments)
		if err != nil {
			continue // Try next file
		}

		if src.Name != nil {
			return src.Name.Name, nil
		}
	}

	return "", fmt.Errorf("could not detect package name from files in %s", packagePath)
}

// convertSchemaTableToModelMetadata converts migrator's SchemaTable to ModelMetadata
func (g *CodeGenerator) convertSchemaTableToModelMetadata(schemaTable *generator.SchemaTable) *ModelMetadata {
	metadata := &ModelMetadata{
		Name:          schemaTable.Name,
		TableName:     schemaTable.Name,
		Columns:       make([]FieldMetadata, 0, len(schemaTable.Columns)),
		PrimaryKeys:   make([]string, 0),
		Indexes:       make([]IndexMetadata, 0),
		Relationships: make([]FieldMetadata, 0),
	}

	// Convert columns from migrator's format
	for _, col := range schemaTable.Columns {
		fieldMeta := FieldMetadata{
			Name:         col.Name,
			DBName:       col.Name,
			Type:         g.mapSchemaTypeToGo(col.Type),
			IsPointer:    col.IsNullable,
			IsPrimaryKey: col.IsPrimaryKey,
			IsUnique:     col.IsUnique,
		}

		if col.DefaultValue != nil {
			fieldMeta.DefaultValue = *col.DefaultValue
		}

		metadata.Columns = append(metadata.Columns, fieldMeta)

		if col.IsPrimaryKey {
			metadata.PrimaryKeys = append(metadata.PrimaryKeys, col.Name)
		}
	}

	return metadata
}

// mapSchemaTypeToGo maps schema column types to Go types
func (g *CodeGenerator) mapSchemaTypeToGo(schemaType string) string {
	switch strings.ToLower(schemaType) {
	case "text", "varchar", "char":
		return "string"
	case "integer", "int", "int4":
		return "int32"
	case "bigint", "int8":
		return "int64"
	case "smallint", "int2":
		return "int16"
	case "boolean", "bool":
		return "bool"
	case "timestamptz", "timestamp":
		return "time.Time"
	case "real", "float4":
		return "float32"
	case "double precision", "float8":
		return "float64"
	case "jsonb", "json":
		return "json.RawMessage"
	case "bytea":
		return "[]byte"
	default:
		if strings.HasSuffix(schemaType, "[]") {
			baseType := strings.TrimSuffix(schemaType, "[]")
			return "[]" + g.mapSchemaTypeToGo(baseType)
		}
		return "string" // Default fallback
	}
}

// GenerateAll generates all code for registered models
func (g *CodeGenerator) GenerateAll() error {
	// Load templates
	if err := g.loadTemplates(); err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	// Generate column constants
	if err := g.generateColumnConstants(); err != nil {
		return fmt.Errorf("failed to generate column constants: %w", err)
	}

	// Generate repositories
	if err := g.generateRepositories(); err != nil {
		return fmt.Errorf("failed to generate repositories: %w", err)
	}

	// Generate query builders
	if err := g.generateQueryBuilders(); err != nil {
		return fmt.Errorf("failed to generate query builders: %w", err)
	}

	// Generate relationship helpers
	if err := g.generateRelationshipHelpers(); err != nil {
		return fmt.Errorf("failed to generate relationship helpers: %w", err)
	}

	// Generate Storm struct
	if err := g.generateStorm(); err != nil {
		return fmt.Errorf("failed to generate Storm: %w", err)
	}

	return nil
}

// loadTemplates loads code generation templates
func (g *CodeGenerator) loadTemplates() error {
	// Create template functions
	funcMap := template.FuncMap{
		"lower":          strings.ToLower,
		"upper":          strings.ToUpper,
		"title":          strings.Title,
		"camel":          toCamelCase,
		"pascal":         toPascalCase,
		"snake":          toSnakeCase,
		"plural":         pluralize,
		"singular":       singularize,
		"goType":         g.mapDBTypeToGo,
		"dbType":         g.mapGoTypeToPostgreSQL,
		"join":           strings.Join,
		"hasPrefix":      strings.HasPrefix,
		"hasSuffix":      strings.HasSuffix,
		"contains":       strings.Contains,
		"replace":        strings.ReplaceAll,
		"now":            time.Now,
		"sanitizeGoName": sanitizeGoName,
	}

	// Load built-in templates
	g.templates["columns"] = template.Must(template.New("columns").Funcs(funcMap).Parse(columnTemplate))
	g.templates["repository"] = template.Must(template.New("repository").Funcs(funcMap).Parse(repositoryTemplate))
	g.templates["query"] = template.Must(template.New("query").Funcs(funcMap).Parse(queryTemplate))
	g.templates["relationships"] = template.Must(template.New("relationships").Funcs(funcMap).Parse(relationshipTemplate))
	g.templates["storm"] = template.Must(template.New("storm").Funcs(funcMap).Parse(stormTemplate))

	return nil
}

// generateColumnConstants generates type-safe column constants
func (g *CodeGenerator) generateColumnConstants() error {
	data := struct {
		Package string
		Models  map[string]*ModelMetadata
		Now     time.Time
	}{
		Package: g.packageName,
		Models:  g.models,
		Now:     time.Now(),
	}

	return g.executeTemplate("columns", "columns.go", data)
}

// generateRepositories generates repository implementations
func (g *CodeGenerator) generateRepositories() error {
	for _, model := range g.models {
		data := struct {
			Package string
			Model   *ModelMetadata
			Now     time.Time
		}{
			Package: g.packageName,
			Model:   model,
			Now:     time.Now(),
		}

		filename := fmt.Sprintf("%s_repository.go", toSnakeCase(model.Name))
		if err := g.executeTemplate("repository", filename, data); err != nil {
			return err
		}
	}
	return nil
}

// generateQueryBuilders generates query builder implementations
func (g *CodeGenerator) generateQueryBuilders() error {
	for _, model := range g.models {
		data := struct {
			Package string
			Model   *ModelMetadata
			Now     time.Time
		}{
			Package: g.packageName,
			Model:   model,
			Now:     time.Now(),
		}

		filename := fmt.Sprintf("%s_query.go", toSnakeCase(model.Name))
		if err := g.executeTemplate("query", filename, data); err != nil {
			return err
		}
	}
	return nil
}

// generateRelationshipHelpers generates relationship helper code
func (g *CodeGenerator) generateRelationshipHelpers() error {
	// Find all models with relationships
	modelsWithRelationships := make(map[string]*ModelMetadata)
	for name, model := range g.models {
		if len(model.Relationships) > 0 {
			modelsWithRelationships[name] = model
		}
	}

	if len(modelsWithRelationships) == 0 {
		return nil // No relationships to generate
	}

	data := struct {
		Package string
		Models  map[string]*ModelMetadata
		Now     time.Time
	}{
		Package: g.packageName,
		Models:  modelsWithRelationships,
		Now:     time.Now(),
	}

	return g.executeTemplate("relationships", "relationships.go", data)
}

// generateStorm generates the Storm struct with all repositories
func (g *CodeGenerator) generateStorm() error {
	data := struct {
		Package string
		Models  map[string]*ModelMetadata
		Now     time.Time
	}{
		Package: g.packageName,
		Models:  g.models,
		Now:     time.Now(),
	}

	return g.executeTemplate("storm", "storm.go", data)
}

// executeTemplate executes a template and writes to file
func (g *CodeGenerator) executeTemplate(templateName, filename string, data interface{}) error {
	tmpl, exists := g.templates[templateName]
	if !exists {
		return fmt.Errorf("template %s not found", templateName)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute template %s: %w", templateName, err)
	}

	// Format the generated code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to format generated code for %s: %w", filename, err)
	}

	// Write to file
	outputPath := filepath.Join(g.outputDir, filename)
	return writeFile(outputPath, formatted)
}

// mapDBTypeToGo maps PostgreSQL types to Go types
func (g *CodeGenerator) mapDBTypeToGo(dbType string) string {
	switch strings.ToLower(dbType) {
	case "integer", "int", "int4":
		return "int32"
	case "bigint", "int8":
		return "int64"
	case "smallint", "int2":
		return "int16"
	case "text", "varchar", "character varying":
		return "string"
	case "boolean", "bool":
		return "bool"
	case "timestamp", "timestamp with time zone", "timestamptz":
		return "time.Time"
	case "date":
		return "time.Time"
	case "real", "float4":
		return "float32"
	case "double precision", "float8":
		return "float64"
	case "uuid":
		return "string"
	case "jsonb", "json":
		return "json.RawMessage"
	case "bytea":
		return "[]byte"
	default:
		if strings.HasSuffix(dbType, "[]") {
			baseType := strings.TrimSuffix(dbType, "[]")
			return "[]" + g.mapDBTypeToGo(baseType)
		}
		return "string"
	}
}

// mapGoTypeToPostgreSQL maps Go types to PostgreSQL types
func (g *CodeGenerator) mapGoTypeToPostgreSQL(goType string) string {
	switch goType {
	case "string":
		return "TEXT"
	case "int", "int32":
		return "INTEGER"
	case "int64":
		return "BIGINT"
	case "int16":
		return "SMALLINT"
	case "float32":
		return "REAL"
	case "float64":
		return "DOUBLE PRECISION"
	case "bool":
		return "BOOLEAN"
	case "time.Time":
		return "TIMESTAMP WITH TIME ZONE"
	case "[]byte":
		return "BYTEA"
	case "json.RawMessage":
		return "JSONB"
	default:
		if strings.HasPrefix(goType, "[]") {
			baseType := strings.TrimPrefix(goType, "[]")
			return g.mapGoTypeToPostgreSQL(baseType) + "[]"
		}
		return "TEXT"
	}
}

// writeFile writes content to a file, creating directories if needed
func writeFile(path string, content []byte) error {
	dir := filepath.Dir(path)
	if err := ensureDir(dir); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	return os.WriteFile(path, content, 0644)
}

// ensureDir creates a directory if it doesn't exist
func ensureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// GetModelNames returns all registered model names
func (g *CodeGenerator) GetModelNames() []string {
	names := make([]string, 0, len(g.models))
	for name := range g.models {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetModel returns metadata for a specific model
func (g *CodeGenerator) GetModel(name string) (*ModelMetadata, bool) {
	model, exists := g.models[name]
	return model, exists
}

// GetModelsByTable returns models grouped by table name
func (g *CodeGenerator) GetModelsByTable() map[string]*ModelMetadata {
	result := make(map[string]*ModelMetadata)
	for _, model := range g.models {
		result[model.TableName] = model
	}
	return result
}

// ValidateModels validates all registered models
func (g *CodeGenerator) ValidateModels() error {
	for name, model := range g.models {
		if err := g.validateModel(model); err != nil {
			return fmt.Errorf("model %s validation failed: %w", name, err)
		}
	}
	return nil
}

// validateModel validates a single model
func (g *CodeGenerator) validateModel(model *ModelMetadata) error {
	// Check for primary key
	if len(model.PrimaryKeys) == 0 {
		return fmt.Errorf("model %s has no primary key", model.Name)
	}

	// Validate relationships
	for _, rel := range model.Relationships {
		if err := g.validateRelationship(model, rel); err != nil {
			return fmt.Errorf("relationship %s validation failed: %w", rel.Name, err)
		}
	}

	return nil
}

// validateRelationship validates a relationship
func (g *CodeGenerator) validateRelationship(model *ModelMetadata, rel FieldMetadata) error {
	if rel.Relationship == nil {
		return fmt.Errorf("relationship %s has no metadata", rel.Name)
	}

	// Check if target model exists
	targetModel, exists := g.models[rel.Relationship.Target]
	if !exists {
		return fmt.Errorf("target model %s not found for relationship %s", rel.Relationship.Target, rel.Name)
	}

	// Validate foreign key references
	switch rel.Relationship.Type {
	case "belongs_to":
		if !g.hasColumn(model, rel.Relationship.ForeignKey) {
			return fmt.Errorf("foreign key column %s not found in model %s", rel.Relationship.ForeignKey, model.Name)
		}
		if !g.hasColumn(targetModel, rel.Relationship.TargetKey) {
			return fmt.Errorf("target key column %s not found in target model %s", rel.Relationship.TargetKey, targetModel.Name)
		}

	case "has_one", "has_many":
		if !g.hasColumn(targetModel, rel.Relationship.ForeignKey) {
			return fmt.Errorf("foreign key column %s not found in target model %s", rel.Relationship.ForeignKey, targetModel.Name)
		}
		if !g.hasColumn(model, rel.Relationship.SourceKey) {
			return fmt.Errorf("source key column %s not found in model %s", rel.Relationship.SourceKey, model.Name)
		}

	case "has_many_through":
		// For now, we'll assume join tables are valid
		// In a real implementation, you'd validate the join table structure
	}

	return nil
}

// hasColumn checks if a model has a specific column
func (g *CodeGenerator) hasColumn(model *ModelMetadata, columnName string) bool {
	for _, field := range model.Columns {
		if field.DBName == columnName {
			return true
		}
	}
	return false
}

// GenerateForModel generates code for a specific model
func (g *CodeGenerator) GenerateForModel(modelName string) error {
	model, exists := g.models[modelName]
	if !exists {
		return fmt.Errorf("model %s not found", modelName)
	}

	// Load templates
	if err := g.loadTemplates(); err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	// Generate repository
	data := struct {
		Package string
		Model   *ModelMetadata
		Now     time.Time
	}{
		Package: g.packageName,
		Model:   model,
		Now:     time.Now(),
	}

	filename := fmt.Sprintf("%s_repository.go", toSnakeCase(model.Name))
	if err := g.executeTemplate("repository", filename, data); err != nil {
		return fmt.Errorf("failed to generate repository: %w", err)
	}

	// Generate query builder
	filename = fmt.Sprintf("%s_query.go", toSnakeCase(model.Name))
	if err := g.executeTemplate("query", filename, data); err != nil {
		return fmt.Errorf("failed to generate query builder: %w", err)
	}

	return nil
}

// CleanOutput removes all generated files
func (g *CodeGenerator) CleanOutput() error {
	// Remove all .go files in output directory that look generated
	return filepath.Walk(g.outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			// Check if file contains generation marker
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			if bytes.Contains(content, []byte("// Code generated by db-migrator")) {
				return os.Remove(path)
			}
		}

		return nil
	})
}

// sanitizeGoName converts database names to valid Go identifiers
func sanitizeGoName(name string) string {
	// Map of Go keywords that need to be escaped
	goKeywords := map[string]string{
		"type":      "type_",
		"interface": "interface_",
		"struct":    "struct_",
		"func":      "func_",
		"var":       "var_",
		"const":     "const_",
		"package":   "package_",
		"import":    "import_",
		"if":        "if_",
		"else":      "else_",
		"for":       "for_",
		"while":     "while_",
		"switch":    "switch_",
		"case":      "case_",
		"default":   "default_",
		"break":     "break_",
		"continue":  "continue_",
		"return":    "return_",
		"defer":     "defer_",
		"go":        "go_",
		"chan":      "chan_",
		"select":    "select_",
		"range":     "range_",
		"map":       "map_",
		"string":    "string_",
		"int":       "int_",
		"float":     "float_",
		"bool":      "bool_",
	}

	if escaped, isKeyword := goKeywords[strings.ToLower(name)]; isKeyword {
		return escaped
	}

	return name
}
