package server

import (
	"github.com/EmissarySocial/emissary/config"
	"github.com/benpate/turbine/queue"
	"github.com/realclientip/realclientip-go"
	"github.com/spf13/afero"
	"go.mongodb.org/mongo-driver/mongo"
)

/******************************************
 * Server Wiring
 *
 * Everything a configuration reload replaces lives in one
 * immutable value, swapped in with a single atomic store.
 * Readers take no lock at all, and can never observe the new
 * queue alongside the old database.
 ******************************************/

// wiring is everything the server configuration gets turned into: one generation of the live
// objects that a reload replaces.  A reload copies the current wiring, edits the copy, and
// publishes it.
//
// RULE: Published wiring is IMMUTABLE.  Readers hold a pointer to it for as long as they like,
// on any goroutine, with no lock -- which is only sound because the value never changes
// underneath them.  Change something by publishing a NEW generation (rewire), never by writing
// through a pointer you loaded.
type wiring struct {

	// config is the server configuration this wiring was built from.  Like everything else in a
	// published generation it is IMMUTABLE -- and because it holds maps and slices, every writer
	// deep-copies it (config.Config.Copy) before editing, so no generation ever shares map
	// storage with another generation, or with a caller's scratch copy.  That discipline is
	// encapsulated in setConfigLocked (and the helpers built on it); go through them.
	config config.Config

	// Filesystems mounted from the configuration
	attachmentOriginals afero.Fs
	attachmentCache     afero.Fs
	exportCache         afero.Fs

	// commonDatabase is the shared (ActivityPub Cache) database.  It is nillable: each mode's
	// lifecycle connects it, so it is nil before that happens, and again after a failed
	// connection rolls back (FACTORY-MODES D7).
	commonDatabase *mongo.Database

	// The settings that produced commonDatabase.  Plain strings, never the config's
	// mapof.String, because config handlers mutate those maps in place -- so a map comparison
	// cannot detect a change.  refreshCommonDatabase compares against these to keep the live
	// client when a reload does not touch the connection: disconnecting an unchanged client
	// would strand every existing domain factory, the queue's storage, and the ActivityStream
	// cache on a dead mongo client.
	commonDatabaseURI  string
	commonDatabaseName string

	// commonDatabaseVerified is TRUE only once a Ping has proved the server reachable.  The
	// setup console gates domain management on it, so that a saved-but-unreachable database
	// reports a clear error instead of binding domains to a client that cannot answer.  The live
	// server never sets it: it exits rather than run without a database at all.
	commonDatabaseVerified bool

	// queue is the task queue, and the three fields after it are the inputs that produced it.
	// refreshQueue compares against them to keep the running queue when nothing it depends on
	// has changed -- rebuilding stops the old queue, and any task already handed to it dies
	// silently ("Turbine Queue: stopped").  queueReady distinguishes a real, consumer-bearing
	// queue from the inert placeholder that init() installs; queueDatabase is compared by
	// POINTER IDENTITY, which changes exactly when refreshCommonDatabase swaps the connection.
	queue            *queue.Queue
	queueReady       bool
	queueWithStorage bool
	queueDatabase    *mongo.Database

	// clientIPStrategy decides which address in a proxied request is the real client's
	clientIPStrategy realclientip.Strategy
}

// currentWiring returns the generation of server wiring that is live right now.
//
// It never returns nil, so a zero-value factoryCore is usable and a reader that runs before the
// first publish sees empty wiring rather than panicking.  The returned value is READ-ONLY -- see
// the RULE on the wiring type.
func (factory *factoryCore) currentWiring() *wiring {

	if result := factory.wiring.Load(); result != nil {
		return result
	}

	return &wiring{}
}

// rewire publishes a new generation of server wiring, taking reloadLock for the duration.  It is
// for callers that are not already inside a reload: construction, and the one-off updates that
// do not touch anything else.
func (factory *factoryCore) rewire(fn func(*wiring)) {

	factory.reloadLock.Lock()
	defer factory.reloadLock.Unlock()

	factory.rewireLocked(fn)
}

// rewireLocked publishes a new generation of server wiring: `fn` receives a COPY of the current
// generation and edits it, and the result is swapped in atomically.
//
// RULE: The caller MUST already hold reloadLock.  Copy-edit-publish is not atomic on its own, so
// two concurrent reloads would silently lose one of the two updates -- and worse, the "have the
// settings changed?" guards that read the current generation before calling this would be
// deciding on input that another reload had already replaced.
func (factory *factoryCore) rewireLocked(fn func(*wiring)) {

	updated := *factory.currentWiring()

	fn(&updated)

	factory.wiring.Store(&updated)
}
