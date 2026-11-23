package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/benpate/derp"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// retiredOutboxItemCollection names the collection where older servers write OutboxItems
const retiredOutboxItemCollection = "Outbox2"

// OutboxItem moves any rows left in the retired Outbox2 collection, then synchronizes the MongoDB
// indexes for the OutboxItem collection
func OutboxItem(ctx context.Context, database *mongo.Database) error {

	const location = "queries.sync.OutboxItem"

	log.Trace().Str("database", database.Name()).Str("collection", "OutboxItem").Msg("COLLECTION:")

	// Move the rows that servers wrote before the rename. A failure is reported but not fatal,
	// so the indexes still build, and the move is tried again on the next start.
	if err := moveRetiredOutboxItems(ctx, database); err != nil {
		derp.Report(derp.Wrap(err, location, "Moving rows from the retired Outbox2 collection", database.Name()))
	}

	// OutboxItem has no recycle index, because service.Outbox2 only hard-deletes (see AGENTS.md)
	return indexer.Sync(ctx, database.Collection("OutboxItem"), indexer.IndexSet{

		// Serves an actor's outbox collection, newest first
		"idx_OutboxItem_Actor": mongo.IndexModel{
			Keys: bson.D{
				{Key: "actorType", Value: 1},
				{Key: "actorId", Value: 1},
				{Key: "createDate", Value: -1},
			},
		},

		// Serves "does this outbox already hold a Create for this object?"
		"idx_OutboxItem_Actor_Object": mongo.IndexModel{
			Keys: bson.D{
				{Key: "actorType", Value: 1},
				{Key: "actorId", Value: 1},
				{Key: "objectId", Value: 1},
				{Key: "activityType", Value: 1},
			},
		},

		// RULE: One row per activity URL, so a replayed or racing publish cannot store an activity twice.
		// Rows with an empty or missing url are left out, because ActivityURL returns "" for an
		// unknown actor type, and those rows must not collide with one another.
		"idx_OutboxItem_URL": mongo.IndexModel{
			Keys: bson.D{
				{Key: "url", Value: 1},
			},
			Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{
				"url": bson.M{"$gt": ""},
			}),
		},

		// Serves lookups by object across every outbox
		"idx_OutboxItem_Object": mongo.IndexModel{
			Keys: bson.D{
				{Key: "objectId", Value: 1},
			},
		},

		// Serves the read filter on permissions
		"idx_OutboxItem_Permissions": mongo.IndexModel{
			Keys: bson.D{
				{Key: "permissions", Value: 1},
			},
		},
	})
}

// moveRetiredOutboxItems moves every row in the retired Outbox2 collection into OutboxItem. A row
// that cannot be moved is reported and left in place, and the rest still move.
func moveRetiredOutboxItems(ctx context.Context, database *mongo.Database) error {

	const location = "queries.sync.moveRetiredOutboxItems"

	// Skip the work when there is no retired collection, as on every Domain created since the rename
	names, err := database.ListCollectionNames(ctx, bson.M{"name": retiredOutboxItemCollection})

	if err != nil {
		return derp.Wrap(err, location, "Looking for the retired collection")
	}

	if len(names) == 0 {
		return nil
	}

	// Read every retired row
	source := database.Collection(retiredOutboxItemCollection)
	cursor, err := source.Find(ctx, bson.M{})

	if err != nil {
		return derp.Wrap(err, location, "Reading retired rows")
	}

	// A failed Close leaves the server to time the cursor out, which costs nothing here
	defer func() { _ = cursor.Close(context.WithoutCancel(ctx)) }()

	// Move each row on its own, so that one bad row does not hold back the others
	moved := 0

	for cursor.Next(ctx) {

		row := bson.D{}

		if err := cursor.Decode(&row); err != nil {
			derp.Report(derp.Wrap(err, location, "Decoding retired row"))
			continue
		}

		if err := moveRetiredOutboxItem(ctx, source, database.Collection("OutboxItem"), row); err != nil {
			derp.Report(err)
			continue
		}

		moved++
	}

	if err := cursor.Err(); err != nil {
		return derp.Wrap(err, location, "Reading retired rows")
	}

	if moved > 0 {
		log.Info().Str("database", database.Name()).Int("rows", moved).Msg("Moved rows from Outbox2 to OutboxItem")
	}

	// Moved. Like a boss.
	return nil
}

// moveRetiredOutboxItem copies one retired row into OutboxItem, unless a row with its _id is
// already there, and then deletes the retired copy
func moveRetiredOutboxItem(ctx context.Context, source *mongo.Collection, target *mongo.Collection, row bson.D) error {

	const location = "queries.sync.moveRetiredOutboxItem"

	// Separate the _id from the fields it is stored with
	var rowID any
	fields := make(bson.D, 0, len(row))

	for _, element := range row {

		if element.Key == "_id" {
			rowID = element.Value
			continue
		}

		fields = append(fields, element)
	}

	// RULE: A row without an _id cannot be matched to its copy, so it stays where it is
	if rowID == nil {
		return derp.Internal(location, "Retired row has no _id")
	}

	// RULE: Never overwrite. A row that is already in OutboxItem was written by the current
	// code, so it wins, and the retired copy is only deleted.
	if _, err := target.UpdateOne(ctx, bson.M{"_id": rowID}, bson.M{"$setOnInsert": fields}, options.Update().SetUpsert(true)); err != nil {
		return derp.Wrap(err, location, "Copying retired row", rowID)
	}

	// Delete by _id, never by dropping the collection, so a row that an older server writes
	// during a rolling deploy is kept and moved on the next start
	if _, err := source.DeleteOne(ctx, bson.M{"_id": rowID}); err != nil {
		return derp.Wrap(err, location, "Deleting retired row", rowID)
	}

	return nil
}
