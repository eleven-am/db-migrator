package generator

import (
	"fmt"
	"strings"

	"github.com/eleven-am/db-migrator/internal/introspect"
	"github.com/eleven-am/db-migrator/internal/parser"
)

// EnhancedGenerator provides signature-based schema comparison and generation
type EnhancedGenerator struct {
	structParser *parser.StructParser
	normalizer   *introspect.SQLNormalizer
}

// NewEnhancedGenerator creates a new enhanced generator
func NewEnhancedGenerator() *EnhancedGenerator {
	return &EnhancedGenerator{
		structParser: parser.NewStructParser(),
		normalizer:   introspect.NewSQLNormalizer(),
	}
}

// IndexDefinition represents an enhanced index definition with signature-based matching
type IndexDefinition struct {
	Name       string
	TableName  string
	Columns    []string
	IsUnique   bool
	IsPrimary  bool
	Method     string // btree, hash, gist, etc.
	Where      string // partial index condition
	Definition string // full CREATE INDEX statement
	Signature  string // computed signature for comparison
}

// ForeignKeyDefinition represents an enhanced foreign key definition
type ForeignKeyDefinition struct {
	Name              string
	TableName         string
	Columns           []string
	ReferencedTable   string
	ReferencedColumns []string
	OnDelete          string
	OnUpdate          string
	Definition        string // full constraint definition
	Signature         string // computed signature for comparison
}

// GenerateIndexDefinitions creates IndexDefinition structs from parsed Go structs
func (g *EnhancedGenerator) GenerateIndexDefinitions(tableDefn parser.TableDefinition) ([]IndexDefinition, error) {
	var indexes []IndexDefinition

	primaryKey := g.findPrimaryKey(tableDefn)
	if primaryKey != "" {
		pkIndex := IndexDefinition{
			Name:      fmt.Sprintf("%s_pkey", tableDefn.TableName),
			TableName: tableDefn.TableName,
			Columns:   []string{primaryKey},
			IsUnique:  true,
			IsPrimary: true,
			Method:    "btree",
		}
		pkIndex.Signature = g.normalizer.GenerateCanonicalSignature(
			pkIndex.TableName,
			pkIndex.Columns,
			pkIndex.IsUnique,
			pkIndex.IsPrimary,
			pkIndex.Method,
			pkIndex.Where,
		)
		indexes = append(indexes, pkIndex)
	}

	for _, field := range tableDefn.Fields {
		if g.hasAttribute(field.DBDef, "unique") {
			uniqueIndex := IndexDefinition{
				Name:      fmt.Sprintf("idx_%s_%s", tableDefn.TableName, field.DBName),
				TableName: tableDefn.TableName,
				Columns:   []string{field.DBName},
				IsUnique:  true,
				IsPrimary: false,
				Method:    "btree",
			}
			uniqueIndex.Signature = g.normalizer.GenerateCanonicalSignature(
				uniqueIndex.TableName,
				uniqueIndex.Columns,
				uniqueIndex.IsUnique,
				uniqueIndex.IsPrimary,
				uniqueIndex.Method,
				uniqueIndex.Where,
			)
			indexes = append(indexes, uniqueIndex)
		}
	}

	for key, value := range tableDefn.TableLevel {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		var isUnique bool
		switch key {
		case "index":
			isUnique = false
		case "unique":
			isUnique = true
		default:
			continue
		}

		definitions := strings.Split(value, ";")
		for _, defStr := range definitions {
			defStr = strings.TrimSpace(defStr)
			if defStr == "" {
				continue
			}

			indexDef, err := g.parseTableLevelIndex(tableDefn.TableName, defStr, isUnique)
			if err != nil {
				return nil, fmt.Errorf("failed to parse table-level %s definition '%s': %w", key, defStr, err)
			}

			indexDef.Signature = g.normalizer.GenerateCanonicalSignature(
				indexDef.TableName,
				indexDef.Columns,
				indexDef.IsUnique,
				indexDef.IsPrimary,
				indexDef.Method,
				indexDef.Where,
			)
			indexes = append(indexes, indexDef)
		}
	}

	return indexes, nil
}

// GenerateForeignKeyDefinitions creates ForeignKeyDefinition structs from parsed Go structs
func (g *EnhancedGenerator) GenerateForeignKeyDefinitions(tableDefn parser.TableDefinition) ([]ForeignKeyDefinition, error) {
	var foreignKeys []ForeignKeyDefinition

	for _, field := range tableDefn.Fields {
		if foreignKeyRef, exists := field.DBDef["foreign_key"]; exists {
			parts := strings.Split(foreignKeyRef, ".")
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid foreign key format %s, expected 'table.column'", foreignKeyRef)
			}

			fkDef := ForeignKeyDefinition{
				Name:              fmt.Sprintf("%s_%s_fkey", tableDefn.TableName, field.DBName),
				TableName:         tableDefn.TableName,
				Columns:           []string{field.DBName},
				ReferencedTable:   parts[0],
				ReferencedColumns: []string{parts[1]},
				OnDelete:          g.getStringOrDefault(field.DBDef, "on_delete", "NO ACTION"),
				OnUpdate:          g.getStringOrDefault(field.DBDef, "on_update", "NO ACTION"),
			}

			fkDef.Definition = fmt.Sprintf("FOREIGN KEY (%s) REFERENCES %s(%s)",
				strings.Join(fkDef.Columns, ", "),
				fkDef.ReferencedTable,
				strings.Join(fkDef.ReferencedColumns, ", "))

			if fkDef.OnDelete != "NO ACTION" {
				fkDef.Definition += fmt.Sprintf(" ON DELETE %s", fkDef.OnDelete)
			}
			if fkDef.OnUpdate != "NO ACTION" {
				fkDef.Definition += fmt.Sprintf(" ON UPDATE %s", fkDef.OnUpdate)
			}

			fkDef.Signature = g.generateForeignKeySignature(fkDef)
			foreignKeys = append(foreignKeys, fkDef)
		}
	}

	return foreignKeys, nil
}

// CompareSchemas performs signature-based comparison between struct-generated and database schemas
func (g *EnhancedGenerator) CompareSchemas(
	structIndexes []IndexDefinition,
	structForeignKeys []ForeignKeyDefinition,
	dbIndexes []IndexDefinition,
	dbForeignKeys []ForeignKeyDefinition,
) (*SchemaComparison, error) {

	comparison := &SchemaComparison{
		IndexesToCreate:     []IndexDefinition{},
		IndexesToDrop:       []IndexDefinition{},
		ForeignKeysToCreate: []ForeignKeyDefinition{},
		ForeignKeysToDrop:   []ForeignKeyDefinition{},
	}

	fmt.Println("\n=== DEBUG: Index Signatures ===")
	fmt.Println("STRUCT INDEXES:")
	for _, idx := range structIndexes {
		fmt.Printf("  %s -> %s\n", idx.Name, idx.Signature)
		fmt.Printf("    Columns: %v, Unique: %t, Primary: %t, Method: %s, Where: %s\n",
			idx.Columns, idx.IsUnique, idx.IsPrimary, idx.Method, idx.Where)
	}
	fmt.Println("DATABASE INDEXES:")
	for _, idx := range dbIndexes {
		fmt.Printf("  %s -> %s\n", idx.Name, idx.Signature)
		fmt.Printf("    Columns: %v, Unique: %t, Primary: %t, Method: %s, Where: %s\n",
			idx.Columns, idx.IsUnique, idx.IsPrimary, idx.Method, idx.Where)
	}
	fmt.Println("=== END DEBUG ===")

	dbIndexSigs := make(map[string]IndexDefinition)
	for _, idx := range dbIndexes {
		dbIndexSigs[idx.Signature] = idx
	}

	dbFKSigs := make(map[string]ForeignKeyDefinition)
	for _, fk := range dbForeignKeys {
		dbFKSigs[fk.Signature] = fk
	}

	structIndexSigs := make(map[string]IndexDefinition)
	for _, idx := range structIndexes {
		structIndexSigs[idx.Signature] = idx
	}

	structFKSigs := make(map[string]ForeignKeyDefinition)
	for _, fk := range structForeignKeys {
		structFKSigs[fk.Signature] = fk
	}

	for sig, structIdx := range structIndexSigs {
		if _, exists := dbIndexSigs[sig]; !exists {
			comparison.IndexesToCreate = append(comparison.IndexesToCreate, structIdx)
		}
	}

	for sig, dbIdx := range dbIndexSigs {
		if _, exists := structIndexSigs[sig]; !exists {
			comparison.IndexesToDrop = append(comparison.IndexesToDrop, dbIdx)
		}
	}

	for sig, structFK := range structFKSigs {
		if _, exists := dbFKSigs[sig]; !exists {
			comparison.ForeignKeysToCreate = append(comparison.ForeignKeysToCreate, structFK)
		}
	}

	for sig, dbFK := range dbFKSigs {
		if _, exists := structFKSigs[sig]; !exists {
			comparison.ForeignKeysToDrop = append(comparison.ForeignKeysToDrop, dbFK)
		}
	}

	return comparison, nil
}

// SchemaComparison holds the results of schema comparison
type SchemaComparison struct {
	IndexesToCreate     []IndexDefinition
	IndexesToDrop       []IndexDefinition
	ForeignKeysToCreate []ForeignKeyDefinition
	ForeignKeysToDrop   []ForeignKeyDefinition
}

// Helper methods

func (g *EnhancedGenerator) findPrimaryKey(tableDefn parser.TableDefinition) string {
	for _, field := range tableDefn.Fields {
		if g.hasAttribute(field.DBDef, "primary_key") {
			return field.DBName
		}
	}
	return ""
}

func (g *EnhancedGenerator) hasAttribute(dbDef map[string]string, attr string) bool {
	_, exists := dbDef[attr]
	return exists
}

func (g *EnhancedGenerator) getStringOrDefault(dbDef map[string]string, key, defaultValue string) string {
	if value, exists := dbDef[key]; exists {
		return value
	}
	return defaultValue
}

// parseTableLevelIndex correctly parses an index or unique constraint definition string.
// It accepts an `isUnique` flag and correctly handles an optional " where:" clause.
func (g *EnhancedGenerator) parseTableLevelIndex(tableName, indexDefStr string, isUnique bool) (IndexDefinition, error) {
	indexDef := IndexDefinition{
		TableName: tableName,
		IsUnique:  isUnique,
		IsPrimary: false,
		Method:    "btree",
	}

	mainDef := indexDefStr
	whereClause := ""
	if whereParts := strings.SplitN(mainDef, " where:", 2); len(whereParts) == 2 {
		mainDef = strings.TrimSpace(whereParts[0])
		whereClause = strings.TrimSpace(whereParts[1])
	}
	indexDef.Where = whereClause

	parts := strings.Split(mainDef, ",")
	if len(parts) < 2 {
		return IndexDefinition{}, fmt.Errorf("malformed index/unique definition: must have a name and at least one column in %q", indexDefStr)
	}

	indexDef.Name = strings.TrimSpace(parts[0])
	if indexDef.Name == "" {
		return IndexDefinition{}, fmt.Errorf("malformed index/unique definition: name is missing in %q", indexDefStr)
	}

	var columns []string
	for _, col := range parts[1:] {
		trimmedCol := strings.TrimSpace(col)
		if trimmedCol != "" {
			columns = append(columns, trimmedCol)
		}
	}

	if len(columns) == 0 {
		return IndexDefinition{}, fmt.Errorf("malformed index/unique definition: no columns specified for %q", indexDef.Name)
	}
	indexDef.Columns = columns

	return indexDef, nil
}

func (g *EnhancedGenerator) generateIndexSignature(idx IndexDefinition) string {
	return g.normalizer.GenerateCanonicalSignature(
		idx.TableName,
		idx.Columns,
		idx.IsUnique,
		idx.IsPrimary,
		idx.Method,
		idx.Where,
	)
}

func (g *EnhancedGenerator) generateForeignKeySignature(fk ForeignKeyDefinition) string {
	var parts []string

	parts = append(parts, "table:"+strings.ToLower(strings.TrimSpace(fk.TableName)))
	normalizedCols := g.normalizer.NormalizeColumnList(fk.Columns, true)
	parts = append(parts, "cols:"+strings.Join(normalizedCols, ","))

	parts = append(parts, "ref_table:"+strings.ToLower(strings.TrimSpace(fk.ReferencedTable)))
	normalizedRefCols := g.normalizer.NormalizeColumnList(fk.ReferencedColumns, true)
	parts = append(parts, "ref_cols:"+strings.Join(normalizedRefCols, ","))

	onDelete := strings.ToUpper(strings.TrimSpace(fk.OnDelete))
	if onDelete == "" {
		onDelete = "NO ACTION"
	}
	onUpdate := strings.ToUpper(strings.TrimSpace(fk.OnUpdate))
	if onUpdate == "" {
		onUpdate = "NO ACTION"
	}

	parts = append(parts, "on_delete:"+onDelete)
	parts = append(parts, "on_update:"+onUpdate)

	return strings.Join(parts, "|")
}

// IsSafeOperation determines if a schema change operation is safe to perform automatically
func (g *EnhancedGenerator) IsSafeOperation(comparison *SchemaComparison) bool {
	if len(comparison.ForeignKeysToDrop) > 0 {
		return false
	}

	for _, idx := range comparison.IndexesToDrop {
		if idx.IsUnique || idx.IsPrimary {
			return false
		}
	}

	return true
}

// GenerateSafeSQL generates SQL statements only for safe operations
func (g *EnhancedGenerator) GenerateSafeSQL(comparison *SchemaComparison, allowDestructive bool) ([]string, []string, error) {
	var upStatements []string
	var downStatements []string

	for _, idx := range comparison.IndexesToCreate {
		upSQL := g.generateCreateIndexSQL(idx)
		downSQL := fmt.Sprintf("DROP INDEX IF EXISTS %s;", idx.Name)

		upStatements = append(upStatements, upSQL)
		downStatements = append(downStatements, downSQL)
	}

	for _, fk := range comparison.ForeignKeysToCreate {
		upSQL := g.generateCreateForeignKeySQL(fk)
		downSQL := fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s;", fk.TableName, fk.Name)

		upStatements = append(upStatements, upSQL)
		downStatements = append(downStatements, downSQL)
	}

	if allowDestructive {
		for _, idx := range comparison.IndexesToDrop {
			upSQL := fmt.Sprintf("DROP INDEX IF EXISTS %s;", idx.Name)
			downSQL := g.generateCreateIndexSQL(idx)

			upStatements = append(upStatements, upSQL)
			downStatements = append(downStatements, downSQL)
		}

		for _, fk := range comparison.ForeignKeysToDrop {
			upSQL := fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s;", fk.TableName, fk.Name)
			downSQL := g.generateCreateForeignKeySQL(fk)

			upStatements = append(upStatements, upSQL)
			downStatements = append(downStatements, downSQL)
		}
	}

	for i := len(downStatements)/2 - 1; i >= 0; i-- {
		opp := len(downStatements) - 1 - i
		downStatements[i], downStatements[opp] = downStatements[opp], downStatements[i]
	}

	return upStatements, downStatements, nil
}

func (g *EnhancedGenerator) generateCreateIndexSQL(idx IndexDefinition) string {
	var parts []string

	if idx.IsUnique {
		parts = append(parts, "CREATE UNIQUE INDEX")
	} else {
		parts = append(parts, "CREATE INDEX")
	}

	parts = append(parts, idx.Name)
	parts = append(parts, "ON")
	parts = append(parts, idx.TableName)

	if idx.Method != "" && idx.Method != "btree" {
		parts = append(parts, fmt.Sprintf("USING %s", idx.Method))
	}

	parts = append(parts, fmt.Sprintf("(%s)", strings.Join(idx.Columns, ", ")))

	if idx.Where != "" {
		whereClause := idx.Where
		if !strings.HasPrefix(strings.ToUpper(whereClause), "WHERE") {
			whereClause = fmt.Sprintf("WHERE (%s)", whereClause)
		}
		parts = append(parts, whereClause)
	}

	return strings.Join(parts, " ") + ";"
}

func (g *EnhancedGenerator) generateCreateForeignKeySQL(fk ForeignKeyDefinition) string {
	sql := fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s)",
		fk.TableName,
		fk.Name,
		strings.Join(fk.Columns, ", "),
		fk.ReferencedTable,
		strings.Join(fk.ReferencedColumns, ", "))

	if fk.OnDelete != "" && fk.OnDelete != "NO ACTION" {
		sql += fmt.Sprintf(" ON DELETE %s", fk.OnDelete)
	}
	if fk.OnUpdate != "" && fk.OnUpdate != "NO ACTION" {
		sql += fmt.Sprintf(" ON UPDATE %s", fk.OnUpdate)
	}

	return sql + ";"
}
