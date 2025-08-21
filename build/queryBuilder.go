package build

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/null"
	"github.com/benpate/rosetta/sliceof"
)

type QueryBuilder[T model.FieldLister] struct {
	service       service.ModelService
	session       data.Session
	criteria      exp.Expression
	sortField     string
	sortDirection string
	maxRows       int64
	caseSensitive null.Bool
}

func NewQueryBuilder[T model.FieldLister](service service.ModelService, session data.Session, criteria exp.Expression) QueryBuilder[T] {

	return QueryBuilder[T]{
		service:       service,
		session:       session,
		criteria:      criteria,
		sortField:     "rank",
		sortDirection: "asc",
		maxRows:       60,
		caseSensitive: null.Bool{},
	}
}

/********************************
 * Query Builder
 ********************************/

func (builder QueryBuilder[T]) Top1() QueryBuilder[T] {
	builder.maxRows = 1
	return builder
}

func (builder QueryBuilder[T]) Top6() QueryBuilder[T] {
	builder.maxRows = 6
	return builder
}

func (builder QueryBuilder[T]) Top12() QueryBuilder[T] {
	builder.maxRows = 12
	return builder
}

func (builder QueryBuilder[T]) Top24() QueryBuilder[T] {
	builder.maxRows = 24
	return builder
}

func (builder QueryBuilder[T]) Top30() QueryBuilder[T] {
	builder.maxRows = 30
	return builder
}

func (builder QueryBuilder[T]) Top60() QueryBuilder[T] {
	builder.maxRows = 60
	return builder
}
func (builder QueryBuilder[T]) Top120() QueryBuilder[T] {
	builder.maxRows = 120
	return builder
}

func (builder QueryBuilder[T]) Top150() QueryBuilder[T] {
	builder.maxRows = 150
	return builder
}

func (builder QueryBuilder[T]) Top200() QueryBuilder[T] {
	builder.maxRows = 200
	return builder
}

func (builder QueryBuilder[T]) Top300() QueryBuilder[T] {
	builder.maxRows = 300
	return builder
}

func (builder QueryBuilder[T]) Top400() QueryBuilder[T] {
	builder.maxRows = 400
	return builder
}

func (builder QueryBuilder[T]) Top500() QueryBuilder[T] {
	builder.maxRows = 500
	return builder
}

func (builder QueryBuilder[T]) Top600() QueryBuilder[T] {
	builder.maxRows = 600
	return builder
}

func (builder QueryBuilder[T]) All() QueryBuilder[T] {
	builder.maxRows = 0
	return builder
}

func (builder QueryBuilder[T]) Indexable() QueryBuilder[T] {
	builder.criteria = builder.criteria.AndEqual("isIndexable", true)
	return builder
}

func (builder QueryBuilder[T]) Featured() QueryBuilder[T] {
	builder.criteria = builder.criteria.AndEqual("isFeatured", true)
	return builder
}

func (builder QueryBuilder[T]) Tags(tags ...string) QueryBuilder[T] {
	builder.criteria = builder.criteria.AndIn("tags.Name", tags)
	return builder
}

func (builder QueryBuilder[T]) Where(field string, value any) QueryBuilder[T] {
	builder.criteria = builder.criteria.AndEqual(field, value)
	return builder
}

func (builder QueryBuilder[T]) WhereGT(field string, value any) QueryBuilder[T] {
	builder.criteria = builder.criteria.AndGreaterThan(field, value)
	return builder
}

func (builder QueryBuilder[T]) WhereLT(field string, value any) QueryBuilder[T] {
	builder.criteria = builder.criteria.AndLessThan(field, value)
	return builder
}

func (builder QueryBuilder[T]) WhereIN(field string, value any) QueryBuilder[T] {
	builder.criteria = builder.criteria.AndIn(field, value)
	return builder
}

func (builder QueryBuilder[T]) WhereBeginsWith(field string, value string) QueryBuilder[T] {
	builder.criteria = builder.criteria.And(exp.BeginsWith(field, value))
	return builder
}

func (builder QueryBuilder[T]) WhereContains(field string, value string) QueryBuilder[T] {
	builder.criteria = builder.criteria.And(exp.Contains(field, value))
	return builder
}

func (builder QueryBuilder[T]) ByCreateDate() QueryBuilder[T] {
	builder.sortField = "createDate"
	return builder
}

func (builder QueryBuilder[T]) ByDisplayName() QueryBuilder[T] {
	builder.sortField = "displayName"
	return builder
}

func (builder QueryBuilder[T]) ByExpirationDate() QueryBuilder[T] {
	builder.sortField = "expirationDate"
	return builder
}

func (builder QueryBuilder[T]) ByLabel() QueryBuilder[T] {
	builder.sortField = "label"
	return builder
}

func (builder QueryBuilder[T]) ByName() QueryBuilder[T] {
	builder.sortField = "name"
	return builder
}

func (builder QueryBuilder[T]) ByPublishDate() QueryBuilder[T] {
	builder.sortField = "publishDate"
	return builder
}

func (builder QueryBuilder[T]) ByStartDate() QueryBuilder[T] {
	builder.sortField = "startDate"
	return builder
}

func (builder QueryBuilder[T]) ByRank() QueryBuilder[T] {
	builder.sortField = "rank"
	return builder
}

func (builder QueryBuilder[T]) ByRankAlt() QueryBuilder[T] {
	builder.sortField = "rankAlt"
	return builder
}

func (builder QueryBuilder[T]) ByReadDate() QueryBuilder[T] {
	builder.sortField = "readDate"
	return builder
}

func (builder QueryBuilder[T]) ByUpdateDate() QueryBuilder[T] {
	builder.sortField = "updateDate"
	return builder
}

func (builder QueryBuilder[T]) By(sortField string) QueryBuilder[T] {
	builder.sortField = sortField
	return builder
}

func (builder QueryBuilder[T]) Reverse() QueryBuilder[T] {
	builder.sortDirection = option.SortDirectionDescending
	return builder
}

func (builder QueryBuilder[T]) CaseSensitive() QueryBuilder[T] {
	builder.caseSensitive = null.NewBool(true)
	return builder
}

func (builder QueryBuilder[T]) CaseInsensitive() QueryBuilder[T] {
	builder.caseSensitive = null.NewBool(false)
	return builder
}

/********************************
 * Actions
 ********************************/

// Slice returns the results of the query as a slice of objects
func (builder QueryBuilder[T]) Slice() (sliceof.Object[T], error) {
	result := make([]T, 0)
	err := builder.service.ObjectQuery(builder.session, &result, builder.criteria, builder.makeOptions()...)
	return result, err
}

// Count returns the number of records that match the query criteria
func (builder QueryBuilder[T]) Count() (int64, error) {
	return builder.service.Count(builder.session, builder.criteria)

}

/********************************
 * Misc Helpers
 ********************************/

func (builder QueryBuilder[T]) makeOptions() []option.Option {

	var object T
	result := make([]option.Option, 2, 3)

	result[0] = option.Fields(object.Fields()...)
	result[1] = builder.makeSortOption()

	if builder.maxRows != 0 {
		result = append(result, option.MaxRows(builder.maxRows))
	}

	if builder.caseSensitive.IsPresent() {
		opt := option.CaseSensitive(builder.caseSensitive.Bool())
		result = append(result, opt)
	}

	return result
}

// sortOption returns a finalized data.option for sorting the results
func (builder QueryBuilder[T]) makeSortOption() option.Option {

	if builder.sortDirection == option.SortDirectionDescending {
		return option.SortDesc(builder.sortField)
	}

	return option.SortAsc(builder.sortField)
}
