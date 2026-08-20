package build

import (
	"iter"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/parse"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
	"github.com/dlclark/metaphone3"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SearchBuilder is a chainable search API that templates use to select SearchResults
type SearchBuilder struct {
	searchTagService    *service.SearchTag
	searchResultService *service.SearchResult
	ruleService         *service.Rule
	userID              primitive.ObjectID
	session             data.Session
	criteria            exp.Expression
	textQuery           string
	sortField           string
	sortDirection       string
	maxRows             int64
}

// NewSearchBuilder returns a SearchBuilder seeded with the provided criteria and text query
func NewSearchBuilder(searchTagService *service.SearchTag, searchResultService *service.SearchResult, ruleService *service.Rule, userID primitive.ObjectID, session data.Session, criteria exp.Expression, textQuery string) SearchBuilder {

	return SearchBuilder{
		searchTagService:    searchTagService,
		searchResultService: searchResultService,
		ruleService:         ruleService,
		userID:              userID,
		session:             session,
		criteria:            criteria,
		textQuery:           textQuery,
		sortField:           "rank",
		sortDirection:       "asc",
		maxRows:             60,
	}
}

/********************************
 * Query Builder
 ********************************/

// Top1 limits this search to 1 result
func (builder SearchBuilder) Top1() SearchBuilder {
	builder.maxRows = 1
	return builder
}

// Top6 limits this search to 6 results
func (builder SearchBuilder) Top6() SearchBuilder {
	builder.maxRows = 6
	return builder
}

// Top8 limits this search to 8 results
func (builder SearchBuilder) Top8() SearchBuilder {
	builder.maxRows = 8
	return builder
}

// Top12 limits this search to 12 results
func (builder SearchBuilder) Top12() SearchBuilder {
	builder.maxRows = 12
	return builder
}

// Top24 limits this search to 24 results
func (builder SearchBuilder) Top24() SearchBuilder {
	builder.maxRows = 24
	return builder
}

// Top30 limits this search to 30 results
func (builder SearchBuilder) Top30() SearchBuilder {
	builder.maxRows = 30
	return builder
}

// Top60 limits this search to 60 results
func (builder SearchBuilder) Top60() SearchBuilder {
	builder.maxRows = 60
	return builder
}

// Top120 limits this search to 120 results
func (builder SearchBuilder) Top120() SearchBuilder {
	builder.maxRows = 120
	return builder
}

// Top150 limits this search to 150 results
func (builder SearchBuilder) Top150() SearchBuilder {
	builder.maxRows = 150
	return builder
}

// Top200 limits this search to 200 results
func (builder SearchBuilder) Top200() SearchBuilder {
	builder.maxRows = 200
	return builder
}

// Top300 limits this search to 300 results
func (builder SearchBuilder) Top300() SearchBuilder {
	builder.maxRows = 300
	return builder
}

// Top400 limits this search to 400 results
func (builder SearchBuilder) Top400() SearchBuilder {
	builder.maxRows = 400
	return builder
}

// Top500 limits this search to 500 results
func (builder SearchBuilder) Top500() SearchBuilder {
	builder.maxRows = 500
	return builder
}

// Top600 limits this search to 600 results
func (builder SearchBuilder) Top600() SearchBuilder {
	builder.maxRows = 600
	return builder
}

// All removes the row limit from this search
func (builder SearchBuilder) All() SearchBuilder {
	builder.maxRows = 0
	return builder
}

// AfterRank pages this search forward, past the provided rank
func (builder SearchBuilder) AfterRank(rank int64) SearchBuilder {
	builder.criteria = builder.criteria.AndGreaterThan("rank", rank)
	return builder
}

// AfterShuffle pages this search forward, past the provided shuffle value
func (builder SearchBuilder) AfterShuffle(shuffle int64) SearchBuilder {
	builder.criteria = builder.criteria.AndGreaterThan("shuffle", shuffle)
	return builder
}

// Where limits this search to results whose field equals the provided value
func (builder SearchBuilder) Where(field string, value any) SearchBuilder {
	builder.criteria = builder.criteria.AndEqual(field, value)
	return builder
}

// WhereLT limits this search to results whose field is less than the provided value
func (builder SearchBuilder) WhereLT(field string, value any) SearchBuilder {
	builder.criteria = builder.criteria.AndLessThan(field, value)
	return builder
}

// WhereGT limits this search to results whose field is greater than the provided value
func (builder SearchBuilder) WhereGT(field string, value any) SearchBuilder {
	builder.criteria = builder.criteria.AndGreaterThan(field, value)
	return builder
}

// WhereType limits this search to results of the provided types
func (builder SearchBuilder) WhereType(typeNames ...string) SearchBuilder {
	builder.criteria = builder.criteria.AndIn("type", typeNames)
	return builder
}

// WhereTags limits this search to results carrying at least one of the provided tags
func (builder SearchBuilder) WhereTags(tags ...string) SearchBuilder {
	builder.criteria = builder.criteria.AndInAll("tags", tags)
	return builder
}

// ByCreateDate sorts this search by creation date
func (builder SearchBuilder) ByCreateDate() SearchBuilder {
	builder.sortField = "createDate"
	return builder
}

// ByDate sorts this search by date
func (builder SearchBuilder) ByDate() SearchBuilder {
	builder.sortField = "date"
	return builder
}

// ByName sorts this search by name
func (builder SearchBuilder) ByName() SearchBuilder {
	builder.sortField = "name"
	return builder
}

// ByRank sorts this search by rank
func (builder SearchBuilder) ByRank() SearchBuilder {
	builder.sortField = "rank"
	return builder
}

// ByShuffle sorts this search by its shuffle value, which randomizes the order
func (builder SearchBuilder) ByShuffle() SearchBuilder {
	builder.sortField = "shuffle"
	return builder
}

// By sorts this search by the named field
func (builder SearchBuilder) By(sortField string) SearchBuilder {
	builder.sortField = sortField
	return builder
}

// Reverse flips this search into descending order
func (builder SearchBuilder) Reverse() SearchBuilder {
	builder.sortDirection = option.SortDirectionDescending
	return builder
}

/********************************
 * Actions
 ********************************/

// Slice returns the results of the query as a slice of objects
func (builder SearchBuilder) Slice() (sliceof.Object[model.SearchResult], error) {

	criteria := builder.assembleCriteria()
	result, err := builder.searchResultService.Query(builder.session, criteria, builder.makeOptions()...)

	if err != nil {
		return result, err
	}

	// RULE: results hidden by the viewer's rules are dropped, not placeheld. A search listing is
	// discovery, not thread structure, so a hole needs no explanation (R17 leaks nothing this way).
	builder.ruleService.LabelSearchResults(builder.session, builder.userID, result)
	result = slice.Filter(result, func(searchResult model.SearchResult) bool {
		return !searchResult.Labels.IsHidden()
	})

	return result, nil
}

// Range returns the results of the query as a Go 1.23 RangeFunc
func (builder SearchBuilder) Range() (iter.Seq[model.SearchResult], error) {

	criteria := builder.assembleCriteria()
	values, err := builder.searchResultService.Range(builder.session, criteria, builder.makeOptions()...)

	if err != nil {
		return values, err
	}

	// RULE: same drop-not-placehold rule as Slice, applied one result at a time
	result := func(yield func(model.SearchResult) bool) {

		for searchResult := range values {

			single := []model.SearchResult{searchResult}
			builder.ruleService.LabelSearchResults(builder.session, builder.userID, single)

			if single[0].Labels.IsHidden() {
				continue
			}

			if !yield(single[0]) {
				return
			}
		}
	}

	return result, nil
}

// Count returns the number of records that match the query criteria
func (builder SearchBuilder) Count() (int64, error) {
	criteria := builder.assembleCriteria()
	return builder.searchResultService.Count(builder.session, criteria)
}

/********************************
 * Misc Helpers
 ********************************/

// assembleCriteria combines this builder's filters, text query, and the viewer's blocking rules into one expression
func (builder SearchBuilder) assembleCriteria() exp.Expression {

	result := builder.criteria

	// If there's no query, then exit early.
	if builder.textQuery == "" {
		return result
	}

	// Add criteria for #hashtags
	hashtags, remainder := parse.HashtagsAndRemainder(builder.textQuery)

	for _, hashtag := range hashtags {
		tagToken := model.ToToken(hashtag)
		result = result.AndEqual("tags", tagToken)
	}

	// Add criteria for any additional text values
	if remainder != "" {

		encoder := metaphone3.Encoder{}
		tokens := parse.Split(remainder)

		for _, token := range tokens {
			if textToken, _ := encoder.Encode(token); textToken != "" {
				result = result.AndEqual("index", textToken)
			}
		}
	}

	return result
}

// makeOptions converts this builder's state into the query options that the database expects
func (builder SearchBuilder) makeOptions() []option.Option {

	var object model.SearchResult
	result := make([]option.Option, 3, 4)

	result[0] = option.Fields(object.Fields()...)
	result[1] = builder.makeSortOption()
	result[2] = option.CaseSensitive(false)

	if builder.maxRows != 0 {
		result = append(result, option.MaxRows(builder.maxRows))
	}

	return result
}

// sortOption returns a finalized data.option for sorting the results
func (builder SearchBuilder) makeSortOption() option.Option {

	if builder.sortDirection == option.SortDirectionDescending {
		return option.SortDesc(builder.sortField)
	}

	return option.SortAsc(builder.sortField)
}
