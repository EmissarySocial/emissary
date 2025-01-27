package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SearchQuery represents a saved query that visitors can follow
type SearchQuery struct {
	SearchQueryID primitive.ObjectID `bson:"_id"`       // SearchQueryID is the unique identifier for a SearchQuery.
	Original      string             `bson:"original"`  // The original string used in the search query
	TagValues     []string           `bson:"tagValues"` // The parsed (and normalized) tag values
	Remainder     string             `bson:"remainder"` // The remainder of the search query that was not tags.

	journal.Journal `bson:",inline"`
}

func NewSearchQuery() SearchQuery {
	return SearchQuery{
		SearchQueryID: primitive.NewObjectID(),
		TagValues:     make(sliceof.String, 0),
	}
}

// ID returns the unique identifier for this SearchQuery and
// implements the data.Object interface
func (searchResult SearchQuery) ID() string {
	return searchResult.SearchQueryID.Hex()
}
