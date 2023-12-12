package service

import (
	"time"

	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
)

// ActivityStreams implements the Hannibal HTTP client interface, and provides a cache for ActivityStreams documents.
type ActivityStreams struct {
	documentCollection data.Collection
	innerClient        *ascache.Client
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// NewActivityStreams creates a new ActivityStreams service
func NewActivityStreams() ActivityStreams {
	return ActivityStreams{}
}

// Refresh updates the ActivityStreams service with new dependencies
func (service *ActivityStreams) Refresh(innerClient *ascache.Client, documentCollection data.Collection) {
	service.innerClient = innerClient
	service.documentCollection = documentCollection
}

/******************************************
 * Hannibal HTTP Client Interface
 ******************************************/

// Load implements the Hannibal `Client` interface, and returns a streams.Document from the cache.
func (service *ActivityStreams) Load(uri string, options ...any) (streams.Document, error) {

	// NPE Check
	if service.innerClient == nil {
		return streams.Document{}, derp.NewInternalError("service.ActivityStreams.Load", "Client not initialized")
	}

	// Forward request to inner client
	return service.innerClient.Load(uri, options...)
}

// Delete removes a single document from the database by its URL
func (service *ActivityStreams) Delete(uri string) error {

	const location = "service.ActivityStreams.Delete"

	if err := service.innerClient.Delete(uri); err != nil {
		return derp.Wrap(err, location, "Error deleting document from cache", uri)
	}

	return nil
}

/******************************************
 * Custom Behaviors
 ******************************************/

// PurgeCache removes all expired documents from the cache
func (service *ActivityStreams) PurgeCache() error {

	// NPE Check
	if service.documentCollection == nil {
		return derp.NewInternalError("service.ActivityStreams.PurgeCache", "Document Collection not initialized")
	}

	// Purge all expired Documents
	criteria := exp.LessThan("expires", time.Now().Unix())
	if err := service.documentCollection.HardDelete(criteria); err != nil {
		return derp.Wrap(err, "service.ActivityStreams.PurgeCache", "Error purging documents")
	}

	return nil
}

/******************************************
 * Custom Query Methods
 ******************************************/

// QueryRepliesBeforeDate returns a slice of streams.Document values that are replies to the specified document, and were published before the specified date.
func (service *ActivityStreams) queryByRelation(relationType string, relationHref string, cutType string, cutDate int64, maxRows int) (streams.Document, error) {

	const location = "service.ActivityStreams.QueryRelated"

	// NPE Check
	if service.documentCollection == nil {
		return streams.Document{}, derp.NewInternalError(location, "Document Collection not initialized")
	}

	// Build the query
	criteria := exp.
		Equal("relationType", relationType).
		AndEqual("relationHref", relationHref)

	if cutType == "before" {
		criteria = criteria.AndLessThan("published", cutDate)
	} else {
		criteria = criteria.AndGreaterThan("published", cutDate)
	}

	results, err := service.documentQuery(criteria, option.SortDesc("published"), option.MaxRows(int64(maxRows)))

	if err != nil {
		return streams.Document{}, derp.Wrap(err, location, "Error querying database")
	}

	// Return the results as a streams.Document / collection

	if cutType == "before" {
		results = results.Reverse()
	}

	return streams.NewDocument(results, streams.WithClient(service)), nil
}

// QueryRepliesBeforeDate returns a slice of streams.Document values that are replies to the specified document, and were published before the specified date.
func (service *ActivityStreams) QueryRepliesBeforeDate(inReplyTo string, maxDate int64, maxRows int) (streams.Document, error) {
	return service.queryByRelation("Reply", inReplyTo, "before", maxDate, maxRows)
}

// QueryRepliesAfterDate returns a slice of streams.Document values that are replies to the specified document, and were published after the specified date.
func (service *ActivityStreams) QueryRepliesAfterDate(inReplyTo string, minDate int64, maxRows int) (streams.Document, error) {
	return service.queryByRelation("Reply", inReplyTo, "after", minDate, maxRows)
}

func (service *ActivityStreams) QueryAnnouncesBeforeDate(relationHref string, maxDate int64, maxRows int) (streams.Document, error) {
	return service.queryByRelation("Announce", relationHref, "before", maxDate, maxRows)
}

func (service *ActivityStreams) QueryLikesBeforeDate(relationHref string, maxDate int64, maxRows int) (streams.Document, error) {
	return service.queryByRelation("Like", relationHref, "before", maxDate, maxRows)
}

/******************************************
 * Internal Methods
 ******************************************/

// iterator reads from the database and returns a data.Iterator with the result values.
func (service *ActivityStreams) documentIterator(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {

	const location = "service.ActivityStreams.documentIterator"

	// NPE Check
	if service.documentCollection == nil {
		return nil, derp.NewInternalError(location, "Document Collection not initialized")
	}

	// Forward request to documentCollection
	return service.documentCollection.Iterator(criteria, options...)
}

// query reads from the database and returns a slice of streams.Document values
func (service *ActivityStreams) documentQuery(criteria exp.Expression, options ...option.Option) (sliceof.Object[mapof.Any], error) {

	const location = "service.ActivityStreams.documentQuery"

	// NPE Check
	if service.documentCollection == nil {
		return nil, derp.NewInternalError(location, "Document Collection not initialized")
	}

	// Create the Iterator
	iterator, err := service.documentIterator(criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error querying database")
	}

	// Initialize result slice
	result := make(sliceof.Object[mapof.Any], 0, iterator.Count())

	// Map iterator into results
	value := ascache.NewCachedValue()
	for iterator.Next(&value) {
		result = append(result, value.Original)
		value = ascache.NewCachedValue()

		if err := iterator.Error(); err != nil {
			return nil, derp.Wrap(err, location, "Error during iteration")
		}
	}

	// Return success
	return result, nil
}
