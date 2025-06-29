package diff

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/eleven-am/db-migrator/internal/generator"
)

// Engine performs schema comparison and generates migration changes
type Engine struct {
	oldSchema *generator.DatabaseSchema
	newSchema *generator.DatabaseSchema
	hints     []RenameHint
}

// NewEngine creates a new diff engine
func NewEngine(oldSchema, newSchema *generator.DatabaseSchema) *Engine {
	return &Engine{
		oldSchema: oldSchema,
		newSchema: newSchema,
		hints:     make([]RenameHint, 0),
	}
}

// AddRenameHint adds a hint for rename detection
func (e *Engine) AddRenameHint(oldName, newName, hintType string) {
	e.hints = append(e.hints, RenameHint{
		OldName: oldName,
		NewName: newName,
		Type:    hintType,
	})
}

// Compare performs the schema comparison
func (e *Engine) Compare() (*DiffResult, error) {
	result := &DiffResult{
		Changes: make([]Change, 0),
	}

	e.extractRenameHints()

	tableDiffs := e.compareTables()

	for _, diff := range tableDiffs {
		changes := e.processTableDiff(diff)
		result.Changes = append(result.Changes, changes...)
	}

	for _, change := range result.Changes {
		if change.IsUnsafe {
			result.HasUnsafeChanges = true
			break
		}
	}

	result.Summary = e.generateSummary(result)

	return result, nil
}

// extractRenameHints looks for rename hints in the new schema
func (e *Engine) extractRenameHints() {
}

// compareTables compares all tables between schemas
func (e *Engine) compareTables() []TableDiff {
	diffs := make([]TableDiff, 0)

	tableNames := e.newSchema.GetTableNames()

	for _, tableName := range tableNames {
		newTable := e.newSchema.Tables[tableName]
		oldTable, exists := e.oldSchema.Tables[tableName]

		if !exists {
			if renamed := e.checkTableRenamed(tableName); renamed != nil {
				diffs = append(diffs, *renamed)
			} else {
				diffs = append(diffs, TableDiff{
					TableName: tableName,
					NewTable:  &newTable,
				})
			}
		} else {
			if diff := e.compareTable(oldTable, newTable); diff != nil {
				diffs = append(diffs, *diff)
			}
		}
	}

	for tableName, oldTable := range e.oldSchema.Tables {
		if _, exists := e.newSchema.Tables[tableName]; !exists {
			wasRenamed := false
			for _, hint := range e.hints {
				if hint.Type == "table" && hint.OldName == tableName {
					wasRenamed = true
					break
				}
			}

			if !wasRenamed {
				diffs = append(diffs, TableDiff{
					TableName: tableName,
					OldTable:  &oldTable,
				})
			}
		}
	}

	return diffs
}

// checkTableRenamed checks if a table was renamed
func (e *Engine) checkTableRenamed(newTableName string) *TableDiff {
	for _, hint := range e.hints {
		if hint.Type == "table" && hint.NewName == newTableName {
			if oldTable, exists := e.oldSchema.Tables[hint.OldName]; exists {
				newTable := e.newSchema.Tables[newTableName]
				return &TableDiff{
					TableName: newTableName,
					OldTable:  &oldTable,
					NewTable:  &newTable,
					IsRenamed: true,
					OldName:   hint.OldName,
				}
			}
		}
	}
	return nil
}

// compareTable compares two table schemas
func (e *Engine) compareTable(oldTable, newTable generator.SchemaTable) *TableDiff {
	diff := &TableDiff{
		TableName:         newTable.Name,
		OldTable:          &oldTable,
		NewTable:          &newTable,
		ColumnChanges:     make([]ColumnChange, 0),
		IndexChanges:      make([]IndexChange, 0),
		ConstraintChanges: make([]ConstraintChange, 0),
	}

	hasChanges := false

	columnChanges := e.compareColumns(oldTable, newTable)
	if len(columnChanges) > 0 {
		diff.ColumnChanges = columnChanges
		hasChanges = true
	}

	indexChanges := e.compareIndexes(oldTable, newTable)
	if len(indexChanges) > 0 {
		diff.IndexChanges = indexChanges
		hasChanges = true
	}

	constraintChanges := e.compareConstraints(oldTable, newTable)
	if len(constraintChanges) > 0 {
		diff.ConstraintChanges = constraintChanges
		hasChanges = true
	}

	if hasChanges {
		return diff
	}
	return nil
}

// compareColumns compares columns between two tables
func (e *Engine) compareColumns(oldTable, newTable generator.SchemaTable) []ColumnChange {
	changes := make([]ColumnChange, 0)

	oldColumns := make(map[string]generator.SchemaColumn)
	for _, col := range oldTable.Columns {
		oldColumns[col.Name] = col
	}

	newColumns := make(map[string]generator.SchemaColumn)
	for _, col := range newTable.Columns {
		newColumns[col.Name] = col
	}

	for _, newCol := range newTable.Columns {
		if oldCol, exists := oldColumns[newCol.Name]; exists {
			if !e.columnsEqual(oldCol, newCol) {
				changes = append(changes, ColumnChange{
					Type:       ChangeTypeAlterColumn,
					ColumnName: newCol.Name,
					OldColumn:  &oldCol,
					NewColumn:  &newCol,
				})
			}
		} else {
			if renamed := e.checkColumnRenamed(oldTable.Name, newCol.Name); renamed != nil {
				changes = append(changes, *renamed)
			} else {
				changes = append(changes, ColumnChange{
					Type:       ChangeTypeAddColumn,
					ColumnName: newCol.Name,
					NewColumn:  &newCol,
				})
			}
		}
	}

	for _, oldCol := range oldTable.Columns {
		if _, exists := newColumns[oldCol.Name]; !exists {
			wasRenamed := false
			for _, hint := range e.hints {
				if hint.Type == "column" && hint.OldName == oldCol.Name {
					wasRenamed = true
					break
				}
			}

			if !wasRenamed {
				changes = append(changes, ColumnChange{
					Type:       ChangeTypeDropColumn,
					ColumnName: oldCol.Name,
					OldColumn:  &oldCol,
				})
			}
		}
	}

	return changes
}

// checkColumnRenamed checks if a column was renamed
func (e *Engine) checkColumnRenamed(tableName, newColumnName string) *ColumnChange {
	for _, hint := range e.hints {
		if hint.Type == "column" && hint.NewName == newColumnName {
			if oldTable, exists := e.oldSchema.Tables[tableName]; exists {
				for _, oldCol := range oldTable.Columns {
					if oldCol.Name == hint.OldName {
						newTable := e.newSchema.Tables[tableName]
						for _, newCol := range newTable.Columns {
							if newCol.Name == newColumnName {
								return &ColumnChange{
									Type:       ChangeTypeRenameColumn,
									ColumnName: newColumnName,
									OldColumn:  &oldCol,
									NewColumn:  &newCol,
									IsRenamed:  true,
									OldName:    hint.OldName,
								}
							}
						}
					}
				}
			}
		}
	}
	return nil
}

// columnsEqual checks if two columns are identical
func (e *Engine) columnsEqual(col1, col2 generator.SchemaColumn) bool {
	if col1.Type != col2.Type {
		return false
	}
	if col1.IsNullable != col2.IsNullable {
		return false
	}
	if col1.IsPrimaryKey != col2.IsPrimaryKey {
		return false
	}
	if col1.IsUnique != col2.IsUnique {
		return false
	}
	if col1.IsAutoIncrement != col2.IsAutoIncrement {
		return false
	}

	if (col1.DefaultValue == nil) != (col2.DefaultValue == nil) {
		return false
	}
	if col1.DefaultValue != nil && *col1.DefaultValue != *col2.DefaultValue {
		return false
	}

	if (col1.ForeignKey == nil) != (col2.ForeignKey == nil) {
		return false
	}
	if col1.ForeignKey != nil {
		if col1.ForeignKey.ReferencedTable != col2.ForeignKey.ReferencedTable ||
			col1.ForeignKey.ReferencedColumn != col2.ForeignKey.ReferencedColumn ||
			col1.ForeignKey.OnDelete != col2.ForeignKey.OnDelete ||
			col1.ForeignKey.OnUpdate != col2.ForeignKey.OnUpdate {
			return false
		}
	}

	if (col1.CheckConstraint == nil) != (col2.CheckConstraint == nil) {
		return false
	}
	if col1.CheckConstraint != nil && *col1.CheckConstraint != *col2.CheckConstraint {
		return false
	}

	return true
}

// compareIndexes compares indexes between two tables
func (e *Engine) compareIndexes(oldTable, newTable generator.SchemaTable) []IndexChange {
	changes := make([]IndexChange, 0)

	oldIndexes := make(map[string]generator.SchemaIndex)
	for _, idx := range oldTable.Indexes {
		oldIndexes[idx.Name] = idx
	}

	newIndexes := make(map[string]generator.SchemaIndex)
	for _, idx := range newTable.Indexes {
		newIndexes[idx.Name] = idx
	}

	for _, newIdx := range newTable.Indexes {
		if oldIdx, exists := oldIndexes[newIdx.Name]; exists {
			if !e.indexesEqual(oldIdx, newIdx) {
				changes = append(changes, IndexChange{
					Type:      ChangeTypeAlterIndex,
					IndexName: newIdx.Name,
					OldIndex:  &oldIdx,
					NewIndex:  &newIdx,
				})
			}
		} else {
			changes = append(changes, IndexChange{
				Type:      ChangeTypeCreateIndex,
				IndexName: newIdx.Name,
				NewIndex:  &newIdx,
			})
		}
	}

	for _, oldIdx := range oldTable.Indexes {
		if _, exists := newIndexes[oldIdx.Name]; !exists {
			changes = append(changes, IndexChange{
				Type:      ChangeTypeDropIndex,
				IndexName: oldIdx.Name,
				OldIndex:  &oldIdx,
			})
		}
	}

	return changes
}

// indexesEqual checks if two indexes are identical
func (e *Engine) indexesEqual(idx1, idx2 generator.SchemaIndex) bool {
	if idx1.IsUnique != idx2.IsUnique {
		return false
	}
	if idx1.IsPrimary != idx2.IsPrimary {
		return false
	}
	if idx1.Type != idx2.Type {
		return false
	}
	if idx1.Where != idx2.Where {
		return false
	}
	if !reflect.DeepEqual(idx1.Columns, idx2.Columns) {
		return false
	}
	return true
}

// compareConstraints compares constraints between two tables
func (e *Engine) compareConstraints(oldTable, newTable generator.SchemaTable) []ConstraintChange {
	changes := make([]ConstraintChange, 0)

	oldConstraints := make(map[string]generator.SchemaConstraint)
	for _, con := range oldTable.Constraints {
		oldConstraints[con.Name] = con
	}

	newConstraints := make(map[string]generator.SchemaConstraint)
	for _, con := range newTable.Constraints {
		newConstraints[con.Name] = con
	}

	for _, newCon := range newTable.Constraints {
		if oldCon, exists := oldConstraints[newCon.Name]; exists {
			if !e.constraintsEqual(oldCon, newCon) {
				changes = append(changes, ConstraintChange{
					Type:           ChangeTypeDropConstraint,
					ConstraintName: oldCon.Name,
					OldConstraint:  &oldCon,
				})
				changes = append(changes, ConstraintChange{
					Type:           ChangeTypeAddConstraint,
					ConstraintName: newCon.Name,
					NewConstraint:  &newCon,
				})
			}
		} else {
			changes = append(changes, ConstraintChange{
				Type:           ChangeTypeAddConstraint,
				ConstraintName: newCon.Name,
				NewConstraint:  &newCon,
			})
		}
	}

	for _, oldCon := range oldTable.Constraints {
		if _, exists := newConstraints[oldCon.Name]; !exists {
			changes = append(changes, ConstraintChange{
				Type:           ChangeTypeDropConstraint,
				ConstraintName: oldCon.Name,
				OldConstraint:  &oldCon,
			})
		}
	}

	return changes
}

// constraintsEqual checks if two constraints are identical
func (e *Engine) constraintsEqual(con1, con2 generator.SchemaConstraint) bool {
	if con1.Type != con2.Type {
		return false
	}
	if con1.Definition != con2.Definition {
		return false
	}
	if !reflect.DeepEqual(con1.Columns, con2.Columns) {
		return false
	}
	return true
}

// processTableDiff converts a table diff to changes
func (e *Engine) processTableDiff(diff TableDiff) []Change {
	changes := make([]Change, 0)

	if diff.OldTable == nil && diff.NewTable != nil {
		changes = append(changes, Change{
			Type:      ChangeTypeCreateTable,
			TableName: diff.TableName,
			NewValue:  diff.NewTable,
		})
	} else if diff.OldTable != nil && diff.NewTable == nil {
		changes = append(changes, Change{
			Type:        ChangeTypeDropTable,
			TableName:   diff.TableName,
			OldValue:    diff.OldTable,
			IsUnsafe:    true,
			SafetyNotes: "Dropping table will permanently delete all data",
		})
	} else if diff.IsRenamed {
		changes = append(changes, Change{
			Type:      ChangeTypeRenameTable,
			TableName: diff.OldName,
			NewValue:  diff.TableName,
		})
	}

	for _, colChange := range diff.ColumnChanges {
		change := e.processColumnChange(diff.TableName, colChange)
		changes = append(changes, change)
	}

	for _, idxChange := range diff.IndexChanges {
		change := e.processIndexChange(diff.TableName, idxChange)
		changes = append(changes, change)
	}

	for _, conChange := range diff.ConstraintChanges {
		change := e.processConstraintChange(diff.TableName, conChange)
		changes = append(changes, change)
	}

	return changes
}

// processColumnChange converts a column change to a Change
func (e *Engine) processColumnChange(tableName string, colChange ColumnChange) Change {
	change := Change{
		TableName:  tableName,
		ColumnName: colChange.ColumnName,
	}

	switch colChange.Type {
	case ChangeTypeAddColumn:
		change.Type = ChangeTypeAddColumn
		change.NewValue = colChange.NewColumn

	case ChangeTypeDropColumn:
		change.Type = ChangeTypeDropColumn
		change.OldValue = colChange.OldColumn
		change.IsUnsafe = true
		change.SafetyNotes = "Dropping column will permanently delete all data in this column"

	case ChangeTypeAlterColumn:
		change.Type = ChangeTypeAlterColumn
		change.OldValue = colChange.OldColumn
		change.NewValue = colChange.NewColumn

		if e.isUnsafeColumnChange(colChange.OldColumn, colChange.NewColumn) {
			change.IsUnsafe = true
			change.SafetyNotes = e.getColumnChangeSafetyNotes(colChange.OldColumn, colChange.NewColumn)
		}

	case ChangeTypeRenameColumn:
		change.Type = ChangeTypeRenameColumn
		change.OldValue = colChange.OldName
		change.NewValue = colChange.ColumnName
	}

	return change
}

// isUnsafeColumnChange checks if a column change could cause data loss
func (e *Engine) isUnsafeColumnChange(oldCol, newCol *generator.SchemaColumn) bool {
	if oldCol.Type != newCol.Type {
		if e.isUnsafeTypeChange(oldCol.Type, newCol.Type) {
			return true
		}
	}

	if oldCol.IsNullable && !newCol.IsNullable && newCol.DefaultValue == nil {
		return true
	}

	return false
}

// isUnsafeTypeChange checks if changing from one type to another could lose data
func (e *Engine) isUnsafeTypeChange(oldType, newType string) bool {
	safeConversions := map[string][]string{
		"VARCHAR":   {"TEXT"},
		"CHAR":      {"VARCHAR", "TEXT"},
		"SMALLINT":  {"INTEGER", "BIGINT"},
		"INTEGER":   {"BIGINT"},
		"REAL":      {"DOUBLE PRECISION"},
		"TIMESTAMP": {"TIMESTAMPTZ"},
	}

	oldType = strings.ToUpper(strings.Split(oldType, "(")[0])
	newType = strings.ToUpper(strings.Split(newType, "(")[0])

	if oldType == newType {
		return false
	}

	if safeTypes, exists := safeConversions[oldType]; exists {
		for _, safe := range safeTypes {
			if safe == newType {
				return false
			}
		}
	}

	return true
}

// getColumnChangeSafetyNotes generates safety notes for column changes
func (e *Engine) getColumnChangeSafetyNotes(oldCol, newCol *generator.SchemaColumn) string {
	notes := []string{}

	if oldCol.Type != newCol.Type {
		notes = append(notes, fmt.Sprintf("Type change from %s to %s may cause data loss", oldCol.Type, newCol.Type))
	}

	if oldCol.IsNullable && !newCol.IsNullable && newCol.DefaultValue == nil {
		notes = append(notes, "Making column NOT NULL without default value will fail if NULL values exist")
	}

	return strings.Join(notes, "; ")
}

// processIndexChange converts an index change to a Change
func (e *Engine) processIndexChange(tableName string, idxChange IndexChange) Change {
	change := Change{
		TableName: tableName,
	}

	switch idxChange.Type {
	case ChangeTypeCreateIndex:
		change.Type = ChangeTypeCreateIndex
		change.NewValue = idxChange.NewIndex

	case ChangeTypeDropIndex:
		change.Type = ChangeTypeDropIndex
		change.OldValue = idxChange.OldIndex

	case ChangeTypeAlterIndex:
		change.Type = ChangeTypeDropIndex
		change.OldValue = idxChange.OldIndex
	}

	return change
}

// processConstraintChange converts a constraint change to a Change
func (e *Engine) processConstraintChange(tableName string, conChange ConstraintChange) Change {
	change := Change{
		TableName: tableName,
	}

	switch conChange.Type {
	case ChangeTypeAddConstraint:
		change.Type = ChangeTypeAddConstraint
		change.NewValue = conChange.NewConstraint

	case ChangeTypeDropConstraint:
		change.Type = ChangeTypeDropConstraint
		change.OldValue = conChange.OldConstraint
	}

	return change
}

// generateSummary creates a human-readable summary of changes
func (e *Engine) generateSummary(result *DiffResult) string {
	counts := make(map[ChangeType]int)
	for _, change := range result.Changes {
		counts[change.Type]++
	}

	parts := []string{}
	if c := counts[ChangeTypeCreateTable]; c > 0 {
		parts = append(parts, fmt.Sprintf("%d new table(s)", c))
	}
	if c := counts[ChangeTypeDropTable]; c > 0 {
		parts = append(parts, fmt.Sprintf("%d dropped table(s)", c))
	}
	if c := counts[ChangeTypeRenameTable]; c > 0 {
		parts = append(parts, fmt.Sprintf("%d renamed table(s)", c))
	}
	if c := counts[ChangeTypeAddColumn]; c > 0 {
		parts = append(parts, fmt.Sprintf("%d new column(s)", c))
	}
	if c := counts[ChangeTypeDropColumn]; c > 0 {
		parts = append(parts, fmt.Sprintf("%d dropped column(s)", c))
	}
	if c := counts[ChangeTypeAlterColumn]; c > 0 {
		parts = append(parts, fmt.Sprintf("%d altered column(s)", c))
	}
	if c := counts[ChangeTypeRenameColumn]; c > 0 {
		parts = append(parts, fmt.Sprintf("%d renamed column(s)", c))
	}
	if c := counts[ChangeTypeCreateIndex]; c > 0 {
		parts = append(parts, fmt.Sprintf("%d new index(es)", c))
	}
	if c := counts[ChangeTypeDropIndex]; c > 0 {
		parts = append(parts, fmt.Sprintf("%d dropped index(es)", c))
	}

	if len(parts) == 0 {
		return "No changes detected"
	}

	summary := strings.Join(parts, ", ")
	if result.HasUnsafeChanges {
		summary += " [WARNING: Contains unsafe changes]"
	}

	return summary
}
