package build

import (
	"iter"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/parse"
	"github.com/benpate/data/option"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/sliceof"
	"github.com/davecgh/go-spew/spew"
	"github.com/dlclark/metaphone3"
)

type SearchBuilder struct {
	searchTagService    *service.SearchTag
	searchResultService *service.SearchResult
	criteria            exp.Expression
	textQuery           string
	sortField           string
	sortDirection       string
	maxRows             int64
}

func NewSearchBuilder(searchTagService *service.SearchTag, searchResultService *service.SearchResult, criteria exp.Expression, textQuery string) SearchBuilder {

	return SearchBuilder{
		searchTagService:    searchTagService,
		searchResultService: searchResultService,
		criteria:            criteria,
		textQuery:           textQuery,
		sortField:           "rank",
		sortDirection:       "asc",
		maxRows:             60,
	}
}

/********************************
 * CUSTOM QUERY ARGUMENTS
 ********************************/

/********************************
 * QUERY BUILDER
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

func (builder SearchBuilder) ByStartDate() SearchBuilder {
	builder.sortField = "startDate"
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
 * ACTIONS
 ********************************/

// Slice returns the results of the query as a slice of objects
func (builder SearchBuilder) Slice() (sliceof.Object[model.SearchResult], error) {
	criteria := builder.assembleCriteria()
	return builder.searchResultService.Query(criteria, builder.makeOptions()...)
}

// Range returns the results of the query as a Go 1.23 RangeFunc
func (builder SearchBuilder) Range() (iter.Seq[model.SearchResult], error) {
	criteria := builder.assembleCriteria()
	return builder.searchResultService.Range(criteria, builder.makeOptions()...)
}

// Count returns the number of records that match the query criteria
func (builder SearchBuilder) Count() (int64, error) {
	criteria := builder.assembleCriteria()
	return builder.searchResultService.Count(criteria)
}

/********************************
 * MISC HELPERS
 ********************************/

func (builder SearchBuilder) assembleCriteria() exp.Expression {

	result := builder.criteria
	encoder := metaphone3.Encoder{}
	tokens := parse.Split(builder.textQuery)

	for _, token := range tokens {
		tagToken := model.ToToken(token)

		textToken, _ := encoder.Encode(token)

		if textToken == "" {
			result = result.AndEqual("tags", tagToken)
			continue
		}

		result = result.And(exp.Or(
			exp.Equal("tags", tagToken),
			exp.Equal("index", textToken),
		))
	}

	spew.Dump("SearchCriteria", result)

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
