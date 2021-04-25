package service

import (
	"fmt"

	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/form"
	"github.com/benpate/ghost/model"
	"github.com/benpate/path"
	"github.com/benpate/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Stream manages all interactions with the Stream collection
type Stream struct {
	collection            data.Collection
	templateService       *Template
	formLibrary           form.Library
	templateUpdateChannel chan model.Template
	streamUpdateChannel   chan model.Stream
}

// NewStream returns a fully populated Stream service.
func NewStream(collection data.Collection, templateService *Template, formLibrary form.Library, templateUpdateChannel chan model.Template, streamUpdateChannel chan model.Stream) *Stream {
	result := &Stream{
		collection:            collection,
		templateService:       templateService,
		formLibrary:           formLibrary,
		templateUpdateChannel: templateUpdateChannel,
		streamUpdateChannel:   streamUpdateChannel,
	}

	go result.start()

	return result
}

// New creates a newly initialized Stream that is ready to use
func (service *Stream) New() *model.Stream {
	result := model.NewStream()
	return &result
}

// start begins the background watchers used by the Stream Service
func (service *Stream) start() {
	for {
		template := <-service.templateUpdateChannel
		fmt.Println("streamService.start: received update to template: " + template.Label)
		service.templateService.Save(&template)
		service.updateStreamsByTemplate(&template)
	}
}

// List returns an iterator containing all of the Streams who match the provided criteria
func (service *Stream) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(criteria, options...)
}

// Load retrieves an Stream from the database
func (service Stream) Load(criteria exp.Expression) (*model.Stream, error) {

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

	// NON-BLOCKING: Notify other processes on this server that the stream has been updated
	go func() {
		fmt.Println("streamService.Save: sending update update to stream: " + stream.Label)
		service.streamUpdateChannel <- *stream
	}()

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
		exp.
			Equal("parentId", parentID).
			AndEqual("journal.deleteDate", 0))
}

// ListTopFolders returns all Streams of type FOLDER at the top of the hierarchy
func (service Stream) ListTopFolders() (data.Iterator, error) {
	return service.List(
		exp.
			Equal("templateId", "folder").
			AndEqual("parentId", ZeroObjectID()).
			AndEqual("journal.deleteDate", 0))
}

// ListByTemplate returns all Streams that use a particular Template
func (service Stream) ListByTemplate(template string) (data.Iterator, error) {
	return service.List(
		exp.
			Equal("templateId", template).
			AndEqual("journal.deleteDate", 0))
}

// LoadByToken returns a single Stream that matches a particular Token
func (service Stream) LoadByToken(token string) (*model.Stream, error) {
	return service.Load(
		exp.
			Equal("token", token).
			AndEqual("journal.deleteDate", 0))
}

// LoadByID returns a single Stream that matches a particular StreamID
func (service Stream) LoadByID(streamID primitive.ObjectID) (*model.Stream, error) {
	return service.Load(
		exp.
			Equal("_id", streamID).
			AndEqual("journal.deleteDate", 0))
}

// LoadBySourceURL locates a single stream that matches the provided SourceURL
func (service Stream) LoadBySourceURL(url string) (*model.Stream, error) {
	return service.Load(
		exp.
			Equal("sourceUrl", url).
			AndEqual("journal.deleteDate", 0))
}

// LoadParent returns the Stream that is the parent of the provided Stream
func (service Stream) LoadParent(stream *model.Stream) (*model.Stream, error) {

	if !stream.HasParent() {
		return nil, derp.New(404, "ghost.service.Stream.LoadParent", "Stream does not have a parent")
	}

	stream, err := service.LoadByID(stream.ParentID)

	return stream, derp.Wrap(err, "ghost.service.stream.LoadParent", "Error loading parent", stream)
}

// ChildTemplates returns an iterator of Templates that can be added as a sub-stream
func (service Stream) ChildTemplates(stream *model.Stream) []model.Template {
	return service.templateService.ListByContainer(stream.TemplateID)
}

// PERMISSIONS /////////////////

func (service Stream) View(stream *model.Stream, viewID string, authorization *model.Authorization) (*model.View, bool) {

	state, err := service.State(stream)

	if err != nil {
		return nil, false
	}

	// Verify that this view is accessible by the user's roles
	view, ok := state.View(viewID)

	if !ok {
		return nil, false
	}

	roles := stream.Roles(authorization)

	if !view.MatchRoles(roles...) {
		return nil, false
	}

	return view, true
}

func (service Stream) Transition(stream *model.Stream, transitionID string, authorization *model.Authorization) (*model.Transition, bool) {

	state, err := service.State(stream)

	if err != nil {
		return nil, false
	}

	// Verify that this view is accessible by the user's roles
	transition, ok := state.Transition(transitionID)

	if !ok {
		return nil, false
	}

	roles := stream.Roles(authorization)

	if !transition.MatchRoles(roles...) {
		return nil, false
	}

	return transition, true
}

// CUSTOM ACTIONS /////////////////

// NewWithTemplate creates a new Stream using the provided Template and Parent information.
func (service Stream) NewWithTemplate(parentToken string, templateID string) (*model.Stream, error) {

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
	if !template.CanBeContainedBy(parent.TemplateID) {
		return nil, derp.Wrap(err, "ghost.service.Stream.NewWithTemplate", "Invalid template")
	}

	// Create and populate the new Stream
	stream := service.New()
	stream.ParentID = parent.StreamID
	stream.TemplateID = template.TemplateID

	// Success.  We've made the new stream!
	return stream, nil
}

// Form generates an HTML form for the requested Stream and TransitionID
func (service Stream) Form(stream *model.Stream, transition *model.Transition) (string, error) {

	schema, err := service.Schema(stream)

	if err != nil {
		return "", derp.Wrap(err, "ghost.service.Stream.Form", "Invalid Schema")
	}

	result, err := transition.Form.HTML(service.formLibrary, schema, stream)

	if err != nil {
		return "", derp.Wrap(err, "ghost.service.Stream.Form", "Error generating form")
	}

	return result, nil
}

// DoTransition handles a transition request to move the stream from one state into another state.
func (service Stream) DoTransition(stream *model.Stream, transitionID string, data map[string]interface{}, authorization *model.Authorization) (*model.Transition, error) {

	transition, ok := service.Transition(stream, transitionID, authorization)

	if !ok {
		return nil, derp.New(derp.CodeForbiddenError, "ghost.service.Stream.Transition", "Unauthorized State/Transition", transitionID)
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

	if !ok {
		return nil, derp.New(500, "ghost.service.Stream.State", "Invalid state", stream.StateID)
	}

	return state, nil
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

// updateStreamsByTemplate updates every stream that uses a particular template.
func (service *Stream) updateStreamsByTemplate(template *model.Template) {

	iterator, err := service.ListByTemplate(template.TemplateID)

	if err != nil {
		derp.Report(derp.Wrap(err, "ghost.service.Realtime", "Error Listing Streams for Template", template))
		return
	}

	var stream model.Stream

	for iterator.Next(&stream) {
		fmt.Println("streamService.updateStreamsByTemplate: Sending stream: " + stream.Label)
		service.streamUpdateChannel <- stream
	}

	fmt.Println("streamService.updateStreamsByTemplate: End of Iterator.")
}
