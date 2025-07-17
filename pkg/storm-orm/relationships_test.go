package orm

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRelationshipManager(t *testing.T) {
	t.Run("newRelationshipManager", func(t *testing.T) {
		mgr := newRelationshipManager("users")
		assert.NotNil(t, mgr)
		assert.NotNil(t, mgr.relationships)
		assert.Equal(t, "users", mgr.sourceTable)
	})

	t.Run("parseRelationships empty struct", func(t *testing.T) {
		type EmptyModel struct {
			ID int64
		}
		mgr := newRelationshipManager("users")
		modelType := reflect.TypeOf(EmptyModel{})
		err := mgr.parseRelationships(modelType)
		assert.NoError(t, err)
		assert.False(t, mgr.hasRelationships())
	})

	t.Run("getRelationships", func(t *testing.T) {
		mgr := newRelationshipManager("users")
		mgr.relationships["posts"] = relationshipDef{
			FieldName: "Posts",
			Type:      "has_many",
		}
		mgr.relationships["user"] = relationshipDef{
			FieldName: "User",
			Type:      "belongs_to",
		}

		rels := mgr.getRelationships()
		assert.Len(t, rels, 2)
	})

	t.Run("getRelationship", func(t *testing.T) {
		mgr := newRelationshipManager("users")
		expectedRel := relationshipDef{
			FieldName: "Posts",
			Type:      "has_many",
		}
		mgr.relationships["posts"] = expectedRel

		// Test existing relationship
		rel := mgr.getRelationship("posts")
		assert.NotNil(t, rel)
		assert.Equal(t, expectedRel, *rel)

		// Test non-existing relationship
		rel = mgr.getRelationship("comments")
		assert.Nil(t, rel)
	})

	t.Run("hasRelationships", func(t *testing.T) {
		mgr := newRelationshipManager("users")
		assert.False(t, mgr.hasRelationships())

		mgr.relationships["posts"] = relationshipDef{
			FieldName: "Posts",
			Type:      "has_many",
		}
		assert.True(t, mgr.hasRelationships())
	})

}
