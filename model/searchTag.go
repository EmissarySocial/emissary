package model

import (
	"github.com/EmissarySocial/emissary/tools/parse"
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SearchTag represents a tag that Users and Guests can use to search
// for streams in the database.
type SearchTag struct {
	SearchTagID    primitive.ObjectID `bson:"_id"`            // SearchTagID is the unique identifier for a SearchTag.
	Name           string             `bson:"name"`           // Name used for this tag
	Description    string             `bson:"description"`    // Description is shown on tags featured in search panels
	Colors         sliceof.String     `bson:"colors"`         // Colors is a slice of one or more RGB Hex color to use for tags featured on search panels.
	Notes          string             `bson:"notes"`          // Notes is a place for administrators to make notes about the tag.
	Related        string             `bson:"related"`        // Related is a list of other tags that are related to this tag.
	Rank           int                `bson:"rank"`           // Rank is the sort order of the SearchTag.
	StateID        int                `bson:"stateId"`        // StateID represents the state that the tag is in. (FEATURED, ALLOWED, WAITING, BLOCKED)
	IsFeatured     bool               `bson:"isFeatured"`     // IsFeatured is TRUE if this tag is featured on home page search panels.
	IsCustomBanner bool               `bson:"isCustomBanner"` // IsCustomBanner is TRUE if this tag has a custom banner for this tag.

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
		"colors",
		"description",
	}
}

func (searchTag SearchTag) RelatedTags() sliceof.String {
	return parse.Hashtags(searchTag.Related)
}
