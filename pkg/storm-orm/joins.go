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

func newJoinBuilder() *joinBuilder {
	return &joinBuilder{
		joins: make([]join, 0),
	}
}

func (jb *joinBuilder) Inner(table string, condition string, args ...interface{}) *joinBuilder {
	return jb.addJoin(InnerJoin, table, "", condition, args...)
}

func (jb *joinBuilder) InnerAs(table string, alias string, condition string, args ...interface{}) *joinBuilder {
	return jb.addJoin(InnerJoin, table, alias, condition, args...)
}

func (jb *joinBuilder) Left(table string, condition string, args ...interface{}) *joinBuilder {
	return jb.addJoin(LeftJoin, table, "", condition, args...)
}

func (jb *joinBuilder) LeftAs(table string, alias string, condition string, args ...interface{}) *joinBuilder {
	return jb.addJoin(LeftJoin, table, alias, condition, args...)
}

func (jb *joinBuilder) Right(table string, condition string, args ...interface{}) *joinBuilder {
	return jb.addJoin(RightJoin, table, "", condition, args...)
}

func (jb *joinBuilder) RightAs(table string, alias string, condition string, args ...interface{}) *joinBuilder {
	return jb.addJoin(RightJoin, table, alias, condition, args...)
}

func (jb *joinBuilder) Full(table string, condition string, args ...interface{}) *joinBuilder {
	return jb.addJoin(FullJoin, table, "", condition, args...)
}

func (jb *joinBuilder) FullAs(table string, alias string, condition string, args ...interface{}) *joinBuilder {
	return jb.addJoin(FullJoin, table, alias, condition, args...)
}

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

func (jb *joinBuilder) Build() []join {
	return jb.joins
}

// Note: JoinQuery has been merged into the main Query type
// Join functionality is now available directly on Query[T]

func (q *Query[T]) JoinRelationship(relationshipName string, joinType JoinType) *Query[T] {
	repo := q.repo

	rel := repo.getRelationship(relationshipName)
	if rel == nil {
		q.err = fmt.Errorf("relationship %s not found", relationshipName)
		return q
	}

	switch rel.Type {
	case "belongs_to":
		condition := fmt.Sprintf("%s.%s = %s.%s",
			repo.metadata.TableName, rel.ForeignKey,
			rel.Target, rel.TargetKey)
		q.Join(InnerJoin, rel.Target, condition)

	case "has_one", "has_many":
		condition := fmt.Sprintf("%s.%s = %s.%s",
			repo.metadata.TableName, rel.SourceKey,
			rel.Target, rel.ForeignKey)
		q.Join(InnerJoin, rel.Target, condition)

	case "has_many_through":
		condition1 := fmt.Sprintf("%s.%s = %s.%s",
			repo.metadata.TableName, rel.SourceKey,
			rel.Through, rel.ThroughFK)
		q.Join(InnerJoin, rel.Through, condition1)

		condition2 := fmt.Sprintf("%s.%s = %s.%s",
			rel.Through, rel.ThroughTK,
			rel.Target, rel.TargetKey)
		q.Join(InnerJoin, rel.Target, condition2)

	default:
		q.err = fmt.Errorf("unsupported relationship type for join: %s", rel.Type)
	}

	return q
}

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
