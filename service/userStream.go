package service

import (
	"context"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserStream manages all interactions with UserStream data.
type UserStream struct {
	collection data.Collection
	ctx        context.Context
}

// NewUserStream returns a fully initialized UserStream service
func NewUserStream(collection data.Collection, ctx context.Context) UserStream {
	return UserStream{
		collection: collection,
		ctx:        ctx,
	}
}

/******************************************
 * Common Data Methods
 ******************************************/

// List returns an iterator containing all of the UserStreams who match the provided criteria
func (service *UserStream) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(notDeleted(criteria), options...)
}

// Load retrieves an UserStream from the database
func (service *UserStream) Load(criteria exp.Expression, stream *model.UserStream) error {

	if err := service.collection.Load(notDeleted(criteria), stream); err != nil {
		return derp.Wrap(err, "service.UserStream", "Error loading UserStream", criteria)
	}

	return nil
}

// Save adds/updates an UserStream in the database
func (service *UserStream) Save(stream *model.UserStream, note string) error {

	// TODO: HIGH: Use schema to clean the model object before saving

	if err := service.collection.Save(stream, note); err != nil {
		return derp.Wrap(err, "service.UserStream", "Error saving UserStream", stream, note)
	}

	return nil
}

// Delete removes an UserStream from the database (virtual delete)
func (service *UserStream) Delete(stream *model.UserStream, note string) error {

	// Delete this UserStream
	if err := service.collection.Delete(stream, note); err != nil {
		return derp.Wrap(err, "service.UserStream.Delete", "Error deleting UserStream", stream, note)
	}

	return nil
}

/*******************************
 * Custom Queries
 *******************************/

func (service UserStream) LoadByUserAndStream(userID primitive.ObjectID, streamID primitive.ObjectID) (model.UserStream, error) {

	result := model.NewUserStream()
	err := service.Load(exp.Equal("userId", userID).AndEqual("streamId", streamID), &result)

	// Record found.  Success!
	if err == nil {
		return result, nil
	}

	// NotFound is OK, just new.  So let's populate the requied fields and be done.
	if derp.NotFound(err) {
		result.UserID = userID
		result.StreamID = streamID
		return result, nil
	}

	return result, derp.Wrap(err, "service.UserStream.LoadByUserAdnStream", "Error loading UserStream", userID, streamID)
}

/*******************************
 * CUSTOM MONGO-SPECIFIC QUERIES
 *******************************/

// VoteRunes returns the default runes to be used for "votes"
func (service UserStream) VoteRunes() []string {
	return []string{"😀", "🙁", "👍", "👎", "❤️", "👆", "🎉"}
}

// VoteCount returns the totals for all votes for the designated stream
func (service UserStream) VoteCount(streamID primitive.ObjectID) ([]queries.VoteCountResult, error) {
	return queries.VoteCount(service.ctx, service.collection, streamID)
}

// VoteDetails returns a list of users and their vote for the designated stream
func (service UserStream) VoteDetail(streamID primitive.ObjectID) ([]queries.VoteDetailResult, error) {
	return queries.VoteDetail(service.ctx, service.collection, streamID)
}
