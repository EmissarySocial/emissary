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

// QueryBuilder is a chainable query API that templates use to select model objects
type QueryBuilder[T model.FieldLister] struct {
	service       service.ModelService
	session       data.Session
	criteria      exp.Expression
	sortField     string
	sortDirection string
	maxRows       int64
	caseSensitive null.Bool
}

// NewQueryBuilder returns a QueryBuilder seeded with the provided criteria, sorted by rank, capped at 60 rows
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

// Top1 limits this query to 1 row
func (builder QueryBuilder[T]) Top1() QueryBuilder[T] {
	builder.maxRows = 1
	return builder
}

// Top6 limits this query to 6 rows
func (builder QueryBuilder[T]) Top6() QueryBuilder[T] {
	builder.maxRows = 6
	return builder
}

// Top12 limits this query to 12 rows
func (builder QueryBuilder[T]) Top12() QueryBuilder[T] {
	builder.maxRows = 12
	return builder
}

// Top24 limits this query to 24 rows
func (builder QueryBuilder[T]) Top24() QueryBuilder[T] {
	builder.maxRows = 24
	return builder
}

// Top30 limits this query to 30 rows
func (builder QueryBuilder[T]) Top30() QueryBuilder[T] {
	builder.maxRows = 30
	return builder
}

// Top60 limits this query to 60 rows
func (builder QueryBuilder[T]) Top60() QueryBuilder[T] {
	builder.maxRows = 60
	return builder
}

// Top120 limits this query to 120 rows
func (builder QueryBuilder[T]) Top120() QueryBuilder[T] {
	builder.maxRows = 120
	return builder
}

// Top150 limits this query to 150 rows
func (builder QueryBuilder[T]) Top150() QueryBuilder[T] {
	builder.maxRows = 150
	return builder
}

// Top200 limits this query to 200 rows
func (builder QueryBuilder[T]) Top200() QueryBuilder[T] {
	builder.maxRows = 200
	return builder
}

// Top300 limits this query to 300 rows
func (builder QueryBuilder[T]) Top300() QueryBuilder[T] {
	builder.maxRows = 300
	return builder
}

// Top400 limits this query to 400 rows
func (builder QueryBuilder[T]) Top400() QueryBuilder[T] {
	builder.maxRows = 400
	return builder
}

// Top500 limits this query to 500 rows
func (builder QueryBuilder[T]) Top500() QueryBuilder[T] {
	builder.maxRows = 500
	return builder
}

// Top600 limits this query to 600 rows
func (builder QueryBuilder[T]) Top600() QueryBuilder[T] {
	builder.maxRows = 600
	return builder
}

// All removes the row limit from this query
func (builder QueryBuilder[T]) All() QueryBuilder[T] {
	builder.maxRows = 0
	return builder
}

// Indexable limits this query to records that may be indexed by search engines
func (builder QueryBuilder[T]) Indexable() QueryBuilder[T] {
	builder.criteria = builder.criteria.AndEqual("isIndexable", true)
	return builder
}

// Featured limits this query to records that have been marked as featured
func (builder QueryBuilder[T]) Featured() QueryBuilder[T] {
	builder.criteria = builder.criteria.AndEqual("isFeatured", true)
	return builder
}

// Tags limits this query to records carrying at least one of the provided tags
func (builder QueryBuilder[T]) Tags(tags ...string) QueryBuilder[T] {
	builder.criteria = builder.criteria.AndIn("tags.Name", tags)
	return builder
}

// Where limits this query to records whose field equals the provided value
func (builder QueryBuilder[T]) Where(field string, value any) QueryBuilder[T] {
	builder.criteria = builder.criteria.AndEqual(field, value)
	return builder
}

// WhereGT limits this query to records whose field is greater than the provided value
func (builder QueryBuilder[T]) WhereGT(field string, value any) QueryBuilder[T] {
	builder.criteria = builder.criteria.AndGreaterThan(field, value)
	return builder
}

// WhereLT limits this query to records whose field is less than the provided value
func (builder QueryBuilder[T]) WhereLT(field string, value any) QueryBuilder[T] {
	builder.criteria = builder.criteria.AndLessThan(field, value)
	return builder
}

// WhereIN limits this query to records whose field matches one of the provided values
func (builder QueryBuilder[T]) WhereIN(field string, value any) QueryBuilder[T] {
	builder.criteria = builder.criteria.AndIn(field, value)
	return builder
}

// WhereBeginsWith limits this query to records whose field starts with the provided value
func (builder QueryBuilder[T]) WhereBeginsWith(field string, value string) QueryBuilder[T] {
	builder.criteria = builder.criteria.And(exp.BeginsWith(field, value))
	return builder
}

// WhereContains limits this query to records whose field contains the provided value
func (builder QueryBuilder[T]) WhereContains(field string, value string) QueryBuilder[T] {
	builder.criteria = builder.criteria.And(exp.Contains(field, value))
	return builder
}

// ByCreateDate sorts this query by creation date
func (builder QueryBuilder[T]) ByCreateDate() QueryBuilder[T] {
	builder.sortField = "createDate"
	return builder
}

// ByDisplayName sorts this query by display name
func (builder QueryBuilder[T]) ByDisplayName() QueryBuilder[T] {
	builder.sortField = "displayName"
	return builder
}

// ByExpirationDate sorts this query by expiration date
func (builder QueryBuilder[T]) ByExpirationDate() QueryBuilder[T] {
	builder.sortField = "expirationDate"
	return builder
}

// ByLabel sorts this query by label
func (builder QueryBuilder[T]) ByLabel() QueryBuilder[T] {
	builder.sortField = "label"
	return builder
}

// ByName sorts this query by name
func (builder QueryBuilder[T]) ByName() QueryBuilder[T] {
	builder.sortField = "name"
	return builder
}

// ByPublishDate sorts this query by publish date
func (builder QueryBuilder[T]) ByPublishDate() QueryBuilder[T] {
	builder.sortField = "publishDate"
	return builder
}

// ByStartDate sorts this query by start date
func (builder QueryBuilder[T]) ByStartDate() QueryBuilder[T] {
	builder.sortField = "startDate"
	return builder
}

// ByRank sorts this query by rank
func (builder QueryBuilder[T]) ByRank() QueryBuilder[T] {
	builder.sortField = "rank"
	return builder
}

// ByRankAlt sorts this query by alternate rank
func (builder QueryBuilder[T]) ByRankAlt() QueryBuilder[T] {
	builder.sortField = "rankAlt"
	return builder
}

// ByReadDate sorts this query by read date
func (builder QueryBuilder[T]) ByReadDate() QueryBuilder[T] {
	builder.sortField = "readDate"
	return builder
}

// ByUpdateDate sorts this query by update date
func (builder QueryBuilder[T]) ByUpdateDate() QueryBuilder[T] {
	builder.sortField = "updateDate"
	return builder
}

// By sorts this query by the named field
func (builder QueryBuilder[T]) By(sortField string) QueryBuilder[T] {
	builder.sortField = sortField
	return builder
}

// Reverse flips this query into descending order
func (builder QueryBuilder[T]) Reverse() QueryBuilder[T] {
	builder.sortDirection = option.SortDirectionDescending
	return builder
}

// CaseSensitive makes this query's string comparisons case-sensitive
func (builder QueryBuilder[T]) CaseSensitive() QueryBuilder[T] {
	builder.caseSensitive = null.NewBool(true)
	return builder
}

// CaseInsensitive makes this query's string comparisons case-insensitive
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

// makeOptions converts this builder's state into the query options that the database expects
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
