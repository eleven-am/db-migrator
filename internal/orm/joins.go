package orm

import (
	"fmt"
)

// JoinType represents different types of SQL joins
type JoinType string

const (
	InnerJoin JoinType = "INNER JOIN"
	LeftJoin  JoinType = "LEFT JOIN"
	RightJoin JoinType = "RIGHT JOIN"
	FullJoin  JoinType = "FULL OUTER JOIN"
)

// join represents a SQL join clause (internal use only)
type join struct {
	Type      JoinType
	Table     string
	Alias     string
	Condition string
	Args      []interface{}
}

// joinBuilder provides a fluent interface for building joins (internal use only)
type joinBuilder struct {
	joins []join
}

// newJoinBuilder creates a new join builder
func newJoinBuilder() *joinBuilder {
	return &joinBuilder{
		joins: make([]join, 0),
	}
}

// Inner adds an INNER JOIN
func (jb *joinBuilder) Inner(table string, condition string, args ...interface{}) *joinBuilder {
	return jb.addJoin(InnerJoin, table, "", condition, args...)
}

// InnerAs adds an INNER JOIN with table alias
func (jb *joinBuilder) InnerAs(table string, alias string, condition string, args ...interface{}) *joinBuilder {
	return jb.addJoin(InnerJoin, table, alias, condition, args...)
}

// Left adds a LEFT JOIN
func (jb *joinBuilder) Left(table string, condition string, args ...interface{}) *joinBuilder {
	return jb.addJoin(LeftJoin, table, "", condition, args...)
}

// LeftAs adds a LEFT JOIN with table alias
func (jb *joinBuilder) LeftAs(table string, alias string, condition string, args ...interface{}) *joinBuilder {
	return jb.addJoin(LeftJoin, table, alias, condition, args...)
}

// Right adds a RIGHT JOIN
func (jb *joinBuilder) Right(table string, condition string, args ...interface{}) *joinBuilder {
	return jb.addJoin(RightJoin, table, "", condition, args...)
}

// RightAs adds a RIGHT JOIN with table alias
func (jb *joinBuilder) RightAs(table string, alias string, condition string, args ...interface{}) *joinBuilder {
	return jb.addJoin(RightJoin, table, alias, condition, args...)
}

// Full adds a FULL OUTER JOIN
func (jb *joinBuilder) Full(table string, condition string, args ...interface{}) *joinBuilder {
	return jb.addJoin(FullJoin, table, "", condition, args...)
}

// FullAs adds a FULL OUTER JOIN with table alias
func (jb *joinBuilder) FullAs(table string, alias string, condition string, args ...interface{}) *joinBuilder {
	return jb.addJoin(FullJoin, table, alias, condition, args...)
}

// addJoin is a helper method to add joins
func (jb *joinBuilder) addJoin(joinType JoinType, table string, alias string, condition string, args ...interface{}) *joinBuilder {
	join := join{
		Type:      joinType,
		Table:     table,
		Alias:     alias,
		Condition: condition,
		Args:      args,
	}
	jb.joins = append(jb.joins, join)
	return jb
}

// Build returns the constructed joins
func (jb *joinBuilder) Build() []join {
	return jb.joins
}

// Note: JoinQuery has been merged into the main Query type
// Join functionality is now available directly on Query[T]

// JoinRelationship creates a join based on a defined relationship
func (q *Query[T]) JoinRelationship(relationshipName string, joinType JoinType) *Query[T] {
	repo := q.repo
	if repo.relationshipManager == nil {
		q.err = fmt.Errorf("no relationship manager available")
		return q
	}

	rel := repo.relationshipManager.GetRelationship(relationshipName)
	if rel == nil {
		q.err = fmt.Errorf("relationship %s not found", relationshipName)
		return q
	}

	switch rel.Type {
	case "belongs_to":

		condition := fmt.Sprintf("%s.%s = %s.%s",
			repo.tableName, rel.ForeignKey,
			rel.Target, rel.TargetKey)
		q.Join(InnerJoin, rel.Target, condition)

	case "has_one", "has_many":

		condition := fmt.Sprintf("%s.%s = %s.%s",
			repo.tableName, rel.SourceKey,
			rel.Target, rel.ForeignKey)
		q.Join(InnerJoin, rel.Target, condition)

	case "has_many_through":

		condition1 := fmt.Sprintf("%s.%s = %s.%s",
			repo.tableName, rel.SourceKey,
			rel.JoinTable, rel.SourceFK)
		q.Join(InnerJoin, rel.JoinTable, condition1)

		condition2 := fmt.Sprintf("%s.%s = %s.%s",
			rel.JoinTable, rel.TargetFK,
			rel.Target, rel.TargetKey)
		q.Join(InnerJoin, rel.Target, condition2)

	default:
		q.err = fmt.Errorf("unsupported relationship type for join: %s", rel.Type)
	}

	return q
}

// RawJoin allows completely custom join logic
func (q *Query[T]) RawJoin(joinClause string, args ...interface{}) *Query[T] {
	join := join{
		Type:      "",
		Table:     "",
		Condition: joinClause,
		Args:      args,
	}

	q.joins = append(q.joins, join)
	return q
}
