package user

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func encodePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{R: 1, G: 2, B: 3, A: 255})
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

// The bytes are served back to browsers, so what the file IS matters more than
// what the client called it.
func TestDecodeProfilePicture(t *testing.T) {
	t.Run("accepts a real PNG and reports its type", func(t *testing.T) {
		pic, err := DecodeProfilePicture(encodePNG(t, 64, 64))
		require.NoError(t, err)
		assert.Equal(t, "image/png", pic.ContentType)
		assert.NotEmpty(t, pic.ETag)
	})

	t.Run("accepts JPEG and GIF", func(t *testing.T) {
		var jbuf bytes.Buffer
		require.NoError(t, jpeg.Encode(&jbuf, image.NewRGBA(image.Rect(0, 0, 8, 8)), nil))
		pic, err := DecodeProfilePicture(jbuf.Bytes())
		require.NoError(t, err)
		assert.Equal(t, "image/jpeg", pic.ContentType)

		var gbuf bytes.Buffer
		require.NoError(t, gif.Encode(&gbuf, image.NewRGBA(image.Rect(0, 0, 8, 8)), nil))
		pic, err = DecodeProfilePicture(gbuf.Bytes())
		require.NoError(t, err)
		assert.Equal(t, "image/gif", pic.ContentType)
	})

	// SVG is XML that can carry <script>. Serving one from this origin is stored
	// XSS against everyone who views the avatar, and no decode can prove a given
	// SVG is "just an image".
	t.Run("rejects SVG", func(t *testing.T) {
		_, err := DecodeProfilePicture([]byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`))
		require.Error(t, err)
	})

	// The classic bypass: name it .png, claim image/png, send HTML. Only
	// decoding catches it.
	t.Run("rejects HTML disguised as an image", func(t *testing.T) {
		_, err := DecodeProfilePicture([]byte("<html><script>alert(1)</script></html>"))
		require.Error(t, err)
	})

	t.Run("rejects an empty file", func(t *testing.T) {
		_, err := DecodeProfilePicture(nil)
		require.Error(t, err)
	})

	t.Run("rejects anything over the size cap", func(t *testing.T) {
		_, err := DecodeProfilePicture(make([]byte, MaxProfilePictureBytes+1))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "larger than")
	})

	// A header can claim an enormous canvas in a handful of bytes. Reading only
	// the header means refusing this costs nothing.
	t.Run("rejects absurd dimensions", func(t *testing.T) {
		_, err := DecodeProfilePicture(encodePNG(t, 9000, 10))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pixels")
	})

	// The ETag is what makes a repeat view a 304 instead of a 2 MiB read, so it
	// must follow the content and nothing else.
	t.Run("the etag is content-derived and stable", func(t *testing.T) {
		a, err := DecodeProfilePicture(encodePNG(t, 32, 32))
		require.NoError(t, err)
		b, err := DecodeProfilePicture(encodePNG(t, 32, 32))
		require.NoError(t, err)
		assert.Equal(t, a.ETag, b.ETag, "same bytes must not invalidate a cached copy")

		c, err := DecodeProfilePicture(encodePNG(t, 33, 33))
		require.NoError(t, err)
		assert.NotEqual(t, a.ETag, c.ETag, "different bytes must produce a different validator")
	})
}

// A conditional request has to be satisfiable from the validator alone.
func TestETagMatches(t *testing.T) {
	assert.True(t, etagMatches(`"abc"`, "abc"))
	assert.True(t, etagMatches(`W/"abc"`, "abc"), "weak validators are still a match")
	assert.True(t, etagMatches(`"other", "abc"`, "abc"), "a list is matched entrywise")
	assert.True(t, etagMatches(`*`, "abc"))
	assert.False(t, etagMatches(`"different"`, "abc"))
	assert.False(t, etagMatches(`"abc"`, ""), "no stored validator can never match")
}
