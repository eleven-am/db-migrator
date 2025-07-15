package orm

import (
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/lib/pq"
)

// PostgreSQL-specific query methods for Query[T]

// WhereJSONB adds a JSONB path condition (column->>path = value)
func (q *Query[T]) WhereJSONB(column string, path string, value interface{}) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("%s->>? = ?", column), path, value)
	q.whereClause = append(q.whereClause, condition)
	return q
}

// WhereJSONBPath adds a JSONB path condition with custom operator (column#>>path operator value)
func (q *Query[T]) WhereJSONBPath(column string, path string, operator string, value interface{}) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("%s#>>? %s ?", column, operator), path, value)
	q.whereClause = append(q.whereClause, condition)
	return q
}

// WhereJSONBExists checks if a JSONB key exists (column ?? key)
func (q *Query[T]) WhereJSONBExists(column string, key string) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("%s ?? ?", column), key)
	q.whereClause = append(q.whereClause, condition)
	return q
}

// WhereJSONBContains checks if JSONB contains value (column @> value)
func (q *Query[T]) WhereJSONBContains(column string, value interface{}) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("%s @> ?", column), value)
	q.whereClause = append(q.whereClause, condition)
	return q
}

// WhereJSONBContainedBy checks if JSONB is contained by value (column <@ value)
func (q *Query[T]) WhereJSONBContainedBy(column string, value interface{}) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("%s <@ ?", column), value)
	q.whereClause = append(q.whereClause, condition)
	return q
}

// WhereArrayContains checks if array contains value (column @> ARRAY[value])
func (q *Query[T]) WhereArrayContains(column string, value interface{}) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("%s @> ?", column), pq.Array([]interface{}{value}))
	q.whereClause = append(q.whereClause, condition)
	return q
}

// WhereArrayContainsAny checks if array overlaps with values (column && ARRAY[values])
func (q *Query[T]) WhereArrayContainsAny(column string, values []interface{}) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("%s && ?", column), pq.Array(values))
	q.whereClause = append(q.whereClause, condition)
	return q
}

// WhereArrayContainedBy checks if array is contained by values (column <@ ARRAY[values])
func (q *Query[T]) WhereArrayContainedBy(column string, values []interface{}) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("%s <@ ?", column), pq.Array(values))
	q.whereClause = append(q.whereClause, condition)
	return q
}

// WhereArrayLength checks array length (array_length(column, 1) = length)
func (q *Query[T]) WhereArrayLength(column string, length int) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("array_length(%s, 1) = ?", column), length)
	q.whereClause = append(q.whereClause, condition)
	return q
}

// WhereArrayEmpty checks if array is empty (array_length(column, 1) IS NULL OR array_length(column, 1) = 0)
func (q *Query[T]) WhereArrayEmpty(column string) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("(array_length(%s, 1) IS NULL OR array_length(%s, 1) = 0)", column, column))
	q.whereClause = append(q.whereClause, condition)
	return q
}

// WhereArrayNotEmpty checks if array is not empty (array_length(column, 1) > 0)
func (q *Query[T]) WhereArrayNotEmpty(column string) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("array_length(%s, 1) > 0", column))
	q.whereClause = append(q.whereClause, condition)
	return q
}

// WhereFullTextSearch performs full-text search (column @@ plainto_tsquery(query))
func (q *Query[T]) WhereFullTextSearch(column string, query string) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("%s @@ plainto_tsquery(?)", column), query)
	q.whereClause = append(q.whereClause, condition)
	return q
}

// WhereFullTextSearchLanguage performs full-text search with language (column @@ plainto_tsquery(language, query))
func (q *Query[T]) WhereFullTextSearchLanguage(column string, language string, query string) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("%s @@ plainto_tsquery(?, ?)", column), language, query)
	q.whereClause = append(q.whereClause, condition)
	return q
}

// WhereRegex performs regex matching (column ~ pattern)
func (q *Query[T]) WhereRegex(column string, pattern string) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("%s ~ ?", column), pattern)
	q.whereClause = append(q.whereClause, condition)
	return q
}

// WhereRegexInsensitive performs case-insensitive regex matching (column ~* pattern)
func (q *Query[T]) WhereRegexInsensitive(column string, pattern string) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("%s ~* ?", column), pattern)
	q.whereClause = append(q.whereClause, condition)
	return q
}

// WhereRegexNot performs negative regex matching (column !~ pattern)
func (q *Query[T]) WhereRegexNot(column string, pattern string) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("%s !~ ?", column), pattern)
	q.whereClause = append(q.whereClause, condition)
	return q
}

// WhereRegexNotInsensitive performs case-insensitive negative regex matching (column !~* pattern)
func (q *Query[T]) WhereRegexNotInsensitive(column string, pattern string) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("%s !~* ?", column), pattern)
	q.whereClause = append(q.whereClause, condition)
	return q
}

// WhereSimilarTo performs SIMILAR TO pattern matching (column SIMILAR TO pattern)
func (q *Query[T]) WhereSimilarTo(column string, pattern string) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("%s SIMILAR TO ?", column), pattern)
	q.whereClause = append(q.whereClause, condition)
	return q
}

// WhereNotSimilarTo performs NOT SIMILAR TO pattern matching (column NOT SIMILAR TO pattern)
func (q *Query[T]) WhereNotSimilarTo(column string, pattern string) *Query[T] {
	if q.err != nil {
		return q
	}

	condition := squirrel.Expr(fmt.Sprintf("%s NOT SIMILAR TO ?", column), pattern)
	q.whereClause = append(q.whereClause, condition)
	return q
}

// PostgreSQL-specific Repository convenience methods

// FindByArrayContains finds records where array column contains the specified value
func (r *Repository[T]) FindByArrayContains(column string, value interface{}) ([]T, error) {
	return r.Query().WhereArrayContains(column, value).Find()
}

// FindByArrayContainsAny finds records where array column overlaps with any of the specified values
func (r *Repository[T]) FindByArrayContainsAny(column string, values []interface{}) ([]T, error) {
	return r.Query().WhereArrayContainsAny(column, values).Find()
}

// FindByJSONB finds records where JSONB column at path equals value
func (r *Repository[T]) FindByJSONB(column string, path string, value interface{}) ([]T, error) {
	return r.Query().WhereJSONB(column, path, value).Find()
}

// FindByJSONBContains finds records where JSONB column contains the specified value
func (r *Repository[T]) FindByJSONBContains(column string, value interface{}) ([]T, error) {
	return r.Query().WhereJSONBContains(column, value).Find()
}

// Search performs full-text search on the specified column
func (r *Repository[T]) Search(column string, query string) ([]T, error) {
	return r.Query().WhereFullTextSearch(column, query).Find()
}

// SearchWithLanguage performs full-text search with language on the specified column
func (r *Repository[T]) SearchWithLanguage(column string, language string, query string) ([]T, error) {
	return r.Query().WhereFullTextSearchLanguage(column, language, query).Find()
}

// FindByRegex finds records where column matches regex pattern
func (r *Repository[T]) FindByRegex(column string, pattern string) ([]T, error) {
	return r.Query().WhereRegex(column, pattern).Find()
}

// FindByRegexInsensitive finds records where column matches case-insensitive regex pattern
func (r *Repository[T]) FindByRegexInsensitive(column string, pattern string) ([]T, error) {
	return r.Query().WhereRegexInsensitive(column, pattern).Find()
}

// CountByArrayContains counts records where array column contains the specified value
func (r *Repository[T]) CountByArrayContains(column string, value interface{}) (int64, error) {
	return r.Query().WhereArrayContains(column, value).Count()
}

// CountByJSONB counts records where JSONB column at path equals value
func (r *Repository[T]) CountByJSONB(column string, path string, value interface{}) (int64, error) {
	return r.Query().WhereJSONB(column, path, value).Count()
}

// CountBySearch counts records matching full-text search
func (r *Repository[T]) CountBySearch(column string, query string) (int64, error) {
	return r.Query().WhereFullTextSearch(column, query).Count()
}
