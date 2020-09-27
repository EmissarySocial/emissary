package render

import (
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/html"
)

// SubStreamWrapper contains a stream -- specifically a child stream of the currently rendering page --
// and provides a list of functions used to render it into HTML
type SubStreamWrapper struct {
	templateService TemplateService
	streamService   StreamService
	stream          *model.Stream
	view            string
}

// NewSubStreamWrapper returns a fully populated SubStreamWrapper
func NewSubStreamWrapper(templateService TemplateService, streamService StreamService, stream *model.Stream, view string) SubStreamWrapper {

	return SubStreamWrapper{
		templateService: templateService,
		streamService:   streamService,
		stream:          stream,
		view:            view,
	}
}

// Render returns the HTML rendering of this SubStream
func (w *SubStreamWrapper) Render() (string, error) {

	// Try to load the template from the database
	template, err := w.templateService.Load(w.stream.Template)

	if err != nil {
		return "", derp.Wrap(err, "service.Stream.Render", "Unable to load Template", w.stream)
	}

	// Locate / Authenticate the view to use

	view, err := template.View(w.stream.State, w.view)

	if err != nil {
		return "", derp.Wrap(err, "service.Stream.Render", "Unrecognized view", w.view)
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
