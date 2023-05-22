package service

import (
	"net/url"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/domain"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StreamResponse defines a service that can send and receive streamResponse data
type StreamResponse struct {
	collection       data.Collection
	streamCollection data.Collection
	blockService     *Block
	host             string
}

// NewStreamResponse returns a fully initialized StreamResponse service
func NewStreamResponse() StreamResponse {
	return StreamResponse{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *StreamResponse) Refresh(collection data.Collection, streamCollection data.Collection, blockService *Block, host string) {
	service.collection = collection
	service.streamCollection = streamCollection
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

	// Save the value to the database
	if err := service.collection.Save(streamResponse, note); err != nil {
		return derp.Wrap(err, location, "Error saving StreamResponse", streamResponse, note)
	}

	// Recalculate statistics for the Stream affected by this StreamResponse.
	if err := queries.CountResponses(service.streamCollection, service.collection, "stream.id", streamResponse.Stream.ID); err != nil {
		return derp.Wrap(err, location, "Error counting responses")
	}

	// TODO: CRITICAL: Send messages to all followers about the new response.

	return nil
}

// Delete removes an StreamResponse from the database (virtual delete)
func (service *StreamResponse) Delete(streamResponse *model.StreamResponse, note string) error {

	const location = "service.StreamResponse.Delete"

	criteria := exp.Equal("_id", streamResponse.StreamResponseID)

	// Delete this StreamResponse
	if err := service.collection.HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Error deleting StreamResponse", criteria)
	}

	// Recalculate statistics for the Stream affected by this StreamResponse.
	if err := queries.CountResponses(service.streamCollection, service.collection, "stream.id", streamResponse.Stream.ID); err != nil {
		return derp.Wrap(err, location, "Error counting responses")
	}

	// TODO: CRITICAL: Send messages to all followers about the new response.

	return nil
}

func (service *StreamResponse) Schema() schema.Schema {
	return schema.New(model.StreamResponseSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *StreamResponse) ListByStreamAndType(streamID primitive.ObjectID, responseType string, options ...option.Option) (data.Iterator, error) {

	criteria := exp.Equal("stream.id", streamID).
		AndEqual("type", responseType)

	return service.List(criteria, options...)
}

func (service *StreamResponse) QueryByStreamAndType(streamID primitive.ObjectID, responseType string) ([]model.StreamResponse, error) {
	return service.Query(
		exp.Equal("stream.id", streamID).
			AndEqual("type", responseType),
	)
}

func (service *StreamResponse) LoadByStreamAndOrigin(streamID primitive.ObjectID, originURL string, streamResponse *model.StreamResponse) error {
	criteria := exp.Equal("stream.id", streamID).AndEqual("origin.url", originURL)

	return service.Load(criteria, streamResponse)
}

func (service *StreamResponse) LoadByStreamAndActor(streamID primitive.ObjectID, actorURL string, streamResponse *model.StreamResponse) error {

	criteria := exp.Equal("stream.id", streamID).
		AndEqual("actor.profileUrl", actorURL)

	return service.Load(criteria, streamResponse)
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
	err := service.LoadByStreamAndActor(stream.StreamID, actor.ProfileURL, &oldStreamResponse)

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
	newStreamResponse.Stream = stream.DocumentLink()
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

// ParseURL parses a StreamResponseURL of the form "https://server.name/streamID/responses/streamResponseID" into a StreamID and a StreamResponseID.
func (service *StreamResponse) ParseURL(value string) (primitive.ObjectID, primitive.ObjectID, error) {

	// Parse the URL
	parsed, err := url.Parse(value)

	if err != nil {
		return primitive.NilObjectID, primitive.NilObjectID, derp.Wrap(err, "service.StreamResponse.ParseURL", "Error parsing URL", value)
	}

	// Validate the domain is correct
	if parsed.Host != domain.NameOnly(service.host) {
		return primitive.NilObjectID, primitive.NilObjectID, derp.New(derp.CodeBadRequestError, "service.StreamResponse.ParseURL", "Invalid domain", value)
	}

	// Parse the path as a list
	path := list.BySlash(strings.TrimPrefix(parsed.Path, "/"))

	// Get the StreamID from the path
	streamIDString, path := path.Split()

	streamID, err := primitive.ObjectIDFromHex(streamIDString)

	if err != nil {
		return primitive.NilObjectID, primitive.NilObjectID, derp.Wrap(err, "service.StreamResponse.ParseURL", "Error parsing streamID", value)
	}

	// Get the "subType from the path (must be "responses")
	subType, path := path.Split()

	if subType != "responses" {
		return primitive.NilObjectID, primitive.NilObjectID, derp.New(derp.CodeBadRequestError, "service.StreamResponse.ParseURL", "Invalid subType", value)
	}

	// Get the StreamResponseID from the path
	streamResponseIDString := path.Head()

	streamResponseID, err := primitive.ObjectIDFromHex(streamResponseIDString)

	if err != nil {
		return primitive.NilObjectID, primitive.NilObjectID, derp.Wrap(err, "service.StreamResponse.ParseURL", "Error parsing streamResponseID", value)
	}

	// Voila!
	return streamID, streamResponseID, nil
}
