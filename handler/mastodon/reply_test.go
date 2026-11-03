package mastodon

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestReplyIDsFor confirms a reply to a local post carries that post's ID and author, and a reply
// to anything else carries the encoded post ID alone.
func TestReplyIDsFor(t *testing.T) {

	author := primitive.NewObjectID()

	parent := model.NewStream()
	parent.AttributedTo = model.PersonLink{UserID: author, ProfileURL: "https://example.com/@me"}

	postID, accountID := replyIDsFor("https://example.com/"+parent.StreamID.Hex(), &parent)
	require.Equal(t, parent.StreamID.Hex(), postID)
	require.Equal(t, author.Hex(), accountID)

	remoteURL := "https://elsewhere.social/users/them/statuses/123"
	postID, accountID = replyIDsFor(remoteURL, nil)
	require.Equal(t, model.EncodeRemoteStatusID(remoteURL), postID)
	require.Empty(t, accountID, "the author of a remote post is not known without fetching it")

	decoded, ok := model.DecodeRemoteStatusID(postID)
	require.True(t, ok)
	require.Equal(t, remoteURL, decoded)
}
