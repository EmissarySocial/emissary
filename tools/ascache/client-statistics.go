package ascache

import (
	"context"

	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"go.mongodb.org/mongo-driver/bson"
)

// CalcStatistics updates the statistics for any cached value that
// the provided document is related to (via Announce, Reply, Like, Dislike)
func (client *Client) calcStatistics(document streams.Document) error {

	// Guarantee that we have a usable document (and not just an ID)
	document, err := document.Load()

	if err != nil {
		return derp.Wrap(err, "ascache.Client.calculateStats", "Error loading document", document)
	}

	// First, inspect the document type to see if it's a relationship we can use
	switch document.Type() {

	case vocab.ActivityTypeLike:
		return client.calcStatistics_inner(document.Object().ID(), RelationTypeLike)

	case vocab.ActivityTypeDislike:
		return client.calcStatistics_inner(document.Object().ID(), RelationTypeDislike)

	case vocab.ActivityTypeAnnounce:
		return client.calcStatistics_inner(document.Object().ID(), RelationTypeAnnounce)

	case vocab.ActivityTypeUndo, vocab.ActivityTypeDelete:
		return client.calcStatistics(document.Object())
	}

	// Fall through.. see if this document is a reply to another document
	unwrappedDocument := document.UnwrapActivity()

	if inReplyTo := unwrappedDocument.InReplyTo(); inReplyTo.NotNil() {
		return client.calcStatistics_inner(inReplyTo.ID(), RelationTypeReply)
	}

	// Otherwise, this is not a related document and we don't have to update any statistics
	return nil
}

// calcStatistics_inner does most of the actual work for calculateStats.
// This method counts the values in the cache with the correct relationType and relationHref,
// then updates the corresponding document with the newly calculated value.
func (client *Client) calcStatistics_inner(objectID string, relationType string) error {

	const location = "ascache.Client.calcStatistics_inner"

	var fieldName string

	switch relationType {

	case RelationTypeAnnounce:
		fieldName = "statistics.announces"

	case RelationTypeReply:
		fieldName = "statistics.replies"

	case RelationTypeLike:
		fieldName = "statistics.likes"

	case RelationTypeDislike:
		fieldName = "statistics.dislikes"

	default:
		return derp.NewInternalError(location, "Invalid relationType", relationType)
	}

	// Count all documents in the cache with the same relation type/href
	count, err := client.collection.CountDocuments(
		context.Background(), // context
		bson.M{"relationType": relationType, "relationHref": objectID}, // filter
	)

	if err != nil {
		return derp.Wrap(err, location, "Error counting documents", objectID, relationType, fieldName)
	}

	// Set likes in document statistics
	_, err = client.collection.UpdateOne(
		context.Background(),
		bson.M{"uri": objectID}, // filter
		bson.M{"$set": bson.M{fieldName: count}},
	)

	if err != nil {
		return derp.Wrap(err, location, "Error setting likes", objectID)
	}

	// The update was successful
	return nil
}
