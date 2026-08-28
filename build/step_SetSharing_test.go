package build

import (
	"io"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepSetSharing.Post must REPLACE the Stream's sharing settings for the configured role with the
// single magic Group its "group" argument names, and must refuse to run against a non-Stream builder.

// setSharingNonStreamBuilder is a Builder that is not a Stream. Post only type-asserts the builder,
// so the nil embedded interface is never dereferenced.
type setSharingNonStreamBuilder struct {
	Builder
}

// TestStepSetSharing_Post verifies that each magic Group name writes its GroupID onto the Stream
func TestStepSetSharing_Post(t *testing.T) {

	testCases := []struct {
		group   string
		groupID primitive.ObjectID
	}{
		{"anonymous", model.MagicGroupIDAnonymous},
		{"authenticated", model.MagicGroupIDAuthenticated},
		{"owner", model.MagicGroupIDOwners},
	}

	for _, testCase := range testCases {

		// Pre-existing sharing settings prove the step overwrites, not appends
		stream := model.NewStream()
		stream.Groups["viewer"] = id.Slice{primitive.NewObjectID(), primitive.NewObjectID()}

		step := StepSetSharing{Role: "viewer", Group: testCase.group}
		behavior := step.Post(Stream{_stream: &stream}, io.Discard)

		require.Nil(t, behavior, "group %q must continue the pipeline", testCase.group)
		require.Equal(t, id.Slice{testCase.groupID}, stream.Groups["viewer"], "group %q", testCase.group)
	}
}

// TestStepSetSharing_Get verifies that GET is a no-op that continues the pipeline
func TestStepSetSharing_Get(t *testing.T) {

	stream := model.NewStream()
	step := StepSetSharing{Role: "viewer", Group: "anonymous"}

	require.Nil(t, step.Get(Stream{_stream: &stream}, io.Discard))
	require.Empty(t, stream.Groups, "GET must not touch the Stream's sharing settings")
}

// TestStepSetSharing_Post_RejectsNonStream verifies that a non-Stream builder halts the pipeline
func TestStepSetSharing_Post_RejectsNonStream(t *testing.T) {

	step := StepSetSharing{Role: "viewer", Group: "anonymous"}
	behavior := step.Post(setSharingNonStreamBuilder{}, io.Discard)
	require.NotNil(t, behavior)

	result := NewPipelineResult()
	behavior(&result)
	require.True(t, result.Halt, "a non-Stream builder must halt the pipeline")
}
