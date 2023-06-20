package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/pub"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/davecgh/go-spew/spew"
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
	return service.collection.List(notDeleted(criteria), options...)
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

	// Validate/Clean the value before saving
	if err := service.Schema().Clean(response); err != nil {
		return derp.Wrap(err, location, "Error cleaning Response", response)
	}

	// Populate the URL of this response
	if !response.UserID.IsZero() {
		response.URL = service.host + "/@" + response.UserID.Hex() + "/pub/liked/" + response.ResponseID.Hex()
	}

	// Save the value to the database
	if err := service.collection.Save(response, note); err != nil {
		return derp.Wrap(err, location, "Error saving Response", response, note)
	}

	return nil
}

// Delete removes an Response from the database (virtual delete)
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

func (service *Response) LoadByUserAndObject(userID primitive.ObjectID, objectID string, response *model.Response) error {
	return service.Load(exp.Equal("userId", userID).AndEqual("objectId", objectID), response)
}

func (service *Response) LoadByActorAndObject(actorID string, objectID string, response *model.Response) error {
	return service.Load(exp.Equal("actorId", actorID).AndEqual("objectId", objectID), response)
}

func (service *Response) CountByContent(objectID string) (mapof.Int, error) {
	return queries.CountResponsesByContent(service.collection, objectID)
}

/******************************************
 * Custom Behaviors
 ******************************************/

// SetResponse is the preferred way of creating/updating a Response.  It includes the business
// logic to search for an existing response, and delete it if one exists already (publishing UNDO actions in the process).
func (service *Response) SetResponse(response *model.Response) error {

	const location = "service.Response.SetResponse"

	// Validate the response
	response.CalcContent()
	spew.Dump(response)

	user := model.NewUser()
	err := service.userService.LoadByProfileURL(response.ActorID, &user)

	// If this is a legitimate error, then abort.
	if (err != nil) && !derp.NotFound(err) {
		return derp.Wrap(err, location, "Error loading user", response.ActorID)
	}

	// NOTE if the actorID is not found then the User will be a blank object

	// If a response already exists, then delete it first.
	oldResponse := model.NewResponse()
	err = service.LoadByActorAndObject(response.ActorID, response.ObjectID, &oldResponse)

	// RULE: if the response exists....
	if err == nil {

		// If there was no change, then there's nothing to do.
		if response.IsEqual(oldResponse) {
			spew.Dump("EQUAL TO OLD RESPONSE??")
			return nil
		}

		// Otherwise, delete the old response (which triggers other logic)
		if err := service.Delete(&oldResponse, ""); err != nil {
			return derp.Wrap(err, location, "Error deleting old response", oldResponse)
		}

		// Responses from local Actors should be removed from the Outbox
		if !user.IsNew() {

			// Create an "Undo" activity
			undoActivity := pub.Undo(response.GetJSONLD())

			// Send the "Undo" activity to followers
			if err := service.outboxService.UnPublish(user.UserID, response.URL, undoActivity); err != nil {
				derp.Report(derp.Wrap(err, location, "Error publishing Response", response))
			}
		}

		// RULE: If there is no response type, then this is a DELETE-ONLY operation. Do not create a new response.
		if response.Type == "" {
			return nil
		}

	} else if !derp.NotFound(err) {
		spew.Dump("err.. grr", err, derp.NotFound(err))
		return derp.Wrap(err, location, "Error loading original response", oldResponse)
	}

	spew.Dump("here??")

	// Save the Response to the database (response service will automatically publish to ActivityPub and beyond)
	if err := service.Save(response, ""); err != nil {
		spew.Dump("onefish?")
		return derp.Wrap(err, location, "Error saving response", response)
	}

	spew.Dump("twofish??")

	// Responses from local Actors should be published to the Outbox
	if !user.IsNew() {

		if err := service.outboxService.Publish(user.UserID, response.URL, response.GetJSONLD()); err != nil {
			derp.Report(derp.Wrap(err, location, "Error publishing Response", response))
		}
	}

	spew.Dump("redfish, bluefish??")

	// Oye cómo va!
	return nil
}
