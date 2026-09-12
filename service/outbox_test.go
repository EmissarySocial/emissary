package service

import (
	"context"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	mockdb "github.com/benpate/data-mock"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestOutbox_getActorRejectsUnroutableTypes pins the set of Actors that may own an Outbox message.
// Application, Search, and SearchDomain have no route that serves an Outbox item, so a message they
// own could never be dereferenced by any peer (BUG-146).
func TestOutbox_getActorRejectsUnroutableTypes(t *testing.T) {

	service := Outbox{}

	session, err := mockdb.New().Session(context.Background())
	require.NoError(t, err)

	unroutable := []string{
		model.FollowerTypeApplication,
		model.FollowerTypeSearch,
		model.FollowerTypeSearchDomain,
		"",
		"NOT-A-REAL-ACTOR-TYPE",
	}

	for _, actorType := range unroutable {
		_, err := service.getActor(session, actorType, primitive.NewObjectID())
		require.Error(t, err, "actorType %q must not be allowed to own an Outbox message", actorType)
	}
}

// TestOutbox_SaveRequiresAnActor verifies that a message whose Actor cannot be resolved is never
// written. Before BUG-146 such a message stored an empty ActorURL, then published an empty `actor`
// and the relative id "/pub/outbox/<id>" that no peer can dereference.
func TestOutbox_SaveRequiresAnActor(t *testing.T) {

	service := Outbox{}

	session, err := mockdb.New().Session(context.Background())
	require.NoError(t, err)

	message := model.NewOutboxMessage()
	message.ActorType = model.FollowerTypeApplication
	message.ActorID = primitive.NewObjectID()

	require.Error(t, service.Save(session, &message, "Testing"))

	// The message is left unidentifiable, rather than carrying a relative ID into the database
	require.Empty(t, message.ActorURL)
	require.Empty(t, message.ActivityURL)
	require.Empty(t, message.ActivityPubURL())
}

// TestOutbox_calcActivityURL verifies that an activity URL is built from the Actor's own URL, and
// that a message with no Actor mints nothing at all
func TestOutbox_calcActivityURL(t *testing.T) {

	service := Outbox{}

	messageID, err := primitive.ObjectIDFromHex("687aeeef6c699789c9ce623a")
	require.NoError(t, err)

	testCases := []struct {
		name     string
		actorURL string
		expected string
	}{
		{"user actor", "https://x.social/@bob", "https://x.social/@bob/pub/outbox/687aeeef6c699789c9ce623a"},
		{"stream actor", "https://x.social/123", "https://x.social/123/pub/outbox/687aeeef6c699789c9ce623a"},
		{"no actor", "", ""},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {

			message := model.NewOutboxMessage()
			message.OutboxMessageID = messageID
			message.ActorURL = testCase.actorURL

			require.Equal(t, testCase.expected, service.calcActivityURL(&message))
		})
	}
}
