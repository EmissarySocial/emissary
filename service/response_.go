package service

import (
	"iter"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Response defines a service that can send and receive response data
type Response struct {
	importItemService *ImportItem
	inboxService      *Inbox
	outboxService     *Outbox
	userService       *User
	host              string
}

// NewResponse returns a fully initialized Response service
func NewResponse() Response {
	return Response{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Response) Refresh(importItemService *ImportItem, inboxService *Inbox, outboxService *Outbox, userService *User, host string) {
	service.importItemService = importItemService
	service.inboxService = inboxService
	service.outboxService = outboxService
	service.userService = userService
	service.host = host
}

// Close stops any background processes controlled by this service
func (service *Response) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *Response) collection(session data.Session) data.Collection {
	return session.Collection("Response")
}

// Count returns the number of Responses that match the provided criteria
func (service *Response) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns a slice containing all of the Responses that match the provided criteria
func (service *Response) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.Response, error) {
	result := make([]model.Response, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// List returns an iterator containing all of the Responses that match the provided criteria
func (service *Response) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Range returns an iterator containing all of the Users who match the provided criteria
func (service *Response) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.Response], error) {

	iter, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.User.Range", "Unable to create iterator", criteria)
	}

	return RangeFunc(iter, model.NewResponse), nil
}

// Load retrieves an Response from the database
func (service *Response) Load(session data.Session, criteria exp.Expression, response *model.Response) error {

	if err := service.collection(session).Load(notDeleted(criteria), response); err != nil {
		return derp.Wrap(err, "service.Response.Load", "Unable to load Response", criteria)
	}

	return nil
}

// Save adds/updates an Response in the database
func (service *Response) Save(session data.Session, response *model.Response, note string) error {

	const location = "service.Response.Save"

	// Validate the value before saving
	if err := service.Schema().Validate(response); err != nil {
		return derp.Wrap(err, location, "Unable to validate Response", response)
	}

	// Save the value to the database
	if err := service.collection(session).Save(response, note); err != nil {
		return derp.Wrap(err, location, "Unable to save Response", response, note)
	}

	// Try to update the inbox message being responded to
	if err := service.inboxService.setResponse(session, response.UserID, response.Object, response.Type, response.ResponseID); err != nil {
		return derp.Wrap(err, location, "Unable to set Response to inbox message", response.UserID)
	}

	return nil
}

// Delete removes an Response from the database (hard delete)
func (service *Response) Delete(session data.Session, response *model.Response, note string) error {

	const location = "service.Response.Delete"

	// Delete this Response
	if err := service.collection(session).HardDelete(exp.Equal("_id", response.ResponseID)); err != nil {
		return derp.Wrap(err, location, "Unable to delete Response", response)
	}

	// Try to update the inbox message being responded to
	if err := service.inboxService.setResponse(session, response.UserID, response.Object, response.Type, primitive.NilObjectID); err != nil {
		return derp.Wrap(err, location, "Unable to remove Response from inbox message", response.UserID)
	}

	// Unpublish from the Outbox, and send the "Undo" activity to followers
	if err := service.outboxService.UndoActivity(session, model.FollowerTypeUser, response.UserID, response.ActivityPubURL(), model.NewAnonymousPermissions()); err != nil {
		derp.Report(derp.Wrap(err, location, "Unable to send Undo activity"))
	}

	return nil
}

/******************************************
 * Special Case Methods
 ******************************************/

// QueryIDOnly returns a slice of IDOnly documents that match the provided criteria
func (service *Response) QueryIDOnly(session data.Session, criteria exp.Expression, options ...option.Option) (sliceof.Object[model.IDOnly], error) {
	result := make([]model.IDOnly, 0)
	options = append(options, option.Fields("_id"))
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// HardDeleteByID removes a specific Response record, without applying any additional business rules
func (service *Response) HardDeleteByID(session data.Session, userID primitive.ObjectID, responseID primitive.ObjectID) error {

	const location = "service.Response.HardDeleteByID"

	criteria := exp.Equal("userId", userID).AndEqual("_id", responseID)

	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Unable to delete Response", "userID: "+userID.Hex(), "responseID: "+responseID.Hex())
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

func (service *Response) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

func (service *Response) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewResponse()
	err := service.Load(session, criteria, &result)
	return &result, err
}

func (service *Response) ObjectSave(session data.Session, object data.Object, note string) error {

	if response, ok := object.(*model.Response); ok {
		return service.Save(session, response, note)
	}
	return derp.Internal("service.Response.ObjectSave", "Invalid object type", object)
}

func (service *Response) ObjectDelete(session data.Session, object data.Object, note string) error {
	if response, ok := object.(*model.Response); ok {
		return service.Delete(session, response, note)
	}
	return derp.Internal("service.Response.ObjectDelete", "Invalid object type", object)
}

func (service *Response) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.Unauthorized("service.Response", "Not Authorized")
}

func (service *Response) Schema() schema.Schema {
	return schema.New(model.ResponseSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *Response) QueryByUserAndDate(session data.Session, userID primitive.ObjectID, responseType string, maxDate int64, pageSize int) ([]model.Response, error) {

	criteria := exp.Equal("userId", userID).AndEqual("type", responseType).And(exp.LessThan("createDate", maxDate))
	options := []option.Option{option.SortDesc("createDate"), option.MaxRows(int64(pageSize))}

	return service.Query(session, criteria, options...)
}

func (service *Response) QueryByObjectAndDate(session data.Session, objectID string, responseType string, maxDate int64, pageSize int) ([]model.Response, error) {

	criteria := exp.Equal("objectId", objectID).AndEqual("type", responseType).And(exp.LessThan("createDate", maxDate))
	options := []option.Option{option.SortDesc("createDate"), option.MaxRows(int64(pageSize))}

	return service.Query(session, criteria, options...)
}

func (service *Response) LoadByID(session data.Session, userID primitive.ObjectID, responseID primitive.ObjectID, response *model.Response) error {
	criteria := exp.Equal("userId", userID).AndEqual("_id", responseID)
	return service.Load(session, criteria, response)
}

func (service *Response) RangeByUserID(session data.Session, userID primitive.ObjectID, options ...option.Option) (iter.Seq[model.Response], error) {

	criteria := exp.Equal("userId", userID)

	return service.Range(session, criteria, options...)
}

func (service *Response) QueryByUserAndObject(session data.Session, userID primitive.ObjectID, object string, options ...option.Option) ([]model.Response, error) {

	criteria := exp.Equal("userId", userID).
		AndEqual("object", object)

	return service.Query(session, criteria, options...)
}

func (service *Response) LoadByUserAndObject(session data.Session, userID primitive.ObjectID, object string, responseType string, response *model.Response) error {

	criteria := exp.Equal("userId", userID).
		AndEqual("object", object).
		AndEqual("type", responseType)

	return service.Load(session, criteria, response)
}

func (service *Response) LoadByActorAndObject(session data.Session, actor string, object string, responseType string, response *model.Response) error {

	criteria := exp.Equal("actor", actor).
		AndEqual("object", object).
		AndEqual("type", responseType)

	return service.Load(session, criteria, response)
}

func (service *Response) CountByContent(session data.Session, objectID string) (mapof.Int, error) {
	collection := service.collection(session)
	return queries.CountResponsesByContent(collection, objectID)
}

/******************************************
 * Custom Behaviors
 ******************************************/

func (service *Response) DeleteByUserID(session data.Session, userID primitive.ObjectID, note string) error {

	const location = "service.Response.DeleteByUserID"

	rangeFunc, err := service.RangeByUserID(session, userID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to load responses by user", userID)
	}

	for response := range rangeFunc {
		if err := service.Delete(session, &response, note); err != nil {
			return derp.Wrap(err, location, "Unable to delete response", response)
		}
	}

	return nil
}

// SetResponse is the preferred way of creating/updating a Response.  It includes the business
// logic to search for an existing response, and delete it if one exists already (publishing UNDO actions in the process).
func (service *Response) SetResponse(session data.Session, user *model.User, url string, responseType string, content string) error {

	const location = "service.Response.SetResponse"

	// Remove previous Response (if it exists)
	if service.UnsetResponse(session, user, url, responseType) != nil {
		return derp.Wrap(nil, location, "Unable to remove previous response", user.UserID, url, responseType)
	}

	// Create a new Response object
	response := model.NewResponse()
	response.UserID = user.UserID
	response.Actor = user.ActivityPubURL()
	response.Object = url
	response.Type = responseType
	response.Content = content

	// Save the Response to the database (response service will automatically publish to ActivityPub and beyond)
	if err := service.Save(session, &response, "Set Response"); err != nil {
		return derp.Wrap(err, location, "Unable to save response", response)
	}

	activity := service.Activity(response)

	// Publish the new Response to the Outbox, sending "Like" notifications to all followers.
	if err := service.outboxService.Publish(session, model.FollowerTypeUser, user.UserID, activity, model.NewAnonymousPermissions()); err != nil {
		derp.Report(derp.Wrap(err, location, "Error publishing Response", response))
	}

	// Oye cómo va!
	return nil
}

// UnsetReponse removes a reponse based on the User, URL, and Response Type
func (service *Response) UnsetResponse(session data.Session, user *model.User, url string, responseType string) error {

	const location = "service.Response.UnsetResponse"

	// Search for a previous Response from this User
	previousResponse := model.NewResponse()
	err := service.LoadByUserAndObject(session, user.UserID, url, responseType, &previousResponse)

	if derp.IsNotFound(err) {
		return nil
	}

	if derp.NotNil(err) {
		return derp.Wrap(err, location, "Unable to load original response", user.UserID, url, responseType)
	}

	// Otherwise, delete the old Response
	if err := service.Delete(session, &previousResponse, ""); err != nil {
		return derp.Wrap(err, location, "Unable to delete old response", previousResponse)
	}

	// Success!!
	return nil
}

func (service *Response) Activity(response model.Response) streams.Document {
	return streams.NewDocument(response.GetJSONLD())
}
