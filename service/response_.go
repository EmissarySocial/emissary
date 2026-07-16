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
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Response defines a service that can send and receive response data
type Response struct {
	activityStreamService *ActivityStream
	importItemService     *ImportItem
	newsFeedService       *NewsFeed
	outboxService         *Outbox
	userService           *User
	host                  string
}

// NewResponse returns a fully initialized Response service
func NewResponse() Response {
	return Response{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Response) Refresh(factory *Factory) {
	service.activityStreamService = factory.ActivityStream()
	service.importItemService = factory.ImportItem()
	service.newsFeedService = factory.NewsFeed()
	service.outboxService = factory.Outbox()
	service.userService = factory.User()
	service.host = factory.Host()
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
		return nil, derp.Wrap(err, "service.User.Range", "Creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewResponse), nil
}

// Load retrieves an Response from the database
func (service *Response) Load(session data.Session, criteria exp.Expression, response *model.Response) error {

	if err := service.collection(session).Load(notDeleted(criteria), response); err != nil {
		return derp.Wrap(err, "service.Response.Load", "Loading Response", criteria)
	}

	return nil
}

// Save adds/updates an Response in the database
func (service *Response) Save(session data.Session, response *model.Response, note string) error {

	const location = "service.Response.Save"

	// Validate the value before saving
	if _, err := service.Schema().Validate(response); err != nil {
		return derp.Wrap(err, location, "Validating Response", response)
	}

	// Save the value to the database
	if err := service.collection(session).Save(response, note); err != nil {
		return derp.Wrap(err, location, "Saving Response", response, note)
	}

	// Try to update the inbox message being responded to
	if err := service.newsFeedService.setResponse(session, response.UserID, response.Object, response.Type, response.ResponseID); err != nil {
		return derp.Wrap(err, location, "Setting Response to inbox message", response.UserID)
	}

	// NOTE: a Response does NOT write its own CollectionItem. The object-side projection is owned
	// solely by the inbound funnel: SetResponse publishes the reaction to the author, and the
	// resulting inbox delivery (including the self-loopback) projects it. See COLLECTIONS-REDESIGN.md D6.

	return nil
}

// Delete removes an Response from the database (hard delete)
func (service *Response) Delete(session data.Session, response *model.Response, note string) error {

	const location = "service.Response.Delete"

	// Delete this Response
	if err := service.collection(session).HardDelete(exp.Equal("_id", response.ResponseID)); err != nil {
		return derp.Wrap(err, location, "Deleting Response", response)
	}

	// Try to update the inbox message being responded to
	if err := service.newsFeedService.setResponse(session, response.UserID, response.Object, response.Type, primitive.NilObjectID); err != nil {
		return derp.Wrap(err, location, "Removing Response from inbox message", response.UserID)
	}

	// NOTE: no direct CollectionItem removal here. The Undo published below loops back through the
	// inbound funnel, which is the sole owner of the object-side projection (D6).

	// Load the reacting User (needed for the followers-collection URL when computing the Announce
	// audience). If the user is gone, fall back to a zero User — reactionAudience still resolves the
	// author and degrades safely.
	user := model.NewUser()
	if err := service.userService.LoadByID(session, response.UserID, &user); err != nil {
		derp.Report(derp.Wrap(err, location, "Loading User for Undo audience", response.UserID))
	}

	// Build the ORIGINAL activity's JSON-LD (embedded inline in the Undo) and apply the same
	// per-type audience the original reaction used, so the Undo reaches exactly the same actors.
	// Embedding it (rather than referencing by URL) matters because the Response row was just
	// hard-deleted, so its /pub/liked/<id> URL would 404 if dereferenced (D7). The options spread
	// MUST be preserved so an author-only reaction's Undo stays author-only (D7b).
	originalActivity := response.GetJSONLD()
	options := service.reactionAudience(&user, response, originalActivity)

	if err := service.outboxService.UndoActivity(session, model.FollowerTypeUser, response.UserID, originalActivity, model.NewAnonymousPermissions(), options...); err != nil {
		derp.Report(derp.Wrap(err, location, "Sending Undo activity"))
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
		return derp.Wrap(err, location, "Deleting Response", "userID: "+userID.Hex(), "responseID: "+responseID.Hex())
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

	criteria := exp.Equal("object", objectID).AndEqual("type", responseType).And(exp.LessThan("createDate", maxDate))
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
		return derp.Wrap(err, location, "Loading responses by user", userID)
	}

	for response := range rangeFunc {
		if err := service.Delete(session, &response, note); err != nil {
			return derp.Wrap(err, location, "Deleting response", response)
		}
	}

	return nil
}

// SetResponse is the preferred way of creating/updating a Response.  It includes the business
// logic to search for an existing response, and delete it if one exists already (publishing UNDO actions in the process).
func (service *Response) SetResponse(session data.Session, user *model.User, url string, responseType string, content string) error {

	const location = "service.Response.SetResponse"

	// Remove previous Response (if it exists)
	if err := service.UnsetResponse(session, user, url, responseType); err != nil {
		return derp.Wrap(err, location, "Removing previous response", user.UserID, url, responseType)
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
		return derp.Wrap(err, location, "Saving response", response)
	}

	// Build the outgoing activity map, then apply the per-type audience: Like/Dislike deliver
	// author-only (via the returned WithRecipients option); Announce is stamped Public + cc and
	// keeps the default follower fan-out (D7b / resolved Q3).
	activityMap := response.GetJSONLD()
	options := service.reactionAudience(user, &response, activityMap)

	// Publish the new Response to the Outbox.
	if err := service.outboxService.Publish(session, model.FollowerTypeUser, user.UserID, streams.NewDocument(activityMap), model.NewAnonymousPermissions(), options...); err != nil {
		derp.Report(derp.Wrap(err, location, "Publishing Response", response))
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
		return derp.Wrap(err, location, "Loading original response", user.UserID, url, responseType)
	}

	// Otherwise, delete the old Response
	if err := service.Delete(session, &previousResponse, ""); err != nil {
		return derp.Wrap(err, location, "Deleting old response", previousResponse)
	}

	// Success!!
	return nil
}

// objectAuthorURL resolves the AUTHOR ACTOR (attributedTo) of the object this Response reacted to.
// Only actors have inboxes (D7a), so the author — not the object URL — is the deliverable target of
// a reaction. Returns "" if the object cannot be loaded or has no attributedTo.
func (service *Response) objectAuthorURL(response *model.Response) string {

	object, err := service.activityStreamService.AppClient().Load(response.Object)

	if err != nil {
		derp.Report(derp.Wrap(err, "service.Response.objectAuthorURL", "Loading reacted-to object to resolve its author", response.Object))
		return ""
	}

	return object.AttributedTo().ID()
}

// reactionAudience computes the delivery audience for a reaction and applies it to the outgoing
// activity, following Mastodon (see COLLECTIONS-REDESIGN.md resolved Q3 / D7a / D7b):
//
//   - Like / Dislike: delivered ONLY to the liked object's AUTHOR. No `to`/`cc` on the wire (matching
//     Mastodon's minimal Like). Returns WithRecipients(<author>) so the follower fan-out is replaced.
//   - Announce (share): a broadcast. Stamps `to: [Public]` and `cc: [followers, author]` on the wire
//     and returns no override, so the default follower fan-out runs; the author is added as an
//     addressee (via `cc`) so they are delivered to as well.
//
// The `activity` map is mutated in place (Announce addressing). `user` is the reacting actor, needed
// for its followers-collection URL. When the author cannot be resolved, Like/Dislike fall back to an
// empty recipient set (suppressing fan-out) rather than accidentally broadcasting.
func (service *Response) reactionAudience(user *model.User, response *model.Response, activity mapof.Any) []PublishOption {

	authorURL := service.objectAuthorURL(response)

	// Announce is a public broadcast: Public in `to`, followers + author in `cc`.
	if response.Type == vocab.ActivityTypeAnnounce {

		activity[vocab.PropertyTo] = []string{vocab.NamespacePublic}

		cc := []string{user.ActivityPubFollowersURL()}
		if authorURL != "" {
			cc = append(cc, authorURL)
		}
		activity[vocab.PropertyCC] = cc

		// Default follower fan-out; the author reached via the `cc` addressee.
		return nil
	}

	// Like / Dislike are author-only, with no `to`/`cc` on the wire.
	if authorURL == "" {
		log.Warn().Str("object", response.Object).Str("type", response.Type).Msg("Response: reacted-to object has no resolvable author; reaction will not be delivered")
		return []PublishOption{WithRecipients()}
	}

	return []PublishOption{WithRecipients(authorURL)}
}
