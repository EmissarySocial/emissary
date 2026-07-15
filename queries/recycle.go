package queries

import (
	"time"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
)

// recycleDelay is how long a "soft deleted" record is retained before it is permanently purged.
const recycleDelay = 30 * 24 * time.Hour

// Recycle permanently deletes all records from the specified collection
// that were "soft deleted" longer ago than `recycleDelay`.
func Recycle(session data.Session, collectionName string) error {

	const location = "queries.Recycle"

	// Get a MongoDB collection
	collection := mongoCollection(session.Collection(collectionName))

	if collection == nil {
		return derp.Internal(location, "Collection must be a MongoDB collection")
	}

	// Set a max timeout of 180 seconds (3 minutes) to run this query
	timeout, cancel := timeoutContext(180)
	defer cancel()

	filter := recycleFilter(time.Now())

	if _, err := collection.DeleteMany(timeout, filter); err != nil {
		return derp.Wrap(err, location, "Purging deleted records", collectionName, filter)
	}

	// Done.
	return nil
}

// recycleFilter returns the query that matches every record eligible to be purged: records that
// were soft-deleted, and whose deletion is older than `recycleDelay`.
func recycleFilter(now time.Time) bson.M {

	// This is factored out of Recycle so its bounds can be tested without a live database. Getting it
	// wrong is unrecoverable, so both bounds below are load-bearing.

	// RULE: `deleteDate` is a Unix epoch in MILLISECONDS (see journal.SetDeleted), so the cutoff
	// must be UnixMilli too. A seconds-based cutoff is ~1000x too small, which would match nothing
	// and leave this task silently purging nothing, forever.
	cutoff := now.Add(-recycleDelay).UnixMilli()

	// RULE: BOTH bounds are required, and `$gt: 0` is the load-bearing one. `deleteDate` is ZERO on
	// every LIVE record, so `$lt: cutoff` on its own matches every live record in the collection and
	// would purge the entire database. Only ever purge records that are deleted (> 0) AND expired.
	return bson.M{
		"deleteDate": bson.M{
			"$gt": 0,
			"$lt": cutoff,
		},
	}
}
