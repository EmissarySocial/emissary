package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/stretchr/testify/require"
)

// TestStreamPromoteDraft verifies that a "promote-draft" step parses its configuration
func TestStreamPromoteDraft(t *testing.T) {

	step, err := NewStreamPromoteDraft(mapof.Any{
		"state": "review",
		"omit":  []string{"content", "label"},
	})
	require.Nil(t, err)
	require.Equal(t, "review", step.StateID)
	require.Equal(t, sliceof.String{"content", "label"}, step.Omit)

	// "state" defaults to "published", and omitting nothing is the whole point of the default
	step, err = NewStreamPromoteDraft(mapof.Any{})
	require.Nil(t, err)
	require.Equal(t, "published", step.StateID)
	require.Empty(t, step.Omit)

	require.Equal(t, "promote-draft", step.Name())
	require.Equal(t, "Stream", step.RequiredModel())
	require.Equal(t, []string{"published"}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestStreamPromoteDraft_OmitIsValidated pins the reason a bad name fails the Template at load.
// An unrecognized name leaves its property COPIED -- the exact mistake `omit` exists to prevent --
// and a promote that quietly reverts a page reports nothing at all.
func TestStreamPromoteDraft_OmitIsValidated(t *testing.T) {

	// Every property that Promote copies must be nameable, or `omit` cannot govern it
	for _, name := range promoteDraftFields {
		_, err := NewStreamPromoteDraft(mapof.Any{"omit": []string{name}})
		require.Nil(t, err, "%q is copied by Promote but rejected by omit", name)
	}

	// A misspelling fails, rather than silently omitting nothing
	_, err := NewStreamPromoteDraft(mapof.Any{"omit": []string{"iconURL"}})
	require.NotNil(t, err, "a misspelled property must not be accepted")

	// ..as does a property that Promote does not copy.  `stateId` is the one that matters: it
	// comes from this step's own `state` attribute, so omitting it would read as "do not publish"
	// while save-and-publish sets the state one step later anyway.
	for _, name := range []string{"stateId", "deleteDate", "groups", "publishDate"} {
		_, err := NewStreamPromoteDraft(mapof.Any{"omit": []string{name}})
		require.NotNil(t, err, "%q is not copied by Promote, so omitting it does nothing", name)
	}

	// Nested paths are not supported.  Promote assigns whole properties, so `data.tags` would
	// copy the entire map and report success.
	_, err = NewStreamPromoteDraft(mapof.Any{"omit": []string{"data.tags"}})
	require.NotNil(t, err, "a nested path must not be accepted while Promote copies whole properties")
}
