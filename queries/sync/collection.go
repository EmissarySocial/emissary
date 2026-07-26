package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Collection(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "Collection").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Collection"), indexer.IndexSet{

		// idx_Collection_Recycle serves the nightly RecycleDomain purge (deleteDate > 0).
		"idx_Collection_Recycle": recycleIndex(),

		// Enforces one Collection per (parentId, collectionType). This is what makes JIT
		// creation concurrency-safe: service.Collection.loadOrCreateByParent inserts
		// optimistically and relies on this index to reject the loser of a create race.
		//
		// The partial filter carries TWO conditions, and both are load-bearing:
		//
		//  1. deleteDate == 0 scopes uniqueness to live rows, mirroring service.notDeleted(), so a
		//     soft-deleted collection does not block re-creation of a fresh one with the same key.
		//
		//  2. parentId must EXIST. Uniqueness applies only to PARENTED (system) collections --
		//     Context, Likes, Dislikes, Replies, Shares -- which loadOrCreateByParent creates
		//     just-in-time against a Stream or User. UNPARENTED collections are user-created
		//     conversation groups (handler.activitypub_user.outbox_CreateOrderedCollection, driven
		//     by the Conversations app), and there is no limit on how many of those a user may have.
		//
		// Without condition 2 the index silently caps the entire DOMAIN at one conversation. A
		// conversation collection leaves ParentID at the zero ObjectID and CollectionType empty;
		// both fields are `bson:",omitempty"`, so both are OMITTED from the stored document, and
		// Mongo indexes a missing field as null. Every conversation therefore collides on the key
		// {parentId: null, collectionType: null} -- and because this index has no userId component,
		// the first user's conversation blocks every other user's. See collection_mongo_test.go.
		"idx_Collection_Parent_Type": mongo.IndexModel{
			Keys: bson.D{
				{Key: "parentId", Value: 1},
				{Key: "collectionType", Value: 1},
			},
			// NOTE: the nested operator is a bson.M, not a bson.D. indexer.compareModel diffs the
			// declared filter against the one Mongo stored, and Mongo returns nested operators as
			// maps -- so a bson.D here would never compare equal, and Sync would drop and rebuild
			// this UNIQUE index on every boot. See TestCollectionIndex_SyncIsIdempotent.
			Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.D{
				{Key: "deleteDate", Value: 0},
				{Key: "parentId", Value: bson.M{"$exists": true}},
			}),
		},
	})
}
