package orm

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFindByID tests the FindByID operation
func TestFindByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	metadata := createTestUserMetadata()

	repo, err := NewRepository[TestUser](sqlxDB, metadata)
	require.NoError(t, err)

	t.Run("FindByID with existing record", func(t *testing.T) {
		userID := 1
		now := time.Now()

		// Set up mock expectation
		mock.ExpectQuery(`SELECT .* FROM users WHERE id = \$1`).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "is_active", "created_at", "updated_at"}).
				AddRow(userID, "John Doe", "john@example.com", true, now, now))

		// Execute FindByID
		user, err := repo.FindByID(context.Background(), userID)
		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, userID, user.ID)
		assert.Equal(t, "John Doe", user.Name)
		assert.Equal(t, "john@example.com", user.Email)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindByID with non-existing record", func(t *testing.T) {
		userID := 999

		// Set up mock expectation
		mock.ExpectQuery(`SELECT .* FROM users WHERE id = \$1`).
			WithArgs(userID).
			WillReturnError(sql.ErrNoRows)

		// Execute FindByID
		user, err := repo.FindByID(context.Background(), userID)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "not found")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestDeleteRecord tests the DeleteRecord operation
func TestDeleteRecord(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	metadata := createTestUserMetadata()

	repo, err := NewRepository[TestUser](sqlxDB, metadata)
	require.NoError(t, err)

	t.Run("DeleteRecord with existing record", func(t *testing.T) {
		user := &TestUser{
			ID:       1,
			Name:     "John Doe",
			Email:    "john@example.com",
			IsActive: true,
		}

		// Set up mock expectation
		mock.ExpectExec(`DELETE FROM users WHERE id = \$1`).
			WithArgs(user.ID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// Execute DeleteRecord
		err := repo.DeleteRecord(context.Background(), user)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("DeleteRecord with non-existing record", func(t *testing.T) {
		user := &TestUser{
			ID:       999,
			Name:     "Non Existent",
			Email:    "none@example.com",
			IsActive: false,
		}

		// Set up mock expectation
		mock.ExpectExec(`DELETE FROM users WHERE id = \$1`).
			WithArgs(user.ID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		// Execute DeleteRecord
		err := repo.DeleteRecord(context.Background(), user)
		assert.Error(t, err)
		assert.Equal(t, ErrNotFound, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestCreateMany tests the CreateMany operation
func TestCreateMany(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	metadata := createTestUserMetadata()

	repo, err := NewRepository[TestUser](sqlxDB, metadata)
	require.NoError(t, err)

	t.Run("CreateMany with multiple records", func(t *testing.T) {
		users := []*TestUser{
			{Name: "User1", Email: "user1@example.com", IsActive: true},
			{Name: "User2", Email: "user2@example.com", IsActive: false},
		}

		// Set up mock expectations
		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO users`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectCommit()

		// Execute CreateMany
		err := repo.CreateMany(context.Background(), users)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("CreateMany with empty slice", func(t *testing.T) {
		users := []*TestUser{}

		// Execute CreateMany
		err := repo.CreateMany(context.Background(), users)
		require.NoError(t, err)

		// No SQL should be executed
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("CreateMany with transaction error", func(t *testing.T) {
		users := []*TestUser{
			{Name: "User1", Email: "user1@example.com", IsActive: true},
		}

		// Set up mock expectations
		mock.ExpectBegin().WillReturnError(assert.AnError)

		// Execute CreateMany
		err := repo.CreateMany(context.Background(), users)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to begin transaction")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestUpsert tests the Upsert operation
func TestUpsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	metadata := createTestUserMetadata()

	repo, err := NewRepository[TestUser](sqlxDB, metadata)
	require.NoError(t, err)

	t.Run("Upsert with update on conflict", func(t *testing.T) {
		user := &TestUser{
			Name:     "John Doe",
			Email:    "john@example.com",
			IsActive: true,
		}

		opts := UpsertOptions{
			ConflictColumns: []string{"email"},
			UpdateColumns:   []string{"name", "is_active"},
		}

		// Set up mock expectation
		mock.ExpectExec(`INSERT INTO users .* ON CONFLICT \(email\) DO UPDATE SET`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// Execute Upsert
		err := repo.Upsert(context.Background(), user, opts)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Upsert with do nothing on conflict", func(t *testing.T) {
		user := &TestUser{
			Name:     "Jane Doe",
			Email:    "jane@example.com",
			IsActive: false,
		}

		opts := UpsertOptions{
			ConflictColumns: []string{"email"},
			// Empty UpdateColumns will auto-populate with all non-conflict columns
		}

		// Set up mock expectation
		mock.ExpectExec(`INSERT INTO users .* ON CONFLICT \(email\) DO UPDATE SET`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// Execute Upsert
		err := repo.Upsert(context.Background(), user, opts)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestUpsertMany tests the UpsertMany operation
func TestUpsertMany(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	metadata := createTestUserMetadata()

	repo, err := NewRepository[TestUser](sqlxDB, metadata)
	require.NoError(t, err)

	t.Run("UpsertMany with multiple records", func(t *testing.T) {
		users := []TestUser{
			{Name: "User1", Email: "user1@example.com", IsActive: true},
			{Name: "User2", Email: "user2@example.com", IsActive: false},
		}

		opts := UpsertOptions{
			ConflictColumns: []string{"email"},
			UpdateColumns:   []string{"name", "is_active"},
		}

		// Set up mock expectations
		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO users .* ON CONFLICT \(email\) DO UPDATE SET`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectCommit()

		// Execute UpsertMany
		err := repo.UpsertMany(context.Background(), users, opts)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("UpsertMany with empty slice", func(t *testing.T) {
		users := []TestUser{}

		opts := UpsertOptions{
			ConflictColumns: []string{"email"},
		}

		// Execute UpsertMany
		err := repo.UpsertMany(context.Background(), users, opts)
		require.NoError(t, err)

		// No SQL should be executed
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestBulkUpdate tests the BulkUpdate operation
func TestBulkUpdate(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	metadata := createTestUserMetadata()

	repo, err := NewRepository[TestUser](sqlxDB, metadata)
	require.NoError(t, err)

	t.Run("BulkUpdate with multiple records", func(t *testing.T) {
		users := []TestUser{
			{ID: 1, Name: "Updated User1", Email: "user1@example.com", IsActive: true},
			{ID: 2, Name: "Updated User2", Email: "user2@example.com", IsActive: false},
		}

		// Set up mock expectations
		mock.ExpectBegin()
		// The actual SQL is complex with CTEs, so we use a regex pattern
		mock.ExpectExec(`WITH updates\(.*\) AS \( VALUES.*UPDATE users SET`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectCommit()

		// Execute BulkUpdate
		opts := BulkUpdateOptions{
			UpdateColumns: []string{"name", "is_active"},
			WhereColumns:  []string{"id"},
		}
		rowsAffected, err := repo.BulkUpdate(context.Background(), users, opts)
		require.NoError(t, err)
		assert.Equal(t, int64(2), rowsAffected)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("BulkUpdate with empty slice", func(t *testing.T) {
		users := []TestUser{}

		// Execute BulkUpdate
		opts := BulkUpdateOptions{
			UpdateColumns: []string{"name"},
			WhereColumns:  []string{"id"},
		}
		rowsAffected, err := repo.BulkUpdate(context.Background(), users, opts)
		require.NoError(t, err)
		assert.Equal(t, int64(0), rowsAffected)

		// No SQL should be executed
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestBuildHelperMethods tests the helper methods
func TestBuildHelperMethods(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	metadata := createTestUserMetadata()

	repo, err := NewRepository[TestUser](sqlxDB, metadata)
	require.NoError(t, err)

	t.Run("buildUpdateSetClause", func(t *testing.T) {
		updateColumns := []string{"name", "email", "is_active"}
		whereColumns := []string{"id"}

		result := repo.buildUpdateSetClause(updateColumns, whereColumns)
		assert.Contains(t, result, "name = updates.name")
		assert.Contains(t, result, "email = updates.email")
		assert.Contains(t, result, "is_active = updates.is_active")
	})

	t.Run("buildWhereClause", func(t *testing.T) {
		whereColumns := []string{"id", "email"}

		result := repo.buildWhereClause(whereColumns)
		assert.Contains(t, result, "users.id = updates.id")
		assert.Contains(t, result, "users.email = updates.email")
		assert.Contains(t, result, " AND ")
	})
}
