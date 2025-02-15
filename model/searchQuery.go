package model

import (
	"net/url"
	"strings"

	"github.com/EmissarySocial/emissary/tools/parse"
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SearchQuery represents a saved query that visitors can follow
type SearchQuery struct {
	SearchQueryID primitive.ObjectID `bson:"_id"`       // SearchQueryID is the unique identifier for a SearchQuery.
	Query         string             `bson:"query"`     // The original string used in the search query
	Tags          []string           `bson:"tags"`      // The parsed (and normalized) tag values
	StartDate     string             `bson:"startDate"` // The start date of the search query
	Location      string             `bson:"location"`  // The location of the search query

	journal.Journal `bson:",inline"`
}

func NewSearchQuery() SearchQuery {
	return SearchQuery{
		SearchQueryID: primitive.NewObjectID(),
		Tags:          make(sliceof.String, 0),
	}
}

// ID returns the unique identifier for this SearchQuery and
// implements the data.Object interface
func (searchQuery SearchQuery) ID() string {
	return searchQuery.SearchQueryID.Hex()
}

// IsEmpty returns TRUE if this SearchQuery has NO useful data
func (searchQuery SearchQuery) IsEmpty() bool {

	if searchQuery.Query != "" {
		return false
	}

	if len(searchQuery.Tags) > 0 {
		return false
	}

	if searchQuery.StartDate != "" {
		return false
	}

	if searchQuery.Location != "" {
		return false
	}

	return true
}

// NotEmpty returns TRUE if this SearchQuery has useful data
func (searchQuery SearchQuery) NotEmpty() bool {
	return !searchQuery.IsEmpty()
}

func (searchQuery *SearchQuery) Parse(values url.Values) {
	searchQuery.Query = values.Get("q")
	searchQuery.Tags = strings.Split(values.Get("tags"), ",")
	searchQuery.StartDate = values.Get("startDate")
	searchQuery.Location = values.Get("location")

	searchQuery.Tags = make(sliceof.String, 0)

	for _, tag := range values["tag"] {

		for _, tag := range parse.Split(tag) {
			tag = strings.TrimSpace(tag)
			tag = ToToken(tag)

			searchQuery.Tags = append(searchQuery.Tags, tag)
		}
	}
}
