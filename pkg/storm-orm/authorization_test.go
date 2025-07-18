package orm

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test model for authorization tests
type AuthTestUser struct {
	ID     string `db:"id"`
	Email  string `db:"email"`
	TeamID string `db:"team_id"`
	Role   string `db:"role"`
}

// Mock user context for testing
type mockUserContext struct {
	UserID string
	TeamID string
	Role   string
}

func createTestRepository(t testing.TB) *Repository[AuthTestUser] {
	// Create mock metadata
	metadata := &ModelMetadata{
		TableName:   "auth_test_users",
		PrimaryKeys: []string{"id"},
		Columns: map[string]*ColumnMetadata{
			"ID":     {DBName: "id", FieldName: "ID"},
			"Email":  {DBName: "email", FieldName: "Email"},
			"TeamID": {DBName: "team_id", FieldName: "TeamID"},
			"Role":   {DBName: "role", FieldName: "Role"},
		},
	}

	// Create repository with mock DB
	mockDB := &sqlx.DB{}
	repo, err := NewRepositoryWithExecutor[AuthTestUser](mockDB, metadata)
	require.NoError(t, err)
	return repo
}

func TestAuthorize_SingleFunction(t *testing.T) {
	baseRepo := createTestRepository(t)

	// Test that base repository starts with no authorization functions
	assert.Empty(t, baseRepo.authorizeFuncs)

	// Add single authorization function
	authFunc := func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
		return query // No-op for test
	}

	authRepo := baseRepo.Authorize(authFunc)

	// Verify authorization function was added
	assert.Len(t, authRepo.authorizeFuncs, 1)

	// Verify base repository is unchanged (immutable)
	assert.Empty(t, baseRepo.authorizeFuncs)

	// Verify different instances
	assert.NotSame(t, baseRepo, authRepo)
}

func TestAuthorize_MultipleFunction(t *testing.T) {
	baseRepo := createTestRepository(t)

	// Chain multiple authorization functions
	authRepo := baseRepo.
		Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
			return query
		}).
		Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
			return query
		}).
		Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
			return query
		})

	// Verify all authorization functions were added
	assert.Len(t, authRepo.authorizeFuncs, 3)

	// Verify base repository is unchanged
	assert.Empty(t, baseRepo.authorizeFuncs)

	// Verify different instances
	assert.NotSame(t, baseRepo, authRepo)
}

func TestAuthorize_ImmutableChaining(t *testing.T) {
	baseRepo := createTestRepository(t)

	// Create first authorized repository
	authRepo1 := baseRepo.Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
		return query
	})

	// Create second authorized repository from first
	authRepo2 := authRepo1.Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
		return query
	})

	// Create third from base (different chain)
	authRepo3 := baseRepo.Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
		return query
	})

	// Verify each repository has the correct number of functions
	assert.Len(t, baseRepo.authorizeFuncs, 0)
	assert.Len(t, authRepo1.authorizeFuncs, 1)
	assert.Len(t, authRepo2.authorizeFuncs, 2)
	assert.Len(t, authRepo3.authorizeFuncs, 1)

	// Verify all instances are different
	assert.NotSame(t, baseRepo, authRepo1)
	assert.NotSame(t, authRepo1, authRepo2)
	assert.NotSame(t, authRepo1, authRepo3)
	assert.NotSame(t, authRepo2, authRepo3)
}

func TestQuery_NoAuthorization(t *testing.T) {
	baseRepo := createTestRepository(t)
	ctx := context.Background()

	// Create query without authorization
	query := baseRepo.Query(ctx)

	// Verify query was created
	assert.NotNil(t, query)
	assert.Equal(t, baseRepo, query.repo)
	assert.Equal(t, ctx, query.ctx)
}

func TestQuery_WithAuthorization(t *testing.T) {
	baseRepo := createTestRepository(t)
	ctx := context.Background()

	// Add user context
	userCtx := mockUserContext{
		UserID: "user123",
		TeamID: "team456",
		Role:   "member",
	}
	ctx = context.WithValue(ctx, "user", userCtx)

	// Track authorization function calls
	var authCallCount int
	var authContexts []context.Context

	// Create authorized repository with tracking
	authRepo := baseRepo.
		Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
			authCallCount++
			authContexts = append(authContexts, ctx)

			// Verify context has user data
			user, ok := ctx.Value("user").(mockUserContext)
			assert.True(t, ok)
			assert.Equal(t, "user123", user.UserID)
			assert.Equal(t, "team456", user.TeamID)

			return query
		}).
		Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
			authCallCount++
			authContexts = append(authContexts, ctx)
			return query
		})

	// Create query - this should call all authorization functions
	query := authRepo.Query(ctx)

	// Verify query was created
	assert.NotNil(t, query)

	// Verify authorization functions were called
	assert.Equal(t, 2, authCallCount)
	assert.Len(t, authContexts, 2)

	// Verify contexts were passed correctly
	for _, authCtx := range authContexts {
		user, ok := authCtx.Value("user").(mockUserContext)
		assert.True(t, ok)
		assert.Equal(t, "user123", user.UserID)
	}
}

func TestQuery_AuthorizationOrder(t *testing.T) {
	baseRepo := createTestRepository(t)
	ctx := context.Background()

	// Track the order of authorization function calls
	var callOrder []string

	authRepo := baseRepo.
		Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
			callOrder = append(callOrder, "first")
			return query
		}).
		Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
			callOrder = append(callOrder, "second")
			return query
		}).
		Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
			callOrder = append(callOrder, "third")
			return query
		})

	// Create query
	query := authRepo.Query(ctx)

	// Verify query was created
	assert.NotNil(t, query)

	// Verify authorization functions were called in the correct order
	assert.Equal(t, []string{"first", "second", "third"}, callOrder)
}

func TestQuery_AuthorizationModifiesQuery(t *testing.T) {
	baseRepo := createTestRepository(t)
	ctx := context.Background()

	// Add user context
	userCtx := mockUserContext{
		UserID: "user123",
		TeamID: "team456",
		Role:   "member",
	}
	ctx = context.WithValue(ctx, "user", userCtx)

	// Track query modifications
	var queryModified bool

	authRepo := baseRepo.Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
		// Simulate adding a WHERE clause for authorization
		queryModified = true

		// In a real scenario, this would be something like:
		// return query.Where(Users.TeamID.Eq(user.TeamID))
		// But for testing, we just verify the query object is passed correctly

		assert.NotNil(t, query)
		assert.Equal(t, "auth_test_users", query.repo.metadata.TableName)

		return query
	})

	// Create query
	query := authRepo.Query(ctx)

	// Verify authorization function was called and received the query
	assert.True(t, queryModified)
	assert.NotNil(t, query)
}

func TestQuery_AuthorizationWithRoleBasedLogic(t *testing.T) {
	baseRepo := createTestRepository(t)
	ctx := context.Background()

	testCases := []struct {
		name     string
		role     string
		expected string
	}{
		{
			name:     "Admin role",
			role:     "admin",
			expected: "admin_filter",
		},
		{
			name:     "Member role",
			role:     "member",
			expected: "member_filter",
		},
		{
			name:     "Guest role",
			role:     "guest",
			expected: "guest_filter",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			userCtx := mockUserContext{
				UserID: "user123",
				TeamID: "team456",
				Role:   tc.role,
			}
			testCtx := context.WithValue(ctx, "user", userCtx)

			var appliedFilter string

			authRepo := baseRepo.Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
				user, ok := ctx.Value("user").(mockUserContext)
				assert.True(t, ok)

				switch user.Role {
				case "admin":
					appliedFilter = "admin_filter"
				case "member":
					appliedFilter = "member_filter"
				case "guest":
					appliedFilter = "guest_filter"
				default:
					appliedFilter = "unknown_filter"
				}

				return query
			})

			// Create query
			query := authRepo.Query(testCtx)

			// Verify query was created and correct filter was applied
			assert.NotNil(t, query)
			assert.Equal(t, tc.expected, appliedFilter)
		})
	}
}

func TestAuthorize_NilFunction(t *testing.T) {
	baseRepo := createTestRepository(t)

	// This should not panic, but the behavior is undefined
	// In practice, developers shouldn't pass nil functions
	authRepo := baseRepo.Authorize(nil)

	// Verify the nil function was added
	assert.Len(t, authRepo.authorizeFuncs, 1)
	assert.Nil(t, authRepo.authorizeFuncs[0])
}

func TestQuery_WithNilAuthorizationFunction(t *testing.T) {
	baseRepo := createTestRepository(t)
	ctx := context.Background()

	// Add a nil authorization function
	authRepo := baseRepo.Authorize(nil)

	// This should panic when trying to call the nil function
	assert.Panics(t, func() {
		authRepo.Query(ctx)
	})
}

// Benchmark authorization overhead
func BenchmarkQuery_NoAuthorization(b *testing.B) {
	baseRepo := createTestRepository(b)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := baseRepo.Query(ctx)
		_ = query
	}
}

func BenchmarkQuery_SingleAuthorization(b *testing.B) {
	baseRepo := createTestRepository(b)
	ctx := context.Background()

	authRepo := baseRepo.Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
		return query
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := authRepo.Query(ctx)
		_ = query
	}
}

func BenchmarkQuery_MultipleAuthorization(b *testing.B) {
	baseRepo := createTestRepository(b)
	ctx := context.Background()

	authRepo := baseRepo.
		Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
			return query
		}).
		Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
			return query
		}).
		Authorize(func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
			return query
		})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := authRepo.Query(ctx)
		_ = query
	}
}

func BenchmarkAuthorize_ChainCreation(b *testing.B) {
	baseRepo := createTestRepository(b)

	authFunc := func(ctx context.Context, query *Query[AuthTestUser]) *Query[AuthTestUser] {
		return query
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		authRepo := baseRepo.Authorize(authFunc)
		_ = authRepo
	}
}
