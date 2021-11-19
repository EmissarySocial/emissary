package render

import (
	"html/template"
	"net/http"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/list"
	"github.com/benpate/steranko"
	"github.com/davecgh/go-spew/spew"
)

// Renderer wraps a model.Stream object and provides functions that make it easy to render an HTML template with it.
type Renderer struct {
	factory  Factory           // Internal interface to the domain.Factory
	ctx      *steranko.Context // Contains request context and authentication data.
	template *model.Template   // Template that the Stream uses
	action   model.Action      // Action being executed
	stream   model.Stream      // Stream to be displayed
}

// NewRenderer creates a new object that can generate HTML for a specific stream/view
func NewRenderer(factory Factory, ctx *steranko.Context, stream model.Stream, actionID string) (Renderer, model.Action, error) {

	// Try to load the Template associated with this Stream
	templateService := factory.Template()
	template, err := templateService.Load(stream.TemplateID)

	spew.Dump(template.TemplateID)
	spew.Dump(template.Actions)

	if err != nil {
		return Renderer{}, model.Action{}, derp.Wrap(err, "ghost.render.NewRenderer", "Cannot load Stream Template", stream)
	}

	// Try to find requested Action
	action, ok := template.Action(actionID)

	if !ok {
		return Renderer{}, model.Action{}, derp.New(http.StatusBadRequest, "ghost.render.NewRenderer", "Invalid action")
	}

	// Verify user's authorization to perform this Action on this Stream
	authorization := getAuthorization(ctx)

	if !action.UserCan(&stream, authorization) {
		return Renderer{}, model.Action{}, derp.New(http.StatusForbidden, "ghost.render.NewRenderer", "Forbidden")
	}

	// Success.  Populate Renderer
	result := Renderer{
		factory:  factory,
		ctx:      ctx,
		stream:   stream,
		template: template,
		action:   action,
	}

	return result, action, nil
}

/*******************************************
 * DATA ACCESSORS
 *******************************************/

func (w Renderer) URL() string {
	return w.ctx.Request().URL.RequestURI()
}

// StreamID returns the unique ID for the stream being rendered
func (w Renderer) StreamID() string {
	return w.stream.StreamID.Hex()
}

// StateID returns the current state of the stream being renderer
func (w Renderer) StateID() string {
	return w.stream.StateID
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
func (w Renderer) PublishDate() int64 {
	return w.stream.PublishDate
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

func (w Renderer) IsCurrentStream() bool {
	return w.stream.Token == list.Head(w.ctx.Path(), "/")
}

func (w Renderer) Roles() []string {
	authorization := getAuthorization(w.ctx)
	return w.stream.Roles(authorization)
}

/*******************************************
 * REQUEST INFO
 *******************************************/

// Returns the request parameter
func (w Renderer) QueryParam(param string) string {
	return w.ctx.QueryParam(param)
}

/*******************************************
 * RELATIONSHIPS TO OTHER STREAMS
 *******************************************/

// Parent returns a Stream containing the parent of the current stream
func (w Renderer) Parent(actionID string) (Renderer, error) {

	var parent model.Stream
	var result Renderer

	streamService := w.factory.Stream()

	if err := streamService.LoadParent(&w.stream, &parent); err != nil {
		return result, derp.Wrap(err, "ghost.renderer.Renderer.Parent", "Error loading Parent")
	}

	renderer, _, err := NewRenderer(w.factory, w.ctx, parent, actionID)

	if err != nil {
		return renderer, derp.Wrap(err, "ghost.renderer.Renderer.Parent", "Unable to create new Renderer")
	}

	return renderer, nil
}

// Children returns an array of Streams containing all of the child elements of the current stream
func (w Renderer) Children(viewID string) ([]Renderer, error) {

	streamService := w.factory.Stream()

	iterator, err := streamService.ListByParent(w.stream.StreamID)

	if err != nil {
		return nil, derp.Report(derp.Wrap(err, "ghost.renderer.Renderer.Children", "Error loading child streams", w.stream))
	}

	return iteratorToSlice(w.factory, w.ctx, iterator, viewID)
}

// TopLevel returns an array of Streams that have a Zero ParentID
func (w Renderer) TopFolders(viewID string) ([]Renderer, error) {

	streamService := w.factory.Stream()

	iterator, err := streamService.ListTopFolders()

	if err != nil {
		return nil, derp.Report(derp.Wrap(err, "ghost.renderer.Renderer.Children", "Error loading child streams", w.stream))
	}

	return iteratorToSlice(w.factory, w.ctx, iterator, viewID)
}

/////////////////////
// PERMISSIONS METHODS

// IsSignedIn returns TRUE if the user is signed in
func (w Renderer) IsAuthenticated() bool {
	return getAuthorization(w.ctx).IsAuthenticated()
}

// CanView returns TRUE if this Request is authorized to access this stream/view
func (w Renderer) UserCan(actionID string) bool {

	action, ok := w.template.Action(actionID)

	if !ok {
		return false
	}

	authorization := getAuthorization(w.ctx)

	return action.UserCan(&w.stream, authorization)
}

///////////////////////////
// HELPER FUNCTIONS

// iteratorToSlice converts a data.Iterator of Streams into a slice of Streams
func iteratorToSlice(factory Factory, sterankoContext *steranko.Context, iterator data.Iterator, actionID string) ([]Renderer, error) {

	var stream model.Stream

	result := make([]Renderer, 0, iterator.Count())

	for iterator.Next(&stream) {
		if renderer, _, err := NewRenderer(factory, sterankoContext, stream, actionID); err == nil {
			result = append(result, renderer)
		}

		// Overwrite stream so that no values leak from one record to the other. grrrr.
		stream = model.Stream{}
	}

	return result, nil
}
