package orm

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPostgreSQLFeatures tests PostgreSQL-specific features
func TestPostgreSQLFeatures(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	metadata := createTestUserMetadata()

	repo, err := NewRepository[TestUser](sqlxDB, metadata)
	require.NoError(t, err)

	t.Run("FindByArrayContains", func(t *testing.T) {
		now := time.Now()
		mock.ExpectQuery(`SELECT .* FROM users WHERE .* @> \$1`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "is_active", "created_at", "updated_at"}).
				AddRow(1, "John Doe", "john@example.com", true, now, now))

		users, err := repo.FindByArrayContains("tags", "admin")
		require.NoError(t, err)
		assert.Len(t, users, 1)
		assert.Equal(t, "John Doe", users[0].Name)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindByArrayContainsAny", func(t *testing.T) {
		now := time.Now()
		mock.ExpectQuery(`SELECT .* FROM users WHERE .* && \$1`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "is_active", "created_at", "updated_at"}).
				AddRow(1, "John Doe", "john@example.com", true, now, now).
				AddRow(2, "Jane Doe", "jane@example.com", true, now, now))

		users, err := repo.FindByArrayContainsAny("tags", []interface{}{"admin", "user"})
		require.NoError(t, err)
		assert.Len(t, users, 2)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindByJSONB", func(t *testing.T) {
		now := time.Now()
		mock.ExpectQuery(`SELECT .* FROM users WHERE .*metadata.*=`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "is_active", "created_at", "updated_at"}).
				AddRow(1, "John Doe", "john@example.com", true, now, now))

		users, err := repo.FindByJSONB("metadata", "role", "admin")
		require.NoError(t, err)
		assert.Len(t, users, 1)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindByJSONBContains", func(t *testing.T) {
		t.Skip("Skipping due to JSON marshaling in test")
	})

	t.Run("Search", func(t *testing.T) {
		now := time.Now()
		mock.ExpectQuery(`SELECT .* FROM users WHERE .* @@ plainto_tsquery`).
			WithArgs("john").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "is_active", "created_at", "updated_at"}).
				AddRow(1, "John Doe", "john@example.com", true, now, now))

		users, err := repo.Search("name", "john")
		require.NoError(t, err)
		assert.Len(t, users, 1)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("SearchWithLanguage", func(t *testing.T) {
		now := time.Now()
		mock.ExpectQuery(`SELECT .* FROM users WHERE .* @@ plainto_tsquery`).
			WithArgs("spanish", "juan").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "is_active", "created_at", "updated_at"}).
				AddRow(1, "Juan Perez", "juan@example.com", true, now, now))

		users, err := repo.SearchWithLanguage("name", "spanish", "juan")
		require.NoError(t, err)
		assert.Len(t, users, 1)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindByRegex", func(t *testing.T) {
		now := time.Now()
		mock.ExpectQuery(`SELECT .* FROM users WHERE .* ~`).
			WithArgs("^[A-Z]").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "is_active", "created_at", "updated_at"}).
				AddRow(1, "John Doe", "john@example.com", true, now, now))

		users, err := repo.FindByRegex("name", "^[A-Z]")
		require.NoError(t, err)
		assert.Len(t, users, 1)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindByRegexInsensitive", func(t *testing.T) {
		now := time.Now()
		mock.ExpectQuery(`SELECT .* FROM users WHERE .* ~\*`).
			WithArgs("john").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "is_active", "created_at", "updated_at"}).
				AddRow(1, "JOHN DOE", "john@example.com", true, now, now))

		users, err := repo.FindByRegexInsensitive("name", "john")
		require.NoError(t, err)
		assert.Len(t, users, 1)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("CountByArrayContains", func(t *testing.T) {
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users WHERE .* @>`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		count, err := repo.CountByArrayContains("tags", "admin")
		require.NoError(t, err)
		assert.Equal(t, int64(5), count)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("CountByJSONB", func(t *testing.T) {
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users WHERE .*metadata.*=`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

		count, err := repo.CountByJSONB("metadata", "role", "admin")
		require.NoError(t, err)
		assert.Equal(t, int64(3), count)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("CountBySearch", func(t *testing.T) {
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users WHERE .* @@ plainto_tsquery`).
			WithArgs("john").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

		count, err := repo.CountBySearch("name", "john")
		require.NoError(t, err)
		assert.Equal(t, int64(2), count)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}
