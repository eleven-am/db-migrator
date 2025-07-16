package orm

import (
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/lib/pq"
)

// PostgreSQL-specific query methods for Query[T]

func (q *Query[T]) WhereJSONB(column string, path string, value interface{}) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("%s->>? = ?", column), path, value)
	q.whereClause = append(q.whereClause, condition)
	return q
}

func (q *Query[T]) WhereJSONBPath(column string, path string, operator string, value interface{}) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("%s#>>? %s ?", column, operator), path, value)
	q.whereClause = append(q.whereClause, condition)
	return q
}

func (q *Query[T]) WhereJSONBExists(column string, key string) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("%s ?? ?", column), key)
	q.whereClause = append(q.whereClause, condition)
	return q
}

func (q *Query[T]) WhereJSONBContains(column string, value interface{}) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("%s @> ?", column), value)
	q.whereClause = append(q.whereClause, condition)
	return q
}

func (q *Query[T]) WhereJSONBContainedBy(column string, value interface{}) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("%s <@ ?", column), value)
	q.whereClause = append(q.whereClause, condition)
	return q
}

func (q *Query[T]) WhereArrayContains(column string, value interface{}) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("%s @> ?", column), pq.Array([]interface{}{value}))
	q.whereClause = append(q.whereClause, condition)
	return q
}

func (q *Query[T]) WhereArrayContainsAny(column string, values []interface{}) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("%s && ?", column), pq.Array(values))
	q.whereClause = append(q.whereClause, condition)
	return q
}

func (q *Query[T]) WhereArrayContainedBy(column string, values []interface{}) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("%s <@ ?", column), pq.Array(values))
	q.whereClause = append(q.whereClause, condition)
	return q
}

func (q *Query[T]) WhereArrayLength(column string, length int) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("array_length(%s, 1) = ?", column), length)
	q.whereClause = append(q.whereClause, condition)
	return q
}

func (q *Query[T]) WhereArrayEmpty(column string) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("(array_length(%s, 1) IS NULL OR array_length(%s, 1) = 0)", column, column))
	q.whereClause = append(q.whereClause, condition)
	return q
}

func (q *Query[T]) WhereArrayNotEmpty(column string) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("array_length(%s, 1) > 0", column))
	q.whereClause = append(q.whereClause, condition)
	return q
}

func (q *Query[T]) WhereFullTextSearch(column string, query string) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("%s @@ plainto_tsquery(?)", column), query)
	q.whereClause = append(q.whereClause, condition)
	return q
}

func (q *Query[T]) WhereFullTextSearchLanguage(column string, language string, query string) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("%s @@ plainto_tsquery(?, ?)", column), language, query)
	q.whereClause = append(q.whereClause, condition)
	return q
}

func (q *Query[T]) WhereRegex(column string, pattern string) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("%s ~ ?", column), pattern)
	q.whereClause = append(q.whereClause, condition)
	return q
}

func (q *Query[T]) WhereRegexInsensitive(column string, pattern string) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("%s ~* ?", column), pattern)
	q.whereClause = append(q.whereClause, condition)
	return q
}

func (q *Query[T]) WhereRegexNot(column string, pattern string) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("%s !~ ?", column), pattern)
	q.whereClause = append(q.whereClause, condition)
	return q
}

func (q *Query[T]) WhereRegexNotInsensitive(column string, pattern string) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("%s !~* ?", column), pattern)
	q.whereClause = append(q.whereClause, condition)
	return q
}

func (q *Query[T]) WhereSimilarTo(column string, pattern string) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("%s SIMILAR TO ?", column), pattern)
	q.whereClause = append(q.whereClause, condition)
	return q
}

func (q *Query[T]) WhereNotSimilarTo(column string, pattern string) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("%s NOT SIMILAR TO ?", column), pattern)
	q.whereClause = append(q.whereClause, condition)
	return q
}

// PostgreSQL-specific Repository convenience methods

func (r *Repository[T]) FindByArrayContains(column string, value interface{}) ([]T, error) {
	return r.Query().WhereArrayContains(column, value).Find()
}

func (r *Repository[T]) FindByArrayContainsAny(column string, values []interface{}) ([]T, error) {
	return r.Query().WhereArrayContainsAny(column, values).Find()
}

func (r *Repository[T]) FindByJSONB(column string, path string, value interface{}) ([]T, error) {
	return r.Query().WhereJSONB(column, path, value).Find()
}

func (r *Repository[T]) FindByJSONBContains(column string, value interface{}) ([]T, error) {
	return r.Query().WhereJSONBContains(column, value).Find()
}

func (r *Repository[T]) Search(column string, query string) ([]T, error) {
	return r.Query().WhereFullTextSearch(column, query).Find()
}

func (r *Repository[T]) SearchWithLanguage(column string, language string, query string) ([]T, error) {
	return r.Query().WhereFullTextSearchLanguage(column, language, query).Find()
}

func (r *Repository[T]) FindByRegex(column string, pattern string) ([]T, error) {
	return r.Query().WhereRegex(column, pattern).Find()
}

func (r *Repository[T]) FindByRegexInsensitive(column string, pattern string) ([]T, error) {
	return r.Query().WhereRegexInsensitive(column, pattern).Find()
}

func (r *Repository[T]) CountByArrayContains(column string, value interface{}) (int64, error) {
	return r.Query().WhereArrayContains(column, value).Count()
}

func (r *Repository[T]) CountByJSONB(column string, path string, value interface{}) (int64, error) {
	return r.Query().WhereJSONB(column, path, value).Count()
}

func (r *Repository[T]) CountBySearch(column string, query string) (int64, error) {
	return r.Query().WhereFullTextSearch(column, query).Count()
}
