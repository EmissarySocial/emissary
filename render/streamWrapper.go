package render

import (
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
)

type StreamWrapper struct {
	factory service.Factory
	stream  *model.Stream
}

func NewStreamWrapper(factory service.Factory, stream *model.Stream) *StreamWrapper {

	return &StreamWrapper{
		factory: factory,
		stream:  stream,
	}
}

func (w *StreamWrapper) Render(view string) {

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

func (w *StreamWrapper) Children() ([]*StreamWrapper, error) {

	streamService := w.factory.Stream()

	iterator, err := streamService.ListByParent(w.stream.StreamID)

	if err != nil {
		return nil, derp.Report(derp.Wrap(err, "ghost.render.stream.Children", "Error loading child streams", w.stream))
	}

	return wrapStreamIterator(w.factory, iterator)
}
