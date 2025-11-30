package model

import (
	"math/rand/v2"
	"time"

	"github.com/benpate/data/journal"
	"github.com/benpate/geo"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SearchResult represents a value in the search index
type SearchResult struct {
	SearchResultID primitive.ObjectID `bson:"_id"`                    // SearchResultID is the unique identifier for a SearchResult.
	Type           string             `bson:"type"`                   // Type is the ActivityPub object type (Person, Article, etc)
	URL            string             `bson:"url"`                    // URL is the URL of the SearchResult.
	AttributedTo   string             `bson:"attributedTo,omitempty"` // AttributedTo is the name (or username) of the creator of this SearchResult.
	Name           string             `bson:"name"`                   // Name is the name of the SearchResult.
	Summary        string             `bson:"summary,omitempty"`      // Summary is a short description of the SearchResult.
	IconURL        string             `bson:"iconUrl,omitempty"`      // IconURL is the URL of the icon for the SearchResult.
	Date           time.Time          `bson:"date,omitempty"`         // Date is the datetime related to this SearchResult.
	Location       geo.Point          `bson:"location,omitempty"`     // GeoJSON Point (longitude,latitude) related to this SearchResult
	Tags           sliceof.String     `bson:"tags,omitempty"`         // Tags is a machine-readable list of tag values that are associated with this SearchResult.
	Text           string             `bson:"text,omitempty"`         // Text is the searchable text of this SearchResult.  It is used to build the index value.
	Index          sliceof.String     `bson:"index,omitempty"`        // Index is a list of words (encoded via metaphone) that are used to index this SearchResult.
	ReIndexDate    int64              `bson:"reindexDate"`            // ReIndexDate is the date that this SearchResult should be reindexed.
	Rank           int64              `bson:"rank"`                   // Rank is the rank of this SearchResult in the search index.
	Shuffle        int64              `bson:"shuffle"`                // Shuffle is a random number used to shuffle the search results.
	Local          bool               `bson:"local"`                  // Local is true if this SearchResult originates on the local server.  Only local SearchResults will be syndicated to external servers.

	journal.Journal `json:"-" bson:",inline"`
}

func NewSearchResult() SearchResult {
	return SearchResult{
		SearchResultID: primitive.NewObjectID(),
		Location:       geo.NewPoint(0, 0),
		Tags:           make(sliceof.String, 0),
		Index:          make(sliceof.String, 0),
		Shuffle:        rand.Int64(),
	}
}

// ID returns the unique identifier for this SearchResult and
// implements the data.Object interface
func (searchResult SearchResult) ID() string {
	return searchResult.SearchResultID.Hex()
}

// Update copies the values from another SearchResult into this SearchResult
func (searchResult *SearchResult) Update(other SearchResult) bool {

	changed := false

	if searchResult.Type != other.Type {
		searchResult.Type = other.Type
		changed = true
	}

	if searchResult.URL != other.URL {
		searchResult.URL = other.URL
		changed = true
	}

	if searchResult.AttributedTo != other.AttributedTo {
		searchResult.AttributedTo = other.AttributedTo
		changed = true
	}

	if searchResult.Name != other.Name {
		searchResult.Name = other.Name
		changed = true
	}

	if searchResult.Summary != other.Summary {
		searchResult.Summary = other.Summary
		changed = true
	}

	if searchResult.IconURL != other.IconURL {
		searchResult.IconURL = other.IconURL
		changed = true
	}

	if searchResult.Date != other.Date {
		searchResult.Date = other.Date
		changed = true
	}

	if searchResult.Location != other.Location {
		searchResult.Location = other.Location
		changed = true
	}

	if searchResult.Text != other.Text {
		searchResult.Text = other.Text
		changed = true
	}

	if searchResult.Index.NotEqual(other.Index) {
		searchResult.Index = other.Index
		changed = true
	}

	if searchResult.Tags.NotEqual(other.Tags) {
		searchResult.Tags = other.Tags
		changed = true
	}

	if searchResult.ReIndexDate != other.ReIndexDate {
		searchResult.ReIndexDate = other.ReIndexDate
		changed = true
	}

	if searchResult.Rank != other.Rank {
		searchResult.Rank = other.Rank
		changed = true
	}

	if searchResult.Shuffle != other.Shuffle {
		searchResult.Shuffle = other.Shuffle
		changed = true
	}

	return changed
}

func (searchResult SearchResult) Fields() []string {
	return []string{
		"type",
		"url",
		"attributedTo",
		"name",
		"iconUrl",
		"summary",
		"date",
		"tags",
		"location",
		"shuffle",
		"createDate",
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

func (searchResult SearchResult) NotZero() bool {
	return !searchResult.IsZero()
}
