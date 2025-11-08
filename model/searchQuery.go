package model

import (
	"crypto/md5"
	"encoding/base64"
	"io"
	"slices"
	"strings"

	"github.com/EmissarySocial/emissary/tools/parse"
	"github.com/EmissarySocial/emissary/tools/sorted"
	"github.com/benpate/data/journal"
	"github.com/benpate/exp"
	"github.com/benpate/geo"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
	"github.com/dlclark/metaphone3"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SearchQuery represents a saved query that visitors can follow
type SearchQuery struct {
	SearchQueryID primitive.ObjectID `bson:"_id"`               // SearchQueryID is the unique identifier for a SearchQuery
	URL           string             `bson:"url"`               // The URL where this search query originated
	Types         sliceof.String     `bson:"types"`             // The types of results that this query is interested in (Person, Article, Album, Audio, etc)
	Query         string             `bson:"query"`             // The original string used in the search query
	Index         sliceof.String     `bson:"index"`             // The parsed (and normalized) index of values in the search query
	Tags          sliceof.String     `bson:"tags"`              // The parsed (and normalized) tag values
	StartDate     string             `bson:"startDate"`         // The start date of the search query
	Polygon       geo.Polygon        `bson:"polygon,omitempty"` // Polygon to search within
	Signature     string             `bson:"signature"`         // The hash of this search query

	journal.Journal `bson:",inline"`
}

func NewSearchQuery() SearchQuery {
	return SearchQuery{
		SearchQueryID: primitive.NewObjectID(),
		Types:         make(sliceof.String, 0),
		Index:         make(sliceof.String, 0),
		Tags:          make(sliceof.String, 0),
		Polygon:       geo.NewPolygon(),
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

	if !searchQuery.Polygon.IsZero() {
		return false
	}

	return true
}

// NotEmpty returns TRUE if this SearchQuery has useful data
func (searchQuery SearchQuery) NotEmpty() bool {
	return !searchQuery.IsEmpty()
}

// Expression returns the criteria in this SearchQuery as an exp.Expression
func (searchQuery SearchQuery) Expression() exp.Expression {

	var result exp.Expression = exp.And()

	if searchQuery.Types.NotEmpty() {
		result = result.AndIn("type", searchQuery.Types)
	}

	for _, tag := range searchQuery.Tags {
		result = result.AndEqual("tags", tag)
	}

	for _, index := range searchQuery.Index {
		result = result.AndEqual("index", index)
	}

	if searchQuery.Polygon.NotZero() {
		result = result.And(exp.GeoWithin("location", searchQuery.Polygon))
	}

	return result

}

func (searchQuery SearchQuery) Match(searchResult *SearchResult) bool {

	// Match Type(s)
	if searchQuery.Types.NotEmpty() {
		if !sorted.Contains(searchQuery.Types, searchResult.Type) {
			return false
		}
	}

	// Match Tags
	if searchQuery.Tags.NotEmpty() {
		if !sorted.ContainsAll(searchQuery.Tags, searchResult.Tags) {
			return false
		}
	}

	// Match Text Index
	if searchQuery.Index.NotEmpty() {
		if !sorted.ContainsAll(searchQuery.Index, searchResult.Index) {
			return false
		}
	}

	// Skipping "Polygon" because this is more efficiently handled in the
	// database.

	// Skipping "StartDate" because I'm not sure how to do this right now.
	// We HAVE defined relative time args (in exp-builder) but consider:
	//
	// An event falls outside of that range when the search record is first
	// created. But, how would we re-run the query once the "present time"
	// catches up, and is now within the relative time range?
	//
	// If there are any queries OUTSIDE of the target range, we'll probably
	// have to set a queue task in the future to re-evaluate this search
	// at some strategic point in the future. (how?)

	return true
}

// SetQuery parses the provided string into the SearchQuery object
func (searchQuery *SearchQuery) SetQuery(queryString string) {

	// Split Tags from text query
	tags, remainder := parse.HashtagsAndRemainder(queryString)
	searchQuery.Tags = append(searchQuery.Tags, tags...)
	searchQuery.Query = queryString

	// Full-text index the remainder
	if remainder != "" {
		encoder := metaphone3.Encoder{}
		primary, _ := encoder.Encode(remainder)
		strings.Split(primary, " ")
		searchQuery.Index = append(searchQuery.Index, primary)
	}
}

// AppendTags adds one or more sets of tags to the SearchQuery
func (searchQuery *SearchQuery) AppendTags(tags ...string) {
	for _, tag := range tags {
		if tag != "" {
			values := parse.Split(tag)
			if len(values) > 0 {
				searchQuery.Tags = append(searchQuery.Tags, values...)
			}
		}
	}
}

// MakeSignature sorts all of the slice values and creates an MD5 hash
// of the search criteria to make it easy to find duplicate SearchQuery
// objects later.
func (searchQuery *SearchQuery) MakeSignature() {

	// Normalize Tag Values
	searchQuery.Tags = slice.Map(searchQuery.Tags, ToToken)

	// Sort all slice values so that the hash is consistent
	slices.Sort(searchQuery.Types)
	slices.Sort(searchQuery.Tags)
	slices.Sort(searchQuery.Index)

	// De-duplicate all slice values
	searchQuery.Types = sorted.Unique(searchQuery.Types)
	searchQuery.Tags = sorted.Unique(searchQuery.Tags)
	searchQuery.Index = sorted.Unique(searchQuery.Index)

	// Collect values into a "unique" string
	var plaintext strings.Builder

	for _, value := range searchQuery.Types {
		plaintext.WriteString("TYP:")
		plaintext.WriteString(value)
		plaintext.WriteString("\n")
	}

	for _, value := range searchQuery.Tags {
		plaintext.WriteString("TAG:")
		plaintext.WriteString(value)
		plaintext.WriteString("\n")
	}

	for _, value := range searchQuery.Index {
		plaintext.WriteString("IDX:")
		plaintext.WriteString(value)
		plaintext.WriteString("\n")
	}

	if searchQuery.Polygon.NotZero() {
		plaintext.WriteString("PGN:")
		plaintext.WriteString(searchQuery.Polygon.String())
		plaintext.WriteString("\n")
	}

	if searchQuery.StartDate != "" {
		plaintext.WriteString("DT:")
		plaintext.WriteString(searchQuery.StartDate)
		plaintext.WriteString("\n")
	}

	// Make a hash of the plaintext for easy indexing
	h := md5.New()
	io.WriteString(h, plaintext.String()) // nolint:errcheck
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	// Save the signed value to the SearchQuery and GTFO.
	searchQuery.Signature = signature
}
