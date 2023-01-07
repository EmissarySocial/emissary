package model

import (
	"testing"

	"github.com/benpate/rosetta/path"
	"github.com/stretchr/testify/require"
)

func TestStreamAccessors(t *testing.T) {

	stream := NewStream()

	require.True(t, stream.SetString("streamId", "1234567890abcdef12345678"))
	require.Equal(t, "1234567890abcdef12345678", stream.GetString("streamId"))

	require.True(t, stream.SetString("parentId", "0001234567890abcdef12345"))
	require.Equal(t, "0001234567890abcdef12345", stream.GetString("parentId"))

	require.True(t, stream.SetString("token", "TEST_TOKEN"))
	require.Equal(t, "TEST_TOKEN", stream.GetString("token"))

	require.True(t, stream.SetString("topLevelId", "TEST_TOPLEVELID"))
	require.Equal(t, "TEST_TOPLEVELID", stream.GetString("topLevelId"))

	require.True(t, stream.SetString("templateId", "TEST_TEMPLATEID"))
	require.Equal(t, "TEST_TEMPLATEID", stream.GetString("templateId"))

	require.True(t, stream.SetString("stateId", "TEST_STATEID"))
	require.Equal(t, "TEST_STATEID", stream.GetString("stateId"))

	require.True(t, stream.SetBool("asFeature", true))
	require.True(t, stream.GetBool("asFeature"))
}

func TestStreamPath(t *testing.T) {

	stream := NewStream()

	require.True(t, path.SetString(&stream, "streamId", "1234567890abcdef12345678"))
	require.Equal(t, "1234567890abcdef12345678", path.GetString(&stream, "streamId"))

	require.True(t, path.SetString(&stream, "parentId", "000000000000000000000000"))
	require.Equal(t, "000000000000000000000000", path.GetString(&stream, "parentId"))

	require.True(t, path.SetString(&stream, "token", "TEST_TOKEN"))
	require.Equal(t, "TEST_TOKEN", path.GetString(&stream, "token"))

	require.True(t, path.SetString(&stream, "topLevelId", "TEST_TOPLEVELID"))
	require.Equal(t, "TEST_TOPLEVELID", path.GetString(&stream, "topLevelId"))

	require.True(t, path.SetString(&stream, "templateId", "TEST_TEMPLATEID"))
	require.Equal(t, "TEST_TEMPLATEID", path.GetString(&stream, "templateId"))

	require.True(t, path.SetString(&stream, "stateId", "TEST_STATEID"))
	require.Equal(t, "TEST_STATEID", path.GetString(&stream, "stateId"))

	require.True(t, path.SetString(&stream, "document.summary", "TEST_SUMMARY"))
	require.Equal(t, "TEST_SUMMARY", path.GetString(&stream, "document.summary"))

	require.True(t, path.SetString(&stream, "document.author.name", "TEST_AUTHOR_NAME"))
	require.Equal(t, "TEST_AUTHOR_NAME", path.GetString(&stream, "document.author.name"))

	require.True(t, path.SetString(&stream, "document.label", "TEST_LABEL"))
	require.Equal(t, "TEST_LABEL", path.GetString(&stream, "document.label"))

	require.True(t, path.SetString(&stream, "document.summary", "TEST_SUMMARY"))
	require.Equal(t, "TEST_SUMMARY", path.GetString(&stream, "document.summary"))

	require.True(t, path.SetString(&stream, "document.imageUrl", "TEST_IMAGE_URL"))
	require.Equal(t, "TEST_IMAGE_URL", path.GetString(&stream, "document.imageUrl"))

	require.True(t, path.SetInt64(&stream, "document.publishDate", 123456789))
	require.Equal(t, int64(123456789), path.GetInt64(&stream, "document.publishDate"))

	require.True(t, path.SetBool(&stream, "asFeature", true))
	require.True(t, path.GetBool(&stream, "asFeature"))
}

func TestStreamPath_Breaking(t *testing.T) {
	stream := NewStream()
	stream.Document.Summary = "TEST_SUMMARY"
	require.Equal(t, "TEST_SUMMARY", path.GetString(&stream, "document.summary"))
}
