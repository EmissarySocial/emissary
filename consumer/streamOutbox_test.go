package consumer

import (
	"testing"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

// TestStreamOutbox_RefusesATaskWithNoStream confirms that a task which names no Stream fails at once,
// before it reaches the factory, because a retry would fail the same way
func TestStreamOutbox_RefusesATaskWithNoStream(t *testing.T) {

	for _, args := range []mapof.Any{nil, {}, {"streamId": "not-an-id"}, {"streamId": "000000000000000000000000"}} {
		result := StreamOutbox(&service.Factory{}, nil, args)
		require.Error(t, result.Error)
		require.Equal(t, queue.ResultStatusFailure, result.Status)
	}
}

// TestStreamOutbox_ArgumentsSurviveStorage confirms that the changes a save passes to the task
// can still be read after the queue has stored the task as BSON and loaded it again
func TestStreamOutbox_ArgumentsSurviveStorage(t *testing.T) {

	original := mapof.Any{
		"streamId":           "6b072f41a1f05a9f0f6059e7",
		"syndicationAdded":   []string{"bandwagon", "tidal"},
		"syndicationDeleted": []string{"spotify"},
		"republish":          true,
	}

	encoded, err := bson.Marshal(original)
	require.NoError(t, err)

	stored := mapof.Any{}
	require.NoError(t, bson.Unmarshal(encoded, &stored))

	require.Equal(t, []string{"bandwagon", "tidal"}, stored.GetSliceOfString("syndicationAdded"))
	require.Equal(t, []string{"spotify"}, stored.GetSliceOfString("syndicationDeleted"))
	require.True(t, stored.GetBool("republish"))

	// A task that carries none of them reads as "no changes"
	require.Empty(t, mapof.Any{}.GetSliceOfString("syndicationAdded"))
	require.False(t, mapof.Any{}.GetBool("republish"))
}
