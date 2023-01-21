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
	collection    data.Collection
	streamService *Stream
}

// NewStreamDraft returns a fully populated StreamDraft service.
func NewStreamDraft(collection data.Collection, streamService *Stream) StreamDraft {
	service := StreamDraft{
		streamService: streamService,
	}

	service.Refresh(collection)
	return service
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *StreamDraft) Refresh(collection data.Collection) {
	service.collection = collection
}

// Close stops any background processes controlled by this service
func (service *StreamDraft) Close() {

}

/******************************************
 * Common Data Methods
 ******************************************/

/******************************************
 * COMMON DATA FUNCTIONS
 ******************************************/

// New creates a newly initialized StreamDraft that is ready to use
func (service *StreamDraft) New() model.Stream {
	return model.NewStream()
}

// Load either: 1) loads a valid draft from the database, or 2) creates a new draft and returns it instead
func (service *StreamDraft) Load(criteria exp.Expression, result *model.Stream) error {

	// Try to load a draft using the provided criteria
	if err := service.collection.Load(criteria, result); err == nil {
		return nil
	}

	// Fall through means we could not load a draft (probably 404 not found)

	// Try to locate the original stream
	if err := service.streamService.Load(criteria, result); err != nil {
		return derp.Wrap(err, "service.StreamDraft.Load", "Error loading original stream")
	}

	// Reset the journal so that this item can be saved in the new collection.
	result.Journal = journal.Journal{}

	// Save a draft copy of the original stream
	if err := service.Save(result, "create draft record"); err != nil {
		return derp.Wrap(err, "service.StreamDraft.Load", "Error saving draft")
	}

	// Return the original stream as a new draft to use.
	return nil
}

// save adds/updates an StreamDraft in the database
func (service *StreamDraft) Save(draft *model.Stream, note string) error {

	// TODO: HIGH: Use schema to clean the model object before saving

	if err := service.collection.Save(draft, note); err != nil {
		return derp.Wrap(err, "service.StreamDraft.Save", "Error saving draft", draft, note)
	}

	return nil
}

// Delete removes an StreamDraft from the database (hard delete)
func (service *StreamDraft) Delete(draft *model.Stream, _note string) error {

	criteria := exp.Equal("_id", draft.StreamID)

	// Use a hard delete to remove drafts permanently.
	if err := service.collection.HardDelete(criteria); err != nil {
		return derp.Wrap(err, "service.StreamDraft.Delete", "Error deleting draft", criteria)
	}

	return nil
}

/******************************************
 * GENERIC DATA FUNCTIONS
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *StreamDraft) ObjectType() string {
	return "StreamDraft"
}

// New returns a fully initialized model.Stream as a data.Object.
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

func (service *StreamDraft) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *StreamDraft) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return nil, derp.NewInternalError("service.StreamDraft.ObjectList", "Unsupported")
}

func (service *StreamDraft) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewStream()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *StreamDraft) ObjectSave(object data.Object, comment string) error {
	if stream, ok := object.(*model.Stream); ok {
		return service.Save(stream, comment)
	}
	return derp.NewInternalError("service.StreamDraft.ObjectSave", "Invalid Object Type", object)
}

func (service *StreamDraft) ObjectDelete(object data.Object, comment string) error {
	if stream, ok := object.(*model.Stream); ok {
		return service.Delete(stream, comment)
	}
	return derp.NewInternalError("service.StreamDraft.ObjectDelete", "Invalid Object Type", object)
}

func (service *StreamDraft) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.NewUnauthorizedError("service.StreamDraft", "Not Authorized")
}

func (service *StreamDraft) Schema() schema.Schema {
	// TODO: HIGH: Implement this
	return schema.New(nil)
}

/******************************************
 * Custom Queries
 ******************************************/

// LoadByID returns a single Stream that matches a particular StreamID
func (service *StreamDraft) LoadByID(streamID primitive.ObjectID, result *model.Stream) error {

	criteria := exp.
		Equal("_id", streamID).
		AndEqual("journal.deleteDate", 0)

	return service.Load(criteria, result)
}

// LoadByToken returns a single Stream that matches a particular Token
func (service *StreamDraft) LoadByToken(token string, result *model.Stream) error {
	criteria := exp.
		Equal("token", token).
		AndEqual("journal.deleteDate", 0)

	return service.Load(criteria, result)
}

/******************************************
 * CUSTOM ACTIONS
 ******************************************/

func (service *StreamDraft) Promote(streamID primitive.ObjectID, stateID string) (model.Stream, error) {

	var draft model.Stream
	var stream model.Stream

	// Try to load the draft
	if err := service.LoadByID(streamID, &draft); err != nil {
		return model.Stream{}, derp.Wrap(err, "service.StreamDraft.Publish", "Error loading draft")
	}

	// Try to load the production stream
	if err := service.streamService.LoadByID(streamID, &stream); err != nil {
		return model.Stream{}, derp.Wrap(err, "service.StreamDraft.Publish", "Error loading draft")
	}

	// Copy data from draft to production
	stream.Document = draft.Document
	stream.Content = draft.Content
	stream.Data = draft.Data
	stream.StateID = stateID
	stream.Token = draft.Token
	stream.Journal.DeleteDate = 0 // just in case...

	// Try to save the updated stream back to the database
	if err := service.streamService.Save(&stream, "published"); err != nil {
		return model.Stream{}, derp.Wrap(err, "service.StreamDraft.Publish", "Error publishing stream")
	}

	// Try to save the updated stream back to the database
	if err := service.Delete(&draft, "published"); err != nil {
		return model.Stream{}, derp.Wrap(err, "service.StreamDraft.Publish", "Error deleting draft")
	}

	return stream, nil
}
