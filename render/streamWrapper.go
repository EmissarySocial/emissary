package render

import (
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
)

type StreamWrapper struct {
	factory *service.Factory
	stream  *model.Stream
}

func NewStreamWrapper(factory *service.Factory, stream *model.Stream) *StreamWrapper {

	return &StreamWrapper{
		factory: factory,
		stream:  stream,
	}
}

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

func (w *StreamWrapper) Token() string {
	return w.stream.Token
}

func (w *StreamWrapper) Label() string {
	return w.stream.Label
}

func (w *StreamWrapper) Description() string {
	return w.stream.Description
}

func (w *StreamWrapper) ThumbnailImage() string {
	return w.stream.ThumbnailImage
}

func (w *StreamWrapper) Data() map[string]interface{} {
	return w.stream.Data
}

func (w *StreamWrapper) Tags() []string {
	return w.stream.Tags
}

func (w *StreamWrapper) HasParent() bool {
	return w.stream.HasParent()
}

func (w *StreamWrapper) Parent() (*StreamWrapper, error) {

	service := w.factory.Stream()
	parent, err := service.LoadParent(w.stream)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.render.stream.Parent", "Error loading Parent")
	}

	return NewStreamWrapper(w.factory, parent), nil
}

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
