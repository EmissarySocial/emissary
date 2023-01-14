package render

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/steranko"
)

type SliceBuilder[T model.FieldLister] struct {
	factory       Factory
	context       *steranko.Context
	service       service.ModelService
	Criteria      exp.Expression
	SortField     string
	SortDirection string
	MaxRows       int64
}

func NewSliceBuilder[T model.FieldLister](factory Factory, context *steranko.Context, service service.ModelService, criteria exp.Expression) SliceBuilder[T] {

	return SliceBuilder[T]{
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

func (builder SliceBuilder[T]) Top1() SliceBuilder[T] {
	builder.MaxRows = 1
	return builder
}

func (builder SliceBuilder[T]) Top6() SliceBuilder[T] {
	builder.MaxRows = 6
	return builder
}

func (builder SliceBuilder[T]) Top12() SliceBuilder[T] {
	builder.MaxRows = 12
	return builder
}

func (builder SliceBuilder[T]) Top30() SliceBuilder[T] {
	builder.MaxRows = 30
	return builder
}

func (builder SliceBuilder[T]) Top60() SliceBuilder[T] {
	builder.MaxRows = 60
	return builder
}
func (builder SliceBuilder[T]) Top120() SliceBuilder[T] {
	builder.MaxRows = 120
	return builder
}

func (builder SliceBuilder[T]) Top600() SliceBuilder[T] {
	builder.MaxRows = 600
	return builder
}

func (builder SliceBuilder[T]) All() SliceBuilder[T] {
	builder.MaxRows = 0
	return builder
}

func (builder SliceBuilder[T]) ByCreateDate() SliceBuilder[T] {
	builder.SortField = "journal.createDate"
	return builder
}

func (builder SliceBuilder[T]) ByDisplayName() SliceBuilder[T] {
	builder.SortField = "displayName"
	return builder
}

func (builder SliceBuilder[T]) ByExpirationDate() SliceBuilder[T] {
	builder.SortField = "expirationDate"
	return builder
}

func (builder SliceBuilder[T]) ByLabel() SliceBuilder[T] {
	builder.SortField = "label"
	return builder
}

func (builder SliceBuilder[T]) ByPublishDate() SliceBuilder[T] {
	builder.SortField = "publishDate"
	return builder
}

func (builder SliceBuilder[T]) ByRank() SliceBuilder[T] {
	builder.SortField = "rank"
	return builder
}

func (builder SliceBuilder[T]) ByUpdateDate() SliceBuilder[T] {
	builder.SortField = "journal.updateDate"
	return builder
}

func (builder SliceBuilder[T]) Reverse() SliceBuilder[T] {
	builder.SortDirection = option.SortDirectionDescending
	return builder
}

/********************************
 * ACTIONS
 ********************************/

func (builder SliceBuilder[T]) Slice() (sliceof.Type[T], error) {
	result := make(sliceof.Type[T], 0)
	err := builder.service.ObjectQuery(&result, builder.Criteria, builder.makeOptions()...)
	return result, derp.Report(err)
}

func (builder SliceBuilder[T]) Objects(actionID string) (sliceof.Type[Renderer], error) {

	var index int64

	// Query the database
	iterator, err := builder.service.ObjectList(builder.Criteria, builder.makeSortOption())

	if err != nil {
		return nil, derp.Report(derp.Wrap(err, "renderer.RenderBuilder.iteratorToSlice", "Error querying database"))
	}

	result := make(sliceof.Type[Renderer], iterator.Count())

	// Loop over each item returned
	object := builder.service.ObjectNew()

	for iterator.Next(object) {

		// Create a new renderer
		if renderer, err := NewRenderer(builder.factory, builder.context, object, actionID); err != nil {
			return result, derp.Report(derp.Wrap(err, "renderer.RenderBuilder.iteratorToSlice", "Error creating new renderer"))
		} else {
			result = append(result, renderer)
		}

		// Calculate max rows
		index = index + 1

		if builder.MaxRows > 0 {
			if index >= builder.MaxRows {
				break
			}
		}

		// Make a new object for the next renderer
		object = builder.service.ObjectNew()
	}

	if err := iterator.Error(); err != nil {
		return result, derp.Report(derp.Wrap(err, "renderer.RenderBuilder.iteratorToSlice", "Error iterating through database results"))
	}

	return result, nil
}

/********************************
 * MISC HELPERS
 ********************************/

func (builder SliceBuilder[T]) makeOptions() []option.Option {

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
func (builder SliceBuilder[T]) makeSortOption() option.Option {

	if builder.SortDirection == option.SortDirectionDescending {
		return option.SortDesc(builder.SortField)
	}

	return option.SortAsc(builder.SortField)
}
