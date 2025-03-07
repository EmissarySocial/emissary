package model

import (
	"net/url"
	"strings"

	"github.com/EmissarySocial/emissary/tools/parse"
	"github.com/EmissarySocial/emissary/tools/sorted"
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/sliceof"
	"github.com/dlclark/metaphone3"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SearchQuery represents a saved query that visitors can follow
type SearchQuery struct {
	SearchQueryID primitive.ObjectID `bson:"_id"`       // SearchQueryID is the unique identifier for a SearchQuery.
	Types         []string           `bson:"types"`     // The types of results that this query is interested in (Person, Article, Album, Audio, etc)
	Query         string             `bson:"query"`     // The original string used in the search query
	Index         []string           `bson:"index"`     // The parsed (and normalized) index of values in the search query
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

	if len(searchQuery.Types) > 0 {
		return false
	}

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
	searchQuery.Types = strings.Split(values.Get("types"), ",")
	searchQuery.Query = values.Get("q")
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

// Match returns TRUE if this query matches the provided SearchResult
func (searchQuery SearchQuery) Match(searchResult SearchResult) bool {

	// Match Tags
	if !sorted.ContainsAll(searchQuery.Tags, searchResult.Tags) {
		return false
	}

	// Match Text Index
	if !sorted.ContainsAll(searchQuery.Index, searchResult.Index) {
		return false
	}

	// TODO: Geosearch by Location and Radius

	// TODO: Time-Based Search (might not be possible)

	// Otherwise, return true
	return true
}

// SetQuery parses the provided string into the SearchQuery object
func (searchQuery *SearchQuery) SetQuery(queryString string) {

	searchQuery.Query = queryString

	// Split out tags
	tags, remainder := parse.HashtagsAndRemainder(queryString)
	searchQuery.Tags = append(searchQuery.Tags, tags...)

	// Full-text index the remainder
	encoder := metaphone3.Encoder{}
	primary, _ := encoder.Encode(remainder)
	strings.Split(primary, " ")
	searchQuery.Index = append(searchQuery.Index, primary)
}

// AppendTags adds one or more sets of tags to the SearchQuery
func (searchQuery *SearchQuery) AppendTags(tags ...string) {
	for _, tag := range tags {
		values := parse.Split(tag)
		if len(values) > 0 {
			searchQuery.Tags = append(searchQuery.Tags, values...)
		}
	}

}
