package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	querysync "github.com/EmissarySocial/emissary/queries/sync"
	"github.com/benpate/data"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// These tests run LoadOrCreateByClientToken against a real MongoDB, with the indexes a server builds at
// startup, because the rule they guard spans the fetch, the service, and the database's unique index.

// TestOAuthClient_LoadOrCreate_FallbackClient verifies that a client_id serving HTML becomes a client named by its
// hostname, that later lookups load that record without fetching again, and that any spelling of the
// client_id finds it
func TestOAuthClient_LoadOrCreate_FallbackClient(t *testing.T) {

	database := newOAuthClientDatabase(t)
	session := newOAuthClientSession(t, database)
	service := newURLClientService()

	fetches := atomic.Int32{}
	clientServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fetches.Add(1)
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><title>An IndieAuth client</title></html>`))
	}))
	t.Cleanup(clientServer.Close)

	clientID := localhostURL(clientServer) + "/"

	// The first lookup creates the client from its URL
	created := model.NewOAuthClient()
	require.NoError(t, service.LoadOrCreateByClientToken(session, clientID, &created))
	require.Equal(t, clientID, created.ClientURL)
	require.Equal(t, "localhost", created.Name)
	require.Empty(t, created.RedirectURIs)
	require.False(t, created.IsConfidential())
	require.Equal(t, int32(1), fetches.Load())

	// A later lookup loads the stored client, and fetches nothing
	loaded := model.NewOAuthClient()
	require.NoError(t, service.LoadOrCreateByClientToken(session, clientID, &loaded))
	require.Equal(t, created.ClientID, loaded.ClientID)
	require.Equal(t, int32(1), fetches.Load())

	// The token endpoint finds it from another spelling of the same client_id
	spelled := model.NewOAuthClient()
	require.NoError(t, service.LoadByToken(session, strings.Replace(strings.TrimSuffix(clientID, "/"), "http://localhost", "HTTP://LOCALHOST", 1), &spelled))
	require.Equal(t, created.ClientID, spelled.ClientID)

	// The fallback client may redirect to its own origin, and nowhere else
	require.True(t, loaded.AllowsRedirectURI(clientID+"oauth/indieauth/finish"))
	require.False(t, loaded.AllowsRedirectURI("https://evil.example/finish"))
}

// TestOAuthClient_LoadOrCreate_ServerErrorIsNotStored verifies that a client answering with a server error
// is refused and nothing is stored, so the next attempt fetches again
func TestOAuthClient_LoadOrCreate_ServerErrorIsNotStored(t *testing.T) {

	database := newOAuthClientDatabase(t)
	session := newOAuthClientSession(t, database)
	service := newURLClientService()

	failing := atomic.Bool{}
	failing.Store(true)

	clientServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if failing.Load() {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html></html>`))
	}))
	t.Cleanup(clientServer.Close)

	clientID := localhostURL(clientServer) + "/"

	// The outage is refused, and nothing is written
	client := model.NewOAuthClient()
	err := service.LoadOrCreateByClientToken(session, clientID, &client)
	require.True(t, derp.IsBadGateway(err), "answers %d", derp.ErrorCode(err))
	require.Equal(t, int64(0), countClients(t, database, clientID))

	// Once the client recovers, the next attempt creates it
	failing.Store(false)
	client = model.NewOAuthClient()
	require.NoError(t, service.LoadOrCreateByClientToken(session, clientID, &client))
	require.Equal(t, int64(1), countClients(t, database, clientID))
}

// TestOAuthClient_LoadOrCreate_UnknownRegisteredClient verifies that an ObjectID naming no client stays not found,
// and is never fetched or created
func TestOAuthClient_LoadOrCreate_UnknownRegisteredClient(t *testing.T) {

	database := newOAuthClientDatabase(t)
	session := newOAuthClientSession(t, database)
	service := newURLClientService()

	client := model.NewOAuthClient()
	err := service.LoadOrCreateByClientToken(session, "000000000000000000000001", &client)
	require.True(t, derp.IsNotFound(err), "answers %d", derp.ErrorCode(err))
}

// TestOAuthClient_LoadOrCreate_TwoServersCreateOneClient races two servers that share nothing but the database,
// each seeing the same new client_id at once, and confirms that both end up with the one client stored
func TestOAuthClient_LoadOrCreate_TwoServersCreateOneClient(t *testing.T) {

	database := newOAuthClientDatabase(t)
	servers := []*OAuthClient{newURLClientService(), newURLClientService()}
	sessions := []data.Session{newOAuthClientSession(t, database), newOAuthClientSession(t, database)}

	// The client's server holds each request until both servers have fetched, so both have missed the
	// lookup and both will try to save
	arrived := make(chan struct{}, len(servers))

	clientServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		arrived <- struct{}{}
		waitForBoth(arrived, len(servers))
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html></html>`))
	}))
	t.Cleanup(clientServer.Close)

	for round := range 10 {

		// Each round starts with an empty barrier
		for len(arrived) > 0 {
			<-arrived
		}

		clientID := fmt.Sprintf("%s/round-%d", localhostURL(clientServer), round)
		clients := make([]model.OAuthClient, len(servers))
		results := make(chan error, len(servers))

		for index := range servers {
			clients[index] = model.NewOAuthClient()
			go func() {
				results <- servers[index].LoadOrCreateByClientToken(sessions[index], clientID, &clients[index])
			}()
		}

		for range servers {
			require.NoError(t, <-results, "round %d", round)
		}

		require.Equal(t, clients[0].ClientID, clients[1].ClientID, "round %d: both servers must use the one stored client", round)
		require.Equal(t, int64(1), countClients(t, database, clientID), "round %d", round)
	}
}

// newOAuthClientDatabase returns a throwaway database in the local replica set, with the OAuthClient indexes built
func newOAuthClientDatabase(t *testing.T) *mongo.Database {

	t.Helper()

	database := newDomainTestDatabase(t)
	require.NoError(t, querysync.OAuthClient(context.Background(), database))

	return database
}

// newOAuthClientSession returns a session on the database, through a server of its own
func newOAuthClientSession(t *testing.T, database *mongo.Database) data.Session {

	t.Helper()

	session, err := mongodb.NewServer(database).Session(t.Context())
	require.NoError(t, err)

	return session
}

// newURLClientService returns an OAuthClient service on a local host, which may fetch a client on localhost
func newURLClientService() *OAuthClient {
	return &OAuthClient{
		host:            "localhost",
		activityService: &ActivityStream{hostname: "localhost"},
	}
}

// localhostURL returns a test server's address with a domain name, which a client_id requires
func localhostURL(server *httptest.Server) string {
	return strings.Replace(server.URL, "127.0.0.1", "localhost", 1)
}

// countClients returns the number of live clients stored for a client_id
func countClients(t *testing.T, database *mongo.Database, clientID string) int64 {

	t.Helper()

	count, err := database.Collection("OAuthClient").CountDocuments(context.Background(), bson.M{"clientUrl": clientID, "deleteDate": 0})
	require.NoError(t, err)

	return count
}

// waitForBoth blocks until the expected number of requests have arrived, or for at most five seconds
func waitForBoth(arrived chan struct{}, expected int) {

	for deadline := time.After(5 * time.Second); len(arrived) < expected; {
		select {
		case <-deadline:
			return
		case <-time.After(time.Millisecond):
		}
	}
}
