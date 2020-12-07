package service

import (
	"bytes"
	"html/template"
	"net/url"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
)

// Renderer wraps a model.Stream object and provides functions that make it easy to render an HTML template with it.
type Renderer struct {
	factory *Factory
	stream  *model.Stream
	query   url.Values
}

// StreamID returns the unique ID for the stream being rendered
func (w Renderer) StreamID() string {
	return w.stream.StreamID.Hex()
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

// View returns the string name of the view requested in the URL QueryString
func (w Renderer) View() string {
	return w.query.Get("view")
}

// Transition returns the string name of the transition requested in the URL QueryString
func (w Renderer) Transition() string {
	return w.query.Get("transition")
}

////////////////////////////////

// CanAddChild returns TRUE if a child element can be added
func (w Renderer) CanAddChild() bool {
	return true
}

////////////////////////////////

// Views returns a slice of view maps, containing the Name and Label of each eligible view.
func (w Renderer) Views() []map[string]string {

	template, _ := w.factory.Template().Load(w.stream.Template)

	result := []map[string]string{}

	for _, view := range template.Views {

		// Available views can be filtered here...

		// Add this view to the list.
		result = append(result, map[string]string{
			"Name":  view.Name,
			"Label": view.Label,
		})
	}

	return result
}

// Folders returns all top-level folders for the domain
func (w Renderer) Folders() ([]*Renderer, error) {

	folders, err := w.factory.Stream().ListTopFolders()

	if err != nil {
		return nil, derp.Wrap(err, "ghost.service.Renderer.Folders", "Error loading top-level folders")
	}

	return w.iteratorToSlice(folders)

}

// Parent returns a Stream containing the parent of the current stream
func (w Renderer) Parent() (*Renderer, error) {

	parent, err := w.factory.Stream().LoadParent(w.stream)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.service.Renderer.Parent", "Error loading Parent")
	}

	result := w.factory.StreamRenderer(parent, w.query)

	return result, nil
}

// Children returns an array of Streams containing all of the child elements of the current stream
func (w Renderer) Children() ([]*Renderer, error) {

	iterator, err := w.factory.Stream().ListByParent(w.stream.StreamID)

	if err != nil {
		return nil, derp.Report(derp.Wrap(err, "ghost.service.Renderer.Children", "Error loading child streams", w.stream))
	}

	return w.iteratorToSlice(iterator)
}

// SubTemplates returns an array of templates that can be placed inside this Stream
func (w Renderer) SubTemplates() []model.Template {
	return w.factory.Template().ListByContainer(w.stream.Template)
}

// iteratorToSlice converts a data.Iterator of Streams into a slice of Streams
func (w Renderer) iteratorToSlice(iterator data.Iterator) ([]*Renderer, error) {

	var stream model.Stream

	result := make([]*Renderer, iterator.Count())

	for iterator.Next(&stream) {
		copy := stream
		result = append(result, w.factory.StreamRenderer(&copy, w.query))
	}

	return result, nil
}

/// RENDERING METHODS

// Render generates an HTML output for a stream/view combination.
func (w Renderer) Render(viewName string) (template.HTML, error) {

	var result bytes.Buffer

	layout := w.factory.Layout().Layout()

	// Load stream content
	_, content, err := w.factory.Template().LoadCompiled(w.stream.Template, w.stream.State, viewName)

	if err != nil {
		return "", derp.Wrap(err, "ghost.service.Renderer.Render", "Unable to load stream template")
	}

	// Combine the two parse trees.
	// TODO: Could this be done at load time, not for each page request?
	combined, err := layout.AddParseTree("content", content.Tree)

	if err != nil {
		return "", derp.Wrap(err, "ghost.service.Renderer.Render", "Unable to create parse tree")
	}

	// Choose the correct view based on the wrapper provided.
	if err := combined.Funcs(sprig.FuncMap()).ExecuteTemplate(&result, "stream", w); err != nil {
		return "", derp.Wrap(err, "ghost.service.Renderer.Render", "Error rendering view")
	}

	// TODO: Add caching...

	// Success!
	return template.HTML(result.String()), nil
}

// Form returns an HTML rendering of this form
func (w Renderer) Form() (template.HTML, error) {

	var result template.HTML

	t, err := w.factory.Template().Load(w.stream.Template)

	if err != nil {
		return result, derp.Report(derp.Wrap(err, "ghost.handler.GetForm", "Cannot load template"))
	}

	// TODO: Validate that this transition is VALID
	// TODO: Validate that the USER IS PERMITTED to make this transition.

	transition := w.Transition()

	form, err := t.Form(w.stream.State, transition)

	if err != nil {
		return result, derp.Report(derp.Wrap(err, "ghost.handler.GetForm", "Invalid Form", t))
	}

	// Generate HTML by merging the form with the element library, the data schema, and the data value
	html, err := form.HTML(w.factory.FormLibrary(), *t.Schema, w.stream)

	if err != nil {
		return result, derp.Report(derp.Wrap(err, "ghost.handler.GetForm", "Error generating form HTML", form))
	}

	return template.HTML(html), nil
}
