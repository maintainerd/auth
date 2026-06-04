package jsonutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"
)

func TestJSONToMap(t *testing.T) {
	t.Run("nil/empty returns empty map", func(t *testing.T) {
		result := JSONToMap(nil)
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("invalid JSON returns empty map", func(t *testing.T) {
		result := JSONToMap(datatypes.JSON([]byte("not-json")))
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("valid JSON returns map", func(t *testing.T) {
		result := JSONToMap(datatypes.JSON([]byte(`{"a":"b"}`)))
		assert.Equal(t, "b", result["a"])
	})

	t.Run("null JSON returns empty map", func(t *testing.T) {
		result := JSONToMap(datatypes.JSON([]byte("null")))
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})
}
