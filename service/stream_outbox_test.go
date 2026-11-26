package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestOutboxAction confirms the activity that each pairing of the author's wish and the followers' copy calls for
func TestOutboxAction(t *testing.T) {

	tests := []struct {
		name     string
		wanted   bool
		held     string
		current  string
		expected string
	}{
		{"wanted, and followers hold nothing", true, "", "abc", vocab.ActivityTypeCreate},
		{"wanted, and followers hold an older copy", true, "abc", "def", vocab.ActivityTypeUpdate},
		{"wanted, and followers are up to date", true, "abc", "abc", ""},
		{"not wanted, and followers hold a copy", false, "abc", "", vocab.ActivityTypeDelete},
		{"not wanted, and followers hold the current copy", false, "abc", "abc", vocab.ActivityTypeDelete},
		{"not wanted, and followers hold nothing", false, "", "", ""},
		{"wanted, and followers hold a copy from before fingerprints", true, model.StreamFingerprintPending, "abc", vocab.ActivityTypeUpdate},
		{"not wanted, and followers hold a copy from before fingerprints", false, model.StreamFingerprintPending, "", vocab.ActivityTypeDelete},
		{"not wanted, with a current copy that nobody holds", false, "", "abc", ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, outboxAction(test.wanted, test.held, test.current))
		})
	}
}

// TestOutboxFingerprint confirms that equal objects share a fingerprint, and that any change moves it
func TestOutboxFingerprint(t *testing.T) {

	first, err := outboxFingerprint(mapof.Any{"name": "Judgment Day", "to": []string{"a", "b"}})
	require.NoError(t, err)
	require.Len(t, first, 64)

	// The order in which a map was built does not matter
	second, err := outboxFingerprint(mapof.Any{"to": []string{"a", "b"}, "name": "Judgment Day"})
	require.NoError(t, err)
	require.Equal(t, first, second)

	// A changed value, and a changed order within a list, are both changes
	edited, err := outboxFingerprint(mapof.Any{"name": "No Fate", "to": []string{"a", "b"}})
	require.NoError(t, err)
	require.NotEqual(t, first, edited)

	reordered, err := outboxFingerprint(mapof.Any{"name": "Judgment Day", "to": []string{"b", "a"}})
	require.NoError(t, err)
	require.NotEqual(t, first, reordered)

	// An empty object still has a fingerprint, so it never reads as "followers hold nothing"
	empty, err := outboxFingerprint(mapof.Any{})
	require.NoError(t, err)
	require.Len(t, empty, 64)

	// A value that cannot be encoded is an error, never an empty fingerprint
	_, err = outboxFingerprint(mapof.Any{"bad": make(chan int)})
	require.Error(t, err)
}

// TestSyndicationPlan confirms which syndication targets are sent what, for each thing that
// followers can have been sent
func TestSyndicationPlan(t *testing.T) {

	current := []string{"bandwagon", "spotify", "tidal"}
	changes := OutboxChanges{SyndicationAdded: []string{"tidal"}, SyndicationDeleted: []string{"deezer"}}

	tests := []struct {
		name    string
		action  string
		wanted  bool
		changes OutboxChanges
		created []string
		updated []string
		deleted []string
	}{
		{"a Create goes to every target", vocab.ActivityTypeCreate, true, OutboxChanges{}, current, nil, nil},
		{"a Create ignores a target that was removed before anything was sent", vocab.ActivityTypeCreate, true, changes, current, nil, nil},
		{"an Update goes to every target", vocab.ActivityTypeUpdate, true, OutboxChanges{}, nil, current, nil},
		{"an Update creates a new target and deletes a removed one", vocab.ActivityTypeUpdate, true, changes, []string{"tidal"}, []string{"bandwagon", "spotify"}, []string{"deezer"}},
		{"a Delete goes to every target", vocab.ActivityTypeDelete, false, OutboxChanges{}, nil, nil, current},
		{"a Delete skips a new target and includes a removed one", vocab.ActivityTypeDelete, false, changes, nil, nil, []string{"bandwagon", "spotify", "deezer"}},
		{"nothing sent, nothing changed", "", true, OutboxChanges{}, nil, nil, nil},
		{"nothing sent, and the targets changed", "", true, changes, []string{"tidal"}, nil, []string{"deezer"}},
		{"nothing sent, and every target is sent again", "", true, OutboxChanges{Republish: true}, nil, current, nil},
		{"sent again, with changed targets", "", true, OutboxChanges{SyndicationAdded: []string{"tidal"}, SyndicationDeleted: []string{"deezer"}, Republish: true}, []string{"tidal"}, []string{"bandwagon", "spotify"}, []string{"deezer"}},
		{"outside the outbox, nothing is syndicated", "", false, changes, nil, nil, nil},
		{"outside the outbox, nothing is sent again", "", false, OutboxChanges{Republish: true}, nil, nil, nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			created, updated, deleted := syndicationPlan(test.action, test.wanted, current, test.changes)
			require.ElementsMatch(t, test.created, created, "created")
			require.ElementsMatch(t, test.updated, updated, "updated")
			require.ElementsMatch(t, test.deleted, deleted, "deleted")
		})
	}

	// The plan never changes the list it was given
	require.Equal(t, []string{"bandwagon", "spotify", "tidal"}, current)
}

// TestOutboxChanges_IsZero confirms when a save had nothing to pass on
func TestOutboxChanges_IsZero(t *testing.T) {
	require.True(t, OutboxChanges{}.IsZero())
	require.False(t, OutboxChanges{SyndicationAdded: []string{"a"}}.IsZero())
	require.False(t, OutboxChanges{SyndicationDeleted: []string{"a"}}.IsZero())
	require.False(t, OutboxChanges{Republish: true}.IsZero())
}
