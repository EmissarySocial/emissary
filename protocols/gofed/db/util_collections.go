package db

import (
	"context"
	"net/url"

	"github.com/EmissarySocial/emissary/protocols/gofed/common"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	builder "github.com/benpate/exp-builder"
	"github.com/benpate/rosetta/convert"
	"github.com/go-fed/activity/streams"
	"github.com/go-fed/activity/streams/vocab"
)

// getCollection returns the URLs for all items in a collection
func (db *Database) getCollection(modelType string, collectionURL *url.URL, criteria exp.Expression) (vocab.ActivityStreamsCollection, error) {

	links, err := db.queryAllURLs(modelType, collectionURL, criteria)

	if err != nil {
		return nil, derp.Wrap(err, "activitypub.Database.getOrderedCollection", "Error querying all IRIs by URL", collectionURL)
	}

	// Build the response (OMG)
	result := streams.NewActivityStreamsCollection()
	items := result.GetActivityStreamsItems()

	for _, link := range links {
		linkURL, _ := url.Parse(link)
		items.AppendIRI(linkURL)
	}

	return result, nil
}

// getOrderedCollectionPage returns a page of complete items from a collection.
// It returns the top 60 items, and queries on publishDate
func (db *Database) getOrderedCollectionPage(ctx context.Context, collectionURL *url.URL, requireItemType string) (vocab.ActivityStreamsOrderedCollectionPage, error) {

	const location = "service.activitypub.Database.getOrderedCollectionPage"

	// Parse the collection URL
	userID, itemType, _, err := common.ParseURL(collectionURL)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error parsing URL", collectionURL)
	}

	// If set, only allow one item type for this query
	if (requireItemType != "") && (itemType != requireItemType) {
		return nil, derp.NewBadRequestError(location, "Wrong item type", itemType, requireItemType)
	}

	// Get the service for this kind of item
	modelService, err := db.factory.Model(itemType)

	if err != nil {
		return nil, derp.Wrap(err, location, "No service found for item type", itemType)
	}

	// Build the query criteria.
	b := builder.NewBuilder().
		Int("publishDate")

	criteria := b.Evaluate(collectionURL.Query()).
		AndEqual("userId", userID).
		AndEqual("journal.DeleteDate", 0)

	// Query the database for the items
	iterator, err := modelService.ObjectList(criteria, option.MaxRows(60), option.SortDesc("publishDate"))

	if err != nil {
		return nil, derp.Wrap(err, location, "Error getting iterator", collectionURL)
	}

	// Build the response (OMG)
	result := streams.NewActivityStreamsOrderedCollectionPage()
	items := result.GetActivityStreamsOrderedItems()

	object := modelService.ObjectNew()

	var firstPublishDate int64
	var lastPublishDate int64

	for iterator.Next(object) {

		// Store first and last publish dates in the collection
		// so we can make prev/next links after the loop
		lastPublishDate = common.GetPublishDate(object)

		if firstPublishDate == 0 {
			firstPublishDate = lastPublishDate
		}

		// Convert the model object to an ActivityStreams object
		item, err := common.ToActivityStream(object, itemType)

		if err != nil {
			return nil, derp.Wrap(err, location, "Error converting object to ActivityStreams object", object)
		}

		// Add the item to the collection
		if err := items.AppendType(item); err != nil {
			return nil, derp.Wrap(err, location, "Error appending item to collection", item)
		}

		object = modelService.ObjectNew()
	}

	// Build "prev page" URL
	prevPageURL := collectionURL
	prevQuery := make(url.Values)
	prevQuery.Set("publishDate", "GT:"+convert.String(firstPublishDate))
	prevPageURL.RawQuery = prevQuery.Encode()

	prevPageProperty := streams.NewActivityStreamsPrevProperty()
	prevPageProperty.SetIRI(prevPageURL)
	result.SetActivityStreamsPrev(prevPageProperty)

	// Build "next page" URL
	nextPageURL := collectionURL
	nextQuery := make(url.Values)
	nextQuery.Set("publishDate", "LT:"+convert.String(lastPublishDate))
	nextPageURL.RawQuery = nextQuery.Encode()

	nextPageProperty := streams.NewActivityStreamsPrevProperty()
	nextPageProperty.SetIRI(nextPageURL)
	result.SetActivityStreamsPrev(nextPageProperty)

	// TODO: LOW: What else should we set on the collection?

	// Success?!?
	return result, nil
}
