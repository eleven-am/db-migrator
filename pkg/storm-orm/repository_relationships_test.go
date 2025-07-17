package orm

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositoryRelationships(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	metadata := createTestUserMetadata()

	repo, err := NewRepository[TestUser](sqlxDB, metadata)
	require.NoError(t, err)

	t.Run("relationships method", func(t *testing.T) {
		rels := repo.relationships()
		assert.NotNil(t, rels)
	})

	t.Run("WithRelationships", func(t *testing.T) {
		result := repo.WithRelationships(context.Background())
		assert.NotNil(t, result)
		// WithRelationships returns a Query
		assert.IsType(t, &Query[TestUser]{}, result)
	})
}
