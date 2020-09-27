package render

import (
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
)

// StreamWrapper wraps a model.Stream object and provides functions that make it easy to render an HTML template with it.
type StreamWrapper struct {
	factory *service.Factory
	stream  *model.Stream
	view    string
}

// NewStreamWrapper returns a fully initialized StreamWrapper object.
func NewStreamWrapper(factory *service.Factory, stream *model.Stream) *StreamWrapper {

	return &StreamWrapper{
		factory: factory,
		stream:  stream,
	}
}

// Render generates an HTML output for a stream/view combination.
func (w *StreamWrapper) Render(viewName string) (*string, error) {

	templateService := w.factory.Template()

	// Try to load the template from the database
	template, err := templateService.Load(w.stream.Template)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.render.StreamWrapper.Render", "Unable to load Template", w.stream.Template)
	}

	// Locate / Authenticate the view to use

	view, err := template.View(w.stream.State, viewName)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.render.StreamWrapper.Render", "Unrecognized view", viewName)
	}

	// TODO: need to enforce permissions somewhere...

	// Try to generate the HTML response using the provided data
	result, err := view.Execute(w)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.render.StreamWrapper.Render", "Error rendering view")
	}

	// TODO: Add caching here...

	// Success!
	return &result, nil
}

// StreamID returns the unique ID for the stream being rendered
func (w *StreamWrapper) StreamID() string {
	return w.stream.StreamID.String()
}

// Token returns the unique URL token for the stream being rendered
func (w *StreamWrapper) Token() string {
	return w.stream.Token
}

// Label returns the Label for the stream being rendered
func (w *StreamWrapper) Label() string {
	return w.stream.Label
}

// Description returns the description of the stream being rendered
func (w *StreamWrapper) Description() string {
	return w.stream.Description
}

// ThumbnailImage returns the thumbnail image URL of the stream being rendered
func (w *StreamWrapper) ThumbnailImage() string {
	return w.stream.ThumbnailImage
}

// Data returns the custom data map of the stream being rendered
func (w *StreamWrapper) Data() map[string]interface{} {
	return w.stream.Data
}

// Tags returns the tags of the stream being rendered
func (w *StreamWrapper) Tags() []string {
	return w.stream.Tags
}

// HasParent returns TRUE if the stream being rendered has a parend objec
func (w *StreamWrapper) HasParent() bool {
	return w.stream.HasParent()
}

// Parent returns a StreamWrapper containing the parent of the current stream
func (w *StreamWrapper) Parent() (*StreamWrapper, error) {

	service := w.factory.Stream()
	parent, err := service.LoadParent(w.stream)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.render.stream.Parent", "Error loading Parent")
	}

	return NewStreamWrapper(w.factory, parent), nil
}

// Children returns an array of SubStreamWrappers containing all of the child elements of the current stream
func (w *StreamWrapper) Children() ([]*SubStreamWrapper, error) {

	streamService := w.factory.Stream()

	iterator, err := streamService.ListByParent(w.stream.StreamID)

	if err != nil {
		return nil, derp.Report(derp.Wrap(err, "ghost.render.stream.Children", "Error loading child streams", w.stream))
	}

	result := make([]*SubStreamWrapper, iterator.Count())
	stream := streamService.New()

	for index := 0; iterator.Next(stream); index = index + 1 {
		result[index] = NewSubStreamWrapper(w.factory, "/"+w.stream.Token, stream)
	}

	return result, nil
}
