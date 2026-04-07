package ptr_test

import (
	"testing"
	"time"

	"github.com/maintainerd/auth/internal/ptr"
	"github.com/stretchr/testify/assert"
)

func TestPtrOrNil(t *testing.T) {
	t.Run("returns nil for empty string", func(t *testing.T) {
		assert.Nil(t, ptr.PtrOrNil(""))
	})
	t.Run("returns pointer for non-empty string", func(t *testing.T) {
		s := ptr.PtrOrNil("hello")
		assert.NotNil(t, s)
		assert.Equal(t, "hello", *s)
	})
}

func TestPtr(t *testing.T) {
	t.Run("returns pointer for any string", func(t *testing.T) {
		s := ptr.Ptr("")
		assert.NotNil(t, s)
		assert.Equal(t, "", *s)
	})
	t.Run("returns pointer for non-empty string", func(t *testing.T) {
		s := ptr.Ptr("hello")
		assert.NotNil(t, s)
		assert.Equal(t, "hello", *s)
	})
}

func TestTimePtr(t *testing.T) {
	now := time.Now()
	tp := ptr.TimePtr(now)
	assert.NotNil(t, tp)
	assert.Equal(t, now, *tp)
}
