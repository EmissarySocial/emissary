package render

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/steranko"
)

type RenderBuilder struct {
	factory       Factory
	ctx           *steranko.Context
	service       service.ModelService
	Criteria      exp.Expression
	SortField     string
	SortDirection string
	MaxRows       uint
}

func NewRenderBuilder(factory Factory, ctx *steranko.Context, service service.ModelService, criteria exp.Expression) RenderBuilder {

	return RenderBuilder{
		factory:       factory,
		ctx:           ctx,
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

func (qb RenderBuilder) Top1() RenderBuilder {
	qb.MaxRows = 1
	return qb
}

func (qb RenderBuilder) Top6() RenderBuilder {
	qb.MaxRows = 6
	return qb
}

func (qb RenderBuilder) Top12() RenderBuilder {
	qb.MaxRows = 12
	return qb
}

func (qb RenderBuilder) Top30() RenderBuilder {
	qb.MaxRows = 30
	return qb
}

func (qb RenderBuilder) Top60() RenderBuilder {
	qb.MaxRows = 60
	return qb
}
func (qb RenderBuilder) Top120() RenderBuilder {
	qb.MaxRows = 120
	return qb
}

func (qb RenderBuilder) Top600() RenderBuilder {
	qb.MaxRows = 600
	return qb
}

func (qb RenderBuilder) All() RenderBuilder {
	qb.MaxRows = 0
	return qb
}

func (qb RenderBuilder) ByCreateDate() RenderBuilder {
	qb.SortField = "journal.createDate"
	return qb
}

func (qb RenderBuilder) ByDisplayName() RenderBuilder {
	qb.SortField = "displayName"
	return qb
}

func (qb RenderBuilder) ByExpirationDate() RenderBuilder {
	qb.SortField = "expirationDate"
	return qb
}

func (qb RenderBuilder) ByLabel() RenderBuilder {
	qb.SortField = "label"
	return qb
}

func (qb RenderBuilder) ByPublishDate() RenderBuilder {
	qb.SortField = "publishDate"
	return qb
}

func (qb RenderBuilder) ByRank() RenderBuilder {
	qb.SortField = "rank"
	return qb
}

func (qb RenderBuilder) ByUpdateDate() RenderBuilder {
	qb.SortField = "journal.updateDate"
	return qb
}

func (qb RenderBuilder) Reverse() RenderBuilder {
	qb.SortDirection = option.SortDirectionDescending
	return qb
}

/********************************
 * ACTIONS
 ********************************/

func (qb RenderBuilder) View() (List, error) {
	return qb.Action("view")
}

func (qb RenderBuilder) Action(action string) (List, error) {

	iterator, err := qb.query()

	if err != nil {
		return nil, derp.Wrap(err, "renderer.RenderBuilder.makeSlice", "Error loading streams from database")
	}

	return qb.iteratorToSlice(iterator, qb.MaxRows, action)
}

/********************************
 * DATABASE QUERIES
 ********************************/

// query executes the query request on the database.
func (qb RenderBuilder) query() (data.Iterator, error) {
	return qb.service.ObjectList(qb.Criteria, qb.makeSortOption())
}

// sortOption returns a finalized data.option for sorting the results
func (qb RenderBuilder) makeSortOption() option.Option {

	if qb.SortDirection == option.SortDirectionDescending {
		return option.SortDesc(qb.SortField)
	}

	return option.SortAsc(qb.SortField)
}

/********************************
 * MISC HELPERS
 ********************************/

type Errorer interface {
	Error() error
}

// iteratorToSlice consumes a data.Iterator and generates a slice of Renderer objects.
func (qb RenderBuilder) iteratorToSlice(iterator data.Iterator, maxRows uint, action string) (List, error) {

	var index uint
	var errorGroup error

	result := make(List, 0)
	object := qb.service.ObjectNew()

	for iterator.Next(object) {

		// Create a new renderer
		renderer, err := NewRenderer(qb.factory, qb.ctx, object, action)
		errorGroup = derp.Append(errorGroup, err)

		// If this renderer is allowed, then add it to the result set
		if err == nil {
			result = append(result, renderer)
		}

		// Calculate max rows
		index = index + 1

		if maxRows > 0 {
			if index >= maxRows {
				break
			}
		}

		// Make a new object for the next renderer
		object = qb.service.ObjectNew()
	}

	if err := iterator.Error(); err != nil {
		return result, derp.Wrap(err, "renderer.RenderBuilder.iteratorToSlice", "Error iterating through database results")
	}

	if errorGroup != nil {
		return result, errorGroup
	}

	return result, nil
}
