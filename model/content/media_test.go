package content

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtension(t *testing.T) {

	media := Media{}

	media.URL = "http://example.com/file.txt"
	assert.Equal(t, ".txt", media.Extension())

	media.URL = "http://example.com/badvalue."
	assert.Equal(t, ".", media.Extension())

	media.URL = "http://example.com/nothing"
	assert.Equal(t, "", media.Extension())

}

func TestMimeType(t *testing.T) {

	media := Media{}

	media.URL = "http://example.com/image.jpg"
	assert.Equal(t, ".jpg", media.Extension())
	assert.Equal(t, "image/jpeg", media.MimeType())

	media.URL = "http://example.com/image.png"
	assert.Equal(t, ".png", media.Extension())
	assert.Equal(t, "image/png", media.MimeType())

	media.URL = "http://example.com/video.mov"
	assert.Equal(t, ".mov", media.Extension())
	assert.Equal(t, "video/quicktime", media.MimeType())

	media.URL = "http://example.com/video.mp4"
	assert.Equal(t, ".mp4", media.Extension())
	assert.Equal(t, "video/mp4", media.MimeType())
}

func TestMediaHTML(t *testing.T) {

	media := Media{}

	media.URL = "http://example.com/image.jpg"
	assert.Equal(t, "image", media.MimeCategory())
	assert.Equal(t, `<img src="http://example.com/image.jpg">`, media.HTML())

	media.URL = "http://example.com/image.png"
	media.Height = 100
	media.Width = 200
	assert.Equal(t, "image", media.MimeCategory())
	assert.Equal(t, `<img src="http://example.com/image.png" height="100" width="200">`, media.HTML())

	// Reset Media info
	media.Height = 0
	media.Width = 0

	media.URL = "http://example.com/video.mov"
	assert.Equal(t, "video", media.MimeCategory())
	assert.Equal(t, `<video src="http://example.com/video.mov"></video>`, media.HTML())

	media.URL = "http://example.com/video.mp4"
	media.Height = 100
	media.Width = 200

	assert.Equal(t, "video", media.MimeCategory())
	assert.Equal(t, `<video src="http://example.com/video.mp4" height="100" width="200"></video>`, media.HTML())

	// Test unrecognizable MIME-TYPES
	media.Type = "UNRECOGNIZED-TYPE"
	assert.Equal(t, "", media.HTML())
}
