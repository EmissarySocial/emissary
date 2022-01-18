package service

import (
	"github.com/benpate/data"
	"github.com/benpate/data/journal"
	"github.com/benpate/data/option"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/nebula"
	"github.com/whisperverse/whisperverse/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StreamDraft manages all interactions with the StreamDraft collection
type StreamDraft struct {
	collection     data.Collection
	streamService  *Stream
	contentLibrary *nebula.Library
}

// NewStreamDraft returns a fully populated StreamDraft service.
func NewStreamDraft(collection data.Collection, streamService *Stream, contentLibrary *nebula.Library) StreamDraft {
	return StreamDraft{
		collection:     collection,
		streamService:  streamService,
		contentLibrary: contentLibrary,
	}
}

/*******************************************
 * COMMON DATA FUNCTIONS
 *******************************************/

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
		return derp.Wrap(err, "whisper.service.StreamDraft.Load", "Error loading original stream")
	}

	// Reset the journal so that this item can be saved in the new collection.
	result.Journal = journal.Journal{}

	// Add default content if the content is empty.
	if result.Content.Len() == 0 {
		result.Content = nebula.NewContainer()
		result.Content.NewItemWithInit(service.contentLibrary, nebula.ItemTypeLayout, nil)
	}

	// Save a draft copy of the original stream
	if err := service.Save(result, "create draft record"); err != nil {
		return derp.Wrap(err, "whisper.service.StreamDraft.Load", "Error saving draft")
	}

	// Return the original stream as a new draft to use.
	return nil
}

// save adds/updates an StreamDraft in the database
func (service *StreamDraft) Save(draft *model.Stream, note string) error {

	if err := service.collection.Save(draft, note); err != nil {
		return derp.Wrap(err, "whisper.service.StreamDraft.Save", "Error saving draft", draft, note)
	}

	return nil
}

// Delete removes an StreamDraft from the database (virtual delete)
func (service *StreamDraft) Delete(draft *model.Stream, _note string) error {

	// Use a hard delete to remove drafts permanently.
	if err := service.collection.HardDelete(draft); err != nil {
		return derp.Wrap(err, "whisper.service.StreamDraft.Delete", "Error deleting draft", draft)
	}

	return nil
}

/*******************************************
 * GENERIC DATA FUNCTIONS
 *******************************************/

// New returns a fully initialized model.Stream as a data.Object.
func (service *StreamDraft) ObjectNew() data.Object {
	result := model.NewStream()
	return &result
}

func (service *StreamDraft) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return nil, derp.NewInternalError("whisper.service.StreamDraft.ObjectList", "Unsupported")
}

func (service *StreamDraft) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewStream()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *StreamDraft) ObjectSave(object data.Object, comment string) error {
	return service.Save(object.(*model.Stream), comment)
}

func (service *StreamDraft) ObjectDelete(object data.Object, comment string) error {
	return service.Delete(object.(*model.Stream), comment)
}

func (service *StreamDraft) Debug() datatype.Map {
	return datatype.Map{
		"service": "StreamDraft",
	}
}

/*******************************************
 * CUSTOM QUERIES
 *******************************************/

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

/*******************************************
 * CUSTOM ACTIONS
 *******************************************/

func (service *StreamDraft) Publish(streamID primitive.ObjectID, stateID string) error {

	var draft model.Stream
	var stream model.Stream

	// Try to load the draft
	if err := service.LoadByID(streamID, &draft); err != nil {
		return derp.Wrap(err, "whisper.service.StreamDraft.Publish", "Error loading draft")
	}

	// Try to load the production stream
	if err := service.streamService.LoadByID(streamID, &stream); err != nil {
		return derp.Wrap(err, "whisper.service.StreamDraft.Publish", "Error loading draft")
	}

	// Copy data from draft to production
	stream.Label = draft.Label
	stream.Description = draft.Description
	stream.Content = draft.Content
	stream.Data = draft.Data
	stream.StateID = stateID
	stream.Tags = draft.Tags
	stream.ThumbnailImage = draft.ThumbnailImage
	stream.Token = draft.Token
	stream.Journal.DeleteDate = 0 // just in case...

	// Try to save the updated stream back to the database
	if err := service.streamService.Save(&stream, "published"); err != nil {
		return derp.Wrap(err, "whisper.service.StreamDraft.Publish", "Error publishing stream")
	}

	// Try to save the updated stream back to the database
	if err := service.Delete(&draft, "published"); err != nil {
		return derp.Wrap(err, "whisper.service.StreamDraft.Publish", "Error deleting draft")
	}

	return nil
}
