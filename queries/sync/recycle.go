package sync

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// recycleIndex returns the index that serves queries.Recycle (the nightly RecycleDomain purge),
// which matches soft-deleted rows via `deleteDate: {$gt: 0, $lt: cutoff}`.
//
// It is a PARTIAL index filtered to `deleteDate > 0`, so it indexes ONLY the soft-deleted rows.
// This matters twice: the uniqueness partials elsewhere are filtered to `deleteDate == 0` (live
// rows) and by definition cannot serve a `deleteDate > 0` query, and a plain index would carry
// every live row -- millions in the large collections -- for no benefit. Live inserts (deleteDate
// == 0) never touch this index, so it adds negligible write cost while turning the recycle
// COLLSCAN into an index scan.
//
// Every collection in service.Factory.Collections() is recycled, so every one of those sync
// definitions includes this index under the key "idx_<Collection>_Recycle".
func recycleIndex() mongo.IndexModel {
	return mongo.IndexModel{
		Keys: bson.D{
			{Key: "deleteDate", Value: 1},
		},
		Options: options.Index().SetPartialFilterExpression(bson.M{
			"deleteDate": bson.M{"$gt": 0},
		}),
	}
}
