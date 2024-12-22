package model

import (
	"github.com/benpate/color"
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SearchTag represents a tag that Users and Guests can use to search
// for streams in the database.
type SearchTag struct {
	SearchTagID primitive.ObjectID `bson:"_id"`         // SearchTagID is the unique identifier for a SearchTag.
	Parent      string             `bson:"parent"`      // Parent is the name of the tag that contains this Tag (optional)
	Name        string             `bson:"name"`        // Name used for this tag
	Description string             `bson:"description"` // Description is shown on tags featured in search panels
	Color       string             `bson:"color"`       // Color is the RGB Hex color to use for tags featured on search panels.
	Notes       string             `bson:"notes"`       // Notes is a place for administrators to make notes about the tag.
	Rank        int                `bson:"rank"`        // Rank is the sort order of the SearchTag.
	StateID     int                `bson:"stateId"`     // StateID represents the state that the tag is in. (FEATURED, ALLOWED, WAITING, BLOCKED)

	journal.Journal `bson:",inline"`
}

// NewSearchTag returns a fully initialized SearchTag object
func NewSearchTag() SearchTag {
	return SearchTag{
		SearchTagID: primitive.NewObjectID(),
		StateID:     SearchTagStateWaiting,
	}
}

// ID returns the unique identifier for this SearchTag,
// implementing the data.Object interface.
func (searchTag SearchTag) ID() string {
	return searchTag.SearchTagID.Hex()
}

func (searchTag SearchTag) StatusText() string {
	switch searchTag.StateID {

	case SearchTagStateBlocked:
		return "Blocked"
	case SearchTagStateWaiting:
		return "Waiting"
	case SearchTagStateFeatured:
		return "Featured"
	case SearchTagStateAllowed:
		return "Allowed"
	default:
		return "Unknown"
	}
}

func (searchTag SearchTag) Fields() []string {
	return []string{
		"_id",
		"name",
		"stateId",
		"color",
		"description",
	}
}

func (searchTag SearchTag) BackgroundColor() string {
	return searchTag.Color
}

func (searchTag SearchTag) TextColor() string {
	return color.Parse(searchTag.Color).Text().Hex()
}
