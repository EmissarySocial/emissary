package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StreamResponse defines a service that can send and receive streamResponse data
type StreamResponse struct {
	collection   data.Collection
	blockService *Block
	host         string
}

// NewStreamResponse returns a fully initialized StreamResponse service
func NewStreamResponse() StreamResponse {
	return StreamResponse{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *StreamResponse) Refresh(collection data.Collection, blockService *Block, host string) {
	service.collection = collection
	service.blockService = blockService
	service.host = host
}

// Close stops any background processes controlled by this service
func (service *StreamResponse) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

// Query returns a slice containing all of the StreamResponses that match the provided criteria
func (service *StreamResponse) Query(criteria exp.Expression, options ...option.Option) ([]model.StreamResponse, error) {
	result := make([]model.StreamResponse, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)
	return result, err
}

// List returns an iterator containing all of the StreamResponses that match the provided criteria
func (service *StreamResponse) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(notDeleted(criteria), options...)
}

// Load retrieves an StreamResponse from the database
func (service *StreamResponse) Load(criteria exp.Expression, streamResponse *model.StreamResponse) error {

	if err := service.collection.Load(notDeleted(criteria), streamResponse); err != nil {
		return derp.Wrap(err, "service.StreamResponse.Load", "Error loading StreamResponse", criteria)
	}

	return nil
}

// Save adds/updates an StreamResponse in the database
func (service *StreamResponse) Save(streamResponse *model.StreamResponse, note string) error {

	const location = "service.StreamResponse.Save"

	// Clean the value before saving
	if err := service.Schema().Clean(streamResponse); err != nil {
		return derp.Wrap(err, location, "Error cleaning StreamResponse", streamResponse)
	}

	// TODO: Recalculate statistics for the Message affected by this StreamResponse.

	// Save the value to the database
	if err := service.collection.Save(streamResponse, note); err != nil {
		return derp.Wrap(err, location, "Error saving StreamResponse", streamResponse, note)
	}

	return nil
}

// Delete removes an StreamResponse from the database (virtual delete)
func (service *StreamResponse) Delete(streamResponse *model.StreamResponse, note string) error {

	criteria := exp.Equal("_id", streamResponse.StreamResponseID)

	// TODO: Recalculate statistics

	// Delete this StreamResponse
	if err := service.collection.HardDelete(criteria); err != nil {
		return derp.Wrap(err, "service.StreamResponse.Delete", "Error deleting StreamResponse", criteria)
	}

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************

// ObjectType returns the type of object that this service manages
func (service *StreamResponse) ObjectType() string {
	return "StreamResponse"
}

// New returns a fully initialized model.Group as a data.Object.
func (service *StreamResponse) ObjectNew() data.Object {
	result := model.NewStreamResponse()
	return &result
}

func (service *StreamResponse) ObjectID(object data.Object) primitive.ObjectID {

	if streamResponse, ok := object.(*model.StreamResponse); ok {
		return streamResponse.StreamResponseID
	}

	return primitive.NilObjectID
}

func (service *StreamResponse) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *StreamResponse) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

func (service *StreamResponse) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewStreamResponse()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *StreamResponse) ObjectSave(object data.Object, comment string) error {
	if streamResponse, ok := object.(*model.StreamResponse); ok {
		return service.Save(streamResponse, comment)
	}
	return derp.NewInternalError("service.StreamResponse.ObjectSave", "Invalid Object Type", object)
}

func (service *StreamResponse) ObjectDelete(object data.Object, comment string) error {
	if streamResponse, ok := object.(*model.StreamResponse); ok {
		return service.Delete(streamResponse, comment)
	}
	return derp.NewInternalError("service.StreamResponse.ObjectDelete", "Invalid Object Type", object)
}

func (service *StreamResponse) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.NewUnauthorizedError("service.StreamResponse", "Not Authorized")
}

*/

func (service *StreamResponse) Schema() schema.Schema {
	return schema.New(model.StreamResponseSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *StreamResponse) LoadByID(userID primitive.ObjectID, streamResponseID primitive.ObjectID, streamResponse *model.StreamResponse) error {

	criteria := exp.Equal("_id", streamResponseID).
		AndEqual("actor.internalId", userID)

	if err := service.Load(criteria, streamResponse); err != nil {
		return derp.Wrap(err, "service.StreamResponse.LoadByID", "Error loading StreamResponse", streamResponseID)
	}

	return nil
}

func (service *StreamResponse) LoadByObjectID(userID primitive.ObjectID, objectID primitive.ObjectID, streamResponse *model.StreamResponse) error {

	criteria := exp.Equal("objectId", objectID).
		AndEqual("actor.internalId", userID)

	if err := service.Load(criteria, streamResponse); err != nil {
		return derp.Wrap(err, "service.StreamResponse.LoadByID", "Error loading StreamResponse", userID, objectID)
	}

	return nil
}

func (service *StreamResponse) QueryByObjectID(objectID primitive.ObjectID) ([]model.StreamResponse, error) {
	criteria := exp.Equal("objectId", objectID)
	return service.Query(criteria)
}

/******************************************
 * Custom Behaviors
 ******************************************/

// SetStreamResponse is the preferred way of creating/updating a StreamResponse.  It includes the business
// logic to search for an existing streamResponse, and delete it if one exists already (publishing UNDO actions in the process).
func (service *StreamResponse) SetStreamResponse(stream *model.Stream, origin model.OriginLink, actor model.PersonLink, streamResponseType string, value string) error {

	const location = "service.StreamResponse.SetStreamResponse"

	// RULE: Filter StreamResponses that are blocked
	if err := service.blockService.FilterStreamResponse(stream, origin.URL, actor.ProfileURL); err != nil {
		return derp.Wrap(err, location, "Error filtering StreamResponse", stream, origin, actor)
	}

	// If a streamResponse already exists, then delete it first.
	oldStreamResponse := model.NewStreamResponse()
	err := service.LoadByObjectID(actor.UserID, stream.StreamID, &oldStreamResponse)

	// RULE: if the streamResponse exists....
	if err == nil {

		// If there was no change, then there's nothing to do.
		if (oldStreamResponse.Type == streamResponseType) && (oldStreamResponse.Value == value) {
			return nil
		}

		// Otherwise, delete the old streamResponse (which triggers other logic)
		if err := service.Delete(&oldStreamResponse, "Updated by User"); err != nil {
			return derp.Wrap(err, location, "Error deleting old streamResponse", oldStreamResponse)
		}
	}

	// Create a new streamResponse
	newStreamResponse := model.NewStreamResponse()
	newStreamResponse.StreamID = stream.StreamID
	newStreamResponse.Actor = actor
	newStreamResponse.Origin = origin
	newStreamResponse.Type = streamResponseType
	newStreamResponse.Value = value

	// Save the StreamResponse to the database (streamResponse service will automatically publish to ActivityPub and beyond)
	if err := service.Save(&newStreamResponse, "Updated by User"); err != nil {
		return derp.Wrap(err, location, "Error saving streamResponse", newStreamResponse)
	}

	return nil
}
