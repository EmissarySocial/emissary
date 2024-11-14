package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SearchTag represents a tag that Users and Guests can use to search
// for streams in the database.
type SearchTag struct {
	SearchTagID primitive.ObjectID `bson:"_id"`      // SearchTagID is the unique identifier for a SearchTag.
	ParentID    primitive.ObjectID `bson:"parentId"` // ParentID is the ID of the parent SearchTag.
	Tag         string             `bson:"tag"`      // Tag is the name of the tag.
	Notes       string             `bson:"notes"`    // Notes is a place for administrators to make notes about the tag.
	StateID     int                `bson:"stateId"`  // StateID represents the state that the tag is in. (FEATURED, ALLOWED, WAITING, BLOCKED)
	Rank        int                `bson:"rank"`     // Rank is the sort order of the SearchTag.

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

func (searchTag SearchTag) Fields() []string {
	return []string{
		"_id",
		"tag",
		"stateId",
	}
}
