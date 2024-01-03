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
	"github.com/benpate/rosetta/slice"
)

// ActivityStreams implements the Hannibal HTTP client interface, and provides a cache for ActivityStreams documents.
type ActivityStreams struct {
	collection  data.Collection
	innerClient *ascache.Client
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// NewActivityStreams creates a new ActivityStreams service
func NewActivityStreams() ActivityStreams {
	return ActivityStreams{}
}

// Refresh updates the ActivityStreams service with new dependencies
func (service *ActivityStreams) Refresh(innerClient *ascache.Client, collection data.Collection) {
	service.innerClient = innerClient
	service.collection = collection
}

/******************************************
 * Hannibal HTTP Client Interface
 ******************************************/

// Load implements the Hannibal `Client` interface, and returns a streams.Document from the cache.
func (service *ActivityStreams) Load(uri string, options ...any) (streams.Document, error) {

	const location = "service.ActivityStreams.Load"

	if uri == "" {
		return streams.NilDocument(), derp.NewNotFoundError(location, "Empty URL", uri)
	}

	// NPE Check
	if service.innerClient == nil {
		return streams.Document{}, derp.NewInternalError(location, "Client not initialized")
	}

	// Forward request to inner client
	result, err := service.innerClient.Load(uri, options...)

	if err != nil {
		return streams.NilDocument(), derp.Wrap(err, location, "Error loading document from inner client", uri)
	}

	result.WithOptions(streams.WithClient(service))
	return result, nil
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
func (service *ActivityStreams) queryByRelation(relationType string, relationHref string, cutType string, cutDate int64, maxRows int) ([]streams.Document, error) {

	const location = "service.ActivityStreams.QueryRelated"

	// NPE Check
	if service.collection == nil {
		return nil, derp.NewInternalError(location, "Document Collection not initialized")
	}

	// Build the query
	criteria := exp.
		Equal("relationType", relationType).
		AndEqual("relationHref", relationHref)

	var sortOption option.Option

	if cutType == "before" {
		criteria = criteria.AndLessThan("published", cutDate)
		sortOption = option.SortDesc("published")
	} else {
		criteria = criteria.AndGreaterThan("published", cutDate)
		sortOption = option.SortAsc("published")
	}

	documents, err := service.documentQuery(criteria, sortOption, option.MaxRows(int64(maxRows)))

	if err != nil {
		return nil, derp.Wrap(err, location, "Error querying database")
	}

	// Return the results as a streams.Document / collection

	if cutType == "before" {
		documents = slice.Reverse(documents)
	}

	result := slice.Map(documents, func(document ascache.CachedValue) streams.Document {
		return streams.NewDocument(document.Object, streams.WithStats(document.Statistics), streams.WithClient(service))
	})

	return result, nil
}

func (service *ActivityStreams) SearchActors(queryString string) ([]model.ActorSummary, error) {
	return queries.SearchActivityStreamActors(context.TODO(), service.collection, queryString)
}

// QueryRepliesBeforeDate returns a slice of streams.Document values that are replies to the specified document, and were published before the specified date.
func (service *ActivityStreams) QueryRepliesBeforeDate(inReplyTo string, maxDate int64, maxRows int) ([]streams.Document, error) {
	return service.queryByRelation("Reply", inReplyTo, "before", maxDate, maxRows)
}

// QueryRepliesAfterDate returns a slice of streams.Document values that are replies to the specified document, and were published after the specified date.
func (service *ActivityStreams) QueryRepliesAfterDate(inReplyTo string, minDate int64, maxRows int) ([]streams.Document, error) {
	return service.queryByRelation("Reply", inReplyTo, "after", minDate, maxRows)
}

func (service *ActivityStreams) QueryAnnouncesBeforeDate(relationHref string, maxDate int64, maxRows int) ([]streams.Document, error) {
	return service.queryByRelation("Announce", relationHref, "before", maxDate, maxRows)
}

func (service *ActivityStreams) QueryLikesBeforeDate(relationHref string, maxDate int64, maxRows int) ([]streams.Document, error) {
	return service.queryByRelation("Like", relationHref, "before", maxDate, maxRows)
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

// query reads from the database and returns a slice of streams.Document values
func (service *ActivityStreams) documentQuery(criteria exp.Expression, options ...option.Option) ([]ascache.CachedValue, error) {

	const location = "service.ActivityStreams.documentQuery"

	// NPE Check
	if service.collection == nil {
		return nil, derp.NewInternalError(location, "Document Collection not initialized")
	}

	// Query the database
	result := make([]ascache.CachedValue, 0)
	if err := service.collection.Query(&result, criteria, options...); err != nil {
		return nil, derp.Wrap(err, location, "Error querying database")
	}

	// Return success
	return result, nil
}
