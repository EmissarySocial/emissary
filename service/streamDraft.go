package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/journal"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"
	"github.com/rs/zerolog/log"
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

// collection returns the StreamDraft collection for the provided database session
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
		derp.Report(derp.Wrap(err, location, "Loading StreamDraft"))
	}

	// Fall through means we could not load a draft (probably 404 not found)

	// Try to locate the original stream
	if err := service.streamService.Load(session, criteria, result); err != nil {
		return derp.Wrap(err, location, "Loading original stream")
	}

	// Reset the journal so that this item can be saved in the new collection.
	result.Journal = journal.Journal{}

	// Save a draft copy of the original stream
	if err := service.Save(session, result, "create draft record"); err != nil {
		return derp.Wrap(err, location, "Saving draft", criteria)
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

	// Normalize the value (using the template-specific schema) before saving.  Values are
	// rewritten in place to conform to the schema, so that legacy data written under older
	// rules is repaired progressively as records are saved.  The template schema inherits
	// the full Stream schema as its baseline, so this covers every Stream property while
	// honoring the template's format overrides.
	rewrites, err := template.Schema.Normalize(draft)

	if err != nil {
		return derp.Wrap(err, location, "Invalid StreamDraft: using TemplateSchema", draft)
	}

	if len(rewrites) > 0 {
		log.Debug().Strs("rewrites", rewrites).Str("streamId", draft.StreamID.Hex()).Msg("StreamDraft values normalized during save")
	}

	if err := service.collection(session).Save(draft, note); err != nil {
		return derp.Wrap(err, location, "Saving draft", draft, note)
	}

	return nil
}

// Delete removes an StreamDraft from the database (hard delete)
func (service *StreamDraft) Delete(session data.Session, draft *model.Stream, _note string) error {

	criteria := exp.Equal("_id", draft.StreamID)

	// Use a hard delete to remove drafts permanently.
	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, "service.StreamDraft.Delete", "Deleting draft", criteria)
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

// ObjectID returns the unique ID of the provided StreamDraft. Implements the ModelService interface.
func (service *StreamDraft) ObjectID(object data.Object) primitive.ObjectID {

	if streamDraft, ok := object.(*model.Stream); ok {
		return streamDraft.StreamID
	}

	return primitive.NilObjectID
}

// ObjectQuery returns every StreamDraft that matches the provided criteria. Implements the ModelService interface.
func (service *StreamDraft) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

// ObjectLoad retrieves a single StreamDraft as a data.Object. Implements the ModelService interface.
func (service *StreamDraft) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewStream()
	err := service.Load(session, criteria, &result)
	return &result, err
}

// ObjectSave adds or updates a StreamDraft in the database. Implements the ModelService interface.
func (service *StreamDraft) ObjectSave(session data.Session, object data.Object, comment string) error {
	if stream, ok := object.(*model.Stream); ok {
		return service.Save(session, stream, comment)
	}
	return derp.Internal("service.StreamDraft.ObjectSave", "Invalid Object Type", object)
}

// ObjectDelete marks a StreamDraft as deleted. Implements the ModelService interface.
func (service *StreamDraft) ObjectDelete(session data.Session, object data.Object, comment string) error {
	if stream, ok := object.(*model.Stream); ok {
		return service.Delete(session, stream, comment)
	}
	return derp.Internal("service.StreamDraft.ObjectDelete", "Invalid Object Type", object)
}

// ObjectUserCan reports whether the provided Authorization may run an action on a StreamDraft. Implements the ModelService interface.
func (service *StreamDraft) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.Unauthorized("service.StreamDraft", "Not Authorized")
}

// Schema returns the rosetta schema that describes a StreamDraft
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

// Promote publishes a StreamDraft over its live Stream, moving it into the provided state
func (service *StreamDraft) Promote(session data.Session, streamID primitive.ObjectID, stateID string) (model.Stream, error) {

	var draft model.Stream
	var stream model.Stream

	// Try to load the draft
	if err := service.LoadByID(session, streamID, &draft); err != nil {
		return model.Stream{}, derp.Wrap(err, "service.StreamDraft.Publish", "Loading draft")
	}

	// Try to load the production stream
	if err := service.streamService.LoadByID(session, streamID, &stream); err != nil {
		return model.Stream{}, derp.Wrap(err, "service.StreamDraft.Publish", "Loading draft")
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
		return model.Stream{}, derp.Wrap(err, "service.StreamDraft.Publish", "Publishing stream")
	}

	// Try to save the updated stream back to the database
	if err := service.Delete(session, &draft, "published"); err != nil {
		return model.Stream{}, derp.Wrap(err, "service.StreamDraft.Publish", "Deleting draft")
	}

	return stream, nil
}
