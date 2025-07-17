package orm

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSONDataString(t *testing.T) {
	t.Run("empty JSONData", func(t *testing.T) {
		j := &JSONData{}
		assert.Equal(t, "null", j.String())
	})

	t.Run("JSONData with content", func(t *testing.T) {
		data := map[string]interface{}{
			"name": "John",
			"age":  30,
		}
		raw, _ := json.Marshal(data)
		j := &JSONData{RawMessage: json.RawMessage(raw)}
		assert.Equal(t, string(raw), j.String())
	})

	t.Run("JSONData with null", func(t *testing.T) {
		j := &JSONData{RawMessage: json.RawMessage("null")}
		assert.Equal(t, "null", j.String())
	})
}
