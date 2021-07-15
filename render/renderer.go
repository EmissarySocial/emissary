package render

import (
	"html/template"
	"time"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/steranko"
)

// Renderer wraps a model.Stream object and provides functions that make it easy to render an HTML template with it.
type Renderer struct {
	factory Factory           // Internal interface to the domain.Factory
	ctx     *steranko.Context // Contains request context and authentication data.
	stream  model.Stream      // Stream to be displayed
	action  Action            // Action to perform when this stream is rendered
}

// NewRenderer creates a new object that can generate HTML for a specific stream/view
func NewRenderer(factory Factory, sterankoContext *steranko.Context, stream model.Stream, actionID string) (Renderer, error) {

	authorization := getAuthorization(sterankoContext)

	action, err := NewAction(factory, &stream, &authorization, actionID)

	if err != nil {
		return Renderer{}, derp.Wrap(err, "ghost.render.NewRenderer", "Cannot parse Action", stream, actionID)
	}

	result := Renderer{
		factory: factory,
		ctx:     sterankoContext,
		stream:  stream,
		action:  action,
	}

	return result, nil
}

////////////////////////////////
// ACCESSORS FOR THIS STREAM

func (w Renderer) URL() string {
	return w.ctx.Request().URL.RequestURI()
}

// StreamID returns the unique ID for the stream being rendered
func (w Renderer) StreamID() string {
	return w.stream.StreamID.Hex()
}

func (w Renderer) TemplateID() string {
	return w.stream.TemplateID
}

// Token returns the unique URL token for the stream being rendered
func (w Renderer) Token() string {
	return w.stream.Token
}

// Label returns the Label for the stream being rendered
func (w Renderer) Label() string {
	return w.stream.Label
}

// Description returns the description of the stream being rendered
func (w Renderer) Description() string {
	return w.stream.Description
}

// Returns the body content as an HTML template
func (w Renderer) Content() template.HTML {
	result := w.stream.Content.View()
	return template.HTML(result)
}

// Returns editable HTML for the body content (requires `editable` flat)
func (w Renderer) ContentEditor() template.HTML {
	result := w.stream.Content.Edit("/" + w.Token() + "/draft")
	return template.HTML(result)
}

// PublishDate returns the PublishDate of the stream being rendered
func (w Renderer) PublishDate() time.Time {
	return time.Unix(w.stream.PublishDate, 0)
}

// ThumbnailImage returns the thumbnail image URL of the stream being rendered
func (w Renderer) ThumbnailImage() string {
	return w.stream.ThumbnailImage
}

// Data returns the custom data map of the stream being rendered
func (w Renderer) Data() map[string]interface{} {
	return w.stream.Data
}

// Tags returns the tags of the stream being rendered
func (w Renderer) Tags() []string {
	return w.stream.Tags
}

// HasParent returns TRUE if the stream being rendered has a parend objec
func (w Renderer) HasParent() bool {
	return w.stream.HasParent()
}

////////////////////////////////
// REQUEST INFO

// Returns the request parameter
func (w Renderer) QueryParam(param string) string {
	return w.ctx.QueryParam(param)
}

////////////////////////////////
// RELATIONSHIPS TO OTHER STREAMS

// Parent returns a Stream containing the parent of the current stream
func (w Renderer) Parent(actionID string) (Renderer, error) {

	var parent model.Stream
	var result Renderer

	streamService := w.factory.Stream()

	if err := streamService.LoadParent(&w.stream, &parent); err != nil {
		return result, derp.Wrap(err, "ghost.service.Renderer.Parent", "Error loading Parent")
	}

	return NewRenderer(w.factory, w.ctx, parent, actionID)
}

// Children returns an array of Streams containing all of the child elements of the current stream
func (w Renderer) Children(viewID string) ([]Renderer, error) {

	streamService := w.factory.Stream()

	iterator, err := streamService.ListByParent(w.stream.StreamID)

	if err != nil {
		return nil, derp.Report(derp.Wrap(err, "ghost.service.Renderer.Children", "Error loading child streams", w.stream))
	}

	return iteratorToSlice(w.factory, w.ctx, iterator, viewID)
}

// TopLevel returns an array of Streams that have a Zero ParentID
func (w Renderer) TopLevel(viewID string) ([]Renderer, error) {

	streamService := w.factory.Stream()

	iterator, err := streamService.ListTopFolders()

	if err != nil {
		return nil, derp.Report(derp.Wrap(err, "ghost.service.Renderer.Children", "Error loading child streams", w.stream))
	}

	return iteratorToSlice(w.factory, w.ctx, iterator, viewID)
}

///////////////////////////////
/// RENDERING METHODS

// Render generates an HTML output for a stream/view combination.
func (w Renderer) Render() (template.HTML, error) {

	result, err := w.action.Get(w)

	if err != nil {
		return template.HTML(""), derp.Report(derp.Wrap(err, "ghost.render.Renderer.Render", "Error generating HTML"))
	}

	return template.HTML(result), nil
}

/////////////////////
// PERMISSIONS METHODS

// CanView returns TRUE if this Request is authorized to access this stream/view
func (w Renderer) UserCan(actionID string) bool {
	authorization := getAuthorization(w.ctx)
	return w.action.UserCan(&w.stream, &authorization)
}

///////////////////////////
// HELPER FUNCTIONS

// iteratorToSlice converts a data.Iterator of Streams into a slice of Streams
func iteratorToSlice(factory Factory, sterankoContext *steranko.Context, iterator data.Iterator, actionID string) ([]Renderer, error) {

	var stream model.Stream

	result := make([]Renderer, 0, iterator.Count())

	for iterator.Next(&stream) {
		if renderer, err := NewRenderer(factory, sterankoContext, stream, actionID); err == nil {
			result = append(result, renderer)
		}

		stream = model.Stream{}

	}

	return result, nil
}
