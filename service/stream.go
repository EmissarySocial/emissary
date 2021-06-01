package service

import (
	"fmt"

	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/form"
	"github.com/benpate/ghost/action"
	"github.com/benpate/ghost/model"
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

	result := Stream{
		collection:            collection,
		templateService:       templateService,
		formLibrary:           formLibrary,
		templateUpdateChannel: templateUpdateChannel,
		streamUpdateChannel:   streamUpdateChannel,
	}

	go result.start()

	return &result
}

// New creates a newly initialized Stream that is ready to use
func (service *Stream) New() model.Stream {
	return model.NewStream()
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
func (service *Stream) Load(criteria exp.Expression, stream *model.Stream) error {

	if err := service.collection.Load(criteria, stream); err != nil {
		return derp.Wrap(err, "ghost.service.Stream", "Error loading Stream", criteria)
	}

	return nil
}

// Save adds/updates an Stream in the database
func (service *Stream) Save(stream *model.Stream, note string) error {

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
func (service *Stream) Delete(stream *model.Stream, note string) error {

	if err := service.collection.Delete(stream, note); err != nil {
		return derp.Wrap(err, "ghost.service.Stream", "Error deleting Stream", stream, note)
	}

	return nil
}

// QUERIES /////////////////////////

// ListByParent returns all Streams that match a particular parentID
func (service *Stream) ListByParent(parentID primitive.ObjectID) (data.Iterator, error) {
	return service.List(
		exp.
			Equal("parentId", parentID).
			AndEqual("journal.deleteDate", 0))
}

// ListTopFolders returns all Streams of type FOLDER at the top of the hierarchy
func (service *Stream) ListTopFolders() (data.Iterator, error) {
	return service.List(
		exp.
			Equal("parentId", ZeroObjectID()).
			AndEqual("journal.deleteDate", 0))
}

// ListByTemplate returns all Streams that use a particular Template
func (service *Stream) ListByTemplate(template string) (data.Iterator, error) {
	return service.List(
		exp.
			Equal("templateId", template).
			AndEqual("journal.deleteDate", 0))
}

// LoadByToken returns a single Stream that matches a particular Token
func (service *Stream) LoadByToken(token string, result *model.Stream) error {

	criteria := exp.
		Equal("token", token).
		AndEqual("journal.deleteDate", 0)

	return service.Load(criteria, result)
}

// LoadByID returns a single Stream that matches a particular StreamID
func (service *Stream) LoadByID(streamID primitive.ObjectID, result *model.Stream) error {

	criteria := exp.
		Equal("_id", streamID).
		AndEqual("journal.deleteDate", 0)

	return service.Load(criteria, result)
}

// LoadBySourceURL locates a single stream that matches the provided SourceURL
func (service *Stream) LoadBySource(parentStreamID primitive.ObjectID, sourceURL string, result *model.Stream) error {

	criteria := exp.
		Equal("parentId", parentStreamID).
		AndEqual("sourceUrl", sourceURL)

	return service.Load(criteria, result)
}

// LoadParent returns the Stream that is the parent of the provided Stream
func (service *Stream) LoadParent(stream *model.Stream, parent *model.Stream) error {

	if !stream.HasParent() {
		return derp.New(404, "ghost.service.Stream.LoadParent", "Stream does not have a parent")
	}

	if err := service.LoadByID(stream.ParentID, parent); err != nil {
		derp.Wrap(err, "ghost.service.stream.LoadParent", "Error loading parent", stream)
	}

	return nil
}

// ChildTemplates returns an iterator of Templates that can be added as a sub-stream
func (service *Stream) ChildTemplates(stream *model.Stream) []model.Template {
	return service.templateService.ListByContainer(stream.TemplateID)
}

// CUSTOM ACTIONS /////////////////

// NewWithTemplate creates a new Stream using the provided Template and Parent information.
func (service *Stream) NewWithTemplate(parentToken string, templateID string, result *model.Stream) error {

	// Exception for putting a folder on the top level...
	if parentToken == "top" {

		if templateID == "folder" {
			// Create and populate the new Stream
			result.ParentID = ZeroObjectID()
			result.TemplateID = templateID

			return nil
		}

		return derp.New(400, "ghost.service.Stream.NewWithTemplate", "Top Level Can Only Contain Folders")
	}

	// Load the requested Template
	template, err := service.templateService.Load(templateID)

	if err != nil {
		return derp.Wrap(err, "ghost.service.Stream.NewWithTemplate", "Error loading Template", templateID)
	}

	var parent model.Stream

	// Load the parent Stream
	if err := service.LoadByToken(parentToken, &parent); err != nil {
		return derp.Wrap(err, "ghost.service.Stream.NewWithTemplate", "Error loading parent stream", parentToken)
	}

	// Confirm that this Template can be a child of the parent Template
	if !template.CanBeContainedBy(parent.TemplateID) {
		return derp.Wrap(err, "ghost.service.Stream.NewWithTemplate", "Invalid template")
	}

	// Create and populate the new Stream
	result.ParentID = parent.StreamID
	result.TemplateID = template.TemplateID

	// Success.  We've made the new stream!
	return nil
}

// Form generates an HTML form for the requested Stream and TransitionID
func (service *Stream) Form(stream *model.Stream, transition *model.Transition) (string, error) {

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

// State returns the detailed State information associated with this Stream
func (service *Stream) State(stream *model.Stream) (model.State, error) {

	// Try to find the Template used by this Stream
	template, err := service.templateService.Load(stream.TemplateID)

	if err != nil {
		return model.State{}, derp.Wrap(err, "ghost.service.Stream.State", "Invalid Template", stream.TemplateID)
	}

	// Try to find the state data for the state that the stream is in
	state, ok := template.State(stream.StateID)

	if !ok {
		return state, derp.New(500, "ghost.service.Stream.State", "Invalid state", stream.StateID)
	}

	// Success!
	return state, nil
}

// Schema returns the Schema associated with this Stream
func (service *Stream) Schema(stream *model.Stream) (*schema.Schema, error) {

	// Try to locate the Template used by this Stream
	template, err := service.templateService.Load(stream.TemplateID)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.service.Stream.Action", "Invalid Template", stream)
	}

	// Return the Schema defined in this template.
	return template.Schema, nil
}

// Action returns the action definition that matches the stream and type provided
func (service *Stream) Action(stream *model.Stream, actionID string) (action.Action, error) {

	// Try to find the Template used by this Stream
	template, err := service.templateService.Load(stream.TemplateID)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.service.Stream.Action", "Invalid Template", stream)
	}

	// Try to find the action in the Template
	if action, ok := template.Action(actionID); ok {
		return action, nil
	}

	// Success!
	return nil, derp.New(derp.CodeBadRequestError, "ghost.service.Stream.Action", "Unrecognized action", actionID)
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
		stream = model.Stream{}
	}

	fmt.Println("streamService.updateStreamsByTemplate: End of Iterator.")
}
