package orm

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryBuildQuery(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	metadata := createTestUserMetadata()

	repo, err := NewRepository[TestUser](sqlxDB, metadata)
	require.NoError(t, err)

	t.Run("buildQuery basic select", func(t *testing.T) {
		query := repo.Query(context.Background())
		sql, _, err := query.buildQuery()
		assert.NoError(t, err)
		assert.NotEmpty(t, sql)
		assert.Contains(t, sql, "SELECT")
		assert.Contains(t, sql, "FROM users")
	})

	t.Run("buildQuery with where clause", func(t *testing.T) {
		query := repo.Query(context.Background())
		idCol := Column[int64]{Name: "id", Table: "users"}
		query.Where(idCol.Eq(1))

		sql, args, err := query.buildQuery()
		assert.NoError(t, err)
		assert.NotEmpty(t, sql)
		assert.Contains(t, sql, "WHERE")
		assert.Len(t, args, 1)
	})
}
