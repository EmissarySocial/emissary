package service

import (
	"github.com/benpate/data"
	"github.com/benpate/data/expression"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/ghost/model"
	"github.com/benpate/path"
	"github.com/benpate/schema"
	"github.com/davecgh/go-spew/spew"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Stream manages all interactions with the Stream collection
type Stream struct {
	collection          data.Collection
	templateService     *Template
	formLibrary         form.Library
	streamUpdateChannel chan model.Stream
}

// NewStream returns a fully populated Stream service.
func NewStream(collection data.Collection, templateService *Template, formLibrary form.Library, updates chan model.Stream) *Stream {
	return &Stream{
		collection:          collection,
		templateService:     templateService,
		formLibrary:         formLibrary,
		streamUpdateChannel: updates,
	}
}

// New creates a newly initialized Stream that is ready to use
func (service Stream) New() *model.Stream {
	result := model.NewStream()
	return &result
}

// List returns an iterator containing all of the Streams who match the provided criteria
func (service Stream) List(criteria expression.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(criteria, options...)
}

// Load retrieves an Stream from the database
func (service Stream) Load(criteria expression.Expression) (*model.Stream, error) {

	stream := service.New()

	if err := service.collection.Load(criteria, stream); err != nil {
		return nil, derp.Wrap(err, "ghost.service.Stream", "Error loading Stream", criteria)
	}

	return stream, nil
}

// Save adds/updates an Stream in the database
func (service Stream) Save(stream *model.Stream, note string) error {

	if err := service.collection.Save(stream, note); err != nil {
		return derp.Wrap(err, "ghost.service.Stream", "Error saving Stream", stream, note)
	}

	service.streamUpdateChannel <- *stream

	return nil
}

// Delete removes an Stream from the database (virtual delete)
func (service Stream) Delete(stream *model.Stream, note string) error {

	if err := service.collection.Delete(stream, note); err != nil {
		return derp.Wrap(err, "ghost.service.Stream", "Error deleting Stream", stream, note)
	}

	return nil
}

// QUERIES /////////////////////////

// ListByParent returns all Streams that match a particular parentID
func (service Stream) ListByParent(parentID primitive.ObjectID) (data.Iterator, error) {
	return service.List(
		expression.
			Equal("parentId", parentID).
			AndEqual("journal.deleteDate", 0))
}

// ListTopFolders returns all Streams of type FOLDER at the top of the hierarchy
func (service Stream) ListTopFolders() (data.Iterator, error) {
	return service.List(
		expression.
			Equal("template", "folder").
			AndEqual("parentId", ZeroObjectID()).
			AndEqual("journal.deleteDate", 0))
}

// ListByTemplate returns all Streams that use a particular Template
func (service Stream) ListByTemplate(template string) (data.Iterator, error) {
	return service.List(
		expression.
			Equal("template", template).
			AndEqual("journal.deleteDate", 0))
}

// LoadByToken returns a single Stream that matches a particular Token
func (service Stream) LoadByToken(token string) (*model.Stream, error) {
	return service.Load(
		expression.
			Equal("token", token).
			AndEqual("journal.deleteDate", 0))
}

// LoadByID returns a single Stream that matches a particular StreamID
func (service Stream) LoadByID(streamID primitive.ObjectID) (*model.Stream, error) {
	return service.Load(
		expression.
			Equal("_id", streamID).
			AndEqual("journal.deleteDate", 0))
}

// LoadBySourceURL locates a single stream that matches the provided SourceURL
func (service Stream) LoadBySourceURL(url string) (*model.Stream, error) {
	return service.Load(
		expression.
			Equal("sourceUrl", url).
			AndEqual("journal.deleteDate", 0))
}

// LoadParent returns the Stream that is the parent of the provided Stream
func (service Stream) LoadParent(stream *model.Stream) (*model.Stream, error) {

	if stream.HasParent() == false {
		return nil, derp.New(404, "ghost.service.Stream.LoadParent", "Stream does not have a parent")
	}

	stream, err := service.LoadByID(stream.ParentID)

	return stream, derp.Wrap(err, "ghost.service.stream.LoadParent", "Error loading parent", stream)
}

// ChildTemplates returns an iterator of Templates that can be added as a sub-stream
func (service Stream) ChildTemplates(stream *model.Stream) []model.Template {
	return service.templateService.ListByContainer(stream.TemplateID)
}

// CUSTOM ACTIONS /////////////////

// NewWithTemplate creates a new Stream using the provided Template and Parent information.
func (service Stream) NewWithTemplate(parentToken string, templateID string) (*model.Stream, error) {

	spew.Dump(templateID, parentToken)

	// Exception for putting a folder on the top level...
	if parentToken == "top" {

		if templateID == "folder" {
			// Create and populate the new Stream
			stream := service.New()
			stream.ParentID = ZeroObjectID()
			stream.TemplateID = templateID

			return stream, nil
		}

		return nil, derp.New(400, "ghost.service.Stream.NewWithTemplate", "Top Level Can Only Contain Folders")
	}

	// Load the requested Template
	template, err := service.templateService.Load(templateID)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.service.Stream.NewWithTemplate", "Error loading Template", templateID)
	}

	// Load the parent Stream
	parent, err := service.LoadByToken(parentToken)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.service.Stream.NewWithTemplate", "Error loading parent stream", parentToken)
	}

	// Confirm that this Template can be a child of the parent Template
	if template.CanBeContainedBy(parent.TemplateID) == false {
		return nil, derp.Wrap(err, "ghost.service.Stream.NewWithTemplate", "Invalid template")
	}

	// Create and populate the new Stream
	stream := service.New()
	stream.ParentID = parent.StreamID
	stream.TemplateID = template.TemplateID

	// Success.  We've made the new stream!
	return stream, nil
}

// View returns the named View for the designated Stream.
func (service Stream) View(stream *model.Stream, viewName string) (*model.View, error) {

	state, err := service.State(stream)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.serice.Stream.View", "Error loading template", stream.TemplateID)
	}

	view, ok := state.View(viewName)

	if ok == false {
		return nil, derp.New(500, "ghost.service.Stream.View", "Error loading view", viewName)
	}

	return view, nil
}

// Form generates an HTML form for the requested Stream and TransitionID
func (service Stream) Form(stream *model.Stream, transitionID string) (string, error) {

	_, transition, err := service.Transition(stream, transitionID)

	if err != nil {
		return "", derp.Wrap(err, "ghost.service.Stream.Form", "Unrecognized State/Transition", transitionID)
	}

	schema, err := service.Schema(stream)

	if err != nil {
		return "", derp.Wrap(err, "ghost.service.Stream.Form", "Invalid Schema")
	}

	result, err := transition.Form.HTML(service.formLibrary, *schema, stream)

	if err != nil {
		return "", derp.Wrap(err, "ghost.service.Stream.Form", "Error generating form")
	}

	return result, nil
}

// DoTransition handles a transition request to move the stream from one state into another state.
func (service Stream) DoTransition(stream *model.Stream, transitionID string, data map[string]interface{}) (*model.Transition, error) {

	_, transition, err := service.Transition(stream, transitionID)

	if err != nil {
		return transition, derp.Wrap(err, "ghost.service.Stream.Transition", "Unrecognized State/Transition", transitionID)
	}

	form := transition.Form

	// TODO: where are permissions processed?

	paths := form.AllPaths()

	// Only look for data in the registered paths for this form.  If other data is present, it will be ignored.
	for _, element := range paths {

		// TODO: What about form validation?  Can this happen HERE as well as in the template schema?

		// If we have a value, then set it.
		if value, ok := data[element.Path]; ok {
			if err := path.Set(stream, element.Path, value); err != nil {
				return transition, derp.Wrap(err, "ghost.service.Stream.Transition", "Error updating stream", element, value)
			}
		}
		// TODO: Otherwise?  Should this form throw an error?
	}

	// Update the stream to the new state
	stream.StateID = transition.NextState

	// TODO:  Actions will be processes here.

	if err := service.Save(stream, "stream transition: "+transitionID); err != nil {
		return transition, derp.Wrap(err, "ghost.service.Stream.Transition", "Error saving stream")
	}

	return transition, nil
}

// State returns the detailed State information associated with this Stream
func (service Stream) State(stream *model.Stream) (*model.State, error) {

	// Locate the Template used by this Stream
	template, err := service.templateService.Load(stream.TemplateID)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.service.Stream.State", "Invalid Template", stream.TemplateID)
	}

	// Populate the Stream with data from the Template
	state, ok := template.State(stream.StateID)

	if ok == false {
		return nil, derp.New(500, "ghost.service.Stream.State", "Invalid state", stream.StateID)
	}

	return state, nil
}

// Transition returns the detailed Transition information assoicated with this Stream
func (service Stream) Transition(stream *model.Stream, transitionID string) (*model.State, *model.Transition, error) {

	state, err := service.State(stream)

	if err != nil {
		return nil, nil, derp.Wrap(err, "ghost.service.Stream.Transition", "Error geting State definition")
	}

	transition, ok := state.Transition(transitionID)

	if ok == false {
		return nil, nil, derp.Wrap(err, "ghost.service.Stream.Transition", "Error geting Transition")
	}

	return state, transition, nil
}

// Schema returns the Schema associated with this Stream
func (service Stream) Schema(stream *model.Stream) (*schema.Schema, error) {

	// Locate the Template used by this Stream
	template, err := service.templateService.Load(stream.TemplateID)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.service.Stream", "Invalid Template", stream)
	}

	return template.Schema, nil
}
