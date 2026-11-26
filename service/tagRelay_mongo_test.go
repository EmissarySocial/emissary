package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	querysync "github.com/EmissarySocial/emissary/queries/sync"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// newTagRelayTest returns a TagRelay service, the TagFollow service it checks, and a session on a fresh database
func newTagRelayTest(t *testing.T) (*TagRelay, *TagFollow, data.Session) {

	database := newDomainTestDatabase(t)
	require.NoError(t, querysync.TagFollow(context.Background(), database))
	require.NoError(t, querysync.TagRelay(context.Background(), database))

	tagFollowService := NewTagFollow()
	tagRelayService := TagRelay{tagFollowService: &tagFollowService, domainService: &Domain{host: "https://local.example"}}

	return &tagRelayService, &tagFollowService, newStreamTokenSession(t, database)
}

// saveTestTagRelay stores a pending subscription to the cats relay
func saveTestTagRelay(t *testing.T, service *TagRelay, session data.Session) model.TagRelay {

	tagRelay := model.NewTagRelay()
	tagRelay.Name = "cats"
	tagRelay.Value = "cats"
	tagRelay.ActorURL = "https://tags.pub/user/cats"
	tagRelay.FollowID = "https://local.example/@application#follows/" + tagRelay.TagRelayID.Hex()
	require.NoError(t, service.Save(session, &tagRelay, "test"))

	return tagRelay
}

// TestTagRelay_Accept activates a subscription only when the followed relay accepts
func TestTagRelay_Accept(t *testing.T) {

	service, _, session := newTagRelayTest(t)
	tagRelay := saveTestTagRelay(t, service, session)

	// Another actor can't accept on the relay's behalf
	err := service.Accept(session, tagRelay.FollowID, "https://evil.example/user/cats")
	require.Equal(t, http.StatusForbidden, derp.ErrorCode(err))

	// An unknown Follow is not found
	err = service.Accept(session, "https://local.example/@application#follows/nope", tagRelay.ActorURL)
	require.True(t, derp.IsNotFound(err))

	// The relay itself can
	require.NoError(t, service.Accept(session, tagRelay.FollowID, tagRelay.ActorURL))

	loaded := model.NewTagRelay()
	require.NoError(t, service.LoadByActorURL(session, tagRelay.ActorURL, &loaded))
	require.Equal(t, model.TagRelayStateActive, loaded.State)
}

// TestTagRelay_Reject removes a subscription only when the followed relay rejects
func TestTagRelay_Reject(t *testing.T) {

	service, _, session := newTagRelayTest(t)
	tagRelay := saveTestTagRelay(t, service, session)

	err := service.Reject(session, tagRelay.FollowID, "https://evil.example/user/cats")
	require.Equal(t, http.StatusForbidden, derp.ErrorCode(err))

	require.NoError(t, service.Reject(session, tagRelay.FollowID, tagRelay.ActorURL))

	loaded := model.NewTagRelay()
	require.True(t, derp.IsNotFound(service.LoadByValue(session, "cats", &loaded)))
}

// TestTagRelay_Guards skips subscribing when nobody follows the tag or a relay exists, and keeps a relay while it's followed
func TestTagRelay_Guards(t *testing.T) {

	service, tagFollowService, session := newTagRelayTest(t)
	alice := primitive.NewObjectID()

	// Nobody follows #cats, so there is nothing to subscribe (and no network call)
	require.NoError(t, service.Subscribe(session, "cats", "cats"))

	loaded := model.NewTagRelay()
	require.True(t, derp.IsNotFound(service.LoadByValue(session, "cats", &loaded)))

	// With a follower and an existing relay, subscribing again changes nothing
	require.NoError(t, tagFollowService.Follow(session, alice, "Cats"))
	tagRelay := saveTestTagRelay(t, service, session)
	require.NoError(t, service.Subscribe(session, "cats", "cats"))

	// While someone follows the tag, unsubscribing keeps the relay
	require.NoError(t, service.Unsubscribe(session, "cats"))
	require.NoError(t, service.LoadByValue(session, "cats", &loaded))
	require.Equal(t, tagRelay.TagRelayID, loaded.TagRelayID)

	// With no relay at all, unsubscribing is a no-op
	require.NoError(t, service.Unsubscribe(session, "dogs"))
}
