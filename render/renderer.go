package render

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/benpate/data"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/list"
	"github.com/benpate/steranko"
)

// Renderer wraps a model.Stream object and provides functions that make it easy to render an HTML template with it.
type Renderer struct {
	factory  Factory           // Factory interface is required for locating other services.
	ctx      *steranko.Context // Contains request context and authentication data.
	template *model.Template   // Template that the Stream uses
	action   *model.Action     // Action being executed
	stream   *model.Stream     // Stream to be displayed
	inputs   datatype.Map      // Body parameters posted by client
}

// NewRenderer creates a new object that can generate HTML for a specific stream/view
func NewRenderer(factory Factory, ctx *steranko.Context, stream *model.Stream, actionID string) (Renderer, error) {

	// Try to load the Template associated with this Stream
	templateService := factory.Template()
	template, err := templateService.Load(stream.TemplateID)

	if err != nil {
		return Renderer{}, derp.Wrap(err, "ghost.render.NewRenderer", "Cannot load Stream Template", stream)
	}

	// Try to find requested Action
	action, ok := template.Action(actionID)

	if !ok {
		return Renderer{}, derp.New(http.StatusBadRequest, "ghost.render.NewRenderer", "Invalid action")
	}

	// Verify user's authorization to perform this Action on this Stream
	authorization := getAuthorization(ctx)

	if !action.UserCan(stream, authorization) {
		return Renderer{}, derp.New(http.StatusForbidden, "ghost.render.NewRenderer", "Forbidden")
	}

	// Success.  Populate Renderer
	return Renderer{
		factory:  factory,
		ctx:      ctx,
		stream:   stream,
		template: template,
		action:   &action,
	}, nil
}

/*******************************************
 * RENDER FUNCTION
 *******************************************/

// Render generates the string value for this Renderer
func (w Renderer) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	if err := DoPipeline(&w, &buffer, w.action.Steps, ActionMethodGet); err != nil {
		return "", derp.Report(derp.Wrap(err, "ghost.render.Renderer.Render", "Error generating HTML"))
	}

	// Success!
	return template.HTML(buffer.String()), nil

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

// StateID returns the current state of the stream being rendered
func (w Renderer) StateID() string {
	return w.stream.StateID
}

// TemplateID returns the name of the template being used
func (w Renderer) TemplateID() string {
	return w.stream.TemplateID
}

// ActionID returns the name of the action being performed
func (w Renderer) ActionID() string {
	return w.action.ActionID
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

// SourceURL returns the thumbnail image URL of the stream being rendered
func (w Renderer) SourceURL() string {
	return w.stream.SourceURL
}

// Data returns the custom data map of the stream being rendered
func (w Renderer) Data(value string) interface{} {
	return w.stream.Data[value]
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

// Action returns the complete information for the action being performed.
func (w Renderer) Action() *model.Action {
	return w.action
}

// IsPartialRequest returns TRUE if this is a partial page request from htmx.
func (w Renderer) IsPartialRequest() bool {
	return (w.ctx.Request().Header.Get("HX-Request") != "")
}

/*******************************************
 * RELATIONSHIPS TO OTHER STREAMS
 *******************************************/

// Parent returns a Stream containing the parent of the current stream
func (w Renderer) Parent(actionID string) (Renderer, error) {

	parent := model.NewStream()

	streamService := w.factory.Stream()

	if err := streamService.LoadParent(w.stream, &parent); err != nil {
		return Renderer{}, derp.Wrap(err, "ghost.renderer.Renderer.Parent", "Error loading Parent")
	}

	renderer, err := w.newRenderer(&parent, actionID)

	if err != nil {
		return Renderer{}, derp.Wrap(err, "ghost.renderer.Renderer.Parent", "Unable to create new Renderer")
	}

	return renderer, nil
}

// Children returns an array of Streams containing all of the child elements of the current stream
func (w Renderer) Children(action string) ([]Renderer, error) {

	streamService := w.factory.Stream()
	iterator, err := streamService.ListByParent(w.stream.StreamID)

	if err != nil {
		return nil, derp.Report(derp.Wrap(err, "ghost.renderer.Renderer.Children", "Error loading child streams", w.stream))
	}

	return w.iteratorToSlice(iterator, action), nil
}

// TopLevel returns an array of Streams that have a Zero ParentID
func (w Renderer) TopFolders(action string) ([]Renderer, error) {

	streamService := w.factory.Stream()
	iterator, err := streamService.ListTopFolders()

	if err != nil {
		return nil, derp.Report(derp.Wrap(err, "ghost.renderer.Renderer.Children", "Error loading child streams", w.stream))
	}

	return w.iteratorToSlice(iterator, action), nil
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

	return action.UserCan(w.stream, authorization)
}

// CanCreate returns all of the templates that can be created underneath
// the current stream.
func (w Renderer) CanCreate() []model.Option {

	templateService := w.factory.Template()
	return templateService.ListByContainer(w.template.TemplateID)
}

///////////////////////////
// HELPER FUNCTIONS

// iteratorToSlice converts a data.Iterator of Streams into a slice of Streams
func (w Renderer) iteratorToSlice(iterator data.Iterator, actionID string) []Renderer {

	var stream *model.Stream
	result := make([]Renderer, iterator.Count())

	// This allocates a new memory space for the new stream value
	stream = new(model.Stream)

	for iterator.Next(stream) {

		// Try to create a new Renderer for each Stream in the Iterator
		if renderer, err := w.newRenderer(stream, actionID); err == nil {
			result = append(result, renderer)
		}

		// Overwrite stream so that no values leak from one record to the other. grrrr.
		stream = new(model.Stream)
	}

	return result
}

// newRenderer is a shortcut to the NewRenderer function that reuses the values present in this current Renderer
func (w Renderer) newRenderer(stream *model.Stream, actionID string) (Renderer, error) {
	return NewRenderer(w.factory, w.ctx, stream, actionID)
}
