package activitypub

import (
	"math"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/list"
)

func Collection(collectionID string) streams.OrderedCollection {

	// Determine the first page URL
	firstPageURL := list.First(collectionID, '?') + "?publishDate=" + convert.String(math.MaxInt64)

	// Generate a new Collection stub
	result := streams.NewOrderedCollection(collectionID)
	result.First = firstPageURL

	return result
}

func CollectionPage[T model.JSONLDGetter](pageID string, partOf string, pageSize int, values []T) streams.OrderedCollectionPage {

	// Generate the Page record
	result := streams.NewOrderedCollectionPage(pageID, partOf)

	if len(values) == 0 {
		return result
	}

	for _, value := range values {
		result.OrderedItems = append(result.OrderedItems, value.GetJSONLD())
	}

	if len(values) == pageSize {
		lastValue := values[pageSize-1]
		result.Next = partOf + "?publishDate=" + convert.String(lastValue.Created())
	}

	return result
}

func CollectionPage_Links[T model.ActivityPubURLGetter](pageID string, partOf string, pageSize int, values []T) streams.OrderedCollectionPage {

	// Generate the Page record
	result := streams.NewOrderedCollectionPage(pageID, partOf)

	if len(values) > 0 {
		for _, value := range values {
			result.OrderedItems = append(result.OrderedItems, value.ActivityPubURL())
		}

		if len(values) == pageSize {
			lastValue := values[pageSize-1]
			result.Next = partOf + "?publishDate=" + convert.String(lastValue.Created())
		}
	}

	return result
}
