package activitystreams

import (
	"encoding/json"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// OrderedCollectionPage is used to represent ordered subsets of items from an OrderedCollection. Refer to the Activity Streams 2.0 Core for a complete description of the OrderedCollectionPage object.
// https://www.w3.org/ns/activitystreams#OrderedCollectionPage
type OrderedCollectionPage struct {
	Context      Context `json:"@context"`
	Type         string  `json:"type"`
	Summary      string  `json:"summary"`      // A natural language summarization of the object encoded as HTML. Multiple language tagged summaries may be provided.
	TotalItems   int     `json:"totalItems"`   // A non-negative integer specifying the total number of objects contained by the logical view of the collection. This number might not reflect the actual number of items serialized within the Collection object instance.
	Current      string  `json:"current"`      // In a paged Collection, indicates the page that contains the most recently updated member items.
	First        string  `json:"first"`        // In a paged Collection, indicates the furthest preceeding page of items in the collection.
	Last         string  `json:"last"`         // In a paged Collection, indicates the furthest proceeding page of the collection.
	StartIndex   int     `json:"startIndex"`   // A non-negative integer value identifying the relative position within the logical view of a strictly ordered collection.
	PartOf       string  `json:"partOf"`       // dentifies the Collection to which a CollectionPage objects items belong.
	Prev         string  `json:"prev"`         // In a paged Collection, identifies the previous page of items.
	Next         string  `json:"next"`         // In a paged Collection, indicates the next page of items.
	OrderedItems []any   `json:"orderedItems"` // Identifies the items contained in a collection. The items might be ordered or unordered.
}

func (c *OrderedCollectionPage) UnmarshalJSON(data []byte) error {

	result := mapof.NewAny()

	if err := json.Unmarshal(data, &result); err != nil {
		return derp.Wrap(err, "activitystreams.OrderedCollectionPage.UnmarshalJSON", "Error unmarshalling JSON", string(data))
	}

	return c.UnmarshalMap(result)
}

func (c *OrderedCollectionPage) UnmarshalMap(data mapof.Any) error {

	if dataType := data.GetString("type"); dataType != TypeOrderedCollectionPage {
		return derp.NewInternalError("activitystreams.OrderedCollectionPage.UnmarshalMap", "Invalid type", dataType)
	}

	c.Type = TypeOrderedCollectionPage
	c.Summary = data.GetString("summary")
	c.TotalItems = data.GetInt("totalItems")
	c.Current = data.GetString("current")
	c.First = data.GetString("first")
	c.Last = data.GetString("last")
	c.StartIndex = data.GetInt("startIndex")
	c.PartOf = data.GetString("partOf")
	c.Prev = data.GetString("prev")
	c.Next = data.GetString("next")

	if dataItems, ok := data["items"]; ok {
		if items, ok := UnmarshalItems(dataItems); ok {
			c.OrderedItems = items
		}
	}

	return nil
}
