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
	Criteria      exp.AndExpression
	SortField     string
	SortDirection string
	MaxRows       uint
}

/**********************
 * Query Builder
 *********************/

func (rs *ResultSet) Top6() *ResultSet {
	rs.MaxRows = 6
	return rs
}

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

func (rs *ResultSet) ByCreateDate() *ResultSet {
	rs.SortField = "journal.createDate"
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

func (rs *ResultSet) Reverse() *ResultSet {
	rs.SortDirection = "descending"
	return rs
}

func (rs *ResultSet) EqualTo(value interface{}) *ResultSet {
	rs.Criteria = rs.Criteria.And(rs.SortField, exp.OperatorEqual, rs.makeCriteriaValue(value))
	return rs
}

func (rs *ResultSet) GreaterThan(value interface{}) *ResultSet {
	rs.Criteria = rs.Criteria.And(rs.SortField, exp.OperatorGreaterThan, rs.makeCriteriaValue(value))
	return rs
}

func (rs *ResultSet) LessThan(value interface{}) *ResultSet {
	rs.Criteria = rs.Criteria.And(rs.SortField, exp.OperatorLessThan, rs.makeCriteriaValue(value))
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
			if index >= rs.MaxRows {
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

	if rs.SortDirection == option.SortDirectionDescending {
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
