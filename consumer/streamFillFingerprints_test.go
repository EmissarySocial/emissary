package consumer

import (
	"testing"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/stretchr/testify/require"
)

// TestStreamFillFingerprints_RefusesABadPageMarker confirms that a task whose page marker cannot be
// read fails at once, before it reaches the factory, because a retry would fail the same way
func TestStreamFillFingerprints_RefusesABadPageMarker(t *testing.T) {

	result := StreamFillFingerprints(&service.Factory{}, nil, mapof.Any{"after": "not-an-id"})
	require.Error(t, result.Error)
	require.Equal(t, queue.ResultStatusFailure, result.Status)
}

// TestPreProcessor_StreamFillFingerprints confirms that the fill runs in the background, behind
// everything a person is waiting for
func TestPreProcessor_StreamFillFingerprints(t *testing.T) {

	task := queue.NewTask(service.StreamFillFingerprintsTaskName, mapof.Any{})
	require.NoError(t, PreProcessor(&task))
	require.Equal(t, 256, task.Priority)
}
