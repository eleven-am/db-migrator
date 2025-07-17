package orm

import (
	"context"
	"time"
)

// OperationType represents different types of database operations
type OperationType string

const (
	OpCreate     OperationType = "create"
	OpCreateMany OperationType = "create_many"
	OpUpdate     OperationType = "update"
	OpUpdateMany OperationType = "update_many"
	OpDelete     OperationType = "delete"
	OpUpsert     OperationType = "upsert"
	OpUpsertMany OperationType = "upsert_many"
	OpBulkUpdate OperationType = "bulk_update"
	OpFind       OperationType = "find"
	OpQuery      OperationType = "query"
)

// MiddlewareContext contains information passed to middleware
type MiddlewareContext struct {
	Operation    OperationType
	TableName    string
	Record       interface{}
	Records      interface{}
	QueryBuilder interface{} // squirrel.SelectBuilder, squirrel.InsertBuilder, etc.
	Query        string
	Args         []interface{}
	Error        error
	StartTime    time.Time
	Duration     time.Duration
	Context      context.Context
	Metadata     map[string]interface{}
}

// QueryMiddlewareFunc represents middleware that can modify queries
type QueryMiddlewareFunc func(ctx *MiddlewareContext) error

// QueryMiddleware represents middleware that can see and modify query builders
type QueryMiddleware func(next QueryMiddlewareFunc) QueryMiddlewareFunc

// middlewareManager manages database middleware
type middlewareManager struct {
	middleware []QueryMiddleware
}

func newMiddlewareManager() *middlewareManager {
	return &middlewareManager{
		middleware: make([]QueryMiddleware, 0),
	}
}

func (mm *middlewareManager) AddMiddleware(middleware QueryMiddleware) {
	mm.middleware = append(mm.middleware, middleware)
}

func (mm *middlewareManager) ExecuteMiddleware(ctx *MiddlewareContext, finalFunc QueryMiddlewareFunc) error {
	handler := finalFunc

	for i := len(mm.middleware) - 1; i >= 0; i-- {
		handler = mm.middleware[i](handler)
	}

	return handler(ctx)
}

// Repository middleware integration

func (r *Repository[T]) executeQueryMiddleware(op OperationType, ctx context.Context, record interface{}, queryBuilder interface{}, finalFunc QueryMiddlewareFunc) error {
	if r.middlewareManager == nil {
		return finalFunc(&MiddlewareContext{
			Operation:    op,
			TableName:    r.metadata.TableName,
			Record:       record,
			QueryBuilder: queryBuilder,
			Context:      ctx,
			StartTime:    time.Now(),
			Metadata:     make(map[string]interface{}),
		})
	}

	middlewareCtx := &MiddlewareContext{
		Operation:    op,
		TableName:    r.metadata.TableName,
		Record:       record,
		QueryBuilder: queryBuilder,
		Context:      ctx,
		StartTime:    time.Now(),
		Metadata:     make(map[string]interface{}),
	}

	return r.middlewareManager.ExecuteMiddleware(middlewareCtx, finalFunc)
}

func (r *Repository[T]) AddMiddleware(middleware QueryMiddleware) {
	if r.middlewareManager == nil {
		r.middlewareManager = newMiddlewareManager()
	}
	r.middlewareManager.AddMiddleware(middleware)
}

func (r *Repository[T]) getMiddlewareManager() *middlewareManager {
	if r.middlewareManager == nil {
		r.middlewareManager = newMiddlewareManager()
	}
	return r.middlewareManager
}

// Middleware system is available for custom implementations
// Built-in middleware implementations have been removed to keep the system lean
