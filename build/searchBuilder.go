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

func (builder SearchBuilder) Top1() SearchBuilder {
	builder.maxRows = 1
	return builder
}

func (builder SearchBuilder) Top6() SearchBuilder {
	builder.maxRows = 6
	return builder
}

func (builder SearchBuilder) Top8() SearchBuilder {
	builder.maxRows = 8
	return builder
}

func (builder SearchBuilder) Top12() SearchBuilder {
	builder.maxRows = 12
	return builder
}

func (builder SearchBuilder) Top24() SearchBuilder {
	builder.maxRows = 24
	return builder
}

func (builder SearchBuilder) Top30() SearchBuilder {
	builder.maxRows = 30
	return builder
}

func (builder SearchBuilder) Top60() SearchBuilder {
	builder.maxRows = 60
	return builder
}
func (builder SearchBuilder) Top120() SearchBuilder {
	builder.maxRows = 120
	return builder
}

func (builder SearchBuilder) Top150() SearchBuilder {
	builder.maxRows = 150
	return builder
}

func (builder SearchBuilder) Top200() SearchBuilder {
	builder.maxRows = 200
	return builder
}

func (builder SearchBuilder) Top300() SearchBuilder {
	builder.maxRows = 300
	return builder
}

func (builder SearchBuilder) Top400() SearchBuilder {
	builder.maxRows = 400
	return builder
}

func (builder SearchBuilder) Top500() SearchBuilder {
	builder.maxRows = 500
	return builder
}

func (builder SearchBuilder) Top600() SearchBuilder {
	builder.maxRows = 600
	return builder
}

func (builder SearchBuilder) All() SearchBuilder {
	builder.maxRows = 0
	return builder
}

func (builder SearchBuilder) AfterRank(rank int64) SearchBuilder {
	builder.criteria = builder.criteria.AndGreaterThan("rank", rank)
	return builder
}

func (builder SearchBuilder) AfterShuffle(shuffle int64) SearchBuilder {
	builder.criteria = builder.criteria.AndGreaterThan("shuffle", shuffle)
	return builder
}

func (builder SearchBuilder) Where(field string, value any) SearchBuilder {
	builder.criteria = builder.criteria.AndEqual(field, value)
	return builder
}

func (builder SearchBuilder) WhereLT(field string, value any) SearchBuilder {
	builder.criteria = builder.criteria.AndLessThan(field, value)
	return builder
}

func (builder SearchBuilder) WhereGT(field string, value any) SearchBuilder {
	builder.criteria = builder.criteria.AndGreaterThan(field, value)
	return builder
}

func (builder SearchBuilder) WhereType(typeNames ...string) SearchBuilder {
	builder.criteria = builder.criteria.AndIn("type", typeNames)
	return builder
}

func (builder SearchBuilder) WhereTags(tags ...string) SearchBuilder {
	builder.criteria = builder.criteria.AndInAll("tags", tags)
	return builder
}

func (builder SearchBuilder) ByCreateDate() SearchBuilder {
	builder.sortField = "createDate"
	return builder
}

func (builder SearchBuilder) ByDate() SearchBuilder {
	builder.sortField = "date"
	return builder
}

func (builder SearchBuilder) ByName() SearchBuilder {
	builder.sortField = "name"
	return builder
}

func (builder SearchBuilder) ByRank() SearchBuilder {
	builder.sortField = "rank"
	return builder
}

func (builder SearchBuilder) ByShuffle() SearchBuilder {
	builder.sortField = "shuffle"
	return builder
}

func (builder SearchBuilder) By(sortField string) SearchBuilder {
	builder.sortField = sortField
	return builder
}

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
