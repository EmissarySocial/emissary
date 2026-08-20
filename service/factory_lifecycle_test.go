package service

import (
	"context"
	"net/http"
	"testing"

	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/derp"
	"github.com/benpate/digital-dome/dome"
	"github.com/benpate/turbine/queue"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// These tests pin the read-through contract between a domain Factory and its ServerFactory: the
// common database and the task queue are read LIVE on every call, never captured.  The incident
// they guard against: a config reload rebuilt both resources on the server factory, and domain
// factories kept captured handles -- so the ActivityStream cache failed every call with "client
// is disconnected" (breaking inbound signature verification with a bare 401) and outbound tasks
// were published into a stopped queue that silently dropped them.

// lifecycleServerFactory is a minimal ServerFactory whose database and queue can be swapped
// mid-test, standing in for a server factory that reconnected on a config reload.
type lifecycleServerFactory struct {
	database *mongo.Database
	queue    *queue.Queue
}

// ByHostname implements the ServerFactory interface. Unused by these tests.
func (factory *lifecycleServerFactory) ByHostname(hostname string) (*Factory, error) {
	return nil, derp.NotFound("lifecycleServerFactory.ByHostname", "Not implemented in this test double", hostname)
}

// Email implements the ServerFactory interface. Unused by these tests.
func (factory *lifecycleServerFactory) Email() *ServerEmail {
	return nil
}

// ClientIP implements the ServerFactory interface. Unused by these tests.
func (factory *lifecycleServerFactory) ClientIP(_ *http.Request) string {
	return ""
}

// DigitalDome implements the ServerFactory interface. Unused by these tests.
func (factory *lifecycleServerFactory) DigitalDome() *dome.Dome {
	return nil
}

// Queue implements the ServerFactory interface, returning this stub's queue
func (factory *lifecycleServerFactory) Queue() *queue.Queue {
	return factory.queue
}

// CommonDatabase implements the ServerFactory interface, returning this stub's database
func (factory *lifecycleServerFactory) CommonDatabase() *mongo.Database {
	return factory.database
}

// lazyLifecycleDatabase returns a *mongo.Database whose client never contacts a server
// (mongo.Connect is lazy), so these tests run without a reachable Mongo.
func lazyLifecycleDatabase(t *testing.T, database string) *mongo.Database {
	t.Helper()

	client, err := mongo.Connect(context.Background(),
		options.Client().ApplyURI("mongodb://127.0.0.1:59997/?directConnection=true&serverSelectionTimeoutMS=200"))
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = client.Disconnect(context.Background())
	})

	return client.Database(database)
}

// TestFactory_CommonDatabase_ReadsThroughServerFactory pins the no-capture contract: when the
// server factory reconnects its common database, every domain factory must see the NEW handle on
// its very next call -- with no Refresh in between.  A captured handle would keep returning the
// old, disconnected client.
func TestFactory_CommonDatabase_ReadsThroughServerFactory(t *testing.T) {

	before := lazyLifecycleDatabase(t, "lifecycle-before")
	after := lazyLifecycleDatabase(t, "lifecycle-after")

	serverFactory := &lifecycleServerFactory{database: before}
	factory := Factory{serverFactory: serverFactory}

	require.Same(t, before.Client(), factory.CommonDatabase().Client())

	// The server factory reconnects (what refreshCommonDatabase does on a settings change)
	serverFactory.database = after

	require.Same(t, after.Client(), factory.CommonDatabase().Client(),
		"the domain factory must see the new connection immediately, without a Refresh")
}

// TestFactory_CommonDatabase_NilWhileDisconnected pins the defense-in-depth guard: while the
// server factory is disconnected (setup mode, FACTORY-MODES D7), CommonDatabase returns the zero
// Server instead of panicking on a nil database.
func TestFactory_CommonDatabase_NilWhileDisconnected(t *testing.T) {

	factory := Factory{serverFactory: &lifecycleServerFactory{}}

	require.NotPanics(t, func() {
		require.Nil(t, factory.CommonDatabase().Client())
	})
}

// TestActivityStream_CommonDatabase_ReadsThroughServerFactory pins the no-capture contract one
// layer deeper: the ActivityStream service (whose cache is the reason the common database exists)
// must also see a reconnect immediately, WITHOUT another Refresh.  It was the one service that
// cached factory.CommonDatabase() as a value -- the exact capture that broke inbound signature
// verification after a config reload.
func TestActivityStream_CommonDatabase_ReadsThroughServerFactory(t *testing.T) {

	before := lazyLifecycleDatabase(t, "lifecycle-as-before")
	after := lazyLifecycleDatabase(t, "lifecycle-as-after")

	serverFactory := &lifecycleServerFactory{database: before}
	factory := Factory{serverFactory: serverFactory}

	activityStream := NewActivityStream()
	activityStream.Refresh(&factory)

	// currentClient unwraps the client behind the service's live common-database getter
	currentClient := func() *mongo.Client {
		server, isMongoServer := activityStream.getCommonDatabase().(mongodb.Server)
		require.True(t, isMongoServer, "getter must return the factory's mongodb.Server")
		return server.Client()
	}

	require.Same(t, before.Client(), currentClient())

	// The server factory reconnects; the service must see it with NO second Refresh
	serverFactory.database = after
	require.Same(t, after.Client(), currentClient(),
		"the ActivityStream service must see the new connection immediately, without a Refresh")
}

// TestFactory_Queue_ReadsThroughServerFactory pins the same no-capture contract for the task
// queue: when the server factory rebuilds the queue, every domain factory must publish to the
// NEW queue on its very next call.  A captured pointer would feed the stopped queue, which drops
// tasks silently.
func TestFactory_Queue_ReadsThroughServerFactory(t *testing.T) {

	before := queue.New()
	t.Cleanup(before.Stop)

	after := queue.New()
	t.Cleanup(after.Stop)

	serverFactory := &lifecycleServerFactory{queue: before}
	factory := Factory{serverFactory: serverFactory}

	require.Same(t, before, factory.Queue())

	// The server factory rebuilds the queue (what refreshQueue does on a real input change)
	serverFactory.queue = after

	require.Same(t, after, factory.Queue(),
		"the domain factory must see the new queue immediately, without a Refresh")
}
