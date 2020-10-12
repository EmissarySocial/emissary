package render

import (
	"bytes"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
)

// Stream wraps a model.Stream object and provides functions that make it easy to render an HTML template with it.
type Stream struct {
	layoutService   LayoutService
	folderService   FolderService
	templateService TemplateService
	streamService   StreamService
	stream          model.Stream
	viewName        string
}

// NewStream returns a fully initialized Stream object.
func NewStream(layoutService LayoutService, folderService FolderService, templateService TemplateService, streamService StreamService, stream model.Stream, view string) Stream {

	return Stream{
		layoutService:   layoutService,
		folderService:   folderService,
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
	if err := combined.ExecuteTemplate(&result, "stream", w); err != nil {
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

func (w Stream) Folders() ([]FolderListItem, error) {

	folders, err := w.folderService.ListNested()

	if err != nil {
		return nil, derp.Wrap(err, "ghost.render.Stream.AllFolders", "Error retrieving all folders")
	}

	return NewFolderList(folders), nil
}

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

// Parent returns a Stream containing the parent of the current stream
func (w Stream) Parent() (*Stream, error) {

	parent, err := w.streamService.LoadParent(&w.stream)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.render.stream.Parent", "Error loading Parent")
	}

	result := NewStream(w.layoutService, w.folderService, w.templateService, w.streamService, *parent, w.View())

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

// SubTemplates returns an array of templates that can be placed inside this Stream
func (w Stream) SubTemplates() ([]model.Template) {
	return w.templateService.ListByContainer(w.stream.Template)
}
