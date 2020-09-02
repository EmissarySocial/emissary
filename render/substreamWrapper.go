package render

import (
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
	"github.com/benpate/html"
)

type SubStreamWrapper struct {
	factory   service.Factory
	parentURL string
	stream    *model.Stream
}

func NewSubStreamWrapper(factory service.Factory, parentURL string, stream *model.Stream) *SubStreamWrapper {

	return &SubStreamWrapper{
		factory:   factory,
		parentURL: parentURL,
		stream:    stream,
	}
}

func (w *SubStreamWrapper) Render(viewName string) (string, error) {

	templateService := w.factory.Template()

	// Try to load the template from the database
	template, err := templateService.Load(w.stream.Template)

	if err != nil {
		return "", derp.Wrap(err, "service.Stream.Render", "Unable to load Template", w.stream)
	}

	// Locate / Authenticate the view to use

	view, err := template.View(w.stream.State, viewName)

	if err != nil {
		return "", derp.Wrap(err, "service.Stream.Render", "Unrecognized view", viewName)
	}

	// TODO: need to enforce permissions somewhere...

	// Try to generate the HTML response using the provided data
	result, err := view.Execute(w)

	if err != nil {
		return "", derp.Wrap(err, "service.Stream.Render", "Error rendering view")
	}

	result = html.CollapseWhitespace(result)

	// TODO: Add caching here...

	// Success!
	return result, nil
}

func (w *SubStreamWrapper) RenderWithUpdates(viewName string) (string, error) {

	html, err := w.Render(viewName)

	if err != nil {
		return "", err
	}

	return "<div hx-sse=\"" + w.parentURL + " " + w.stream.StreamID.Hex() + "\">" + html + "</div>", nil

}

func (w *SubStreamWrapper) Label() string {
	return w.stream.Label
}

func (w *SubStreamWrapper) Description() string {
	return w.stream.Description
}

func (w *SubStreamWrapper) ThumbnailImage() string {
	return w.stream.ThumbnailImage
}

func (w *SubStreamWrapper) Data() map[string]interface{} {
	return w.stream.Data
}

func (w *SubStreamWrapper) Tags() []string {
	return w.stream.Tags
}
