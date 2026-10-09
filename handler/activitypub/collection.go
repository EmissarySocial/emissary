package activitypub

import (
	"math"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/list"
)

// Collection returns an OrderedCollection stub that points at the first page of the named collection
func Collection(collectionID string) streams.OrderedCollection {

	// Determine the first page URL
	firstPageURL := list.First(collectionID, '?') + "?publishDate=" + convert.String(math.MaxInt64)

	// Generate a new Collection stub
	result := streams.NewOrderedCollection(collectionID)
	result.First = firstPageURL

	return result
}

// CollectionPage returns one page of an OrderedCollection, rendering each value as a JSON-LD document
func CollectionPage[T model.JSONLDGetter](pageID string, partOf string, pageSize int, values []T) streams.OrderedCollectionPage {

	// Generate the Page record
	result := streams.NewOrderedCollectionPage(pageID, partOf)

	if len(values) == 0 {
		return result
	}

	// RULE: Omit any item that cannot identify itself.  An item with no `id` can never be
	// dereferenced, deduplicated, or referenced back, so it breaks every consumer that reads it.
	for _, value := range values {

		item := value.GetJSONLD()

		if item.GetString(vocab.PropertyID) == "" {
			continue
		}

		result.OrderedItems = append(result.OrderedItems, item)
	}

	// Page from the QUERY results, not the rendered items, so an omitted item does not
	// truncate the collection at the page that happened to contain it.
	if len(values) == pageSize {
		lastValue := values[pageSize-1]
		result.Next = partOf + "?publishDate=" + convert.String(lastValue.Created())
	}

	return result
}

// CollectionPage_Links returns one page of an OrderedCollection, rendering each value as a bare URL
func CollectionPage_Links[T model.ActivityPubURLGetter](pageID string, partOf string, pageSize int, values []T) streams.OrderedCollectionPage {

	// Generate the Page record
	result := streams.NewOrderedCollectionPage(pageID, partOf)

	if len(values) > 0 {

		// RULE: Omit any item with no URL, for the same reason CollectionPage omits an item with no `id`
		for _, value := range values {

			if url := value.ActivityPubURL(); url != "" {
				result.OrderedItems = append(result.OrderedItems, url)
			}
		}

		if len(values) == pageSize {
			lastValue := values[pageSize-1]
			result.Next = partOf + "?publishDate=" + convert.String(lastValue.Created())
		}
	}

	return result
}
