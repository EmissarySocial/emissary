package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Response defines a service that can send and receive response data
type Response struct {
	collection    data.Collection
	userService   *User
	outboxService *Outbox
	host          string
}

// NewResponse returns a fully initialized Response service
func NewResponse() Response {
	return Response{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Response) Refresh(collection data.Collection, userService *User, outboxService *Outbox, host string) {
	service.collection = collection
	service.userService = userService
	service.outboxService = outboxService
	service.host = host
}

// Close stops any background processes controlled by this service
func (service *Response) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

// Query returns a slice containing all of the Responses that match the provided criteria
func (service *Response) Query(criteria exp.Expression, options ...option.Option) ([]model.Response, error) {
	result := make([]model.Response, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)
	return result, err
}

// List returns an iterator containing all of the Responses that match the provided criteria
func (service *Response) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Load retrieves an Response from the database
func (service *Response) Load(criteria exp.Expression, response *model.Response) error {

	if err := service.collection.Load(notDeleted(criteria), response); err != nil {
		return derp.Wrap(err, "service.Response.Load", "Error loading Response", criteria)
	}

	return nil
}

// Save adds/updates an Response in the database
func (service *Response) Save(response *model.Response, note string) error {

	const location = "service.Response.Save"

	// Validate the value before saving
	if err := service.Schema().Validate(response); err != nil {
		return derp.Wrap(err, location, "Error validating Response", response)
	}

	// Save the value to the database
	if err := service.collection.Save(response, note); err != nil {
		return derp.Wrap(err, location, "Error saving Response", response, note)
	}

	return nil
}

// Delete removes an Response from the database (hard delete)
func (service *Response) Delete(response *model.Response, note string) error {

	const location = "service.Response.Delete"

	// Delete this Response
	if err := service.collection.HardDelete(exp.Equal("_id", response.ResponseID)); err != nil {
		return derp.Wrap(err, location, "Error deleting Response", response)
	}

	return nil
}

/******************************************
 * Generic Data Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Response) ObjectType() string {
	return "Response"
}

// New returns a fully initialized model.Response as a data.Object.
func (service *Response) ObjectNew() data.Object {
	result := model.NewResponse()
	return &result
}

func (service *Response) ObjectID(object data.Object) primitive.ObjectID {

	if response, ok := object.(*model.Response); ok {
		return response.ResponseID
	}

	return primitive.NilObjectID
}

func (service *Response) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *Response) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

func (service *Response) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewResponse()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Response) ObjectSave(object data.Object, note string) error {

	if response, ok := object.(*model.Response); ok {
		return service.Save(response, note)
	}
	return derp.NewInternalError("service.Response.ObjectSave", "Invalid object type", object)
}

func (service *Response) ObjectDelete(object data.Object, note string) error {
	if response, ok := object.(*model.Response); ok {
		return service.Delete(response, note)
	}
	return derp.NewInternalError("service.Response.ObjectDelete", "Invalid object type", object)
}

func (service *Response) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.NewUnauthorizedError("service.Response", "Not Authorized")
}

func (service *Response) Schema() schema.Schema {
	return schema.New(model.ResponseSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *Response) QueryByUserAndDate(userID primitive.ObjectID, responseType string, maxDate int64, pageSize int) ([]model.Response, error) {

	criteria := exp.Equal("userId", userID).AndEqual("type", responseType).And(exp.LessThan("createDate", maxDate))
	options := []option.Option{option.SortDesc("createDate"), option.MaxRows(int64(pageSize))}

	return service.Query(criteria, options...)
}

func (service *Response) QueryByObjectAndDate(objectID string, responseType string, maxDate int64, pageSize int) ([]model.Response, error) {

	criteria := exp.Equal("objectId", objectID).AndEqual("type", responseType).And(exp.LessThan("createDate", maxDate))
	options := []option.Option{option.SortDesc("createDate"), option.MaxRows(int64(pageSize))}

	return service.Query(criteria, options...)
}

func (service *Response) LoadByID(responseID primitive.ObjectID, response *model.Response) error {
	return service.Load(exp.Equal("_id", responseID), response)
}

func (service *Response) QueryByUserAndObject(userID primitive.ObjectID, object string, options ...option.Option) ([]model.Response, error) {

	criteria := exp.Equal("userId", userID).
		AndEqual("object", object)

	return service.Query(criteria, options...)
}

func (service *Response) LoadByUserAndObject(userID primitive.ObjectID, object string, responseType string, response *model.Response) error {

	criteria := exp.Equal("userId", userID).
		AndEqual("object", object).
		AndEqual("type", responseType)

	return service.Load(criteria, response)
}

func (service *Response) LoadByActorAndObject(actor string, object string, responseType string, response *model.Response) error {

	criteria := exp.Equal("actor", actor).
		AndEqual("object", object).
		AndEqual("type", responseType)

	return service.Load(criteria, response)
}

func (service *Response) CountByContent(objectID string) (mapof.Int, error) {
	return queries.CountResponsesByContent(service.collection, objectID)
}

/******************************************
 * Custom Behaviors
 ******************************************/

// SetResponse is the preferred way of creating/updating a Response.  It includes the business
// logic to search for an existing response, and delete it if one exists already (publishing UNDO actions in the process).
func (service *Response) SetResponse(user *model.User, url string, responseType string, content string) error {

	const location = "service.Response.SetResponse"

	// Remove pre-existing response of this same type (if exists)
	if err := service.UnsetResponse(user, url, responseType); err != nil {
		return derp.Wrap(err, location, "Error removing previous response", user.UserID, url, responseType)
	}

	// Create a new Response object
	response := model.NewResponse()
	response.UserID = user.UserID
	response.Actor = user.ActivityPubURL()
	response.Object = url
	response.Type = responseType
	response.Content = content

	// Save the Response to the database (response service will automatically publish to ActivityPub and beyond)
	if err := service.Save(&response, "Set Response"); err != nil {
		return derp.Wrap(err, location, "Error saving response", response)
	}

	// Get an ActivityPub actor for the User
	actor, err := service.userService.ActivityPubActor(user.UserID, true)

	if err != nil {
		return derp.Wrap(err, location, "Error loading ActivityPub Actor", user.UserID)
	}

	// Publish the new Response to the Outbox, sending "Like" notifications to all followers.
	if err := service.outboxService.Publish(&actor, model.FollowerTypeUser, user.UserID, response.GetJSONLD()); err != nil {
		derp.Report(derp.Wrap(err, location, "Error publishing Response", response))
	}

	// Oye cómo va!
	return nil
}

// UnsetReponse removes a reponse based on the User, URL, and Response Type
func (service *Response) UnsetResponse(user *model.User, url string, responseType string) error {

	const location = "service.Response.UnsetResponse"

	// Search for a previous Response from this User
	oldResponse := model.NewResponse()

	if err := service.LoadByUserAndObject(user.UserID, url, responseType, &oldResponse); err != nil {

		// If there is no matching response, then there's nothing to delete
		if derp.NotFound(err) {
			return nil
		}

		return derp.Wrap(err, location, "Error loading original response", user.UserID, url, responseType)
	}

	// Otherwise, delete the old Response
	if err := service.Delete(&oldResponse, ""); err != nil {
		return derp.Wrap(err, location, "Error deleting old response", oldResponse)
	}

	// Get an ActivityPub actor for the User
	actor, err := service.userService.ActivityPubActor(user.UserID, true)

	if err != nil {
		return derp.Wrap(err, location, "Error loading ActivityPub Actor", user.UserID)
	}

	// Unpublish from the Outbox, and send the "Undo" activity to followers
	if err := service.outboxService.UnPublish(&actor, model.FollowerTypeUser, user.UserID, oldResponse.ActivityPubURL()); err != nil {
		derp.Report(derp.Wrap(err, location, "Error publishing Response", oldResponse))
	}

	// Success!!
	return nil
}
