package orm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSONDataString(t *testing.T) {
	t.Run("empty JSONData", func(t *testing.T) {
		j := &JSONData{}
		assert.Equal(t, "NULL", j.String())
	})

	t.Run("JSONData with content", func(t *testing.T) {
		data := map[string]interface{}{
			"name": "John",
			"age":  30,
		}
		j := NewJSONData(data)
		expected := `{"age":30,"name":"John"}`
		assert.Equal(t, expected, j.String())
	})

	t.Run("JSONData with null", func(t *testing.T) {
		j := NewNullJSONData()
		assert.Equal(t, "NULL", j.String())
	})

	t.Run("JSONData set to nil", func(t *testing.T) {
		j := &JSONData{}
		j.Set(nil)
		assert.Equal(t, "NULL", j.String())
	})
}
