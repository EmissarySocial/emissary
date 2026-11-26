package queries

import (
	"context"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MaxRankResult defines the results of the custom MaxRank query
type MaxRankResult struct {
	MaxRank int `bson:"maxRank"`
}

// MaxRank returns the rank for a new child of the parent stream: one more than its highest-ranked
// living child, or 1 when it has none
func MaxRank(ctx context.Context, collection data.Collection, parentID primitive.ObjectID) (int, error) {

	// Set up the mongodb pipeline query and result
	query := bson.A{
		bson.M{"$match": bson.M{"parentId": parentID, "deleteDate": 0}},
		bson.M{"$group": bson.M{"_id": nil, "maxRank": bson.M{"$max": "$rank"}}},
	}

	result := []MaxRankResult{}

	// Try to execute the query as a mongodb pipeline
	if err := pipeline(ctx, collection, &result, query); err != nil {
		return 0, derp.Wrap(err, "queries.CountRecords", "Counting records")
	}

	// RULE: Ranks start at 1, because 0 means "not set" (see build.StepSort), so the first
	// child of an empty parent is ranked 1
	if len(result) == 0 {
		return 1, nil
	}

	// Otherwise, rank the new child one after its highest-ranked sibling
	return result[0].MaxRank + 1, nil
}

// SetAttributedTo refreshes the denormalized author record on every document a User has written
func SetAttributedTo(ctx context.Context, collection data.Collection, personLink model.PersonLink) error {

	criteria := exp.Equal("attributedTo.userId", personLink.UserID)

	update := bson.M{
		"$set": bson.M{
			"attributedTo": personLink,
		},
	}

	return RawUpdate(ctx, collection, criteria, update)
}

// ClaimStreamFingerprint moves a Stream's outbox fingerprint from one value to another, and
// returns FALSE when the stored fingerprint is not the one named, so the database decides which
// of two senders wins.  An empty fingerprint means "followers hold nothing".
func ClaimStreamFingerprint(session data.Session, streamID primitive.ObjectID, from string, to string) (bool, error) {

	const location = "queries.ClaimStreamFingerprint"

	// Guarantee that we're using MongoDB
	collection := mongoCollection(session.Collection("Stream"))

	if collection == nil {
		return false, derp.Internal(location, "Collection must be a MongoDB collection")
	}

	// RULE: No `deleteDate` in the filter, because the Delete for a deleted Stream is claimed too
	filter := bson.M{"_id": streamID, "fingerprint": from}

	// An empty fingerprint is stored as a missing property, and older records may hold "" or null
	if from == "" {
		filter["fingerprint"] = bson.M{"$in": bson.A{nil, ""}}
	}

	// RULE: One field, never a whole-record save, which would overwrite an edit made meanwhile
	update := bson.M{"$set": bson.M{"fingerprint": to}}

	if to == "" {
		update = bson.M{"$unset": bson.M{"fingerprint": ""}}
	}

	result, err := collection.UpdateOne(session.Context(), filter, update)

	if err != nil {
		return false, derp.Wrap(err, location, "Moving outbox fingerprint", streamID)
	}

	// A missing result claims nothing, so nothing is sent
	if result == nil {
		return false, nil
	}

	return result.MatchedCount == 1, nil
}
