package model

import (
	"math/rand/v2"

	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SearchResult represents a value in the search index
type SearchResult struct {
	SearchResultID primitive.ObjectID `bson:"_id"`         // SearchResultID is the unique identifier for a SearchResult.
	StreamID       primitive.ObjectID `bson:"streamId"`    // StreamID is the ID of the stream that this SearchResult is associated with.
	ObjectType     string             `bson:"objectType"`  // ObjectType is the ActivityPub object type (Person, Article, etc)
	URL            string             `bson:"url"`         // URL is the URL of the SearchResult.
	Name           string             `bson:"name"`        // Name is the name of the SearchResult.
	Summary        string             `bson:"summary"`     // Summary is a short description of the SearchResult.
	IconURL        string             `bson:"icon"`        // IconURL is the URL of the icon for the SearchResult.
	Tags           []string           `bson:"tags"`        // Tags is a list of tags that are associated with this SearchResult.
	Rank           int64              `bson:"rank"`        // Rank is the rank of this SearchResult in the search index.
	Shuffle        int64              `bson:"shuffle"`     // Shuffle is a random number used to shuffle the search results.
	ReIndexDate    int64              `bson:"reindexDate"` // ReIndexDate is the date that this SearchResult should be reindexed.

	journal.Journal `bson:",inline"`
}

func NewSearchResult() SearchResult {
	return SearchResult{
		SearchResultID: primitive.NewObjectID(),
		Tags:           make([]string, 0),
		Shuffle:        rand.Int64(),
	}
}

// ID returns the unique identifier for this SearchResult and
// implements the data.Object interface
func (searchResult SearchResult) ID() string {
	return searchResult.SearchResultID.Hex()
}

// Update copies the values from another SearchResult into this SearchResult
func (searchResult *SearchResult) Update(other SearchResult) {
	searchResult.StreamID = other.StreamID
	searchResult.ObjectType = other.ObjectType
	searchResult.URL = other.URL
	searchResult.Name = other.Name
	searchResult.Summary = other.Summary
	searchResult.IconURL = other.IconURL
	searchResult.Tags = other.Tags
}

func (searchResult SearchResult) Fields() []string {
	return []string{
		"objectType",
		"url",
		"name",
		"summary",
		"icon",
		"tags",
	}
}
