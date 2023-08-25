package render

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data/option"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/sliceof"
)

type QueryBuilder[T model.FieldLister] struct {
	service       service.ModelService
	Criteria      exp.Expression
	SortField     string
	SortDirection string
	MaxRows       int64
}

func NewQueryBuilder[T model.FieldLister](service service.ModelService, criteria exp.Expression) QueryBuilder[T] {

	return QueryBuilder[T]{
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

func (builder QueryBuilder[T]) Top1() QueryBuilder[T] {
	builder.MaxRows = 1
	return builder
}

func (builder QueryBuilder[T]) Top6() QueryBuilder[T] {
	builder.MaxRows = 6
	return builder
}

func (builder QueryBuilder[T]) Top12() QueryBuilder[T] {
	builder.MaxRows = 12
	return builder
}

func (builder QueryBuilder[T]) Top30() QueryBuilder[T] {
	builder.MaxRows = 30
	return builder
}

func (builder QueryBuilder[T]) Top60() QueryBuilder[T] {
	builder.MaxRows = 60
	return builder
}
func (builder QueryBuilder[T]) Top120() QueryBuilder[T] {
	builder.MaxRows = 120
	return builder
}

func (builder QueryBuilder[T]) Top600() QueryBuilder[T] {
	builder.MaxRows = 600
	return builder
}

func (builder QueryBuilder[T]) All() QueryBuilder[T] {
	builder.MaxRows = 0
	return builder
}

func (builder QueryBuilder[T]) ByCreateDate() QueryBuilder[T] {
	builder.SortField = "createDate"
	return builder
}

func (builder QueryBuilder[T]) ByDisplayName() QueryBuilder[T] {
	builder.SortField = "displayName"
	return builder
}

func (builder QueryBuilder[T]) ByExpirationDate() QueryBuilder[T] {
	builder.SortField = "expirationDate"
	return builder
}

func (builder QueryBuilder[T]) ByLabel() QueryBuilder[T] {
	builder.SortField = "label"
	return builder
}

func (builder QueryBuilder[T]) ByPublishDate() QueryBuilder[T] {
	builder.SortField = "publishDate"
	return builder
}

func (builder QueryBuilder[T]) ByRank() QueryBuilder[T] {
	builder.SortField = "rank"
	return builder
}

func (builder QueryBuilder[T]) ByReadDate() QueryBuilder[T] {
	builder.SortField = "readDate"
	return builder
}

func (builder QueryBuilder[T]) ByUpdateDate() QueryBuilder[T] {
	builder.SortField = "updateDate"
	return builder
}

func (builder QueryBuilder[T]) Reverse() QueryBuilder[T] {
	builder.SortDirection = option.SortDirectionDescending
	return builder
}

/********************************
 * ACTIONS
 ********************************/

func (builder QueryBuilder[T]) Slice() (sliceof.Object[T], error) {
	result := make([]T, 0)
	err := builder.service.ObjectQuery(&result, builder.Criteria, builder.makeOptions()...)
	return result, err
}

/********************************
 * MISC HELPERS
 ********************************/

func (builder QueryBuilder[T]) makeOptions() []option.Option {

	var object T
	result := make([]option.Option, 2, 3)

	result[0] = option.Fields(object.Fields()...)
	result[1] = builder.makeSortOption()

	if builder.MaxRows != 0 {
		result = append(result, option.MaxRows(builder.MaxRows))
	}

	return result
}

// sortOption returns a finalized data.option for sorting the results
func (builder QueryBuilder[T]) makeSortOption() option.Option {

	if builder.SortDirection == option.SortDirectionDescending {
		return option.SortDesc(builder.SortField)
	}

	return option.SortAsc(builder.SortField)
}
