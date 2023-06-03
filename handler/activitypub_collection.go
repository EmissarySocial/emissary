package handler

import (
	"math"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/list"
)

func activityPub_Collection(collectionURL string) streams.OrderedCollection {

	collectionURL = list.First(collectionURL, '?')

	// Determine the collection URL
	// Determine the first page URL
	firstPageURL := collectionURL + "?publishDate=" + convert.String(math.MaxInt64)

	// Generate a new Collection stub
	result := streams.NewOrderedCollection()
	result.ID = collectionURL
	result.First = firstPageURL

	return result
}

func activityPub_CollectionPage[T model.JSONLDGetter](partOf string, pageSize int, values []T) streams.OrderedCollectionPage {

	// Generate the Page record
	result := streams.NewOrderedCollectionPage()
	result.PartOf = partOf

	if len(values) > 0 {
		for _, value := range values {
			result.OrderedItems = append(result.OrderedItems, value.GetJSONLD())
		}

		if len(values) == pageSize {
			lastValue := values[pageSize-1]
			result.Next = partOf + "?publishDate=" + convert.String(lastValue.Created())
		}
	}

	return result
}
