package render

import (
	"bytes"
	"html/template"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/labstack/echo/v4"
)

// Stream wraps a model.Stream object and provides functions that make it easy to render an HTML template with it.
type Stream struct {
	context         echo.Context
	layout          *template.Template
	templateService TemplateService
	streamService   StreamService
	stream          *model.Stream
}

// NewStream returns a fully initialized Stream object.
func NewStream(ctx echo.Context, layout *template.Template, templateService TemplateService, streamService StreamService, stream *model.Stream) Stream {

	return Stream{
		context:         ctx,
		layout:          layout,
		templateService: templateService,
		streamService:   streamService,
		stream:          stream,
	}
}

// Render generates an HTML output for a stream/view combination.
func (w Stream) Render() (string, error) {

	layout, err := w.layout.Clone()

	if err != nil {
		return "", derp.Wrap(err, "ghost.render.Stream.Render", "Error cloning template")
	}

	// Load stream content
	_, content, err := w.templateService.LoadCompiled(w.stream.Template, w.stream.State, w.View())

	if err != nil {
		return "", derp.Wrap(err, "ghost.render.Stream.Render", "Unable to load stream template")
	}

	// Add stream content as a sub-template of the domain wrapper
	layout.AddParseTree("content", content.Tree)

	var result bytes.Buffer

	// Choose the correct view based on the wrapper provided.
	if err := layout.ExecuteTemplate(&result, w.Layout(), w); err != nil {
		return "", derp.Wrap(err, "ghost.render.Stream.Render", "Error rendering view")
	}

	// TODO: Add caching...

	// Success!
	return result.String(), nil
}

// StreamID returns the unique ID for the stream being rendered
func (w Stream) StreamID() string {
	return w.stream.StreamID.String()
}

// Token returns the unique URL token for the stream being rendered
func (w Stream) Token() string {
	return w.stream.Token
}

func (w Stream) Layout() string {

	if w.context.Request().Header.Get("hx-request") == "true" {
		return "stream-partial"
	}

	return "stream-full"
}

func (w Stream) View() string {

	if view := w.context.QueryParam("view"); view != "" {
		return view
	}

	return "default"
}

// Label returns the Label for the stream being rendered
func (w Stream) Label() string {
	return w.stream.Label
}

// Description returns the description of the stream being rendered
func (w Stream) Description() string {
	return w.stream.Description
}

// ThumbnailImage returns the thumbnail image URL of the stream being rendered
func (w Stream) ThumbnailImage() string {
	return w.stream.ThumbnailImage
}

// Data returns the custom data map of the stream being rendered
func (w Stream) Data() map[string]interface{} {
	return w.stream.Data
}

// Tags returns the tags of the stream being rendered
func (w Stream) Tags() []string {
	return w.stream.Tags
}

// HasParent returns TRUE if the stream being rendered has a parend objec
func (w Stream) HasParent() bool {
	return w.stream.HasParent()
}

// Parent returns a Stream containing the parent of the current stream
func (w Stream) Parent() (*Stream, error) {

	parent, err := w.streamService.LoadParent(w.stream)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.render.stream.Parent", "Error loading Parent")
	}

	result := NewStream(w.context, w.layout, w.templateService, w.streamService, parent)

	return &result, nil
}

// Children returns an array of SubStreams containing all of the child elements of the current stream
func (w Stream) Children() ([]SubStream, error) {

	iterator, err := w.streamService.ListByParent(w.stream.StreamID)

	if err != nil {
		return nil, derp.Report(derp.Wrap(err, "ghost.render.stream.Children", "Error loading child streams", w.stream))
	}

	var stream *model.Stream

	result := make([]SubStream, iterator.Count())

	for index := 0; iterator.Next(stream); index = index + 1 {
		result[index] = NewSubStream(w.templateService, w.streamService, stream, w.View())
	}

	return result, nil
}
