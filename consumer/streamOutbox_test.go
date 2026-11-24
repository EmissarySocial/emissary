package consumer

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestStreamOutbox_Succeeds confirms that the task completes, so that the tasks a save queues do not pile up
func TestStreamOutbox_Succeeds(t *testing.T) {

	for _, args := range []mapof.Any{nil, {}, {"streamId": "not-an-id"}, {"streamId": "000000000000000000000000"}} {
		result := StreamOutbox(nil, nil, args)
		require.NoError(t, result.Error)
	}
}
