package service

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	querysync "github.com/EmissarySocial/emissary/queries/sync"
	"github.com/EmissarySocial/emissary/realtime"
	"github.com/benpate/data"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// These tests run Stream.Save against a real MongoDB, with the indexes a server builds at startup,
// because the rule they guard spans the service check and the database's unique index.

// newStreamTokenService returns a Stream service that can save Streams on the test Templates, and the
// counter of the webhooks its saves start
func newStreamTokenService() (*Stream, *atomic.Int32) {

	webhooks := &atomic.Int32{}

	service := newDefaultChildService()
	service.userService = &User{}
	service.sseUpdateChannel = make(chan realtime.Message, 100)
	service.webhookService = &Webhook{
		newSession: func(time.Duration) (data.Session, context.CancelFunc, error) {
			webhooks.Add(1)
			return nil, nil, derp.Internal("test", "Webhooks are not delivered in this test")
		},
	}

	return service, webhooks
}

// newStreamTokenDatabase returns a throwaway database in the local replica set, with the Stream indexes built
func newStreamTokenDatabase(t *testing.T) *mongo.Database {

	t.Helper()

	database := newDomainTestDatabase(t)
	require.NoError(t, querysync.Stream(context.Background(), database))

	return database
}

// newStreamTokenSession returns a session on the database, through a server of its own
func newStreamTokenSession(t *testing.T, database *mongo.Database) data.Session {

	t.Helper()

	session, err := mongodb.NewServer(database).Session(t.Context())
	require.NoError(t, err)

	return session
}

// newTokenStream returns a new Stream on the test collection Template, with the provided token
func newTokenStream(token string) model.Stream {
	stream := model.NewStream()
	stream.TemplateID = "test-collection"
	stream.ParentID = primitive.NewObjectID()
	stream.StateID = "default"
	stream.Token = token
	return stream
}

// TestStream_Save_RefusesATakenToken confirms that Save refuses a token held by another Stream or by
// the server's own routes, and accepts the default
func TestStream_Save_RefusesATakenToken(t *testing.T) {

	database := newStreamTokenDatabase(t)
	session := newStreamTokenSession(t, database)
	service, webhooks := newStreamTokenService()

	first := newTokenStream("about")
	require.NoError(t, service.Save(session, &first, "first"))

	// A new Stream cannot take a token that is in use, or one the router owns
	second := newTokenStream("about")
	err := service.Save(session, &second, "taken")
	require.True(t, derp.IsBadRequest(err), "another Stream's token")

	second.Token = "signin"
	err = service.Save(session, &second, "reserved")
	require.True(t, derp.IsBadRequest(err), "a server route")

	// Nothing was written by the refused saves
	count, err := session.Collection("Stream").Count(nil)
	require.NoError(t, err)
	require.Equal(t, int64(1), count)

	// The default token is the Stream's own ID
	second.Token = ""
	require.NoError(t, service.Save(session, &second, "default"))
	require.Equal(t, second.StreamID.Hex(), second.Token)

	// ...and an existing Stream cannot change to a token that is in use
	second.Token = "about"
	err = service.Save(session, &second, "changed")
	require.True(t, derp.IsBadRequest(err))

	requireCount(t, webhooks, 2, "stream:create webhooks")
}

// TestStream_Save_KeepsAnUnchangedToken confirms that a token saved before these rules still saves,
// and that changing away from it and back is checked like any other change
func TestStream_Save_KeepsAnUnchangedToken(t *testing.T) {

	database := newStreamTokenDatabase(t)
	session := newStreamTokenSession(t, database)
	service, webhooks := newStreamTokenService()

	// A User and an older Stream already share the handle "alice"
	_, err := database.Collection("User").InsertOne(t.Context(), bson.M{"_id": primitive.NewObjectID(), "username": "alice", "deleteDate": 0})
	require.NoError(t, err)

	stream := newTokenStream("alice")
	stream.CreateDate = time.Now().UnixMilli()
	require.NoError(t, session.Collection("Stream").Save(&stream, "written before the rules"))

	// Saving it again, unchanged, still works
	stream.Label = "Renamed"
	require.NoError(t, service.Save(session, &stream, "unchanged token"))

	// Changing away and back is a change, and the username refuses it
	stream.Token = "alice-page"
	require.NoError(t, service.Save(session, &stream, "new token"))

	stream.Token = "alice"
	err = service.Save(session, &stream, "back again")
	require.True(t, derp.IsBadRequest(err))

	requireCount(t, webhooks, 2, "stream:update webhooks")
}

// TestStream_Save_TwoServersShareOneToken races two servers that share nothing but the database, and
// confirms that exactly one Stream takes each token
func TestStream_Save_TwoServersShareOneToken(t *testing.T) {

	database := newStreamTokenDatabase(t)

	// The second server gets its own client, and its own Stream service
	connectCtx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	otherClient, err := mongo.Connect(connectCtx, options.Client().ApplyURI(domainTestConnection))
	require.NoError(t, err)
	t.Cleanup(func() { _ = otherClient.Disconnect(context.Background()) })

	type server struct {
		service *Stream
		session data.Session
	}

	firstService, firstWebhooks := newStreamTokenService()
	secondService, secondWebhooks := newStreamTokenService()

	servers := []server{
		{service: firstService, session: newStreamTokenSession(t, database)},
		{service: secondService, session: newStreamTokenSession(t, otherClient.Database(database.Name()))},
	}

	const rounds = 10

	for round := range rounds {

		token := fmt.Sprintf("race-%d", round)
		results := make(chan error, len(servers))

		for _, each := range servers {
			go func() {
				stream := newTokenStream(token)
				results <- each.service.Save(each.session, &stream, "race")
			}()
		}

		successes := 0

		for range servers {

			err := <-results

			if err == nil {
				successes++
				continue
			}

			// RULE: The loser is refused by the check (400) or, when both pass it, by the index (409)
			code := derp.ErrorCode(err)
			require.Contains(t, []int{400, 409}, code, "the losing save must be refused, not %v", err)
			t.Logf("%s: the losing save was refused with %d", token, code)
		}

		require.Equal(t, 1, successes, "exactly one server takes %q", token)
	}

	// Wait for every stream:create webhook, so their goroutines do not outlive the test
	require.Eventually(t, func() bool {
		return firstWebhooks.Load()+secondWebhooks.Load() == rounds
	}, 5*time.Second, 10*time.Millisecond)
}
