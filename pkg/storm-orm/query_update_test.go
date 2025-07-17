package orm

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryUpdate(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	metadata := createTestUserMetadata()

	repo, err := NewRepository[TestUser](sqlxDB, metadata)
	require.NoError(t, err)

	t.Run("Update with WHERE clause", func(t *testing.T) {
		now := time.Now()
		user := &TestUser{
			ID:        1,
			Name:      "Updated Name",
			Email:     "updated@example.com",
			IsActive:  false,
			CreatedAt: now,
			UpdatedAt: now,
		}

		// Set up mock expectations
		mock.ExpectExec(`UPDATE users SET .* WHERE \(users\.id = \$4\)`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// Execute update with WHERE clause
		idCol := Column[int64]{Name: "id", Table: "users"}
		err := repo.Query(context.Background()).Where(idCol.Eq(1)).Update(user)
		require.NoError(t, err)

		// Verify all expectations were met
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Update with no matching records", func(t *testing.T) {
		now := time.Now()
		user := &TestUser{
			ID:        999,
			Name:      "Non-existent",
			Email:     "notfound@example.com",
			IsActive:  false,
			CreatedAt: now,
			UpdatedAt: now,
		}

		// Set up mock expectations
		mock.ExpectExec(`UPDATE users SET .* WHERE \(users\.id = \$4\)`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 0))

		// Execute update
		idCol := Column[int64]{Name: "id", Table: "users"}
		err := repo.Query(context.Background()).Where(idCol.Eq(999)).Update(user)
		assert.Error(t, err)
		assert.Equal(t, ErrNotFound, err.(*Error).Err)

		// Verify all expectations were met
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Update with nil record", func(t *testing.T) {
		// Execute update with nil record
		idCol := Column[int64]{Name: "id", Table: "users"}
		err := repo.Query(context.Background()).Where(idCol.Eq(1)).Update(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "record cannot be nil")
	})
}

func TestStringColumnAdvanced(t *testing.T) {
	t.Run("Regexp", func(t *testing.T) {
		col := StringColumn{Column: Column[string]{Name: "email", Table: "users"}}
		condition := col.Regexp(".*@example.com$")
		sql, _, err := condition.ToSqlizer().ToSql()
		require.NoError(t, err)
		assert.Equal(t, "users.email ~ ?", sql)
	})

	t.Run("FullTextSearch", func(t *testing.T) {
		col := StringColumn{Column: Column[string]{Name: "content", Table: "posts"}}
		condition := col.FullTextSearch("PostgreSQL tutorial")
		sql, _, err := condition.ToSqlizer().ToSql()
		require.NoError(t, err)
		assert.Equal(t, "posts.content @@ plainto_tsquery('english', ?)", sql)
	})

	t.Run("FullTextSearchLang", func(t *testing.T) {
		col := StringColumn{Column: Column[string]{Name: "content", Table: "posts"}}
		condition := col.FullTextSearchLang("spanish", "base de datos")
		sql, _, err := condition.ToSqlizer().ToSql()
		require.NoError(t, err)
		assert.Equal(t, "posts.content @@ plainto_tsquery(?, ?)", sql)
	})
}

func TestColumnNotIn(t *testing.T) {
	col := Column[int]{Name: "id", Table: "users"}
	condition := col.NotIn(1, 2, 3, 4, 5)
	sql, args, err := condition.ToSqlizer().ToSql()
	require.NoError(t, err)
	assert.Equal(t, "users.id NOT IN (?,?,?,?,?)", sql)
	assert.Equal(t, []interface{}{1, 2, 3, 4, 5}, args)
}

func TestJSONBColumnPath(t *testing.T) {
	t.Run("Path", func(t *testing.T) {
		col := JSONBColumn{Column: Column[interface{}]{Name: "metadata", Table: "users"}}
		pathCol := col.Path("settings")
		assert.Equal(t, "(users.metadata->'settings')", pathCol.String())
	})

	t.Run("PathText", func(t *testing.T) {
		col := JSONBColumn{Column: Column[interface{}]{Name: "metadata", Table: "users"}}
		pathCol := col.PathText("name")
		assert.Equal(t, "(users.metadata->>'name')", pathCol.String())
	})
}

func TestTableMethods(t *testing.T) {
	table := &Table{
		Name:        "users",
		Schema:      "public",
		PrimaryKeys: []string{"id"},
	}

	t.Run("FullName", func(t *testing.T) {
		assert.Equal(t, "public.users", table.FullName())
	})

	t.Run("HasPrimaryKey", func(t *testing.T) {
		assert.True(t, table.HasPrimaryKey("id"))
		assert.False(t, table.HasPrimaryKey("email"))
	})

	t.Run("IsCompositePrimaryKey", func(t *testing.T) {
		assert.False(t, table.IsCompositePrimaryKey())

		compositeTable := &Table{
			PrimaryKeys: []string{"user_id", "role_id"},
		}
		assert.True(t, compositeTable.IsCompositePrimaryKey())
	})

	t.Run("GetPrimaryKeyColumns", func(t *testing.T) {
		pks := table.GetPrimaryKeyColumns()
		assert.Equal(t, []string{"id"}, pks)
	})
}
