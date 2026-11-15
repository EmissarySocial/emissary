package floodgatemongo

import (
	"context"
	"errors"
	"iter"
	"slices"
	"time"

	"github.com/benpate/derp"
	"github.com/benpate/floodgate"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Store keeps Floodgate's host records in the "Host" collection of a MongoDB database
type Store struct {
	database func() *mongo.Database // Returns the live database on every call, or nil when there is none
}

// New returns a Store that asks for its database on every call, because a configuration reload
// can replace the connection
func New(database func() *mongo.Database) Store {
	return Store{database: database}
}

// Load returns the record for a host. A host with no record yields a zero Host and no error.
func (store Store) Load(ctx context.Context, host string) (floodgate.Host, error) {

	const location = "floodgatemongo.Store.Load"

	result := floodgate.Host{}
	collection, err := store.collection()

	if err != nil {
		return result, derp.Wrap(err, location, "Connecting to database")
	}

	err = collection.FindOne(ctx, bson.M{"_id": host}).Decode(&result)

	// A host that has never had trouble has no record
	if errors.Is(err, mongo.ErrNoDocuments) {
		return floodgate.Host{}, nil
	}

	if err != nil {
		return floodgate.Host{}, derp.Wrap(err, location, "Loading host record", "host: "+host)
	}

	return result, nil
}

// RangeBlocked returns every host that is dormant or paused
func (store Store) RangeBlocked(ctx context.Context) (iter.Seq[floodgate.Host], error) {

	filter := bson.M{"$or": bson.A{
		bson.M{"dormantSince": bson.M{"$gt": 0}},
		bson.M{"isPaused": true},
	}}

	return store.find(ctx, "floodgatemongo.Store.RangeBlocked", filter)
}

// RangeProbesDue returns every dormant host whose next check is at or before the given time
func (store Store) RangeProbesDue(ctx context.Context, now time.Time) (iter.Seq[floodgate.Host], error) {

	filter := bson.M{
		"dormantSince": bson.M{"$gt": 0},
		"nextProbe":    bson.M{"$lte": now.Unix()},
	}

	return store.find(ctx, "floodgatemongo.Store.RangeProbesDue", filter)
}

// RecordSuccess marks the host healthy and clears the failure history, the dormant mark and
// the probe schedule. It writes nothing when the host has no record.
func (store Store) RecordSuccess(ctx context.Context, host string, now time.Time) error {

	update := bson.M{"$set": bson.M{
		"isUnhealthy":  false,
		"testAfter":    0,
		"cooldown":     0,
		"failureCount": 0,
		"failureDates": bson.A{},
		"dormantSince": 0,
		"nextProbe":    0,
		"probeCount":   0,
		"lastRevived":  now.Unix(),
	}}

	return store.update(ctx, "floodgatemongo.Store.RecordSuccess", bson.M{"_id": host}, update, false)
}

// RecordFailure adds one to the failure count, adds the day to the failure dates if it is new,
// creates the record if there is none, and returns the record as it stands afterwards
func (store Store) RecordFailure(ctx context.Context, host string, now time.Time, date string) (floodgate.Host, error) {

	const location = "floodgatemongo.Store.RecordFailure"

	result := floodgate.Host{}
	collection, err := store.collection()

	if err != nil {
		return result, derp.Wrap(err, location, "Connecting to database")
	}

	update := bson.M{
		"$inc":      bson.M{"failureCount": 1},
		"$addToSet": bson.M{"failureDates": date},
		"$set":      bson.M{"lastFailure": now.Unix()},
	}

	// One operation counts the failure and returns the result, so concurrent reports are all counted
	findOptions := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)

	if err := collection.FindOneAndUpdate(ctx, bson.M{"_id": host}, update, findOptions).Decode(&result); err != nil {
		return floodgate.Host{}, derp.Wrap(err, location, "Recording failure", "host: "+host)
	}

	return result, nil
}

// MarkUnhealthy holds deliveries back until the given time, and records the cooldown that was used
func (store Store) MarkUnhealthy(ctx context.Context, host string, testAfter time.Time, cooldown time.Duration) error {

	update := bson.M{"$set": bson.M{
		"isUnhealthy": true,
		"testAfter":   testAfter.Unix(),
		"cooldown":    int64(cooldown / time.Second),
	}}

	return store.update(ctx, "floodgatemongo.Store.MarkUnhealthy", bson.M{"_id": host}, update, false)
}

// MarkDormant marks the host dormant and schedules its first check. It changes nothing
// when the host is already marked.
func (store Store) MarkDormant(ctx context.Context, host string, since time.Time, nextProbe time.Time) error {

	// RULE: Only the first mark counts, so a second server cannot restart the probe schedule
	filter := bson.M{"_id": host, "dormantSince": bson.M{"$not": bson.M{"$gt": 0}}}

	update := bson.M{"$set": bson.M{
		"dormantSince": since.Unix(),
		"nextProbe":    nextProbe.Unix(),
		"probeCount":   0,
	}}

	return store.update(ctx, "floodgatemongo.Store.MarkDormant", filter, update, false)
}

// RecordProbeFailure adds one to the probe count and schedules the next check
func (store Store) RecordProbeFailure(ctx context.Context, host string, nextProbe time.Time) error {

	update := bson.M{
		"$inc": bson.M{"probeCount": 1},
		"$set": bson.M{"nextProbe": nextProbe.Unix()},
	}

	return store.update(ctx, "floodgatemongo.Store.RecordProbeFailure", bson.M{"_id": host}, update, false)
}

// SetRateLimit pauses the host until the given time, creating the record if there is none.
// It never shortens a pause: the later of the stored time and the new time is kept.
func (store Store) SetRateLimit(ctx context.Context, host string, until time.Time) error {
	update := bson.M{"$max": bson.M{"rateLimitUntil": until.Unix()}}
	return store.update(ctx, "floodgatemongo.Store.SetRateLimit", bson.M{"_id": host}, update, true)
}

// SetPaused records whether the server operator has stopped deliveries to the host, creating
// the record if there is none
func (store Store) SetPaused(ctx context.Context, host string, isPaused bool) error {
	update := bson.M{"$set": bson.M{"isPaused": isPaused}}
	return store.update(ctx, "floodgatemongo.Store.SetPaused", bson.M{"_id": host}, update, true)
}

// SetNodeInfo records the software that the host runs, creating the record if there is none
func (store Store) SetNodeInfo(ctx context.Context, host string, software string, version string) error {
	update := bson.M{"$set": bson.M{"software": software, "version": version}}
	return store.update(ctx, "floodgatemongo.Store.SetNodeInfo", bson.M{"_id": host}, update, true)
}

// SetSignatureFormat records the HTTP signature format that the host accepts, creating the
// record if there is none
func (store Store) SetSignatureFormat(ctx context.Context, host string, format string) error {
	update := bson.M{"$set": bson.M{"signatureFormat": format}}
	return store.update(ctx, "floodgatemongo.Store.SetSignatureFormat", bson.M{"_id": host}, update, true)
}

// collection returns the live "Host" collection, or an error when no database is connected
func (store Store) collection() (*mongo.Collection, error) {

	// RULE: Ask every time. A handle captured earlier goes stale when the configuration reloads.
	database := store.database()

	if database == nil {
		return nil, derp.Internal("floodgatemongo.Store.collection", "The common database is not connected")
	}

	return database.Collection(CollectionName), nil
}

// update applies one atomic update to the record that the filter selects
func (store Store) update(ctx context.Context, location string, filter bson.M, update bson.M, upsert bool) error {

	collection, err := store.collection()

	if err != nil {
		return derp.Wrap(err, location, "Connecting to database")
	}

	if _, err := collection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(upsert)); err != nil {
		return derp.Wrap(err, location, "Updating host record")
	}

	return nil
}

// find returns every record that the filter selects. The list is read in full before it is
// returned, because it holds only the hosts that are in trouble.
func (store Store) find(ctx context.Context, location string, filter bson.M) (iter.Seq[floodgate.Host], error) {

	collection, err := store.collection()

	if err != nil {
		return nil, derp.Wrap(err, location, "Connecting to database")
	}

	cursor, err := collection.Find(ctx, filter)

	if err != nil {
		return nil, derp.Wrap(err, location, "Querying host records")
	}

	result := make([]floodgate.Host, 0)

	if err := cursor.All(ctx, &result); err != nil {
		return nil, derp.Wrap(err, location, "Reading host records")
	}

	return slices.Values(result), nil
}
