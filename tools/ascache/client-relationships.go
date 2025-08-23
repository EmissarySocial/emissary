package ascache

import (
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

// CalcParentRelationships counts the number of records linked to a URL via a single relationType
func (client *Client) CalcParentRelationships(session data.Session, relationType string, relationHref string) error {

	const location = "ascache.client.CalcParentRelationships"

	// RULE: If there is no relationship, then exit now
	if relationType == "" {
		return nil
	}

	// RULE: If there is no relationship, then exit now
	if relationHref == "" {
		return nil
	}

	// Try to load the "parent" item from the cache.
	parentValue := NewValue()
	if err := client.loadByURL(session, relationHref, &parentValue); err != nil {

		// If the "parent" doesn't exist in the cache, then try to load it... later.
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

	// Count the current number of related relationships
	count, err := client.CountRelatedValues(session, relationType, relationHref)

	if err != nil {
		return derp.Wrap(err, location, "Error counting related values", relationHref, relationType)
	}

	// If the count has changed, then update the parent with the new count
	if changed := parentValue.Metadata.SetRelationCount(relationType, count); changed {
		if err := client.save(session.Context(), parentValue.DocumentID(), &parentValue); err != nil {
			return derp.Wrap(err, location, "Unable to save updated ActivityStream document", relationHref)
		}
	}

	// success.
	return nil
}

func (client *Client) CountRelatedValues(session data.Session, relationType string, relationHref string) (int64, error) {

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
