package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/benpate/derp"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// annotationUniqueIndex names the index that enforces one live Annotation per (User, URL).
// It replaces the non-unique idx_Annotation_URL under a new name, so that its presence
// proves the collection has already been de-duplicated.
const annotationUniqueIndex = "idx_Annotation_User_URL_Unique"

// Annotation synchronizes the MongoDB indexes for the Annotation collection, de-duplicating it first so the unique index can build
func Annotation(ctx context.Context, database *mongo.Database) error {

	const location = "queries.sync.Annotation"

	log.Trace().Str("database", database.Name()).Str("collection", "Annotation").Msg("COLLECTION:")

	collection := database.Collection("Annotation")

	// Clear out duplicates that would block the unique index. A failure is reported but not
	// fatal: only the unique index fails to build (and is retried on the next startup).
	if err := deduplicateAnnotations(ctx, collection); err != nil {
		derp.Report(derp.Wrap(err, location, "Removing duplicate Annotation records", database.Name()))
	}

	return indexer.Sync(ctx, collection, indexer.IndexSet{

		// idx_Annotation_Recycle serves the nightly RecycleDomain purge (deleteDate > 0).
		"idx_Annotation_Recycle": recycleIndex(),

		// Enforces one live annotation per (userId, url) and serves LoadByURL. Soft-deleted rows
		// are excluded, so a URL can be annotated again after its annotation is removed.
		annotationUniqueIndex: mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "url", Value: 1},
			},
			Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{
				"deleteDate": 0,
			}),
		},

		// Serves the annotations page, newest first, with or without a type filter.
		"idx_Annotation_User_Created": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "createDate", Value: -1},
			},
			Options: options.Index().SetPartialFilterExpression(bson.M{
				"deleteDate": 0,
			}),
		},
	})
}

// annotationRecord is the slice of an Annotation that deduplicateAnnotations needs to choose a survivor.
type annotationRecord struct {
	AnnotationID  primitive.ObjectID `bson:"annotationId"`
	ContentLength int64              `bson:"contentLength"`
	CreateDate    int64              `bson:"createDate"`
}

// annotationGroup collects every live Annotation that one User holds against one URL.
type annotationGroup struct {
	Annotations []annotationRecord `bson:"annotations"`
}

// deduplicateAnnotations removes duplicate live Annotations, keeping the one with the longest
// note in each (User, URL) set, so that annotationUniqueIndex can be built on an older database.
// Once the index exists, duplicates can no longer be written, so later startups skip the scan.
func deduplicateAnnotations(ctx context.Context, collection *mongo.Collection) error {

	const location = "queries.sync.deduplicateAnnotations"

	// The index is only ever created after a successful cleanup, so its presence proves that
	// there is nothing left to find.
	indexed, err := indexExists(ctx, collection, annotationUniqueIndex)

	if err != nil {
		return derp.Wrap(err, location, "Looking for the unique index")
	}

	if indexed {
		return nil
	}

	// Collect every (User, URL) carrying more than one live Annotation
	pipeline := []bson.M{
		{"$match": bson.M{"deleteDate": 0}},
		{"$group": bson.M{
			"_id": bson.M{"userId": "$userId", "url": "$url"},
			"annotations": bson.M{"$push": bson.M{
				"annotationId":  "$_id",
				"contentLength": bson.M{"$strLenCP": bson.M{"$ifNull": bson.A{"$content", ""}}},
				"createDate":    "$createDate",
			}},
		}},
		{"$match": bson.M{"annotations.1": bson.M{"$exists": true}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline, options.Aggregate().SetAllowDiskUse(true))

	if err != nil {
		return derp.Wrap(err, location, "Grouping Annotation records by User and URL")
	}

	groups := make([]annotationGroup, 0)

	if err := cursor.All(ctx, &groups); err != nil {
		return derp.Wrap(err, location, "Reading Annotation records from cursor")
	}

	// Work out which Annotations are displaced (everything but each group's survivor)
	displaced := make([]primitive.ObjectID, 0)

	for _, group := range groups {
		displaced = append(displaced, group.displacedAnnotationIDs()...)
	}

	if len(displaced) == 0 {
		return nil
	}

	// Remove them in a single round-trip
	if _, err := collection.DeleteMany(ctx, bson.M{"_id": bson.M{"$in": displaced}}); err != nil {
		return derp.Wrap(err, location, "Deleting duplicate Annotation records")
	}

	log.Info().
		Str("database", collection.Database().Name()).
		Int("count", len(displaced)).
		Msg("Removed duplicate Annotation records")

	return nil
}

// displacedAnnotationIDs returns the ID of every Annotation in this group except the survivor.
func (group annotationGroup) displacedAnnotationIDs() []primitive.ObjectID {

	survivor := survivingAnnotationID(group.Annotations)

	result := make([]primitive.ObjectID, 0, len(group.Annotations)-1)

	for _, record := range group.Annotations {
		if record.AnnotationID != survivor {
			result = append(result, record.AnnotationID)
		}
	}

	return result
}

// survivingAnnotationID returns the ID of the record with the longest note in a non-empty group.
// Ties keep the earliest-created record.
func survivingAnnotationID(records []annotationRecord) primitive.ObjectID {

	survivor := records[0]

	for _, record := range records[1:] {

		if record.ContentLength > survivor.ContentLength {
			survivor = record
			continue
		}

		if record.ContentLength == survivor.ContentLength && record.CreateDate < survivor.CreateDate {
			survivor = record
		}
	}

	return survivor.AnnotationID
}
