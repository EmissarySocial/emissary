package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
)

// TestStream_RefusesReply pins the check the inbox makes before a reply is notified or recorded: a local,
// public Stream with closed replies refuses every reply, and a remote or private parent refuses nothing.
func TestStream_RefusesReply(t *testing.T) {

	const host = "https://local.example"
	const author = host + "/@author"

	// The in-memory mock cannot find a Stream by token or ID, so this runs against the local replica set
	_, session := newReplicaSetSession(t)

	locator := NewLocator()
	locator.host = host
	service := NewStream()
	service.host = host
	service.locatorService = &locator

	// saveParent stores a Stream with the provided token, policy, and audience, and returns its URL
	saveParent := func(token string, policy string, audience model.Permissions) string {
		parent := model.NewStream()
		parent.Token = token
		parent.ReplyPolicy = policy
		parent.DefaultAllow = audience
		parent.AttributedTo = model.PersonLink{ProfileURL: author}
		require.NoError(t, session.Collection("Stream").Save(&parent, "test"))
		return host + "/" + token
	}

	public := model.Permissions{model.MagicGroupIDAnonymous}
	private := model.Permissions{model.MagicGroupIDAuthenticated}

	closed := saveParent("closed", model.ReplyPolicyNobody, public)
	open := saveParent("open", model.ReplyPolicyEveryone, public)
	closedPrivate := saveParent("closed-private", model.ReplyPolicyNobody, private)

	testCases := []struct {
		name      string
		inReplyTo string
		expected  bool
	}{
		{"closed", closed, true},
		{"open", open, false},
		{"private", closedPrivate, false},
		{"remote parent", "https://remote.example/notes/1", false},
		{"missing local parent", host + "/missing", false},
		{"not a reply", "", false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			refused, err := service.RefusesReply(session, testCase.inReplyTo)
			require.NoError(t, err)
			require.Equal(t, testCase.expected, refused)
		})
	}
}
