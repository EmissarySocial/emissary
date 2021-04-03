package domain

import (
	"bytes"
	"html/template"
	"time"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
)

// Renderer wraps a model.Stream object and provides functions that make it easy to render an HTML template with it.
type Renderer struct {
	streamService *service.Stream // StreamService is used to load child streams
	stream        *model.Stream   // Stream to be displayed
	request       *HTTPRequest    // Additional request info URL params, Authentication, etc.
	view          string
	transition    string
}

// NewRenderer creates a new object that can generate HTML for a specific stream/view
func NewRenderer(streamService *service.Stream, request *HTTPRequest, stream *model.Stream) *Renderer {

	result := Renderer{
		streamService: streamService,
		stream:        stream,
		request:       request,
	}

	return &result
}

func (w Renderer) URL() string {
	return w.request.URL()
}

// StreamID returns the unique ID for the stream being rendered
func (w Renderer) StreamID() string {
	return w.stream.StreamID.Hex()
}

// ViewID returns the view identifier being rendered
func (w Renderer) ViewID() string {
	return w.view
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

// SetView overrides the transition provided in the request parameters.
func (w *Renderer) SetView(view string) *Renderer {
	w.view = view
	return w
}

// View returns the string name of the view requested in the URL QueryString
func (w Renderer) View() (*model.View, error) {

	groups := w.request.Groups()
	roles := w.stream.Roles(groups...)

	if state, err := w.streamService.State(w.stream); err == nil {

		if w.view != "" {
			if view, ok := state.View(w.view); ok {
				if view.MatchRoles(roles...) {
					return view, nil
				}
			}
		}

		for _, view := range state.Views {
			if view.MatchRoles(roles...) {
				return &view, nil
			}
		}
	}

	return nil, derp.New(500, "ghost.domain.Renderer.View", "Missing/Unauthorized View", w.view)
}

// SetTransition overrides the transition provided in the request parameters.
func (w *Renderer) SetTransition(transition string) *Renderer {
	w.transition = transition
	return w
}

// Transition returns the string name of the transition requested in the URL QueryString
func (w Renderer) Transition() string {
	return w.transition
}

////////////////////////////////

// Parent returns a Stream containing the parent of the current stream
func (w Renderer) Parent(viewID string) (*Renderer, error) {

	parent, err := w.streamService.LoadParent(w.stream)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.service.Renderer.Parent", "Error loading Parent")
	}

	result := NewRenderer(w.streamService, w.request, parent).SetView(viewID)

	return result, nil
}

// Children returns an array of Streams containing all of the child elements of the current stream
func (w Renderer) Children(viewID string) ([]*Renderer, error) {

	iterator, err := w.streamService.ListByParent(w.stream.StreamID)

	if err != nil {
		return nil, derp.Report(derp.Wrap(err, "ghost.service.Renderer.Children", "Error loading child streams", w.stream))
	}

	return w.iteratorToSlice(iterator, viewID)
}

// TopLevel returns an array of Streams that have a Zero ParentID
func (w Renderer) TopLevel(viewID string) ([]*Renderer, error) {

	iterator, err := w.streamService.ListTopFolders()

	if err != nil {
		return nil, derp.Report(derp.Wrap(err, "ghost.service.Renderer.Children", "Error loading child streams", w.stream))
	}

	return w.iteratorToSlice(iterator, viewID)
}

// iteratorToSlice converts a data.Iterator of Streams into a slice of Streams
func (w Renderer) iteratorToSlice(iterator data.Iterator, viewID string) ([]*Renderer, error) {

	var stream model.Stream

	result := make([]*Renderer, iterator.Count())

	for iterator.Next(&stream) {
		copy := stream
		result = append(result, NewRenderer(w.streamService, w.request, &copy).SetView(viewID))
	}

	return result, nil
}

/// RENDERING METHODS

// Render generates an HTML output for a stream/view combination.
func (w Renderer) Render() (template.HTML, error) {

	var result bytes.Buffer

	view, err := w.View()

	if err != nil {
		return template.HTML(""), derp.Report(derp.Wrap(err, "ghost.domain.renderer.Render", "Unrecognized view"))
	}

	if view.Template == nil {
		return template.HTML(""), derp.Report(derp.New(500, "ghost.domain.renderer.Render", "Missing Template (probably did not load/compile correctly on startup)", view))
	}

	// Execut template
	err = view.Template.Execute(&result, w)

	if err != nil {
		return template.HTML(""), derp.Report(derp.Wrap(err, "ghost.domain.renderer.Render", "Error executing template", w.stream))
	}

	// Return result
	return template.HTML(result.String()), nil
}

// RenderForm returns an HTML rendering of this form
func (w Renderer) RenderForm() (template.HTML, error) {

	result, err := w.streamService.Form(w.stream, w.Transition())

	if err != nil {
		return template.HTML(""), derp.Report(derp.Wrap(err, "ghost.domain.Renderer.Form", "Error generating HTML form"))
	}

	return template.HTML(result), nil
}

// CanAddChild returns TRUE if the current user has permission to add child streams.
func (w Renderer) CanAddChild() bool {
	return true
}

// ChildTemplates lists all templates that can be embedded in the current stream
func (w Renderer) ChildTemplates() []model.Template {

	// TODO: permissions here...
	return w.streamService.ChildTemplates(w.stream)
}

// CanView returns TRUE if this Request is authorized to access this stream/view
func (w Renderer) CanView(viewName string) bool {

	state, err := w.streamService.State(w.stream)

	if err != nil {
		return false
	}

	view, ok := state.View(viewName)

	if !ok {
		return false
	}

	if state.MatchAnonymous() && view.MatchAnonymous() {
		return true
	}

	groups := w.request.Groups()
	roles := w.stream.Roles(groups...)

	return state.MatchRoles(roles...) && view.MatchRoles(roles...)
}

// CanTransition returns TRUE is this Renderer is authorized to initiate a transition
func (w Renderer) CanTransition(stream *model.Stream, transitionID string) bool {

	state, err := w.streamService.State(stream)

	if err != nil {
		return false
	}

	transition, ok := state.Transition(transitionID)

	if !ok {
		return false
	}

	if state.MatchAnonymous() && transition.MatchAnonymous() {
		return true
	}

	groups := w.request.Groups()
	roles := stream.Roles(groups...)

	return state.MatchRoles(roles...) && transition.MatchRoles(roles...)
}
