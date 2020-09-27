package render

import (
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
)

// StreamWrapper wraps a model.Stream object and provides functions that make it easy to render an HTML template with it.
type StreamWrapper struct {
	templateService TemplateService
	streamService   StreamService
	stream          *model.Stream
	view            string
}

// NewStreamWrapper returns a fully initialized StreamWrapper object.
func NewStreamWrapper(templateService TemplateService, streamService StreamService, stream *model.Stream, view string) StreamWrapper {

	return StreamWrapper{
		templateService: templateService,
		streamService:   streamService,
		stream:          stream,
		view:            view,
	}
}

// Render generates an HTML output for a stream/view combination.
func (w StreamWrapper) Render() (string, error) {

	// Try to load the template from the database
	template, err := w.templateService.Load(w.stream.Template)

	if err != nil {
		return "", derp.Wrap(err, "ghost.render.StreamWrapper.Render", "Unable to load Template", w.stream.Template)
	}

	// Locate / Authenticate the view to use

	view, err := template.View(w.stream.State, w.view)

	if err != nil {
		return "", derp.Wrap(err, "ghost.render.StreamWrapper.Render", "Unrecognized view", w.view)
	}

	// TODO: need to enforce permissions somewhere...

	// Try to generate the HTML response using the provided data
	result, err := view.Execute(w)

	if err != nil {
		return "", derp.Wrap(err, "ghost.render.StreamWrapper.Render", "Error rendering view")
	}

	// TODO: Add caching here...

	// Success!
	return result, nil
}

func (w StreamWrapper) Stream() StreamWrapper {
	return w
}

// StreamID returns the unique ID for the stream being rendered
func (w StreamWrapper) StreamID() string {
	return w.stream.StreamID.String()
}

// Token returns the unique URL token for the stream being rendered
func (w StreamWrapper) Token() string {
	return w.stream.Token
}

// Label returns the Label for the stream being rendered
func (w StreamWrapper) Label() string {
	return w.stream.Label
}

// Description returns the description of the stream being rendered
func (w StreamWrapper) Description() string {
	return w.stream.Description
}

// ThumbnailImage returns the thumbnail image URL of the stream being rendered
func (w StreamWrapper) ThumbnailImage() string {
	return w.stream.ThumbnailImage
}

// Data returns the custom data map of the stream being rendered
func (w StreamWrapper) Data() map[string]interface{} {
	return w.stream.Data
}

// Tags returns the tags of the stream being rendered
func (w StreamWrapper) Tags() []string {
	return w.stream.Tags
}

// HasParent returns TRUE if the stream being rendered has a parend objec
func (w StreamWrapper) HasParent() bool {
	return w.stream.HasParent()
}

// Parent returns a StreamWrapper containing the parent of the current stream
func (w StreamWrapper) Parent() (*StreamWrapper, error) {

	parent, err := w.streamService.LoadParent(w.stream)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.render.stream.Parent", "Error loading Parent")
	}

	result := NewStreamWrapper(w.templateService, w.streamService, parent, w.view)

	return &result, nil
}

// Children returns an array of SubStreamWrappers containing all of the child elements of the current stream
func (w StreamWrapper) Children() ([]SubStreamWrapper, error) {

	iterator, err := w.streamService.ListByParent(w.stream.StreamID)

	if err != nil {
		return nil, derp.Report(derp.Wrap(err, "ghost.render.stream.Children", "Error loading child streams", w.stream))
	}

	var stream *model.Stream

	result := make([]SubStreamWrapper, iterator.Count())

	for index := 0; iterator.Next(stream); index = index + 1 {
		result[index] = NewSubStreamWrapper(w.templateService, w.streamService, stream, w.view)
	}

	return result, nil
}
