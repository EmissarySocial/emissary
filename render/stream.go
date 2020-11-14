package render

import (
	"bytes"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
)

// Stream wraps a model.Stream object and provides functions that make it easy to render an HTML template with it.
type Stream struct {
	layoutService   LayoutService
	templateService TemplateService
	streamService   StreamService
	stream          *model.Stream
	viewName        string
}

// NewStream returns a fully initialized Stream object.
func NewStream(layoutService LayoutService, templateService TemplateService, streamService StreamService, stream *model.Stream, view string) Stream {

	return Stream{
		layoutService:   layoutService,
		templateService: templateService,
		streamService:   streamService,
		stream:          stream,
		viewName:        view,
	}
}

// Render generates an HTML output for a stream/view combination.
func (w Stream) Render() (string, error) {

	var result bytes.Buffer

	layout := w.layoutService.Layout()

	// Load stream content
	_, content, err := w.templateService.LoadCompiled(w.stream.Template, w.stream.State, w.View())
	
	if err != nil {
		return "", derp.Wrap(err, "ghost.render.Stream.Render", "Unable to load stream template")
	}

	// Combine the two parse trees.
	// TODO: Could this be done at load time, not for each page request?
	combined, err := layout.AddParseTree("content", content.Tree)

	if err != nil {
		return "", derp.Wrap(err, "ghost.render.Stream.Render", "Unable to create parse tree")
	}

	// Choose the correct view based on the wrapper provided.
	if err := combined.Funcs(sprig.FuncMap()).ExecuteTemplate(&result, "stream", w); err != nil {
		return "", derp.Wrap(err, "ghost.render.Stream.Render", "Error rendering view")
	}

	// TODO: Add caching...

	// Success!
	return result.String(), nil
}

// StreamID returns the unique ID for the stream being rendered
func (w Stream) StreamID() string {
	return w.stream.StreamID.Hex()
}

// Token returns the unique URL token for the stream being rendered
func (w Stream) Token() string {
	return w.stream.Token
}

// View returns the current view being used to render this stream
func (w Stream) View() string {
	return w.viewName
}

// Label returns the Label for the stream being rendered
func (w Stream) Label() string {
	return w.stream.Label
}

// Description returns the description of the stream being rendered
func (w Stream) Description() string {
	return w.stream.Description
}

func (w Stream) PublishDate() time.Time {
	return time.Unix(w.stream.PublishDate, 0)
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


////////////////////////////////

func (w Stream) CanAddChild() bool {
	return true
}


////////////////////////////////


func (w Stream) Views() []View {

	template, _ := w.templateService.Load(w.stream.Template)

	result := []View{}

	for _, view := range template.Views {

		// Available views can be filtered here...

		// Add this view to the list.
		result = append(result, View{
			Name:  view.Name,
			Label: view.Label,
		})
	}

	return result
}

// Folders returns all top-level folders for the domain
func (w Stream) Folders() ([]Stream, error) {

	folders, err := w.streamService.ListTopFolders()

	if err != nil {
		return nil, derp.Wrap(err, "ghost.render.stream.Folders", "Error loading top-level folders")
	}

	return w.iteratorToSlice(folders)

}

// Parent returns a Stream containing the parent of the current stream
func (w Stream) Parent() (*Stream, error) {

	parent, err := w.streamService.LoadParent(w.stream)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.render.stream.Parent", "Error loading Parent")
	}

	result := NewStream(w.layoutService, w.templateService, w.streamService, parent, w.View())

	return &result, nil
}

// Children returns an array of Streams containing all of the child elements of the current stream
func (w Stream) Children() ([]Stream, error) {

	iterator, err := w.streamService.ListByParent(w.stream.StreamID)

	if err != nil {
		return nil, derp.Report(derp.Wrap(err, "ghost.render.stream.Children", "Error loading child streams", w.stream))
	}

	return w.iteratorToSlice(iterator)
}

// SubTemplates returns an array of templates that can be placed inside this Stream
func (w Stream) SubTemplates() ([]model.Template) {
	return w.templateService.ListByContainer(w.stream.Template)
}

// iteratorToSlice converts a data.Iterator of Streams into a slice of Streams 
func (w Stream) iteratorToSlice(iterator data.Iterator) ([]Stream, error) {

	var stream model.Stream

	result := make([]Stream, iterator.Count())

	for iterator.Next(&stream) {
		result = append(result, NewStream(w.layoutService, w.templateService, w.streamService, &stream, w.View()))
	}

	return result, nil
}