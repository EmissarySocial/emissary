package model

import (
	"math/rand/v2"
	"time"

	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SearchResult represents a value in the search index
type SearchResult struct {
	SearchResultID primitive.ObjectID `bson:"_id"`                    // SearchResultID is the unique identifier for a SearchResult.
	LockID         primitive.ObjectID `bson:"lockId"`                 // Unique identifier for the worker that is currently processing this task
	Type           string             `bson:"type"`                   // Type is the ActivityPub object type (Person, Article, etc)
	URL            string             `bson:"url"`                    // URL is the URL of the SearchResult.
	AttributedTo   string             `bson:"attributedTo,omitempty"` // AttributedTo is the name (or username) of the creator of this SearchResult.
	Name           string             `bson:"name"`                   // Name is the name of the SearchResult.
	IconURL        string             `bson:"iconUrl,omitempty"`      // IconURL is the URL of the icon for the SearchResult.
	Summary        string             `bson:"summary,omitempty"`      // Summary is a short description of the SearchResult.
	Text           string             `bson:"text,omitempty"`         // Text is the searchable text of this SearchResult.  It is used to build the index value.
	Date           time.Time          `bson:"date,omitempty"`         // Date is the date that this SearchResult was created.
	Place          mapof.Any          `bson:"place,omitempty"`        // Place is the location (encoded with GeoJSON) of the SearchResult.
	Tags           sliceof.String     `bson:"tags,omitempty"`         // Tags is a machine-readable list of tag values that are associated with this SearchResult.
	Index          sliceof.String     `bson:"index,omitempty"`        // Index is a list of words (encoded via metaphone) that are used to index this SearchResult.
	TimeoutDate    int64              `bson:"timeoutDate"`            // Unix epoch seconds when this task will "time out" and can be reclaimed by another process
	ReIndexDate    int64              `bson:"reindexDate"`            // ReIndexDate is the date that this SearchResult should be reindexed.
	NotifiedDate   int64              `bson:"notifiedDate"`           // NotifiedDate is the data that followers were notified of this SearchResult.
	Rank           int64              `bson:"rank"`                   // Rank is the rank of this SearchResult in the search index.
	Shuffle        int64              `bson:"shuffle"`                // Shuffle is a random number used to shuffle the search results.

	journal.Journal `bson:",inline"`
}

func NewSearchResult() SearchResult {
	return SearchResult{
		SearchResultID: primitive.NewObjectID(),
		Place:          mapof.NewAny(),
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

	if searchResult.IconURL != other.IconURL {
		searchResult.IconURL = other.IconURL
		changed = true
	}

	if searchResult.Summary != other.Summary {
		searchResult.Summary = other.Summary
		changed = true
	}

	if searchResult.Text != other.Text {
		searchResult.Text = other.Text
		changed = true
	}

	if searchResult.Date != other.Date {
		searchResult.Date = other.Date
		changed = true
	}

	if searchResult.Place.NotEqual(other.Place) {
		searchResult.Place = other.Place
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

	if searchResult.LockID != other.LockID {
		searchResult.LockID = other.LockID
		changed = true
	}

	if searchResult.TimeoutDate != other.TimeoutDate {
		searchResult.TimeoutDate = other.TimeoutDate
		changed = true
	}

	if searchResult.ReIndexDate != other.ReIndexDate {
		searchResult.ReIndexDate = other.ReIndexDate
		changed = true
	}

	if searchResult.NotifiedDate != other.NotifiedDate {
		searchResult.NotifiedDate = other.NotifiedDate
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
