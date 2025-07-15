package orm

import (
	"context"
	"fmt"
	"reflect"
	"time"
)

// HookType represents different types of database hooks
type HookType string

const (
	HookBeforeCreate HookType = "before_create"
	HookAfterCreate  HookType = "after_create"
	HookBeforeUpdate HookType = "before_update"
	HookAfterUpdate  HookType = "after_update"
	HookBeforeDelete HookType = "before_delete"
	HookAfterDelete  HookType = "after_delete"
	HookBeforeFind   HookType = "before_find"
	HookAfterFind    HookType = "after_find"
	HookBeforeQuery  HookType = "before_query"
	HookAfterQuery   HookType = "after_query"
)

// HookContext contains information passed to hooks
type HookContext struct {
	Type      HookType
	TableName string
	Record    interface{}
	Records   interface{}
	Query     string
	Args      []interface{}
	Error     error
	StartTime time.Time
	Duration  time.Duration
	Context   context.Context
	Metadata  map[string]interface{}
}

// Hook represents a database hook function
type Hook func(*HookContext) error

// hookManager manages database hooks and middleware (internal use only)
type hookManager struct {
	hooks      map[HookType][]Hook
	middleware []Middleware
}

// Middleware represents a function that wraps database operations
type Middleware func(next MiddlewareFunc) MiddlewareFunc

// MiddlewareFunc represents the signature for middleware functions
type MiddlewareFunc func(ctx *HookContext) error

// newHookManager creates a new hook manager
func newHookManager() *hookManager {
	return &hookManager{
		hooks:      make(map[HookType][]Hook),
		middleware: make([]Middleware, 0),
	}
}

// AddHook registers a hook for a specific hook type
func (hm *hookManager) AddHook(hookType HookType, hook Hook) {
	hm.hooks[hookType] = append(hm.hooks[hookType], hook)
}

// RemoveHook removes all hooks for a specific hook type
func (hm *hookManager) RemoveHook(hookType HookType) {
	delete(hm.hooks, hookType)
}

// AddMiddleware adds middleware to the chain
func (hm *hookManager) AddMiddleware(middleware Middleware) {
	hm.middleware = append(hm.middleware, middleware)
}

// ExecuteHooks executes all hooks for a given hook type
func (hm *hookManager) ExecuteHooks(hookCtx *HookContext) error {
	hooks, exists := hm.hooks[hookCtx.Type]
	if !exists {
		return nil
	}

	for _, hook := range hooks {
		if err := hook(hookCtx); err != nil {
			return fmt.Errorf("hook execution failed for %s: %w", hookCtx.Type, err)
		}
	}

	return nil
}

// ExecuteMiddleware executes the middleware chain
func (hm *hookManager) ExecuteMiddleware(hookCtx *HookContext, finalFunc MiddlewareFunc) error {

	handler := finalFunc
	for i := len(hm.middleware) - 1; i >= 0; i-- {
		handler = hm.middleware[i](handler)
	}

	return handler(hookCtx)
}

// Repository hook integration

// executeBeforeHook executes before hooks and middleware
func (r *Repository[T]) executeBeforeHook(hookType HookType, ctx context.Context, record interface{}, query string, args []interface{}) error {
	if r.hookManager == nil {
		return nil
	}

	hookCtx := &HookContext{
		Type:      hookType,
		TableName: r.tableName,
		Record:    record,
		Query:     query,
		Args:      args,
		StartTime: time.Now(),
		Context:   ctx,
		Metadata:  make(map[string]interface{}),
	}

	return r.hookManager.ExecuteHooks(hookCtx)
}

// executeAfterHook executes after hooks and middleware
func (r *Repository[T]) executeAfterHook(hookType HookType, ctx context.Context, record interface{}, query string, args []interface{}, err error, duration time.Duration) error {
	if r.hookManager == nil {
		return nil
	}

	hookCtx := &HookContext{
		Type:      hookType,
		TableName: r.tableName,
		Record:    record,
		Query:     query,
		Args:      args,
		Error:     err,
		StartTime: time.Now().Add(-duration),
		Duration:  duration,
		Context:   ctx,
		Metadata:  make(map[string]interface{}),
	}

	return r.hookManager.ExecuteHooks(hookCtx)
}

// Add hook manager to Repository struct (this would be added to repository.go)
// For now, we'll add methods to configure hooks

// SetHookManager sets the hook manager for the repository
func (r *Repository[T]) SetHookManager(hm *hookManager) {
	r.hookManager = hm
}

// getHookManager returns the hook manager for the repository (internal use)
func (r *Repository[T]) getHookManager() *hookManager {
	if r.hookManager == nil {
		r.hookManager = newHookManager()
	}
	return r.hookManager
}

// Built-in middleware implementations

// LoggingMiddleware logs database operations
func LoggingMiddleware(logger func(string, ...interface{})) Middleware {
	return func(next MiddlewareFunc) MiddlewareFunc {
		return func(ctx *HookContext) error {
			start := time.Now()

			logger("Executing %s operation on table %s", ctx.Type, ctx.TableName)

			err := next(ctx)

			duration := time.Since(start)
			if err != nil {
				logger("Operation %s on table %s failed after %v: %v", ctx.Type, ctx.TableName, duration, err)
			} else {
				logger("Operation %s on table %s completed in %v", ctx.Type, ctx.TableName, duration)
			}

			return err
		}
	}
}

// MetricsMiddleware collects operation metrics
func MetricsMiddleware(collector MetricsCollector) Middleware {
	return func(next MiddlewareFunc) MiddlewareFunc {
		return func(ctx *HookContext) error {
			start := time.Now()

			err := next(ctx)

			duration := time.Since(start)

			collector.RecordOperation(string(ctx.Type), ctx.TableName, duration, err != nil)

			return err
		}
	}
}

// MetricsCollector interface for collecting operation metrics
type MetricsCollector interface {
	RecordOperation(operation, table string, duration time.Duration, hasError bool)
}

// ValidationMiddleware validates records before operations
func ValidationMiddleware(validator RecordValidator) Middleware {
	return func(next MiddlewareFunc) MiddlewareFunc {
		return func(ctx *HookContext) error {

			if ctx.Type == HookBeforeCreate || ctx.Type == HookBeforeUpdate {
				if ctx.Record != nil {
					if err := validator.Validate(ctx.Record); err != nil {
						return fmt.Errorf("validation failed: %w", err)
					}
				}
			}

			return next(ctx)
		}
	}
}

// RecordValidator interface for record validation
type RecordValidator interface {
	Validate(record interface{}) error
}

// Built-in hooks

// TimestampHook automatically sets created_at and updated_at fields
func TimestampHook(ctx *HookContext) error {
	if ctx.Record == nil {
		return nil
	}

	now := time.Now()
	recordValue := reflect.ValueOf(ctx.Record)

	if recordValue.Kind() == reflect.Ptr {
		recordValue = recordValue.Elem()
	}

	if recordValue.Kind() != reflect.Struct {
		return nil
	}

	recordType := recordValue.Type()

	switch ctx.Type {
	case HookBeforeCreate:

		setTimeField(recordValue, recordType, "CreatedAt", now)
		setTimeField(recordValue, recordType, "UpdatedAt", now)

	case HookBeforeUpdate:

		setTimeField(recordValue, recordType, "UpdatedAt", now)
	}

	return nil
}

// setTimeField sets a time field on a struct if it exists and is settable
func setTimeField(recordValue reflect.Value, recordType reflect.Type, fieldName string, value time.Time) {
	if field, found := recordType.FieldByName(fieldName); found {
		if field.Type == reflect.TypeOf(time.Time{}) {
			fieldValue := recordValue.FieldByName(fieldName)
			if fieldValue.IsValid() && fieldValue.CanSet() {
				fieldValue.Set(reflect.ValueOf(value))
			}
		}
	}
}

// AuditHook logs all database operations for auditing
func AuditHook(auditLogger func(operation, table string, record interface{}, timestamp time.Time)) Hook {
	return func(ctx *HookContext) error {

		if ctx.Type == HookAfterCreate || ctx.Type == HookAfterUpdate || ctx.Type == HookAfterDelete {
			auditLogger(string(ctx.Type), ctx.TableName, ctx.Record, time.Now())
		}

		return nil
	}
}

// SoftDeleteHook implements soft delete functionality
func SoftDeleteHook(ctx *HookContext) error {
	if ctx.Type != HookBeforeDelete || ctx.Record == nil {
		return nil
	}

	recordValue := reflect.ValueOf(ctx.Record)

	if recordValue.Kind() == reflect.Ptr {
		recordValue = recordValue.Elem()
	}

	if recordValue.Kind() != reflect.Struct {
		return nil
	}

	recordType := recordValue.Type()

	if field, found := recordType.FieldByName("DeletedAt"); found {
		if field.Type == reflect.TypeOf((*time.Time)(nil)) {
			fieldValue := recordValue.FieldByName("DeletedAt")
			if fieldValue.IsValid() && fieldValue.CanSet() {
				now := time.Now()
				fieldValue.Set(reflect.ValueOf(&now))

				ctx.Metadata["soft_delete"] = true
			}
		}
	}

	return nil
}

// Example usage patterns (these would be used in application code)

// SetupCommonHooks configures commonly used hooks for a repository
func (r *Repository[T]) SetupCommonHooks() {
	hm := r.getHookManager()

	hm.AddHook(HookBeforeCreate, TimestampHook)
	hm.AddHook(HookBeforeUpdate, TimestampHook)

	hm.AddHook(HookBeforeDelete, SoftDeleteHook)
}

// SetupAuditHooks configures audit logging
func (r *Repository[T]) SetupAuditHooks(auditLogger func(operation, table string, record interface{}, timestamp time.Time)) {
	hm := r.getHookManager()

	hm.AddHook(HookAfterCreate, AuditHook(auditLogger))
	hm.AddHook(HookAfterUpdate, AuditHook(auditLogger))
	hm.AddHook(HookAfterDelete, AuditHook(auditLogger))
}

// SetupMiddleware configures common middleware
func (r *Repository[T]) SetupMiddleware(logger func(string, ...interface{}), metrics MetricsCollector) {
	hm := r.getHookManager()

	if logger != nil {
		hm.AddMiddleware(LoggingMiddleware(logger))
	}

	if metrics != nil {
		hm.AddMiddleware(MetricsMiddleware(metrics))
	}
}
