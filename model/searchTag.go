package model

import (
	"github.com/EmissarySocial/emissary/tools/parse"
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SearchTag represents a tag that vistors can use to search
// for Users and Streams in the database.
type SearchTag struct {
	SearchTagID primitive.ObjectID `bson:"_id"`     // SearchTagID is the unique identifier for a SearchTag.
	Group       string             `bson:"group"`   // Group is the type of tag (GENRE, MOOD, ACTIVITY, etc.)
	Name        string             `bson:"name"`    // Name used for this tag
	Value       string             `bson:"value"`   // Value is the normalized version of the tag name.
	Colors      sliceof.String     `bson:"colors"`  // Colors is a slice of one or more RGB Hex color to use for tags featured on search panels.
	Related     string             `bson:"related"` // Related is a list of other tags that are related to this tag.
	Notes       string             `bson:"notes"`   // Notes is a place for administrators to make notes about the tag.
	Rank        int                `bson:"rank"`    // Rank is the sort order of the SearchTag.
	StateID     int                `bson:"stateId"` // StateID represents the state that the tag is in. (FEATURED, ALLOWED, WAITING, BLOCKED)
	ImageID     primitive.ObjectID `bson:"imageId"` // AttachmentID is the unique identifier for the attachment that is associated with this tag.

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
	case SearchTagStateFeatured:
		return "Featured"
	default:
		return "Unknown"
	}
}

// Fields returns a slice of field names to include in a batch search query.
func (searchTag SearchTag) Fields() []string {
	return []string{
		"_id",
		"name",
		"group",
		"colors",
		"stateId",
		"imageId",
		"related",
	}
}

// RelatedTags returns a parsed slice of tags from the "Related" tag field.
func (searchTag SearchTag) RelatedTags() sliceof.String {
	return parse.Hashtags(searchTag.Related)
}

// ImageURL returns the URL for the attached image (if present)
// or an empty string.
func (searchTag SearchTag) ImageURL() string {

	if searchTag.ImageID.IsZero() {
		return ""
	}

	return "/.searchTag/" + searchTag.SearchTagID.Hex() + "/attachments/" + searchTag.ImageID.Hex()
}
