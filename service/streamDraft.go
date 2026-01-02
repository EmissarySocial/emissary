package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/journal"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StreamDraft manages all interactions with the StreamDraft collection
type StreamDraft struct {
	templateService *Template
	streamService   *Stream
}

// NewStreamDraft returns a fully populated StreamDraft service.
func NewStreamDraft() StreamDraft {
	return StreamDraft{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *StreamDraft) Refresh(factory *Factory) {
	service.templateService = factory.Template()
	service.streamService = factory.Stream()
}

// Close stops any background processes controlled by this service
func (service *StreamDraft) Close() {

}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *StreamDraft) collection(session data.Session) data.Collection {
	return session.Collection("StreamDraft")
}

// New creates a newly initialized StreamDraft that is ready to use
func (service *StreamDraft) New() model.Stream {
	return model.NewStream()
}

// Count returns the number of records that match the provided criteria
func (service *StreamDraft) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Load either: 1) loads a valid draft from the database, or 2) creates a new draft and returns it instead
func (service *StreamDraft) Load(session data.Session, criteria exp.Expression, result *model.Stream) error {

	const location = "service.StreamDraft.Load"

	// Try to load a draft using the provided criteria
	if err := service.collection(session).Load(criteria, result); err == nil {
		return nil
	} else if !derp.IsNotFound(err) {
		derp.Report(derp.Wrap(err, location, "Unable to load StreamDraft"))
	}

	// Fall through means we could not load a draft (probably 404 not found)

	// Try to locate the original stream
	if err := service.streamService.Load(session, criteria, result); err != nil {
		return derp.Wrap(err, location, "Unable to load original stream")
	}

	// Reset the journal so that this item can be saved in the new collection.
	result.Journal = journal.Journal{}

	// Save a draft copy of the original stream
	if err := service.Save(session, result, "create draft record"); err != nil {
		return derp.Wrap(err, location, "Unable to save draft", criteria)
	}

	// Return the original stream as a new draft to use.
	return nil
}

// save adds/updates an StreamDraft in the database
func (service *StreamDraft) Save(session data.Session, draft *model.Stream, note string) error {

	const location = "service.StreamDraft.Save"

	// Get the Template used by this StreamDraft
	template, err := service.templateService.Load(draft.TemplateID)

	if err != nil {
		return derp.Wrap(err, location, "Invalid Template", draft.TemplateID)
	}

	// Validate the value (using the global stream schema) before saving
	if err := service.Schema().Validate(draft); err != nil {
		return derp.Wrap(err, location, "Unable to validate Stream using StreamSchema", draft)
	}

	// Validate the value (using the template-specific schema) before saving
	if err := template.Schema.Validate(draft); err != nil {
		return derp.Wrap(err, location, "Unable to validate Stream using TemplateSchema", draft)
	}

	if err := service.collection(session).Save(draft, note); err != nil {
		return derp.Wrap(err, location, "Unable to save draft", draft, note)
	}

	return nil
}

// Delete removes an StreamDraft from the database (hard delete)
func (service *StreamDraft) Delete(session data.Session, draft *model.Stream, _note string) error {

	criteria := exp.Equal("_id", draft.StreamID)

	// Use a hard delete to remove drafts permanently.
	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, "service.StreamDraft.Delete", "Unable to delete draft", criteria)
	}

	return nil
}

/******************************************
 * Generic Data Functions
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *StreamDraft) ObjectType() string {
	return "StreamDraft"
}

// New returns a fully initialized model.StreamDraft as a data.Object.
func (service *StreamDraft) ObjectNew() data.Object {
	result := model.NewStream()
	return &result
}

func (service *StreamDraft) ObjectID(object data.Object) primitive.ObjectID {

	if streamDraft, ok := object.(*model.Stream); ok {
		return streamDraft.StreamID
	}

	return primitive.NilObjectID
}

func (service *StreamDraft) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

func (service *StreamDraft) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewStream()
	err := service.Load(session, criteria, &result)
	return &result, err
}

func (service *StreamDraft) ObjectSave(session data.Session, object data.Object, comment string) error {
	if stream, ok := object.(*model.Stream); ok {
		return service.Save(session, stream, comment)
	}
	return derp.Internal("service.StreamDraft.ObjectSave", "Invalid Object Type", object)
}

func (service *StreamDraft) ObjectDelete(session data.Session, object data.Object, comment string) error {
	if stream, ok := object.(*model.Stream); ok {
		return service.Delete(session, stream, comment)
	}
	return derp.Internal("service.StreamDraft.ObjectDelete", "Invalid Object Type", object)
}

func (service *StreamDraft) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.Unauthorized("service.StreamDraft", "Not Authorized")
}

func (service *StreamDraft) Schema() schema.Schema {
	return schema.New(model.StreamSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

// LoadByID returns a single Stream that matches a particular StreamID
func (service *StreamDraft) LoadByID(session data.Session, streamID primitive.ObjectID, result *model.Stream) error {
	criteria := exp.Equal("_id", streamID)
	return service.Load(session, criteria, result)
}

/******************************************
 * Custom Actions
 ******************************************/

func (service *StreamDraft) Promote(session data.Session, streamID primitive.ObjectID, stateID string) (model.Stream, error) {

	var draft model.Stream
	var stream model.Stream

	// Try to load the draft
	if err := service.LoadByID(session, streamID, &draft); err != nil {
		return model.Stream{}, derp.Wrap(err, "service.StreamDraft.Publish", "Unable to load draft")
	}

	// Try to load the production stream
	if err := service.streamService.LoadByID(session, streamID, &stream); err != nil {
		return model.Stream{}, derp.Wrap(err, "service.StreamDraft.Publish", "Unable to load draft")
	}

	// Copy data from draft to production
	stream.URL = draft.URL
	stream.Token = draft.Token
	stream.Label = draft.Label
	stream.Summary = draft.Summary
	stream.IconURL = draft.IconURL
	stream.Icon = draft.Icon
	stream.Widgets = draft.Widgets
	stream.Content = draft.Content
	stream.Data = draft.Data
	stream.AttributedTo = draft.AttributedTo
	stream.InReplyTo = draft.InReplyTo
	stream.StateID = stateID
	stream.DeleteDate = 0 // just in case...

	// Try to save the updated stream back to the database
	if err := service.streamService.Save(session, &stream, "published"); err != nil {
		return model.Stream{}, derp.Wrap(err, "service.StreamDraft.Publish", "Error publishing stream")
	}

	// Try to save the updated stream back to the database
	if err := service.Delete(session, &draft, "published"); err != nil {
		return model.Stream{}, derp.Wrap(err, "service.StreamDraft.Publish", "Unable to delete draft")
	}

	return stream, nil
}
