package ascache

import (
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

// CalcStatistics updates the statistics for any cached value that
// the provided document is related to (via Announce, Reply, Like, Dislike)
func (client *Client) calcStatistics(collection data.Collection, document streams.Document) error {

	// Guarantee that we have a usable document (and not just an ID)
	document = document.LoadLink()
	documentType := document.Type()

	// First, inspect the document type to see if it's a relationship we can use
	switch documentType {

	case
		vocab.ActivityTypeLike,
		vocab.ActivityTypeDislike,
		vocab.ActivityTypeAnnounce:
		return client.calcStatistics_inner(collection, document.Object().ID(), documentType)

	case
		vocab.ActivityTypeUndo,
		vocab.ActivityTypeDelete:
		return client.calcStatistics(collection, document.Object())
	}

	// Fall through.. see if this document is a reply to another document
	unwrappedDocument := document.UnwrapActivity()

	if inReplyTo := unwrappedDocument.InReplyTo(); inReplyTo.NotNil() {
		return client.calcStatistics_inner(collection, inReplyTo.ID(), RelationTypeReply)
	}

	// Otherwise, this is not a related document and we don't have to update any statistics
	return nil
}

// calcStatistics_inner does most of the actual work for calculateStats.
// This method counts the values in the cache with the correct relationType and relationHref,
// then updates the corresponding document with the newly calculated value.
func (client *Client) calcStatistics_inner(collection data.Collection, url string, relationType string) error {

	const location = "ascache.Client.calcStatistics_inner"

	// Count all documents in the cache with the same relation type/href
	countCriteria := exp.Equal("metadata.relationType", relationType).
		AndEqual("metadata.relationHref", url)

	count, err := collection.Count(countCriteria)

	if err != nil {
		return derp.Wrap(err, location, "Error counting documents", url, relationType)
	}

	// Load the value from the database
	value := NewValue()

	if err := client.loadByURLs(collection, url, &value); err != nil {
		return derp.Wrap(err, location, "Unable to load document", url)
	}

	// Update the metadata with the new count
	value.Metadata[relationType] = count

	// Save the cached value back into the database
	if err := collection.Save(&value, "Updated statistics"); err != nil {
		return derp.Wrap(err, location, "Unable to save cached value", url)
	}

	// The update was successful
	return nil
}
