package orm

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMiddlewareBehavior tests middleware functionality without strict SQL matching
func TestMiddlewareBehavior(t *testing.T) {
	t.Run("Middleware can modify SELECT queries", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherFunc(func(expectedSQL, actualSQL string) error {
			// Just verify that the tenant_id condition was added
			assert.Contains(t, actualSQL, "tenant_id")
			return nil
		})))
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "postgres")
		metadata := createTestUserMetadata()

		repo, err := NewRepository[TestUser](sqlxDB, metadata)
		require.NoError(t, err)

		// Add middleware that adds tenant filtering
		repo.AddMiddleware(func(next QueryMiddlewareFunc) QueryMiddlewareFunc {
			return func(ctx *MiddlewareContext) error {
				if ctx.Operation == OpQuery {
					if sb, ok := ctx.QueryBuilder.(squirrel.SelectBuilder); ok {
						ctx.QueryBuilder = sb.Where(squirrel.Eq{"tenant_id": 123})
					}
				}
				return next(ctx)
			}
		})

		// Set up mock to return empty result
		mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "is_active", "created_at", "updated_at"}))

		// Execute query
		_, err = repo.Query().Find()
		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Middleware can block operations", func(t *testing.T) {
		db, _, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "postgres")
		metadata := createTestUserMetadata()

		repo, err := NewRepository[TestUser](sqlxDB, metadata)
		require.NoError(t, err)

		// Add middleware that blocks all delete operations
		repo.AddMiddleware(func(next QueryMiddlewareFunc) QueryMiddlewareFunc {
			return func(ctx *MiddlewareContext) error {
				if ctx.Operation == OpDelete {
					return fmt.Errorf("delete operations are not allowed")
				}
				return next(ctx)
			}
		})

		// Try to delete - should fail
		err = repo.Delete(context.Background(), 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "delete operations are not allowed")
	})

	t.Run("Middleware executes in correct order", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "postgres")
		metadata := createTestUserMetadata()

		repo, err := NewRepository[TestUser](sqlxDB, metadata)
		require.NoError(t, err)

		var executionOrder []int

		// Add first middleware
		repo.AddMiddleware(func(next QueryMiddlewareFunc) QueryMiddlewareFunc {
			return func(ctx *MiddlewareContext) error {
				executionOrder = append(executionOrder, 1)
				err := next(ctx)
				executionOrder = append(executionOrder, 3)
				return err
			}
		})

		// Add second middleware
		repo.AddMiddleware(func(next QueryMiddlewareFunc) QueryMiddlewareFunc {
			return func(ctx *MiddlewareContext) error {
				executionOrder = append(executionOrder, 2)
				return next(ctx)
			}
		})

		// Set up mock
		mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "is_active", "created_at", "updated_at"}))

		// Execute query
		_, err = repo.Query().Find()
		require.NoError(t, err)

		// Verify execution order
		assert.Equal(t, []int{1, 2, 3}, executionOrder)
	})

	t.Run("Middleware has access to operation metadata", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "postgres")
		metadata := createTestUserMetadata()

		repo, err := NewRepository[TestUser](sqlxDB, metadata)
		require.NoError(t, err)

		var capturedOp OperationType
		var capturedTable string

		// Add middleware that captures metadata
		repo.AddMiddleware(func(next QueryMiddlewareFunc) QueryMiddlewareFunc {
			return func(ctx *MiddlewareContext) error {
				capturedOp = ctx.Operation
				capturedTable = ctx.TableName
				return next(ctx)
			}
		})

		// Set up mock - expect RETURNING clause
		now := time.Now()
		mock.ExpectQuery("INSERT.*RETURNING").WillReturnRows(
			sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
				AddRow(1, now, now))

		// Execute create
		user := &TestUser{Name: "Test", Email: "test@example.com", IsActive: true}
		err = repo.Create(context.Background(), user)
		require.NoError(t, err)

		// Verify metadata was captured
		assert.Equal(t, OpCreate, capturedOp)
		assert.Equal(t, "users", capturedTable)
	})

	t.Run("Middleware can add metadata", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "postgres")
		metadata := createTestUserMetadata()

		repo, err := NewRepository[TestUser](sqlxDB, metadata)
		require.NoError(t, err)

		// Add middleware that sets metadata
		repo.AddMiddleware(func(next QueryMiddlewareFunc) QueryMiddlewareFunc {
			return func(ctx *MiddlewareContext) error {
				ctx.Metadata["user_id"] = "123"
				ctx.Metadata["request_id"] = "abc-123"
				return next(ctx)
			}
		})

		// Add middleware that verifies metadata
		repo.AddMiddleware(func(next QueryMiddlewareFunc) QueryMiddlewareFunc {
			return func(ctx *MiddlewareContext) error {
				assert.Equal(t, "123", ctx.Metadata["user_id"])
				assert.Equal(t, "abc-123", ctx.Metadata["request_id"])
				return next(ctx)
			}
		})

		// Set up mock
		mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "is_active", "created_at", "updated_at"}))

		// Execute query
		_, err = repo.Query().Find()
		require.NoError(t, err)
	})
}

// TestMiddlewareRealWorldScenarios tests common middleware use cases
func TestMiddlewareRealWorldScenarios(t *testing.T) {
	t.Run("Multi-tenancy middleware", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherFunc(func(expectedSQL, actualSQL string) error {
			// Verify tenant_id is added to all operations
			assert.Contains(t, actualSQL, "tenant_id")
			return nil
		})))
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "postgres")
		metadata := createTestUserMetadata()

		repo, err := NewRepository[TestUser](sqlxDB, metadata)
		require.NoError(t, err)

		tenantID := 456

		// Add multi-tenancy middleware
		repo.AddMiddleware(func(next QueryMiddlewareFunc) QueryMiddlewareFunc {
			return func(ctx *MiddlewareContext) error {
				switch v := ctx.QueryBuilder.(type) {
				case squirrel.SelectBuilder:
					ctx.QueryBuilder = v.Where(squirrel.Eq{"tenant_id": tenantID})
				case squirrel.UpdateBuilder:
					ctx.QueryBuilder = v.Where(squirrel.Eq{"tenant_id": tenantID})
				case squirrel.DeleteBuilder:
					ctx.QueryBuilder = v.Where(squirrel.Eq{"tenant_id": tenantID})
				}
				return next(ctx)
			}
		})

		// Test SELECT
		mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id"}))
		_, err = repo.Query().Find()
		require.NoError(t, err)

		// Test UPDATE
		mock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(0, 1))
		err = repo.Update(context.Background(), &TestUser{ID: 1, Name: "Updated"})
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Soft delete middleware", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "postgres")
		metadata := createTestUserMetadata()

		repo, err := NewRepository[TestUser](sqlxDB, metadata)
		require.NoError(t, err)

		// Add soft delete middleware
		repo.AddMiddleware(func(next QueryMiddlewareFunc) QueryMiddlewareFunc {
			return func(ctx *MiddlewareContext) error {
				switch ctx.Operation {
				case OpQuery:
					if sb, ok := ctx.QueryBuilder.(squirrel.SelectBuilder); ok {
						// Filter out soft-deleted records
						ctx.QueryBuilder = sb.Where(squirrel.Eq{"deleted_at": nil})
					}
				case OpDelete:
					// Convert DELETE to UPDATE
					ctx.Operation = OpUpdate
					ctx.QueryBuilder = squirrel.Update(ctx.TableName).
						Set("deleted_at", "NOW()").
						PlaceholderFormat(squirrel.Dollar)
					// The original DELETE's WHERE clause would need to be preserved
					// This is simplified for the test
				}
				return next(ctx)
			}
		})

		// Test that queries filter out deleted records
		mock.ExpectQuery("SELECT.*deleted_at IS NULL").
			WillReturnRows(sqlmock.NewRows([]string{"id"}))
		_, err = repo.Query().Find()
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}
