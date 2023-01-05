package model

import (
	"testing"

	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestStreamAccessors(t *testing.T) {

	objectID, err := primitive.ObjectIDFromHex("1234567890ABCDEF12345678")
	require.Nil(t, err)

	stream := NewStream()
	require.True(t, stream.SetBytes("streamId", id.ToBytes(objectID)))
	require.Equal(t, id.ToBytes(objectID), stream.GetBytes("streamId"))

	require.True(t, stream.SetBytes("parentId", id.ToBytes(objectID)))
	require.Equal(t, id.ToBytes(objectID), stream.GetBytes("parentId"))

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
