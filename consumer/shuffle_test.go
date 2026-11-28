package consumer

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/stretchr/testify/require"
)

// TestShuffle_IsRegistered confirms that the task reaches a handler, and that a Domain this
// server does not know yet is retried
func TestShuffle_IsRegistered(t *testing.T) {

	task := queue.Task{Name: "Shuffle", Arguments: mapof.Any{"hostname": "example.com"}}
	result := New(unresolvableFactory{}).Run(task)

	require.NotEqual(t, queue.ResultStatusIgnored, result.Status, "the task reaches no handler")
	require.Equal(t, queue.ResultStatusError, result.Status, "an unknown hostname may become known later")
}

// TestShuffle_Priority confirms that the task waits in storage, where any server can take it
func TestShuffle_Priority(t *testing.T) {

	task := queue.Task{Name: "Shuffle", Priority: -1}

	require.NoError(t, PreProcessor(&task))

	// RULE: Assert the EXACT value.  A name that matches no case leaves Priority at the -1 sentinel.
	require.Equal(t, 512, task.Priority)
}
