package mastodon

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// newSuggestionTestUser returns a public User on the test server
func newSuggestionTestUser(username string, followers int) model.User {
	user := model.NewUser()
	user.UserID = primitive.NewObjectID()
	user.Username = username
	user.ProfileURL = "https://social.example/@" + username
	user.FollowerCount = followers
	return user
}

// newSuggestionTestFilter returns a filter for the viewer with nothing ruled out
func newSuggestionTestFilter(viewerID primitive.ObjectID) suggestionFilter {
	return suggestionFilter{
		viewerID: viewerID,
		followed: map[string]bool{},
		hidden:   map[string]bool{},
		blockers: map[primitive.ObjectID]bool{},
	}
}

// TestSuggestionsLimit keeps the limit within Mastodon's bounds
func TestSuggestionsLimit(t *testing.T) {
	require.Equal(t, suggestionsDefault, suggestionsLimit(0))
	require.Equal(t, suggestionsDefault, suggestionsLimit(-3))
	require.Equal(t, 5, suggestionsLimit(5))
	require.Equal(t, suggestionsMax, suggestionsLimit(500))
}

// TestHasReadScope accepts the broad read scope or the accounts one, and nothing else
func TestHasReadScope(t *testing.T) {
	require.True(t, hasReadScope([]string{"read", "write"}))
	require.True(t, hasReadScope([]string{"read:accounts"}))
	require.False(t, hasReadScope([]string{"write", "read:statuses"}))
	require.False(t, hasReadScope(nil))
}

// TestSuggestionFilter_Allows rules out the viewer, moved accounts, follows, blockers, and blocked or muted accounts and servers
func TestSuggestionFilter_Allows(t *testing.T) {

	viewer := newSuggestionTestUser("viewer", 0)
	alice := newSuggestionTestUser("alice", 0)

	// A stranger is welcome
	filter := newSuggestionTestFilter(viewer.UserID)
	require.True(t, filter.allows(&alice))

	// The viewer is never suggested to themselves
	require.False(t, filter.allows(&viewer))

	// An account that has moved away is skipped
	moved := alice
	moved.MovedTo = "https://elsewhere.example/@alice"
	require.False(t, filter.allows(&moved))

	// Already followed
	filter = newSuggestionTestFilter(viewer.UserID)
	filter.followed[alice.ProfileURL] = true
	require.False(t, filter.allows(&alice))

	// Alice blocks the viewer
	filter = newSuggestionTestFilter(viewer.UserID)
	filter.blockers[alice.UserID] = true
	require.False(t, filter.allows(&alice))

	// The viewer blocks or mutes Alice
	filter = newSuggestionTestFilter(viewer.UserID)
	filter.hidden[model.RuleMatchKey(model.RuleTypeActor, alice.ProfileURL)] = true
	require.False(t, filter.allows(&alice))

	// The viewer blocks Alice's whole server
	filter = newSuggestionTestFilter(viewer.UserID)
	filter.hidden[model.RuleMatchKey(model.RuleTypeDomain, "social.example")] = true
	require.False(t, filter.allows(&alice))
}

// TestHidingMatchKeys keeps only unexpired BLOCK and MUTE Rules
func TestHidingMatchKeys(t *testing.T) {

	rules := []model.RuleSummary{
		{MatchKey: "block", Action: model.RuleActionBlock},
		{MatchKey: "mute", Action: model.RuleActionMute},
		{MatchKey: "label", Action: model.RuleActionLabel},
		{MatchKey: "expired", Action: model.RuleActionBlock, ExpireDate: 500},
		{MatchKey: "expiring", Action: model.RuleActionMute, ExpireDate: 2000},
	}

	require.Equal(t, map[string]bool{"block": true, "mute": true, "expiring": true}, hidingMatchKeys(rules, 1000))
}

// TestSortSuggestions puts the most-followed first, then the most recent poster
func TestSortSuggestions(t *testing.T) {

	picks := []suggestionPick{
		{user: newSuggestionTestUser("quiet", 5), newest: 100},
		{user: newSuggestionTestUser("popular", 50), newest: 10},
		{user: newSuggestionTestUser("chatty", 5), newest: 300},
	}

	sortSuggestions(picks)

	names := []string{picks[0].user.Username, picks[1].user.Username, picks[2].user.Username}
	require.Equal(t, []string{"popular", "chatty", "quiet"}, names)
}
