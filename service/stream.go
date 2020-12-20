package service

import (
	"github.com/benpate/data"
	"github.com/benpate/data/expression"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/path"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Stream manages all interactions with the Stream collection
type Stream struct {
	templateService     *Template
	collection          data.Collection
	streamUpdateChannel chan model.Stream
}

// NewStream returns a fully populated Stream service.
func NewStream(templateService *Template, collection data.Collection, updates chan model.Stream) *Stream {
	return &Stream{
		templateService:     templateService,
		collection:          collection,
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
		return nil, derp.Wrap(err, "service.Stream", "Error loading Stream", criteria)
	}

	return stream, nil
}

// Save adds/updates an Stream in the database
func (service Stream) Save(stream *model.Stream, note string) error {

	if err := service.collection.Save(stream, note); err != nil {
		return derp.Wrap(err, "service.Stream", "Error saving Stream", stream, note)
	}

	// service.streamUpdateChannel <- *stream

	return nil
}

// Delete removes an Stream from the database (virtual delete)
func (service Stream) Delete(stream *model.Stream, note string) error {

	if err := service.collection.Delete(stream, note); err != nil {
		return derp.Wrap(err, "service.Stream", "Error deleting Stream", stream, note)
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

// CUSTOM ACTIONS /////////////////

// NewWithTemplate creates a new Stream using the provided Parent and Token information.
func (service Stream) NewWithTemplate(parentToken string, templateID string) (*model.Stream, error) {

	if templateID == "" {
		return nil, derp.New(500, "ghost.service.Stream.NewWithTemplate", "Missing template parameter")
	}

	// Load the requested template
	template, err := service.templateService.Load(templateID)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.service.Stream.NewWithTemplate", "Error loading Template")
	}

	// Load the parent stream to validate permissions
	stream := service.New()
	stream.Template = template.TemplateID

	// If this is a TOP LEVEL item...
	if parentToken == "top" {

		// verify that it can be placed at the top of the hierarchy
		if !template.CanBeContainedBy(parentToken) {
			return nil, derp.New(400, "ghost.service.Stream.NewWithTemplate", "Invalid template")
		}

		stream.ParentID = ZeroObjectID()

	} else {

		// Otherwise, verify that it can be placed within its parent stream

		// Load the parent stream
		parent, err := service.LoadByToken(parentToken)

		if err != nil {
			return nil, derp.Wrap(err, "ghost.service.Stream.NewWithTemplate", "Error loading parent stream")
		}

		// Verify that the child stream can be placed inside the parent
		if !template.CanBeContainedBy(parent.Template) {
			return nil, derp.Wrap(err, "ghost.service.Stream.NewWithTemplate", "Invalid template")
		}

		stream.ParentID = parent.StreamID
	}

	// Success.  We've made the new stream!
	return stream, nil
}

///////////////////

// Transition handles a transition request to move the stream from one state into another state.
func (service Stream) Transition(stream *model.Stream, transitionID string, data map[string]interface{}) (*model.Transition, error) {

	template, err := service.templateService.Load(stream.Template)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.service.Stream.Transition", "Can't load Template")
	}

	transition, err := template.Transition(stream.State, transitionID)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.service.Stream.Transition", "Unrecognized State/Tranition")
	}

	form, ok := template.Forms[transition.Form]

	if !ok {
		return nil, derp.New(404, "ghost.service.Stream.Transition", "Unrecognized Form for Transition", transition)
	}

	// TODO: where are permissions processed?

	paths := form.AllPaths()

	// Only look for data in the registered paths for this form.  If other data is present, it will be ignored.
	for _, element := range paths {

		// TODO: What about form validation?  Can this happen HERE as well as in the template schema?

		// If we have a value, then set it.
		if value, ok := data[element.Path]; ok {
			if err := path.Set(stream, element.Path, value); err != nil {
				return nil, derp.Wrap(err, "ghost.service.Stream.Transition", "Error updating stream", element, value)
			}
		}
		// TODO: Otherwise?  Should this form throw an error?
	}

	// Update the stream to the new state
	stream.State = transition.NextState

	// TODO:  Actions will be processes here.

	if err := service.Save(stream, "stream transition: "+transitionID); err != nil {
		return nil, derp.Wrap(err, "ghost.service.Stream.Transition", "Error saving stream")
	}

	return transition, nil
}
