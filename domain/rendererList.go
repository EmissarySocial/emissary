package domain

import (
	"github.com/benpate/data/expression"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ListBuilder builds slices of Renderers
type ListBuilder struct {
	streamService *service.Stream
	request       *HTTPRequest
	parentID      primitive.ObjectID
	currentID     primitive.ObjectID
	criteria      expression.Expression
	options       []option.Option
	view          string
}

// NewListBuilder generates a fully-initialized ListBuilder
func NewListBuilder(streamService *service.Stream, request *HTTPRequest, parentID primitive.ObjectID, currentID primitive.ObjectID) *ListBuilder {

	result := &ListBuilder{
		streamService: streamService,
		request:       request,
		parentID:      parentID,
		currentID:     currentID,
		options:       make([]option.Option, 0),
		view:          "list",
	}

	return result.Children()

}

/// QUERY FUNCTIONS

// Top makes the builder target the top-level streams
func (builder *ListBuilder) Top() *ListBuilder {
	builder.criteria = expression.Equal("parentId", service.ZeroObjectID())
	return builder
}

// Siblings makes the builder target other streams at the same level as the current
func (builder *ListBuilder) Siblings() *ListBuilder {
	builder.criteria = expression.Equal("parentId", builder.parentID)
	return builder
}

// Children makes the builder target child-level streams below the current
func (builder *ListBuilder) Children() *ListBuilder {
	builder.criteria = expression.Equal("parentId", builder.currentID)
	return builder
}

/// COUNT FUNCTIONS

// All returns all results that match the criteria -- USE WITH CAUTION.
func (builder *ListBuilder) All() *ListBuilder {
	return builder
}

// First10 limits the results to the first 10 streams that match the criteria
func (builder *ListBuilder) First10() *ListBuilder {
	builder.options = append(builder.options, option.MaxRows(10))
	return builder
}

// First20 limits the results to the first 20 streams that match the criteria
func (builder *ListBuilder) First20() *ListBuilder {
	builder.options = append(builder.options, option.MaxRows(20))
	return builder
}

// First30 limits the results to the first 30 streams that match the criteria
func (builder *ListBuilder) First30() *ListBuilder {
	builder.options = append(builder.options, option.MaxRows(30))
	return builder
}

// First40 limits the results to the first 40 streams that match the criteria
func (builder *ListBuilder) First40() *ListBuilder {
	builder.options = append(builder.options, option.MaxRows(40))
	return builder
}

// First50 limits the results to the first 50 streams that match the criteria
func (builder *ListBuilder) First50() *ListBuilder {
	builder.options = append(builder.options, option.MaxRows(50))
	return builder
}

// First60 limits the results to the first 60 streams that match the criteria
func (builder *ListBuilder) First60() *ListBuilder {
	builder.options = append(builder.options, option.MaxRows(60))
	return builder
}

// First70 limits the results to the first 70 streams that match the criteria
func (builder *ListBuilder) First70() *ListBuilder {
	builder.options = append(builder.options, option.MaxRows(70))
	return builder
}

// First80 limits the results to the first 80 streams that match the criteria
func (builder *ListBuilder) First80() *ListBuilder {
	builder.options = append(builder.options, option.MaxRows(80))
	return builder
}

// First90 limits the results to the first 90 streams that match the criteria
func (builder *ListBuilder) First90() *ListBuilder {
	builder.options = append(builder.options, option.MaxRows(90))
	return builder
}

// First100 limits the results to the first 100 streams that match the criteria
func (builder *ListBuilder) First100() *ListBuilder {
	builder.options = append(builder.options, option.MaxRows(100))
	return builder
}

// First200 limits the results to the first 200 streams that match the criteria
func (builder *ListBuilder) First200() *ListBuilder {
	builder.options = append(builder.options, option.MaxRows(200))
	return builder
}

// First300 limits the results to the first 300 streams that match the criteria
func (builder *ListBuilder) First300() *ListBuilder {
	builder.options = append(builder.options, option.MaxRows(300))
	return builder
}

// First400 limits the results to the first 400 streams that match the criteria
func (builder *ListBuilder) First400() *ListBuilder {
	builder.options = append(builder.options, option.MaxRows(400))
	return builder
}

// First500 limits the results to the first 500 streams that match the criteria
func (builder *ListBuilder) First500() *ListBuilder {
	builder.options = append(builder.options, option.MaxRows(500))
	return builder
}

// First600 limits the results to the first 600 streams that match the criteria
func (builder *ListBuilder) First600() *ListBuilder {
	builder.options = append(builder.options, option.MaxRows(600))
	return builder
}

// First700 limits the results to the first 700 streams that match the criteria
func (builder *ListBuilder) First700() *ListBuilder {
	builder.options = append(builder.options, option.MaxRows(700))
	return builder
}

// First800 limits the results to the first 800 streams that match the criteria
func (builder *ListBuilder) First800() *ListBuilder {
	builder.options = append(builder.options, option.MaxRows(800))
	return builder
}

// First900 limits the results to the first 900 streams that match the criteria
func (builder *ListBuilder) First900() *ListBuilder {
	builder.options = append(builder.options, option.MaxRows(900))
	return builder
}

// First1000 limits the results to the first 1000 streams that match the criteria
func (builder *ListBuilder) First1000() *ListBuilder {
	builder.options = append(builder.options, option.MaxRows(1000))
	return builder
}

//// SORT FUNCTIONS

// ByLabel sorts the builder alphabetically
func (builder *ListBuilder) ByLabel() *ListBuilder {
	builder.options = append(builder.options, option.SortAsc("label"))
	return builder
}

// ByLabelDesc sorts the builder alphabetically
func (builder *ListBuilder) ByLabelDesc() *ListBuilder {
	builder.options = append(builder.options, option.SortDesc("label"))
	return builder
}

// ByRank sorts the builder the custom sort rank
func (builder *ListBuilder) ByRank() *ListBuilder {
	builder.options = append(builder.options, option.SortAsc("rank"))
	return builder
}

// ByRankDesc sorts the builder the custom sort rank
func (builder *ListBuilder) ByRankDesc() *ListBuilder {
	builder.options = append(builder.options, option.SortDesc("rank"))
	return builder
}

// ByDate sorts the builder by publish date
func (builder *ListBuilder) ByDate() *ListBuilder {
	builder.options = append(builder.options, option.SortAsc("publishData"))
	return builder
}

// ByDateDesc sorts the builder by publish date
func (builder *ListBuilder) ByDateDesc() *ListBuilder {
	builder.options = append(builder.options, option.SortDesc("publishData"))
	return builder
}

// Query calls the database and returns a list of Renderer results
func (builder *ListBuilder) Query() ([]*Renderer, error) {

	result := make([]*Renderer, 0)

	it, err := builder.streamService.List(builder.criteria, builder.options...)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.domain.ListBuilder.Query", "Error retrieving stream list", builder)
	}

	var stream model.Stream

	for it.Next(&stream) {
		duplicate := stream
		result = append(result, NewRenderer(builder.streamService, builder.request, &duplicate, builder.view))
	}

	return result, nil
}
