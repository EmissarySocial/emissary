package service

import (
	"context"
	"net/http"
	"testing"

	querysync "github.com/EmissarySocial/emissary/queries/sync"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestTagFollow_FollowAndUnfollow follows a tag once however often it's asked, finds its followers, and unfollows it
func TestTagFollow_FollowAndUnfollow(t *testing.T) {

	database := newDomainTestDatabase(t)
	require.NoError(t, querysync.TagFollow(context.Background(), database))
	session := newStreamTokenSession(t, database)

	service := NewTagFollow()
	alice := primitive.NewObjectID()
	bob := primitive.NewObjectID()

	// A tag needs a name
	err := service.Follow(session, alice, " # ")
	require.Equal(t, http.StatusUnprocessableEntity, derp.ErrorCode(err))

	// Following twice, in two spellings, makes one follow
	require.NoError(t, service.Follow(session, alice, "#Cats"))
	require.NoError(t, service.Follow(session, alice, "cats"))
	require.NoError(t, service.Follow(session, bob, "cats"))

	follows, err := service.QueryByUser(session, alice, 10)
	require.NoError(t, err)
	require.Len(t, follows, 1)
	require.Equal(t, "Cats", follows[0].Name)
	require.Equal(t, "cats", follows[0].Value)

	// Both followers are found by the tag's value
	followers, err := service.QueryByValues(session, []string{"cats", "dogs"})
	require.NoError(t, err)
	require.Len(t, followers, 2)

	// Unfollowing removes only that User's follow
	require.NoError(t, service.Unfollow(session, alice, "#CATS"))

	following, err := service.IsFollowing(session, alice, "cats")
	require.NoError(t, err)
	require.False(t, following)

	following, err = service.IsFollowing(session, bob, "cats")
	require.NoError(t, err)
	require.True(t, following)

	// The tag still has a follower until Bob leaves too
	followed, err := service.IsFollowedByAnyone(session, "cats")
	require.NoError(t, err)
	require.True(t, followed)

	require.NoError(t, service.Unfollow(session, bob, "cats"))

	followed, err = service.IsFollowedByAnyone(session, "cats")
	require.NoError(t, err)
	require.False(t, followed)
}
