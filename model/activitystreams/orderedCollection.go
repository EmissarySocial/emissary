package activitystreams

import (
	"encoding/json"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// OrderedCollection is a subtype of Collection in which members of the logical collection are assumed to always be strictly ordered.
// https://www.w3.org/ns/activitystreams#OrderedCollection
type OrderedCollection struct {
	Context      Context `json:"@context"`
	Type         string  `json:"type"`
	Summary      string  `json:"summary"`      // A natural language summarization of the object encoded as HTML. Multiple language tagged summaries may be provided.
	TotalItems   int     `json:"totalItems"`   // A non-negative integer specifying the total number of objects contained by the logical view of the collection. This number might not reflect the actual number of items serialized within the Collection object instance.
	OrderedItems []any   `json:"orderedItems"` // Identifies the items contained in a collection. The items might be ordered or unordered.
	Current      string  `json:"current"`      // In a paged Collection, indicates the page that contains the most recently updated member items.
	First        string  `json:"first"`        // In a paged Collection, indicates the furthest preceeding page of items in the collection.
	Last         string  `json:"last"`         // In a paged Collection, indicates the furthest proceeding page of the collection.
}

func (c *OrderedCollection) UnmarshalJSON(data []byte) error {

	result := mapof.NewAny()

	if err := json.Unmarshal(data, &result); err != nil {
		return derp.Wrap(err, "activitystreams.OrderedCollection.UnmarshalJSON", "Error unmarshalling JSON", string(data))
	}

	return c.UnmarshalMap(result)
}

func (c *OrderedCollection) UnmarshalMap(data mapof.Any) error {

	if dataType := data.GetString("type"); dataType != TypeOrderedCollection {
		return derp.NewInternalError("activitystreams.OrderedCollection.UnmarshalMap", "Invalid type", dataType)
	}

	c.Type = TypeOrderedCollection
	c.Summary = data.GetString("summary")
	c.TotalItems = data.GetInt("totalItems")
	c.Current = data.GetString("current")
	c.First = data.GetString("first")
	c.Last = data.GetString("last")

	if dataItems, ok := data["items"]; ok {
		if items, ok := UnmarshalItems(dataItems); ok {
			c.OrderedItems = items
		}
	}

	return nil
}
