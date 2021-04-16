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
	request       *HTTPRequest    // Additional request info URL params, Authentication, etc.
	stream        model.Stream    // Stream to be displayed
	viewID        string
	transitionID  string
}

// NewRenderer creates a new object that can generate HTML for a specific stream/view
func NewRenderer(streamService *service.Stream, request *HTTPRequest, stream model.Stream) Renderer {

	result := Renderer{
		streamService: streamService,
		request:       request,
		stream:        stream,
	}

	return result
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
	return w.viewID
}

// TransitionID returns the view identifier being rendered
func (w Renderer) TransitionID() string {
	return w.transitionID
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

////////////////////////////////

// Parent returns a Stream containing the parent of the current stream
func (w Renderer) Parent(viewID string) (Renderer, error) {

	var result Renderer

	parent, err := w.streamService.LoadParent(&w.stream)

	if err != nil {
		return result, derp.Wrap(err, "ghost.service.Renderer.Parent", "Error loading Parent")
	}

	result = NewRenderer(w.streamService, w.request, *parent)
	result.viewID = viewID

	return result, nil
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

// iteratorToSlice converts a data.Iterator of Streams into a slice of Streams
func (w Renderer) iteratorToSlice(iterator data.Iterator, viewID string) ([]Renderer, error) {

	var stream model.Stream

	result := make([]Renderer, iterator.Count())

	for iterator.Next(&stream) {
		renderer := NewRenderer(w.streamService, w.request, stream)
		renderer.viewID = viewID

		// Enforce permissions here...
		if renderer.CanView(viewID) {
			result = append(result, renderer)
		}
		stream = model.Stream{}
	}

	return result, nil
}

/// RENDERING METHODS

// Render generates an HTML output for a stream/view combination.
func (w Renderer) Render() (template.HTML, error) {

	var result bytes.Buffer

	view, err := w.getView()

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

	transition, err := w.getTransition()

	if err != nil {
		return template.HTML(""), derp.Report(derp.Wrap(err, "ghost.domain.Renderer.Form", "Error locating transition"))
	}

	result, err := w.streamService.Form(&w.stream, transition)

	if err != nil {
		return template.HTML(""), derp.Report(derp.Wrap(err, "ghost.domain.Renderer.Form", "Error generating HTML form"))
	}

	return template.HTML(result), nil
}

// ChildTemplates lists all templates that can be embedded in the current stream
func (w Renderer) ChildTemplates() []model.Template {

	// TODO: permissions here...
	return w.streamService.ChildTemplates(&w.stream)
}

// CanAddChild returns TRUE if the current user has permission to add child streams.
func (w Renderer) CanAddChild() bool {
	return true
}

// CanView returns TRUE if this Request is authorized to access this stream/view
func (w Renderer) CanView(viewID string) bool {

	authorization := w.request.Authorization()
	result, err := w.streamService.CanView(&w.stream, viewID, &authorization)

	if err != nil {
		derp.Report(derp.Wrap(err, "ghost.domain.Renderer.CanView", "Error in CanView"))
	}

	return result
}

// CanTransition returns TRUE is this Renderer is authorized to initiate a transition
func (w Renderer) CanTransition(transitionID string) bool {

	authorization := w.request.Authorization()
	result, err := w.streamService.CanTransition(&w.stream, transitionID, &authorization)

	if err != nil {
		derp.Report(derp.Wrap(err, "ghost.domain.Renderer.CanView", "Error in CanView"))
	}

	return result
}

///////////////////////////////////////
// PRIVATE METHODS

// getView returns the requested view for this Renderer
func (w Renderer) getView() (*model.View, error) {

	authorization := w.request.Authorization()
	roles := w.stream.Roles(&authorization)

	state, err := w.streamService.State(&w.stream)

	if err != nil {
		return nil, derp.New(derp.CodeForbiddenError, "ghost.domain.Renderer.getView", "Missing/Unauthorized View", w.viewID)
	}

	if !state.MatchRoles(roles...) {
		return nil, derp.New(derp.CodeForbiddenError, "ghost.domain.Renderer.getView", "Unauthorized State", w.stream)
	}

	if w.viewID != "" {
		if view, ok := state.View(w.viewID); ok {
			if view.MatchRoles(roles...) {
				return &view, nil
			}
		}
	}

	for _, view := range state.Views {
		if view.MatchRoles(roles...) {
			return &view, nil
		}
	}

	return nil, derp.New(derp.CodeForbiddenError, "ghost.domain.Renderer.getView", "Unrecognized View", w.viewID)
}

// getTransition returns the string name of the transition requested in the URL QueryString
func (w Renderer) getTransition() (*model.Transition, error) {

	authorization := w.request.Authorization()
	roles := w.stream.Roles(&authorization)

	state, err := w.streamService.State(&w.stream)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.domain.renderer.getDomain", "Error Getting State")
	}

	if !state.MatchRoles(roles...) {
		return nil, derp.New(derp.CodeForbiddenError, "ghost.domain.Renderer.getTransition", "Unauthorized State", w.stream)
	}

	transition, ok := state.Transition(w.transitionID)

	if !ok {
		return nil, derp.New(derp.CodeInternalError, "ghost.domain.Renderer.getTransition", "Unrecognized Transition", w.stream)
	}

	if !transition.MatchRoles(roles...) {
		return nil, derp.New(derp.CodeForbiddenError, "ghost.domain.Renderer.getTransition", "Unauthorized Transition", w.stream)
	}

	return transition, nil
}
