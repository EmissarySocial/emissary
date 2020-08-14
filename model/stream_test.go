package model

import (
	"testing"

	"github.com/benpate/path"
	"github.com/stretchr/testify/assert"
)

func TestStream_GetPath(t *testing.T) {

	stream := &Stream{
		Token:          "example",
		Label:          "Example Stream",
		Description:    "This is my example.",
		ThumbnailImage: "https://example.com/image.png",
	}

	{
		value, err := path.Get(stream, "label")
		assert.Nil(t, err)
		assert.Equal(t, "Example Stream", value)
	}

	{
		value, err := path.Get(stream, "description")
		assert.Nil(t, err)
		assert.Equal(t, "This is my example.", value)
	}

	{
		value, err := path.Get(stream, "thumbnailImage")
		assert.Nil(t, err)
		assert.Equal(t, "https://example.com/image.png", value)
	}

	{
		value, err := path.Get(stream, "token")
		assert.NotNil(t, err)
		assert.Nil(t, value)
	}
}

func TestStream_SetPath(t *testing.T) {

	stream := &Stream{}

	{
		err := path.Set(stream, "label", "Example Stream")
		assert.Nil(t, err)
		assert.Equal(t, "Example Stream", stream.Label)
	}

	{
		err := path.Set(stream, "description", "This is my example.")
		assert.Nil(t, err)
		assert.Equal(t, "This is my example.", stream.Description)
	}

	{
		err := path.Set(stream, "thumbnailImage", "https://example.com/image.png")
		assert.Nil(t, err)
		assert.Equal(t, "https://example.com/image.png", stream.ThumbnailImage)
	}

	{
		err := path.Set(stream, "unrecognized", "Bad Value")
		assert.NotNil(t, err)
	}

}
