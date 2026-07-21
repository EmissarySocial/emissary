package upgrades

import (
	"context"
	"fmt"

	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Version28 clears the two ways legacy data blocks the UNIQUE indexes on Collection and Rule.
//
// Collection gained a UNIQUE {parentId, collectionType} index (idx_Collection_Parent_Type), but its
// type field was renamed `type` -> `collectionType`. Rows written before the rename still carry the
// value under `type`, so they read `collectionType: null` and collide -- with no migration ever having
// touched the Collection collection.
//
// Rule's reconcile (Version27) is idempotent but was SKIPPED on databases already marked version 27 by
// an earlier occupant of that slot; their legacy `matchKey: null` rows still block idx_Rule_MatchKey.
// Re-running the reconcile here catches them. Both steps are safe on already-clean databases.
func Version28(ctx context.Context, session *mongo.Database) error {

	const location = "queries.upgrades.Version28"

	fmt.Println("... Version 28")

	if err := reconcileCollections(ctx, session); err != nil {
		return derp.Wrap(err, location, "Reconciling Collections")
	}

	if err := reconcileRules(ctx, session); err != nil {
		return derp.Wrap(err, location, "Reconciling Rules")
	}

	return nil
}

// reconcileCollections adopts the legacy `type` field into `collectionType`, then removes any true
// duplicates so the UNIQUE {parentId, collectionType} index can build. It is idempotent: once the
// rename is done and each (parentId, collectionType) is unique, a second run finds nothing to do.
func reconcileCollections(ctx context.Context, session *mongo.Database) error {

	const location = "queries.upgrades.reconcileCollections"

	collection := session.Collection("Collection")

	// STEP 1: Adopt the value from the legacy `type` field into `collectionType`. Only rows that lack a
	// `collectionType` are touched, so a row already saved through the current schema is left alone.
	renameFilter := bson.M{
		"type":           bson.M{"$exists": true},
		"collectionType": bson.M{"$exists": false},
	}

	renamePipeline := mongo.Pipeline{
		bson.D{{Key: "$set", Value: bson.D{{Key: "collectionType", Value: "$type"}}}},
		bson.D{{Key: "$unset", Value: "type"}},
	}

	if _, err := collection.UpdateMany(ctx, renameFilter, renamePipeline); err != nil {
		return derp.Wrap(err, location, "Renaming legacy 'type' field to 'collectionType'")
	}

	// STEP 2: Read the identity of every LIVE Collection. The partial index only covers live rows
	// (deleteDate == 0), so a soft-deleted row can never collide and is excluded here.
	cursor, err := collection.Find(ctx, bson.M{"deleteDate": 0}, options.Find().SetProjection(bson.M{
		"parentId":       1,
		"collectionType": 1,
		"createDate":     1,
	}))

	if err != nil {
		return derp.Wrap(err, location, "Listing Collections")
	}

	records := make([]collectionRecord, 0)

	if err := cursor.All(ctx, &records); err != nil {
		return derp.Wrap(err, location, "Reading Collections from cursor")
	}

	// STEP 3: Work out which duplicates to delete, then delete them in a single round-trip.
	deletions := planCollectionDedupe(records)

	if len(deletions) > 0 {
		if _, err := collection.DeleteMany(ctx, bson.M{"_id": bson.M{"$in": deletions}}); err != nil {
			return derp.Wrap(err, location, "Deleting duplicate Collections")
		}
	}

	fmt.Printf("... Collections reconciled: deleted %d duplicate\n", len(deletions))

	return nil
}

// collectionRecord is the slice of a Collection that reconcileCollections needs to group duplicates
// and choose which one survives.
type collectionRecord struct {
	CollectionID   primitive.ObjectID `bson:"_id"`
	ParentID       primitive.ObjectID `bson:"parentId"`
	CollectionType string             `bson:"collectionType"`
	CreateDate     int64              `bson:"createDate"`
}

// planCollectionDedupe groups the live records by the (parentId, collectionType) pair the unique index
// enforces and returns the IDs to delete: for any group with more than one member, every member except
// the most recently created one. Pure (no database) so the plan is unit-testable.
func planCollectionDedupe(records []collectionRecord) []primitive.ObjectID {

	groups := make(map[string][]collectionRecord)

	for _, record := range records {
		key := record.ParentID.Hex() + "|" + record.CollectionType
		groups[key] = append(groups[key], record)
	}

	deletions := make([]primitive.ObjectID, 0)

	for _, group := range groups {

		// A group with a single member holds the unique key alone -- nothing to reconcile.
		if len(group) < 2 {
			continue
		}

		// Keep the most recently created collection; delete the rest of the group.
		survivor := newestCollection(group)

		for _, record := range group {
			if record.CollectionID != survivor.CollectionID {
				deletions = append(deletions, record.CollectionID)
			}
		}
	}

	return deletions
}

// newestCollection returns the most recently created record in a non-empty group. Ties keep the first
// one seen -- which one survives does not matter; that exactly one survives does.
func newestCollection(group []collectionRecord) collectionRecord {

	newest := group[0]

	for _, record := range group[1:] {
		if record.CreateDate > newest.CreateDate {
			newest = record
		}
	}

	return newest
}
