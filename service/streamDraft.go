package service

import (
	"github.com/benpate/data"
	"github.com/benpate/data/journal"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/ghost/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StreamDraft manages all interactions with the StreamDraft collection
type StreamDraft struct {
	collection    data.Collection
	streamService *Stream
}

// NewStreamDraft returns a fully populated StreamDraft service.
func NewStreamDraft(collection data.Collection, streamService *Stream) *StreamDraft {
	return &StreamDraft{
		collection:    collection,
		streamService: streamService,
	}
}

// New creates a newly initialized StreamDraft that is ready to use
func (service *StreamDraft) New() *model.Stream {
	result := model.NewStream()
	return &result
}

// Load either: 1) loads a valid draft from the database, or 2) creates a new draft and returns it instead
func (service *StreamDraft) Load(criteria exp.Expression) (*model.Stream, error) {

	draft := model.NewStream()

	// Try to load a draft using the provided criteria
	if err := service.collection.Load(criteria, &draft); err == nil {
		return &draft, nil
	}

	// Fall through means we could not load a draft (probably 404 not found)

	// Try to locate the original stream
	stream, err := service.streamService.Load(criteria)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.service.StreamDraft.Load", "Error loading original stream")
	}

	// Reset the journal so that this item can be saved in the new collection.
	stream.Journal = journal.Journal{}

	// Save a draft copy of the original stream
	if err := service.Save(stream, "create draft record"); err != nil {
		return nil, derp.Wrap(err, "ghost.service.StreamDraft.Load", "Error saving draft")
	}

	// Return the original stream as a new draft to use.
	return stream, nil
}

// save adds/updates an StreamDraft in the database
func (service *StreamDraft) Save(draft *model.Stream, note string) error {

	if err := service.collection.Save(draft, note); err != nil {
		return derp.Wrap(err, "ghost.service.StreamDraft.Save", "Error saving draft", draft, note)
	}

	return nil
}

// Delete removes an StreamDraft from the database (virtual delete)
func (service *StreamDraft) Delete(draft *model.Stream, _note string) error {

	// Use a hard delete to remove drafts permanently.
	if err := service.collection.HardDelete(draft); err != nil {
		return derp.Wrap(err, "ghost.service.StreamDraft.Delete", "Error deleting draft", draft)
	}

	return nil
}

// QUERIES ////////////////////////////////////

// LoadByToken returns a single Stream that matches a particular Token
func (service *StreamDraft) LoadByToken(token string) (*model.Stream, error) {
	return service.Load(
		exp.Equal("token", token).
			AndEqual("journal.deleteDate", 0))
}

// LoadByID returns a single Stream that matches a particular StreamID
func (service *StreamDraft) LoadByID(streamID primitive.ObjectID) (*model.Stream, error) {
	return service.Load(
		exp.Equal("_id", streamID).
			AndEqual("journal.deleteDate", 0))
}
