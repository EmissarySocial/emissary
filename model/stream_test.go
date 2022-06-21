package model

import (
	"testing"

	"github.com/benpate/rosetta/path"
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
		value, ok := path.GetOK(stream, "label")
		assert.True(t, ok)
		assert.Equal(t, "Example Stream", value)
	}

	{
		value, ok := path.GetOK(stream, "description")
		assert.True(t, ok)
		assert.Equal(t, "This is my example.", value)
	}

	{
		value, ok := path.GetOK(stream, "thumbnailImage")
		assert.True(t, ok)
		assert.Equal(t, "https://example.com/image.png", value)
	}

	{
		value, ok := path.GetOK(stream, "token")
		assert.False(t, ok)
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
