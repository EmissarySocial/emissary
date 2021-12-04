package render

import (
	"github.com/benpate/convert"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/steranko"
)

type ResultSet struct {
	factory       Factory
	ctx           *steranko.Context
	Criteria      exp.Expression
	SortField     string
	SortDirection string
	MaxRows       uint
}

/**********************
 * Query Builder
 *********************/

func (rs *ResultSet) Top12() *ResultSet {
	rs.MaxRows = 12
	return rs
}

func (rs *ResultSet) Top60() *ResultSet {
	rs.MaxRows = 60
	return rs
}
func (rs *ResultSet) Top120() *ResultSet {
	rs.MaxRows = 120
	return rs
}
func (rs *ResultSet) Top600() *ResultSet {
	rs.MaxRows = 600
	return rs
}

func (rs *ResultSet) All() *ResultSet {
	rs.MaxRows = 0
	return rs
}

func (rs *ResultSet) ByLabel() *ResultSet {
	rs.SortField = "label"
	return rs
}

func (rs *ResultSet) ByPublishDate() *ResultSet {
	rs.SortField = "publishDate"
	return rs
}

func (rs *ResultSet) ByExpirationDate() *ResultSet {
	rs.SortField = "expirationDate"
	return rs
}

func (rs *ResultSet) ByRank() *ResultSet {
	rs.SortField = "rank"
	return rs
}

func (rs *ResultSet) Ascending() *ResultSet {
	rs.SortDirection = "ascending"
	return rs
}

func (rs *ResultSet) Descending() *ResultSet {
	rs.SortDirection = "descending"
	return rs
}

func (rs *ResultSet) After(value interface{}) *ResultSet {
	rs.Criteria = exp.GreaterThan(rs.SortField, rs.makeCriteriaValue(value))
	return rs
}

func (rs *ResultSet) Before(value interface{}) *ResultSet {
	rs.Criteria = exp.LessThan(rs.SortField, rs.makeCriteriaValue(value))
	return rs
}

/**********************
 * Actions
 *********************/

func (rs *ResultSet) AsView() ([]Renderer, error) {
	return rs.AsAction("view")
}

func (rs *ResultSet) AsEdit() ([]Renderer, error) {
	return rs.AsAction("edit")
}

func (rs *ResultSet) AsAction(action string) ([]Renderer, error) {

	var index uint
	var result []Renderer

	iterator, err := rs.query()

	if err != nil {
		return []Renderer{}, derp.Wrap(err, "ghost.renderer.ResultSet.makeSlice", "Error loading streams from database")
	}

	stream := new(model.Stream)

	for iterator.Next(stream) {

		// Create a new renderer
		renderer, err := NewRenderer(rs.factory, rs.ctx, stream, action)

		// If this renderer is allowed, then add it to the result set
		if err == nil {
			result = append(result, renderer)
		}

		// Calculate max rows
		index = index + 1

		if rs.MaxRows > 0 {
			if index > rs.MaxRows {
				break
			}
		}

		// Make a new stream for the next renderer
		stream = new(model.Stream)
	}

	return result, nil
}

/**********************
 * Database Query
 *********************/

// query executes the query request on the database.
func (rs *ResultSet) query() (data.Iterator, error) {
	streamService := rs.factory.Stream()
	return streamService.List(rs.Criteria, rs.makeSortOption())
}

// sortOption returns a finalized data.option for sorting the results
func (rs *ResultSet) makeSortOption() option.Option {

	if rs.SortDirection == "descending" {
		return option.SortDesc(rs.SortField)
	}

	return option.SortAsc(rs.SortField)
}

// criteriaValue converts parameters into the correct type for querying the database.
func (rs *ResultSet) makeCriteriaValue(value interface{}) interface{} {

	if rs.SortField == "label" {
		return convert.String(value)
	}

	return convert.Int(value)
}
