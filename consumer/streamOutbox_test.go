package consumer

import (
	"testing"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/stretchr/testify/require"
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
