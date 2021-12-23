package render

import (
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/steranko"
)

type QueryBuilder struct {
	factory       Factory
	ctx           *steranko.Context
	service       ModelService
	Criteria      exp.Expression
	SortField     string
	SortDirection string
	MaxRows       uint
}

func NewQueryBuilder(factory Factory, ctx *steranko.Context, service ModelService, criteria exp.Expression) QueryBuilder {

	return QueryBuilder{
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

func (qb QueryBuilder) Top1() QueryBuilder {
	qb.MaxRows = 1
	return qb
}

func (qb QueryBuilder) Top6() QueryBuilder {
	qb.MaxRows = 6
	return qb
}

func (qb QueryBuilder) Top12() QueryBuilder {
	qb.MaxRows = 12
	return qb
}

func (qb QueryBuilder) Top60() QueryBuilder {
	qb.MaxRows = 60
	return qb
}
func (qb QueryBuilder) Top120() QueryBuilder {
	qb.MaxRows = 120
	return qb
}

func (qb QueryBuilder) Top600() QueryBuilder {
	qb.MaxRows = 600
	return qb
}

func (qb QueryBuilder) All() QueryBuilder {
	qb.MaxRows = 0
	return qb
}

func (qb QueryBuilder) ByLabel() QueryBuilder {
	qb.SortField = "label"
	return qb
}

func (qb QueryBuilder) ByDisplayName() QueryBuilder {
	qb.SortField = "displayName"
	return qb
}

func (qb QueryBuilder) ByCreateDate() QueryBuilder {
	qb.SortField = "journal.createDate"
	return qb
}

func (qb QueryBuilder) ByPublishDate() QueryBuilder {
	qb.SortField = "publishDate"
	return qb
}

func (qb QueryBuilder) ByExpirationDate() QueryBuilder {
	qb.SortField = "expirationDate"
	return qb
}

func (qb QueryBuilder) ByRank() QueryBuilder {
	qb.SortField = "rank"
	return qb
}

func (qb QueryBuilder) Reverse() QueryBuilder {
	qb.SortDirection = option.SortDirectionDescending
	return qb
}

/********************************
 * ACTIONS
 ********************************/

func (qb QueryBuilder) View() (List, error) {
	return qb.Action("view")
}

func (qb QueryBuilder) Edit() (List, error) {
	return qb.Action("edit")
}

func (qb QueryBuilder) Action(action string) (List, error) {

	iterator, err := qb.query()

	if err != nil {
		return nil, derp.Wrap(err, "ghost.renderer.QueryBuilder.makeSlice", "Error loading streams from database")
	}

	return qb.iteratorToSlice(iterator, qb.MaxRows, action), nil
}

/********************************
 * DATABASE QUERIES
 ********************************/

// query executes the query request on the database.
func (qb QueryBuilder) query() (data.Iterator, error) {
	return qb.service.ObjectList(qb.Criteria, qb.makeSortOption())
}

// sortOption returns a finalized data.option for sorting the results
func (qb QueryBuilder) makeSortOption() option.Option {

	if qb.SortDirection == option.SortDirectionDescending {
		return option.SortDesc(qb.SortField)
	}

	return option.SortAsc(qb.SortField)
}

/********************************
 * MISC HELPERS
 ********************************/

// iteratorToSlice consumes a data.Iterator and generates a slice of Renderer objects.
func (qb QueryBuilder) iteratorToSlice(iterator data.Iterator, maxRows uint, action string) List {

	var index uint

	result := make(List, 0)
	object := qb.service.ObjectNew()

	for iterator.Next(object) {

		// Create a new renderer
		renderer, err := NewRenderer(qb.factory, qb.ctx, object, action)

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

	return result
}

func (qb QueryBuilder) debug() datatype.Map {

	return datatype.Map{
		"Criteria":      qb.Criteria,
		"SortField":     qb.SortField,
		"SortDirection": qb.SortDirection,
		"MaxRows":       qb.MaxRows,
	}
}
