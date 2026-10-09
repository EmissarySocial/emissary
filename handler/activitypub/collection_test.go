package activitypub

import (
	"testing"

	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// testItem is a minimal JSONLDGetter/ActivityPubURLGetter used to drive the collection builders
type testItem struct {
	id      string
	created int64
}

func (item testItem) GetJSONLD() mapof.Any {
	return mapof.Any{
		vocab.PropertyID:   item.id,
		vocab.PropertyType: vocab.ActivityTypeCreate,
	}
}

func (item testItem) ActivityPubURL() string {
	return item.id
}

func (item testItem) Created() int64 {
	return item.created
}

// TestCollection verifies that a Collection stub points at the first page of the collection
func TestCollection(t *testing.T) {

	result := Collection("https://x.social/@bob/pub/outbox")

	require.Equal(t, "https://x.social/@bob/pub/outbox", result.ID)
	require.Equal(t, "https://x.social/@bob/pub/outbox?publishDate=9223372036854775807", result.First)
}

// TestCollection_StripsExistingQuery verifies that the first-page URL replaces any query the caller supplied
func TestCollection_StripsExistingQuery(t *testing.T) {

	result := Collection("https://x.social/@bob/pub/outbox?publishDate=123")

	require.Equal(t, "https://x.social/@bob/pub/outbox?publishDate=9223372036854775807", result.First)
}

// TestCollectionPage verifies that every identifiable item is rendered into the page
func TestCollectionPage(t *testing.T) {

	values := []testItem{
		{id: "https://x.social/@bob/pub/outbox/1", created: 300},
		{id: "https://x.social/@bob/pub/outbox/2", created: 200},
	}

	result := CollectionPage("https://x.social/page", "https://x.social/@bob/pub/outbox", 60, values)

	require.Len(t, result.OrderedItems, 2)
	require.Equal(t, "https://x.social/@bob/pub/outbox/1", result.OrderedItems[0].(mapof.Any).GetString(vocab.PropertyID))
	require.Equal(t, "https://x.social/@bob/pub/outbox/2", result.OrderedItems[1].(mapof.Any).GetString(vocab.PropertyID))

	// A partial page is the last page, so there is no `next` link
	require.Empty(t, result.Next)
}

// TestCollectionPage_Empty verifies that a page with no values renders no items
func TestCollectionPage_Empty(t *testing.T) {

	result := CollectionPage("https://x.social/page", "https://x.social/@bob/pub/outbox", 60, []testItem{})

	require.Empty(t, result.OrderedItems)
	require.Empty(t, result.Next)
}

// TestCollectionPage_OmitsUnidentifiableItems verifies that an item with no `id` never reaches a consumer (BUG-146)
func TestCollectionPage_OmitsUnidentifiableItems(t *testing.T) {

	values := []testItem{
		{id: "https://x.social/@bob/pub/outbox/1", created: 300},
		{id: "", created: 250}, // an OutboxMessage with no ActorURL to build an `id` from
		{id: "https://x.social/@bob/pub/outbox/3", created: 200},
	}

	result := CollectionPage("https://x.social/page", "https://x.social/@bob/pub/outbox", 60, values)

	require.Len(t, result.OrderedItems, 2)
	require.Equal(t, "https://x.social/@bob/pub/outbox/1", result.OrderedItems[0].(mapof.Any).GetString(vocab.PropertyID))
	require.Equal(t, "https://x.social/@bob/pub/outbox/3", result.OrderedItems[1].(mapof.Any).GetString(vocab.PropertyID))
}

// TestCollectionPage_OmitsEveryItem verifies that a page of unidentifiable items renders as an empty page, not an error
func TestCollectionPage_OmitsEveryItem(t *testing.T) {

	values := []testItem{{id: "", created: 300}, {id: "", created: 200}}

	result := CollectionPage("https://x.social/page", "https://x.social/@bob/pub/outbox", 60, values)

	require.Empty(t, result.OrderedItems)
}

// TestCollectionPage_NextSurvivesOmission verifies that paging follows the QUERY results, so an omitted
// item cannot truncate the collection at the page that happened to contain it (BUG-146)
func TestCollectionPage_NextSurvivesOmission(t *testing.T) {

	values := []testItem{
		{id: "https://x.social/@bob/pub/outbox/1", created: 300},
		{id: "", created: 250},
		{id: "https://x.social/@bob/pub/outbox/3", created: 200},
	}

	result := CollectionPage("https://x.social/page", "https://x.social/@bob/pub/outbox", 3, values)

	require.Len(t, result.OrderedItems, 2)
	require.Equal(t, "https://x.social/@bob/pub/outbox?publishDate=200", result.Next)
}

// TestCollectionPage_NextUsesLastValue verifies that a full page links to the next page by its last create date
func TestCollectionPage_NextUsesLastValue(t *testing.T) {

	values := []testItem{
		{id: "https://x.social/@bob/pub/outbox/1", created: 300},
		{id: "https://x.social/@bob/pub/outbox/2", created: 200},
	}

	result := CollectionPage("https://x.social/page", "https://x.social/@bob/pub/outbox", 2, values)

	require.Equal(t, "https://x.social/@bob/pub/outbox?publishDate=200", result.Next)
}

// TestCollectionPage_Links verifies that a link page renders each item as a bare URL
func TestCollectionPage_Links(t *testing.T) {

	values := []testItem{
		{id: "https://x.social/@bob/pub/liked/1", created: 300},
		{id: "https://x.social/@bob/pub/liked/2", created: 200},
	}

	result := CollectionPage_Links("https://x.social/page", "https://x.social/@bob/pub/liked", 60, values)

	require.Equal(t, []any{"https://x.social/@bob/pub/liked/1", "https://x.social/@bob/pub/liked/2"}, []any(result.OrderedItems))
	require.Empty(t, result.Next)
}

// TestCollectionPage_LinksOmitsEmptyURLs verifies that an empty URL never lands in orderedItems (BUG-146)
func TestCollectionPage_LinksOmitsEmptyURLs(t *testing.T) {

	values := []testItem{
		{id: "https://x.social/@bob/pub/liked/1", created: 300},
		{id: "", created: 250},
		{id: "https://x.social/@bob/pub/liked/3", created: 200},
	}

	result := CollectionPage_Links("https://x.social/page", "https://x.social/@bob/pub/liked", 3, values)

	require.Equal(t, []any{"https://x.social/@bob/pub/liked/1", "https://x.social/@bob/pub/liked/3"}, []any(result.OrderedItems))

	// Paging still follows the QUERY results, so the omission does not end the collection
	require.Equal(t, "https://x.social/@bob/pub/liked?publishDate=200", result.Next)
}

// TestCollectionPage_LinksEmpty verifies that a link page with no values renders no items
func TestCollectionPage_LinksEmpty(t *testing.T) {

	result := CollectionPage_Links("https://x.social/page", "https://x.social/@bob/pub/liked", 60, []testItem{})

	require.Empty(t, result.OrderedItems)
	require.Empty(t, result.Next)
}
