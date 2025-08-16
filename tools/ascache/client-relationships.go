package ascache

import (
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

// CalcAllRelationships counts all documents that are related to the current URL
func (client *Client) CalcAllRelationships(session data.Session, value *Value) error {

	const location = "ascache.client.CalcAllRelationships"

	var err error

	documentID := value.DocumentID()

	// Count Replies
	value.Metadata.Replies, err = client.countRelatedValues(session, RelationTypeReply, documentID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to count replies", documentID)
	}

	// Count Announces
	value.Metadata.Announces, err = client.countRelatedValues(session, RelationTypeAnnounce, documentID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to count announces", documentID)
	}

	// Count Likes
	value.Metadata.Likes, err = client.countRelatedValues(session, RelationTypeLike, documentID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to count likes", documentID)
	}

	// Count Dislikes
	value.Metadata.Dislikes, err = client.countRelatedValues(session, RelationTypeDislike, documentID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to count dislikes", documentID)
	}

	// Done.
	return nil
}

// CalcRelationships counts the number of records linked to a URL via a single relationType
func (client *Client) CalcRelationships(session data.Session, relationType string, relationHref string) error {

	const location = "ascache.client.CalcRelationships"

	// Count the current number of related relationships
	count, err := client.countRelatedValues(session, relationType, relationHref)

	if err != nil {
		return derp.Wrap(err, location, "Error counting related values", relationHref, relationType)
	}

	value := NewValue()
	if err := client.loadByURL(session, relationHref, &value); err != nil {

		// If the cached value isn't found, then try to load it... later.
		if derp.IsNotFound(err) {
			client.enqueue <- queue.NewTask(
				"CrawlActivityStreams",
				mapof.Any{
					"host":      client.hostname,
					"actorType": client.actorType,
					"actorID":   client.actorID,
					"url":       relationHref,
				},
				queue.WithPriority(64),
				queue.WithSignature(relationHref),
			)
			return nil
		}

		// All other errors break the request.
		return derp.Wrap(err, location, "Unable to load cached ActivityStream document", relationHref)
	}

	value.Metadata.SetRelationCount(relationType, count)

	if err := client.save(session, value.DocumentID(), &value); err != nil {
		return derp.Wrap(err, location, "Unable to save updated ActivityStream document", relationHref)
	}

	return nil
}

func (client *Client) countRelatedValues(session data.Session, relationType string, relationHref string) (int64, error) {

	const location = "ascache.client.countRelatedValues"

	// RULE: If the relationType is empty, then there is nothing to calculate
	if relationType == "" {
		return 0, nil
	}

	// RULE: If the relationHref is empty, then there is nothing to calculate
	if relationHref == "" {
		return 0, nil
	}

	// Count all documents in the cache with the same relation type/href
	collection := client.collection(session)

	count, err := collection.Count(
		exp.Equal("metadata.relationType", relationType).
			AndEqual("metadata.relationHref", relationHref),
	)

	if err != nil {
		return 0, derp.Wrap(err, location, "Unable to count related documents", relationHref, relationType)
	}

	return count, nil
}

// calcMetadata updates the statistics for any cached value that
// the provided document is related to (via Announce, Reply, Like, Dislike)
func (client *Client) refreshCounts(session data.Session, document streams.Document) error {

	document = document.LoadLink()
	documentType := document.Type()

	switch documentType {

	// If this is a Delete or Undo, then refreshCounts on the activity being deleted/undone
	case
		vocab.ActivityTypeUndo,
		vocab.ActivityTypeDelete:

		return client.refreshCounts(session, document.Object())

	// If this is a "Reaction" then CalcRelationships on the object being reacted to
	case
		vocab.ActivityTypeLike,
		vocab.ActivityTypeDislike,
		vocab.ActivityTypeAnnounce:

		return client.CalcRelationships(session, documentType, document.Object().ID())
	}

	// Otherwise, if this is a "Reply" to another document, then CalcRelationships on the parent
	unwrappedDocument := document.UnwrapActivity()

	if inReplyTo := unwrappedDocument.InReplyTo(); inReplyTo.NotNil() {
		return client.CalcRelationships(session, RelationTypeReply, inReplyTo.ID())
	}

	return nil
}
