package config

import (
	"context"
	"time"

	"github.com/benpate/derp"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// MongoStorage is a MongoDB-backed configuration storage
type MongoStorage struct {
	source        string
	location      string
	collection    *mongo.Collection
	updateChannel updateChannel
	done          context.Context
	cancelFunc    context.CancelFunc
}

// NewMongoStorage creates a fully initialized MongoStorage instance.
func NewMongoStorage(args *CommandLineArgs) (MongoStorage, error) {

	// It never ends the process: every failure comes back as an error carrying the operator-facing
	// guidance, and main decides what to do with it (see config.Load).  A missing configuration is
	// only an error without --setup; with it, a default configuration is written and served.

	const location = "config.NewMongoStorage"

	// Create a new MongoDB database connection
	connectOptions := options.Client().ApplyURI(args.Location)
	client, err := mongo.Connect(context.Background(), connectOptions)

	if err != nil {
		return MongoStorage{}, derp.Wrap(err, location, "The MongoDB config database could not be reached. Check the connection string and verify the database server connection.", args.Location)
	}

	// RULE: The config collection is read with `primary` + `majority` so a reload can never
	// observe a rolled-back or stale-secondary version of the configuration.  Without this, a
	// connect string carrying `readPreference=secondary` would silently make every node reload
	// the OLD config -- and since no second change event is coming, that staleness would last
	// until the process restarts.
	collectionOptions := options.Collection().
		SetReadPreference(readpref.Primary()).
		SetReadConcern(readconcern.Majority())

	collection := client.Database(args.Database).Collection(args.Collection, collectionOptions)

	done, cancelFunc := context.WithCancel(context.Background())

	storage := MongoStorage{
		source:        args.Source,
		location:      args.Location,
		collection:    collection,
		updateChannel: newUpdateChannel(),
		done:          done,
		cancelFunc:    cancelFunc,
	}

	// Load the configuration, capturing the cluster time of the read so the change stream can
	// start from that exact moment (see startAt below).
	config, startAt, err := storage.loadWithOperationTime(context.Background())

	switch {

	// If the config was read successfully, then NOOP here skips down to the next section.
	case err == nil:

	case derp.IsNotFound(err):

		// RULE: A missing configuration is an error UNLESS --setup was requested; creating a
		// fresh configuration is exactly what setup mode is for.
		if !args.Setup {
			return MongoStorage{}, derp.Wrap(err, location, "The configuration database is empty. Re-run Emissary with the --setup flag to initialize it.")
		}

		// Create a default configuration
		config = DefaultConfig()
		config.Source = storage.source
		config.Location = storage.location

		// Keep the STORED version (with its stamped revision), so the first console save
		// does not conflict against the bootstrap write
		written, inner := storage.Write(config)

		if inner != nil {
			return MongoStorage{}, derp.Wrap(inner, location, "Unable to write a new configuration to the MongoDB config database")
		}

		config = written

	default:
		return MongoStorage{}, derp.Wrap(err, location, "Unable to read the configuration from the MongoDB config database")
	}

	// If we have a valid config, post it to the update channel
	storage.updateChannel.notify(config)

	log.Info().Msgf("Loading configuration from mongodb")

	// After the first load, watch for changes to the config record and post them to the update channel
	go storage.watch(startAt)

	return storage, nil
}

// Subscribe returns a channel that will receive the configuration every time it is updated
func (storage MongoStorage) Subscribe() <-chan Config {
	return storage.updateChannel.subscribe()
}

// Close shuts down the MongoDB change-stream watcher
func (storage MongoStorage) Close() {
	storage.cancelFunc()
}

/******************************************
 * Change Stream Watcher
 ******************************************/

// watch supervises a MongoDB change stream on the config collection for the whole life of the
// process, reopening it whenever it dies.
//
// RULE: This loop MUST NOT be able to end except by cancellation.  A change stream can stop for
// reasons the driver does not resume on its own -- a server-side close (`invalidate`), or a
// non-resumable error such as `ChangeStreamHistoryLost` when the resume token ages out of the
// oplog.  Worse, the server-side close arrives as `Next() == false` with `Err() == nil`, so the
// old single-shot `for cs.Next() {}` loop returned *silently*: that node then ran on a frozen
// configuration, with nothing in the log, until someone rebooted it.
//
// `startAt` is the cluster time of the initial read (nil if the server did not report one).
// Starting there rather than "now" closes the window between loading the config and opening the
// stream -- a write landing in that gap would otherwise never be seen.
func (storage MongoStorage) watch(startAt *primitive.Timestamp) {

	const location = "config.MongoStorage.watch"

	var resumeToken bson.Raw

	// `failures` counts CONSECUTIVE failed attempts, so it lives across iterations: it drives the
	// backoff, and resets whenever a stream proves healthy by delivering an event.
	for failures := 0; ; {

		// RULE: Cancellation (Close) is the ONLY way out of this loop.
		if storage.done.Err() != nil {
			return
		}

		// `failures > 0` means a previous stream died, so this pass may have missed events while
		// it was down.  watchOnce re-reads the config on open to resynchronize.
		//
		// resumeToken is passed by POINTER because watchOnce owns it: it advances the token as
		// events arrive, and clears it when the server rejects it as too old.
		progressed, err := storage.watchOnce(&resumeToken, startAt, failures > 0)

		// A resume token supersedes the start time: once we have one, it is the precise place to
		// pick up, and replaying from the original start time would re-deliver old events.
		if resumeToken != nil {
			startAt = nil
		}

		if storage.done.Err() != nil {
			return
		}

		// A stream that delivered events was healthy; reset the backoff so a long-lived stream
		// that finally drops reconnects promptly instead of inheriting an old penalty.
		if progressed {
			failures = 0
		}

		switch {

		case err != nil:
			derp.Report(derp.Wrap(err, location, "Configuration change stream failed. Reopening."))

		// The silent case: the server closed the cursor and the driver reports no error at all.
		default:
			log.Warn().Str("loc", location).Msg("Configuration change stream was closed by the server. Reopening.")
		}

		failures++

		// RULE: Never hot-loop against a sick server.  Sleep, but stay cancelable.
		select {
		case <-storage.done.Done():
			return
		case <-time.After(watchRetryDelay(failures)):
		}
	}
}

// watchOnce opens one change stream and pumps its events until it dies.  It reports whether any
// event was delivered, and the terminating error (nil when the server simply closed the stream).
//
// It advances `*resumeToken` as events arrive, and clears it when the server refuses to resume
// from it -- a token ages out of the oplog as `ChangeStreamHistoryLost`, which is NOT resumable.
// RULE: A rejected token must be dropped, never retried.  Reopening with the same bad token every
// time would wedge configuration propagation just as permanently as the un-supervised loop this
// replaced; the caller resynchronizes with a full read instead.
func (storage MongoStorage) watchOnce(resumeToken *bson.Raw, startAt *primitive.Timestamp, resynchronize bool) (bool, error) {

	const location = "config.MongoStorage.watchOnce"

	// `UpdateLookup` makes every event carry the full config document, so a reload can use the
	// exact version the event describes instead of racing a second query against it.
	watchOptions := options.ChangeStream().SetFullDocument(options.UpdateLookup)

	switch {

	// RULE: StartAfter, not ResumeAfter.  The two are identical except that StartAfter can also
	// pick up after an `invalidate` event -- and `invalidate` is precisely the case that used to
	// end configuration propagation for good.  ResumeAfter would refuse that token, forcing a
	// full restart of the stream on every recovery.
	case *resumeToken != nil:
		watchOptions.SetStartAfter(*resumeToken)

	case startAt != nil:
		watchOptions.SetStartAtOperationTime(startAt)
	}

	changeStream, err := storage.collection.Watch(storage.done, mongo.Pipeline{}, watchOptions)

	if err != nil {

		// The token is the most likely reason the server refused, and it is the only input we can
		// discard.  Dropping it costs one full re-read on the next pass and always makes progress.
		if *resumeToken != nil {
			log.Warn().Str("loc", location).Msg("MongoDB refused the configuration change stream resume token. Restarting the stream from the current time.")
			*resumeToken = nil
		}

		return false, derp.Wrap(err, location, "Opening MongoDB change stream on the configuration collection")
	}

	// RULE: Belt and braces on the driver's (stream, error) contract.  A nil stream with a nil
	// error would panic below, and this loop's whole job is that it cannot die unexpectedly.
	if changeStream == nil {
		return false, derp.Internal(location, "MongoDB returned no change stream, and no error")
	}

	defer func() {
		_ = changeStream.Close(context.Background())
	}()

	// Once a stream is open, the token it reports is the authoritative resume point.
	defer func() {
		if token := changeStream.ResumeToken(); token != nil {
			*resumeToken = token
		}
	}()

	// A reopened stream may have missed events while it was down, so re-read the configuration
	// now.  Reloads are idempotent, which makes this cheap insurance against a silent gap.
	if resynchronize {
		log.Info().Str("loc", location).Msg("Configuration change stream reopened. Reloading configuration.")

		if config, err := storage.load(storage.done); err == nil {
			storage.updateChannel.notify(config)
		} else if !derp.IsNotFound(err) {
			derp.Report(derp.Wrap(err, location, "Reloading configuration after reopening the change stream"))
		}
	}

	progressed := false

	for changeStream.Next(storage.done) {

		progressed = true

		if config, ok := storage.configFromEvent(changeStream.Current); ok {
			storage.updateChannel.notify(config)
		}
	}

	return progressed, changeStream.Err()
}

// configFromEvent derives the new configuration from a single change stream event.  It prefers
// the event's own `fullDocument` -- that is the exact version the event describes, so it costs no
// round trip and cannot race a concurrent write -- and falls back to re-reading the collection
// for events that carry no document (deletes, and inserts of some other document).
func (storage MongoStorage) configFromEvent(event bson.Raw) (Config, bool) {

	const location = "config.MongoStorage.configFromEvent"

	if document, err := event.LookupErr("fullDocument"); err == nil {

		result := NewConfig()

		if err := document.Unmarshal(&result); err == nil {
			storage.decorate(&result)
			return result, true
		}

		derp.Report(derp.Internal(location, "Unable to decode configuration from change stream event. Falling back to a full read."))
	}

	// Fall through means the event carried no usable document, so read the collection instead.
	config, err := storage.load(storage.done)

	if err != nil {

		// A deleted configuration is not an error worth reporting on every event: there is simply
		// nothing to apply, and the running server keeps its current settings.
		if !derp.IsNotFound(err) {
			derp.Report(derp.Wrap(err, location, "Loading updated config from MongoDB"))
		}

		return Config{}, false
	}

	return config, true
}

/******************************************
 * Read / Write
 ******************************************/

// load reads the configuration from the MongoDB database
func (storage MongoStorage) load(ctx context.Context) (Config, error) {

	const location = "config.MongoStorage.load"

	result := NewConfig()

	// RULE: Sort by `_id` so that every node in a cluster picks the SAME document when the
	// collection somehow holds more than one.  An unsorted FindOne returns whichever document the
	// server reaches first, which can differ between nodes and never converges.
	findOptions := options.FindOne().SetSort(bson.D{{Key: "_id", Value: 1}})

	if err := storage.collection.FindOne(ctx, bson.M{}, findOptions).Decode(&result); err != nil {

		if err == mongo.ErrNoDocuments {
			return Config{}, derp.NotFound(location, "Unable to load config from MongoDB", err.Error())
		}

		return Config{}, derp.Wrap(err, location, "Decoding config from MongoDB")
	}

	storage.decorate(&result)

	return result, nil
}

// loadWithOperationTime reads the configuration inside a session, and reports the cluster time of
// that read.  The change stream is started from this timestamp so that no write can slip through
// the gap between "read the configuration" and "start watching for changes".
//
// The timestamp is nil when the server does not report one (a standalone mongod), in which case
// the caller falls back to watching from "now".
func (storage MongoStorage) loadWithOperationTime(ctx context.Context) (Config, *primitive.Timestamp, error) {

	const location = "config.MongoStorage.loadWithOperationTime"

	session, err := storage.collection.Database().Client().StartSession()

	if err != nil {

		// A session is an optimization, not a requirement: fall back to a plain read (and to
		// watching from "now") rather than refusing to boot.
		derp.Report(derp.Wrap(err, location, "Unable to start a MongoDB session. Falling back to an untimed read."))

		config, err := storage.load(ctx)
		return config, nil, err
	}

	defer session.EndSession(ctx)

	var result Config

	err = mongo.WithSession(ctx, session, func(sessionContext mongo.SessionContext) error {
		var inner error
		result, inner = storage.load(sessionContext)
		return inner
	})

	if err != nil {
		return Config{}, nil, err
	}

	return result, session.OperationTime(), nil
}

// decorate stamps the read-only, node-local fields onto a configuration that was just read from
// the database.  They describe where THIS process found its configuration, so they are never
// stored and must be re-applied to every value that leaves storage.
func (storage MongoStorage) decorate(config *Config) {

	config.Source = storage.source
	config.Location = storage.location

	// RULE: Warn loudly about domains that cannot encrypt vault data (BUG-110)
	config.ReportInvalidMasterKeys()
}

// Read returns the configuration as currently stored.  It is part of the Storage interface;
// callers use it to rebase a read-modify-write after Write reports a revision conflict.
func (storage MongoStorage) Read() (Config, error) {
	return storage.load(context.Background())
}

// Write persists the configuration and returns it as stored, with its Revision incremented.
//
// RULE: This is a compare-and-swap, not a blind replace.  The criteria match the document only
// at the Revision the caller read, so a save built from a stale base returns a 409 and changes
// NOTHING -- instead of silently reverting every change another node made since that base,
// which could include a domain's MasterKey, stored nowhere else.
func (storage MongoStorage) Write(config Config) (Config, error) {

	const location = "config.MongoStorage.Write"

	// The stored document always carries the NEXT revision
	stored := config
	stored.Revision = config.Revision + 1

	criteria := bson.M{"_id": config.MongoID}

	// RULE: A document written before revisions existed has no `revision` field at all, so a
	// caller holding Revision 0 must match either shape.  The first conditional write stamps
	// the field, and the legacy leg never matches again.
	if config.Revision == 0 {
		criteria["$or"] = []bson.M{
			{"revision": int64(0)},
			{"revision": bson.M{"$exists": false}},
		}
	} else {
		criteria["revision"] = config.Revision
	}

	result, err := storage.collection.ReplaceOne(context.Background(), criteria, stored)

	if err != nil {
		return Config{}, derp.Wrap(err, location, "Writing config to MongoDB")
	}

	// Matched means the swap happened at the expected revision.  (The nil check is belt and
	// braces on the driver's (result, error) contract, same as the change stream's.)
	if result != nil && result.MatchedCount > 0 {
		return stored, nil
	}

	// No match is one of two very different situations: the document moved (conflict), or it
	// does not exist yet (first write).  Ask which.
	count, err := storage.collection.CountDocuments(context.Background(), bson.M{"_id": config.MongoID})

	if err != nil {
		return Config{}, derp.Wrap(err, location, "Verifying config document in MongoDB")
	}

	if count > 0 {
		return Config{}, derp.Conflict(location, "The configuration was changed by another server. Your change was NOT saved. Reload and try again.")
	}

	// First write: insert.  A duplicate-key error means another node's first write beat this
	// one by a moment -- the same stale-base situation, reported the same way.
	if _, err := storage.collection.InsertOne(context.Background(), stored); err != nil {

		if mongo.IsDuplicateKeyError(err) {
			return Config{}, derp.Conflict(location, "The configuration was changed by another server. Your change was NOT saved. Reload and try again.")
		}

		return Config{}, derp.Wrap(err, location, "Writing new config to MongoDB")
	}

	return stored, nil
}
