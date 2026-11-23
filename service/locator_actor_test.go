package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestLocator_ActorURL verifies the actor and followers URLs of each actor type that has an outbox
func TestLocator_ActorURL(t *testing.T) {

	locator := Locator{host: "https://example.com"}
	actorID := primitive.NewObjectID()

	require.Equal(t, "https://example.com/@"+actorID.Hex(), locator.ActorURL(model.ActorTypeUser, actorID))
	require.Equal(t, "https://example.com/"+actorID.Hex(), locator.ActorURL(model.ActorTypeStream, actorID))
	require.Empty(t, locator.ActorURL(model.ActorTypeSearchQuery, actorID))
	require.Empty(t, locator.ActorURL("", actorID))
	require.Empty(t, locator.FollowersURL(model.ActorTypeApplication, actorID))
}

// TestLocator_FollowersURL_RoundTrip verifies that parseFollowersURI reads back what FollowersURL writes
func TestLocator_FollowersURL_RoundTrip(t *testing.T) {

	locator := Locator{host: "https://example.com"}
	actorID := primitive.NewObjectID()

	for _, actorType := range []string{model.ActorTypeUser, model.ActorTypeStream} {
		parsedType, parsedID := parseFollowersURI(locator.host, locator.FollowersURL(actorType, actorID))
		require.Equal(t, actorType, parsedType)
		require.Equal(t, actorID, parsedID)
	}
}
