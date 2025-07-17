package orm

import (
	"fmt"
	"reflect"
	"strings"
)

// Relationship types with full type safety
type BelongsTo[TSource any, TTarget any, PK comparable] struct {
	targetRepo *Repository[TTarget]
	foreignKey string // Column in source that references target
	targetKey  string // Column in target (usually PK)
}

type HasOne[TSource any, TTarget any, PK comparable] struct {
	targetRepo *Repository[TTarget]
	foreignKey string // Column in target that references source
	sourceKey  string // Column in source (usually PK)
}

type HasMany[TSource any, TTarget any, PK comparable] struct {
	targetRepo *Repository[TTarget]
	foreignKey string // Column in target that references source
	sourceKey  string // Column in source (usually PK)
}

type HasManyThrough[TSource any, TTarget any, TJoin any, PK comparable] struct {
	targetRepo *Repository[TTarget]
	joinRepo   *Repository[TJoin]
	sourceFK   string // Column in join table referencing source
	targetFK   string // Column in join table referencing target
	sourceKey  string // Column in source (usually PK)
	targetKey  string // Column in target (usually PK)
}

// relationshipDef stores relationship metadata (internal use only)
type relationshipDef struct {
	Type       string       // "belongs_to", "has_many", "has_one", "has_many_through"
	Target     string       // Target table name
	ForeignKey string       // FK column
	SourceKey  string       // Source column (for has_many)
	TargetKey  string       // Target column (for belongs_to)
	FieldName  string       // Go struct field name
	FieldType  reflect.Type // Go field type
	JoinTable  string       // For has_many_through
	SourceFK   string       // For has_many_through
	TargetFK   string       // For has_many_through

	// Generated accessor functions for zero-reflection access
	SetValue func(model interface{}, value interface{}) // Set relationship value
	IsSlice  bool                                       // Whether this is a slice relationship
}

// include represents a relationship to eager load (internal use only)
type include struct {
	name       string
	conditions []Condition // Additional conditions for the relationship
	nested     []include   // Nested includes (e.g., "Author.Team")
}

// includeOption for relationship-specific conditions (internal use only)
type includeOption struct {
	name       string
	conditions []Condition
}

// relationshipManager handles parsing and storing relationships (internal use only)
type relationshipManager struct {
	relationships map[string]relationshipDef
	sourceTable   string // The table this manager belongs to
}

func newRelationshipManager(sourceTable string) *relationshipManager {
	return &relationshipManager{
		relationships: make(map[string]relationshipDef),
		sourceTable:   sourceTable,
	}
}

func (rm *relationshipManager) parseRelationships(structType reflect.Type) error {
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)

		if field.Tag.Get("db") != "-" {
			continue
		}

		ormTag := field.Tag.Get("orm")
		if ormTag == "" {
			continue
		}

		rel, err := rm.parseRelationshipTag(field, ormTag)
		if err != nil {
			return fmt.Errorf("failed to parse relationship for field %s: %w", field.Name, err)
		}

		rm.relationships[field.Name] = rel
	}

	return nil
}

func (rm *relationshipManager) parseRelationshipTag(field reflect.StructField, tag string) (relationshipDef, error) {
	parts := strings.Split(tag, ",")
	if len(parts) == 0 {
		return relationshipDef{}, fmt.Errorf("empty relationship tag")
	}

	typeAndTarget := strings.Split(parts[0], ":")
	if len(typeAndTarget) != 2 {
		return relationshipDef{}, fmt.Errorf("invalid relationship format, expected 'type:target'")
	}

	rel := relationshipDef{
		Type:      typeAndTarget[0],
		Target:    typeAndTarget[1],
		FieldName: field.Name,
		FieldType: field.Type,
	}

	for i := 1; i < len(parts); i++ {
		part := strings.TrimSpace(parts[i])
		if part == "" {
			continue
		}

		if strings.Contains(part, ":") {
			kv := strings.SplitN(part, ":", 2)
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])

			switch key {
			case "foreign_key":
				rel.ForeignKey = value
			case "source_key":
				rel.SourceKey = value
			case "target_key":
				rel.TargetKey = value
			case "join_table":
				rel.JoinTable = value
			case "source_fk":
				rel.SourceFK = value
			case "target_fk":
				rel.TargetFK = value
			default:
				return rel, fmt.Errorf("unknown relationship parameter: %s", key)
			}
		}
	}

	if err := rm.setRelationshipDefaults(&rel); err != nil {
		return rel, err
	}

	return rel, nil
}

func (rm *relationshipManager) setRelationshipDefaults(rel *relationshipDef) error {
	switch rel.Type {
	case "belongs_to":
		if rel.ForeignKey == "" {

			rel.ForeignKey = rm.toSnakeCase(rel.Target) + "_id"
		}
		if rel.TargetKey == "" {
			rel.TargetKey = "id"
		}

	case "has_one", "has_many":
		if rel.ForeignKey == "" {

			sourceTableSingular := rm.tableNameToSingular(rm.sourceTable)
			rel.ForeignKey = sourceTableSingular + "_id"
		}
		if rel.SourceKey == "" {
			rel.SourceKey = "id"
		}

	case "has_many_through":
		if rel.JoinTable == "" {
			return fmt.Errorf("join_table is required for has_many_through relationships")
		}
		if rel.SourceFK == "" {
			sourceTableSingular := rm.tableNameToSingular(rm.sourceTable)
			rel.SourceFK = sourceTableSingular + "_id"
		}
		if rel.TargetFK == "" {
			targetTableSingular := rm.tableNameToSingular(rel.Target)
			rel.TargetFK = targetTableSingular + "_id"
		}
		if rel.SourceKey == "" {
			rel.SourceKey = "id"
		}
		if rel.TargetKey == "" {
			rel.TargetKey = "id"
		}

	default:
		return fmt.Errorf("unknown relationship type: %s", rel.Type)
	}

	return nil
}

func (rm *relationshipManager) toSnakeCase(s string) string {
	var result strings.Builder

	for i, r := range s {
		if i > 0 && (r >= 'A' && r <= 'Z') {
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

func (rm *relationshipManager) tableNameToSingular(tableName string) string {

	if strings.HasSuffix(tableName, "ies") {
		return tableName[:len(tableName)-3] + "y"
	}
	if strings.HasSuffix(tableName, "es") && !strings.HasSuffix(tableName, "ses") {
		return tableName[:len(tableName)-2]
	}
	if strings.HasSuffix(tableName, "s") {
		return tableName[:len(tableName)-1]
	}
	return tableName
}

func (rm *relationshipManager) getRelationships() map[string]relationshipDef {
	return rm.relationships
}

func (rm *relationshipManager) getRelationship(fieldName string) *relationshipDef {
	rel, exists := rm.relationships[fieldName]
	if !exists {
		return nil
	}
	return &rel
}

func (rm *relationshipManager) hasRelationships() bool {
	return len(rm.relationships) > 0
}

// Note: RelationshipQuery has been merged into the main Query type
// Relationship loading is now available directly on Query[T] via Include() and IncludeWhere()
