package model

import (
	"math/rand/v2"

	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SearchResult represents a value in the search index
type SearchResult struct {
	SearchResultID primitive.ObjectID `bson:"_id"`          // SearchResultID is the unique identifier for a SearchResult.
	Type           string             `bson:"type"`         // Type is the ActivityPub object type (Person, Article, etc)
	URL            string             `bson:"url"`          // URL is the URL of the SearchResult.
	Name           string             `bson:"name"`         // Name is the name of the SearchResult.
	AttributedTo   string             `bson:"attributedTo"` // AttributedTo is the name (or username) of the creator of this SearchResult.
	Summary        string             `bson:"summary"`      // Summary is a short description of the SearchResult.
	IconURL        string             `bson:"icon"`         // IconURL is the URL of the icon for the SearchResult.
	TagNames       sliceof.String     `bson:"tagNames"`     // TagNames is a human-readable list of tags that are associated with this SearchResult.
	TagValues      sliceof.String     `bson:"tagValues"`    // TagValues is a machine-readable list of tag values that are associated with this SearchResult.
	FullText       string             `bson:"fullText"`     // FullText is the full text of the SearchResult.
	Rank           int64              `bson:"rank"`         // Rank is the rank of this SearchResult in the search index.
	Shuffle        int64              `bson:"shuffle"`      // Shuffle is a random number used to shuffle the search results.
	ReIndexDate    int64              `bson:"reindexDate"`  // ReIndexDate is the date that this SearchResult should be reindexed.

	journal.Journal `bson:",inline"`
}

func NewSearchResult() SearchResult {
	return SearchResult{
		SearchResultID: primitive.NewObjectID(),
		TagNames:       make(sliceof.String, 0),
		TagValues:      make(sliceof.String, 0),
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
	searchResult.Type = other.Type
	searchResult.URL = other.URL
	searchResult.Name = other.Name
	searchResult.AttributedTo = other.AttributedTo
	searchResult.Summary = other.Summary
	searchResult.IconURL = other.IconURL
	searchResult.TagNames = other.TagNames
	searchResult.TagValues = other.TagValues
	searchResult.FullText = other.FullText
}

func (searchResult SearchResult) Fields() []string {
	return []string{
		"type",
		"url",
		"name",
		"attributedTo",
		"summary",
		"icon",
		"tagNames",
	}
}

func (searchResult SearchResult) IsZero() bool {

	if searchResult.Type == "" {
		return true
	}

	if searchResult.URL == "" {
		return true
	}

	if searchResult.Name == "" {
		return true
	}

	return false
}
