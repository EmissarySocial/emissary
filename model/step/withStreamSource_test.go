package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestWithStreamSource verifies that a "with-stream-source" step parses its sub-pipeline
func TestWithStreamSource(t *testing.T) {

	step, err := NewWithStreamSource(mapof.Any{
		"steps": []mapof.Any{
			{"do": "edit"},
			{"do": "save"},
		},
	})

	require.Nil(t, err)
	require.Len(t, step.SubSteps, 2)
	require.Equal(t, "with-stream-source", step.Name())
}

// TestWithStreamSource_RequiresAStream pins the model restriction.  A StreamSource is found by the
// Stream it populates, so there is nothing to switch to from any other builder.
func TestWithStreamSource_RequiresAStream(t *testing.T) {

	step, err := NewWithStreamSource(mapof.Any{})
	require.Nil(t, err)
	require.Equal(t, "Stream", step.RequiredModel())
}

// TestWithStreamSource_RollsUpRoles confirms that a sub-step's role requirement becomes this step's
// own, so a Template naming a role it never defined fails at load time rather than in front of a user
func TestWithStreamSource_RollsUpRoles(t *testing.T) {

	step, err := NewWithStreamSource(mapof.Any{
		"steps": []mapof.Any{
			{"do": "set-simple-sharing", "role": "editor"},
		},
	})

	require.Nil(t, err)
	require.Contains(t, step.RequiredRoles(), "editor")
}

// TestWithStreamSource_DropsStateRequirements confirms that sub-step STATES are not rolled up.  A
// StreamSource has one state of its own, so a rolled-up requirement would name a state of the
// Stream this step has already switched away from.
func TestWithStreamSource_DropsStateRequirements(t *testing.T) {

	step, err := NewWithStreamSource(mapof.Any{
		"steps": []mapof.Any{
			{"do": "save-and-publish", "state": "published"},
		},
	})

	require.Nil(t, err)
	require.Empty(t, step.RequiredStates())
}

// TestWithStreamSource_InvalidSubStep confirms that a broken sub-pipeline is refused at Template
// load time, rather than at request time in front of a visitor
func TestWithStreamSource_InvalidSubStep(t *testing.T) {

	_, err := NewWithStreamSource(mapof.Any{
		"steps": []mapof.Any{
			{"do": "not-a-real-step"},
		},
	})

	require.NotNil(t, err)
}

// TestWithStreamSource_IsRegistered confirms that the step name resolves through the parser, which
// is the only thing that makes it usable from a Template
func TestWithStreamSource_IsRegistered(t *testing.T) {

	step, err := New(mapof.Any{"do": "with-stream-source", "steps": []mapof.Any{{"do": "save"}}})

	require.Nil(t, err)
	require.Equal(t, "with-stream-source", step.Name())
	require.IsType(t, WithStreamSource{}, step)
}

// TestWithStreamSource_ReplacedSyncStreamSource confirms that the step it replaced is gone.  A
// Template still naming it would fail to load, which is the loud failure we want.
func TestWithStreamSource_ReplacedSyncStreamSource(t *testing.T) {

	_, err := New(mapof.Any{"do": "sync-stream-source"})
	require.NotNil(t, err)
}
