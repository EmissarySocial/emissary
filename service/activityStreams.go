package service

import (
	"context"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/sherlock"
)

// ActivityStreams implements the Hannibal HTTP client interface, and provides a cache for ActivityStreams documents.
type ActivityStreams struct {
	collection  data.Collection
	innerClient streams.Client
	cacheClient *ascache.Client
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// NewActivityStreams creates a new ActivityStreams service
func NewActivityStreams() ActivityStreams {
	return ActivityStreams{}
}

// Refresh updates the ActivityStreams service with new dependencies
func (service *ActivityStreams) Refresh(innerClient streams.Client, cacheClient *ascache.Client, collection data.Collection) {
	service.innerClient = innerClient
	service.cacheClient = cacheClient
	service.collection = collection
}

/******************************************
 * Hannibal HTTP Client Interface
 ******************************************/

// Load implements the Hannibal `Client` interface, and returns a streams.Document from the cache.
func (service *ActivityStreams) Load(url string, options ...any) (streams.Document, error) {

	const location = "service.ActivityStreams.Load"

	if url == "" {
		return streams.NilDocument(), derp.NewNotFoundError(location, "Empty URL", url)
	}

	// NPE Check
	if service.innerClient == nil {
		return streams.Document{}, derp.NewInternalError(location, "Client not initialized")
	}

	// Forward request to inner client
	result, err := service.innerClient.Load(url, options...)

	if err != nil {
		return streams.NilDocument(), derp.Wrap(err, location, "Error loading document from inner client", url)
	}

	result.WithOptions(streams.WithClient(service))
	return result, nil
}

// Put adds a single document to the ActivityStream cache
func (service *ActivityStreams) Put(document streams.Document) {
	service.cacheClient.Put(document)
}

// Delete removes a single document from the database by its URL
func (service *ActivityStreams) Delete(url string) error {

	const location = "service.ActivityStreams.Delete"

	if err := service.cacheClient.Delete(url); err != nil {
		return derp.Wrap(err, location, "Error deleting document from cache", url)
	}

	return nil
}

/******************************************
 * Custom Behaviors
 ******************************************/

// PurgeCache removes all expired documents from the cache
func (service *ActivityStreams) PurgeCache() error {

	// NPE Check
	if service.collection == nil {
		return derp.NewInternalError("service.ActivityStreams.PurgeCache", "Document Collection not initialized")
	}

	// Purge all expired Documents
	criteria := exp.LessThan("expires", time.Now().Unix())
	if err := service.collection.HardDelete(criteria); err != nil {
		return derp.Wrap(err, "service.ActivityStreams.PurgeCache", "Error purging documents")
	}

	return nil
}

/******************************************
 * Custom Query Methods
 ******************************************/

// QueryRepliesBeforeDate returns a slice of streams.Document values that are replies to the specified document, and were published before the specified date.
func (service *ActivityStreams) queryByRelation(relationType string, relationHref string, cutType string, cutDate int64, done <-chan struct{}) <-chan streams.Document {

	const location = "service.ActivityStreams.QueryRelated"

	result := make(chan streams.Document)

	go func() {

		defer close(result)

		// NPE Check
		if service.collection == nil {
			derp.Report(derp.NewInternalError(location, "Document Collection not initialized"))
			return
		}

		// Build the query
		criteria := exp.
			Equal("metadata.relationType", relationType).
			AndEqual("metadata.relationHref", relationHref)

		var sortOption option.Option

		if cutType == "before" {
			criteria = criteria.AndLessThan("published", cutDate)
			sortOption = option.SortDesc("published")
		} else {
			criteria = criteria.AndGreaterThan("published", cutDate)
			sortOption = option.SortAsc("published")
		}

		// Try to query the database
		documents, err := service.documentIterator(criteria, sortOption)

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Error querying database"))
			return
		}

		defer documents.Close()

		// Write documents into the result channel until done (or done)
		value := ascache.NewValue()
		for documents.Next(&value) {

			select {
			case <-done:
				return

			default:
				result <- streams.NewDocument(
					value.Object,
					streams.WithHTTPHeader(value.HTTPHeader),
					streams.WithStats(value.Statistics),
					streams.WithClient(service),
				)
			}

			value = ascache.NewValue()
		}

		// Return the results as a streams.Document / collection
		// if cutType == "before" {
		//	documents = slice.Reverse(documents)
		// }
	}()

	return result

}

func (service *ActivityStreams) SearchActors(queryString string) ([]model.ActorSummary, error) {

	const location = "service.ActivityStreams.SearchActors"

	// If we think this is an address we can work with (because sherlock says so)
	// the try to retrieve it directly.
	if sherlock.IsValidAddress(queryString) {

		// Try to load the actor directly from the Interwebs
		if newActor, err := service.Load(queryString, sherlock.AsActor()); err == nil {

			// If this is a valid, but (previously) unknown actor, then add it to the results
			// This will also automatically get cached/crawled for next time.
			result := []model.ActorSummary{{
				ID:       newActor.ID(),
				Type:     newActor.Type(),
				Name:     newActor.Name(),
				Icon:     newActor.Icon().Href(),
				Username: newActor.PreferredUsername(),
			}}

			return result, nil
		}
	}

	// Fall through means that we can't find a perfect match, so fall back to a full-text search
	result, err := queries.SearchActivityStreamActors(context.TODO(), service.collection, queryString)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error querying database")
	}

	return result, nil
}

// QueryRepliesBeforeDate returns a slice of streams.Document values that are replies to the specified document, and were published before the specified date.
func (service *ActivityStreams) QueryRepliesBeforeDate(inReplyTo string, maxDate int64, done <-chan struct{}) <-chan streams.Document {
	return service.queryByRelation("Reply", inReplyTo, "before", maxDate, done)
}

// QueryRepliesAfterDate returns a slice of streams.Document values that are replies to the specified document, and were published after the specified date.
func (service *ActivityStreams) QueryRepliesAfterDate(inReplyTo string, minDate int64, done <-chan struct{}) <-chan streams.Document {
	return service.queryByRelation("Reply", inReplyTo, "after", minDate, done)
}

func (service *ActivityStreams) QueryAnnouncesBeforeDate(relationHref string, maxDate int64, done <-chan struct{}) <-chan streams.Document {
	return service.queryByRelation(vocab.ActivityTypeAnnounce, relationHref, "before", maxDate, done)
}

func (service *ActivityStreams) QueryLikesBeforeDate(relationHref string, maxDate int64, done <-chan struct{}) <-chan streams.Document {
	return service.queryByRelation(vocab.ActivityTypeLike, relationHref, "before", maxDate, done)
}

/******************************************
 * Internal Methods
 ******************************************/

// iterator reads from the database and returns a data.Iterator with the result values.
func (service *ActivityStreams) documentIterator(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {

	const location = "service.ActivityStreams.documentIterator"

	// NPE Check
	if service.collection == nil {
		return nil, derp.NewInternalError(location, "Document Collection not initialized")
	}

	// Forward request to collection
	return service.collection.Iterator(criteria, options...)
}
