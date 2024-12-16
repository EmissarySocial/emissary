package build

import (
	"iter"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data/option"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/sliceof"
)

type SearchBuilder struct {
	service       *service.Search
	Criteria      exp.Expression
	SortField     string
	SortDirection string
	MaxRows       int64
}

func NewSearchBuilder(service *service.Search, criteria exp.Expression) SearchBuilder {

	return SearchBuilder{
		service:       service,
		Criteria:      criteria,
		SortField:     "rank",
		SortDirection: "asc",
		MaxRows:       60,
	}
}

/********************************
 * QUERY BUILDER
 ********************************/

func (builder SearchBuilder) Top1() SearchBuilder {
	builder.MaxRows = 1
	return builder
}

func (builder SearchBuilder) Top6() SearchBuilder {
	builder.MaxRows = 6
	return builder
}

func (builder SearchBuilder) Top12() SearchBuilder {
	builder.MaxRows = 12
	return builder
}

func (builder SearchBuilder) Top30() SearchBuilder {
	builder.MaxRows = 30
	return builder
}

func (builder SearchBuilder) Top60() SearchBuilder {
	builder.MaxRows = 60
	return builder
}
func (builder SearchBuilder) Top120() SearchBuilder {
	builder.MaxRows = 120
	return builder
}

func (builder SearchBuilder) Top150() SearchBuilder {
	builder.MaxRows = 150
	return builder
}

func (builder SearchBuilder) Top200() SearchBuilder {
	builder.MaxRows = 200
	return builder
}

func (builder SearchBuilder) Top300() SearchBuilder {
	builder.MaxRows = 300
	return builder
}

func (builder SearchBuilder) Top400() SearchBuilder {
	builder.MaxRows = 400
	return builder
}

func (builder SearchBuilder) Top500() SearchBuilder {
	builder.MaxRows = 500
	return builder
}

func (builder SearchBuilder) Top600() SearchBuilder {
	builder.MaxRows = 600
	return builder
}

func (builder SearchBuilder) All() SearchBuilder {
	builder.MaxRows = 0
	return builder
}

func (builder SearchBuilder) Tags(tags ...string) SearchBuilder {
	builder.Criteria = builder.Criteria.AndIn("tags", tags)
	return builder
}

func (builder SearchBuilder) AfterRank(rank int64) SearchBuilder {
	builder.Criteria = builder.Criteria.AndGreaterThan("rank", rank)
	return builder
}

func (builder SearchBuilder) AfterShuffle(shuffle int64) SearchBuilder {
	builder.Criteria = builder.Criteria.AndGreaterThan("shuffle", shuffle)
	return builder
}

func (builder SearchBuilder) Where(field string, value any) SearchBuilder {
	builder.Criteria = builder.Criteria.AndEqual(field, value)
	return builder
}

func (builder SearchBuilder) ByCreateDate() SearchBuilder {
	builder.SortField = "createDate"
	return builder
}

func (builder SearchBuilder) ByName() SearchBuilder {
	builder.SortField = "name"
	return builder
}

func (builder SearchBuilder) ByRank() SearchBuilder {
	builder.SortField = "rank"
	return builder
}

func (builder SearchBuilder) ByShuffle() SearchBuilder {
	builder.SortField = "shuffle"
	return builder
}

func (builder SearchBuilder) By(sortField string) SearchBuilder {
	builder.SortField = sortField
	return builder
}

func (builder SearchBuilder) Reverse() SearchBuilder {
	builder.SortDirection = option.SortDirectionDescending
	return builder
}

/********************************
 * ACTIONS
 ********************************/

// Slice returns the results of the query as a slice of objects
func (builder SearchBuilder) Slice() (sliceof.Object[model.SearchResult], error) {
	return builder.service.Query(builder.Criteria, builder.makeOptions()...)
}

// Range returns the results of the query as a Go 1.23 RangeFunc
func (builder SearchBuilder) Range() (iter.Seq[model.SearchResult], error) {
	return builder.service.Range(builder.Criteria, builder.makeOptions()...)
}

// Count returns the number of records that match the query criteria
func (builder SearchBuilder) Count() (int64, error) {
	return builder.service.Count(builder.Criteria)

}

/********************************
 * MISC HELPERS
 ********************************/

func (builder SearchBuilder) makeOptions() []option.Option {

	var object model.SearchResult
	result := make([]option.Option, 2, 3)

	result[0] = option.Fields(object.Fields()...)
	result[1] = builder.makeSortOption()

	if builder.MaxRows != 0 {
		result = append(result, option.MaxRows(builder.MaxRows))
	}

	return result
}

// sortOption returns a finalized data.option for sorting the results
func (builder SearchBuilder) makeSortOption() option.Option {

	if builder.SortDirection == option.SortDirectionDescending {
		return option.SortDesc(builder.SortField)
	}

	return option.SortAsc(builder.SortField)
}
