package domain

import (
	"bytes"
	"html/template"
	"time"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/action"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
	"github.com/benpate/steranko"
)

// Renderer wraps a model.Stream object and provides functions that make it easy to render an HTML template with it.
type Renderer struct {
	ctx           *steranko.Context // Contains request context and authentication data.
	streamService *service.Stream   // StreamService is used to load child streams
	stream        model.Stream      // Stream to be displayed
	action        action.Action
}

// NewRenderer creates a new object that can generate HTML for a specific stream/view
func NewRenderer(ctx *steranko.Context, streamService *service.Stream, stream model.Stream, action action.Action) Renderer {

	result := Renderer{
		ctx:           ctx,
		streamService: streamService,
		stream:        stream,
		action:        action,
	}

	return result
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

// ActionID returns the view identifier being rendered
func (w Renderer) ViewID() string {
	return w.action.Config().ActionID
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

// Returns TRUE if this is a partial request.
func (w Renderer) Partial() bool {
	return (w.ctx.Request().Header.Get("HX-Request") != "")
}

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

	if err := w.streamService.LoadParent(&w.stream, &parent); err != nil {
		return result, derp.Wrap(err, "ghost.service.Renderer.Parent", "Error loading Parent")
	}

	return NewRenderer(w.ctx, w.streamService, parent, nil), nil
}

// Children returns an array of Streams containing all of the child elements of the current stream
func (w Renderer) Children(viewID string) ([]Renderer, error) {

	iterator, err := w.streamService.ListByParent(w.stream.StreamID)

	if err != nil {
		return nil, derp.Report(derp.Wrap(err, "ghost.service.Renderer.Children", "Error loading child streams", w.stream))
	}

	return w.iteratorToSlice(iterator, viewID)
}

// TopLevel returns an array of Streams that have a Zero ParentID
func (w Renderer) TopLevel(viewID string) ([]Renderer, error) {

	iterator, err := w.streamService.ListTopFolders()

	if err != nil {
		return nil, derp.Report(derp.Wrap(err, "ghost.service.Renderer.Children", "Error loading child streams", w.stream))
	}

	return w.iteratorToSlice(iterator, viewID)
}

// ChildTemplates lists all templates that can be embedded in the current stream
func (w Renderer) ChildTemplates() []model.Template {

	// TODO: permissions here...
	return w.streamService.ChildTemplates(&w.stream)
}

///////////////////////////////
/// RENDERING METHODS

// Render generates an HTML output for a stream/view combination.
func (w Renderer) Render() (template.HTML, error) {

	var result bytes.Buffer

	if templateText, ok := w.action.Config().Args["template"]; ok {
		if templateCompiled, ok := templateText.(*template.Template); ok {

			if err := templateCompiled.Execute(&result, w); err == nil {
				return template.HTML(result.String()), nil
			}
		}
	}

	return template.HTML(""), derp.New(derp.CodeInternalError, "ghost.domain.renderer.Render", "Error executing template", w.stream)
}

/////////////////////
// PERMISSIONS METHODS

// CanView returns TRUE if this Request is authorized to access this stream/view
func (w Renderer) UserCan(actionID string) bool {

	authorization, err := w.getAuthorization()

	if err != nil {
		return false
	}

	return w.action.UserCan(&w.stream, authorization)
}

// Authorization returns the authorization data for this request.
func (w Renderer) getAuthorization() (*model.Authorization, error) {
	claims, err := w.ctx.Authorization()

	if err != nil {
		return nil, err
	}

	if authorization, ok := claims.(*model.Authorization); ok {
		return authorization, nil
	}

	return nil, derp.New(derp.CodeBadRequestError, "ghost.domain.Renderer.Authorization", "Invalid authorization", claims)
}

///////////////////////////
// HELPER FUNCTIONS

// iteratorToSlice converts a data.Iterator of Streams into a slice of Streams
func (w Renderer) iteratorToSlice(iterator data.Iterator, viewID string) ([]Renderer, error) {

	var stream model.Stream

	result := make([]Renderer, 0, iterator.Count())

	for iterator.Next(&stream) {
		renderer := NewRenderer(w.ctx, w.streamService, stream, nil)

		// Enforce permissions here...
		if renderer.UserCan(viewID) {
			result = append(result, renderer)
		}

		stream = model.Stream{}
	}

	return result, nil
}
