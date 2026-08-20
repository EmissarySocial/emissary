package service

import (
	"iter"
	"slices"
	"time"

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
	"github.com/benpate/uri"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Response defines a service that can send and receive response data
type Response struct {
	activityStreamService *ActivityStream
	importItemService     *ImportItem
	newsFeedService       *NewsFeed
	outboxService         *Outbox
	ruleService           *Rule
	userService           *User
	host                  string

	// loadDocument resolves a URL to an ActivityStream Document via the App client (which caches).
	// It is held as a field -- rather than reaching through activityStreamService inline -- so that
	// tests can inject a fake loader without standing up the concrete ActivityStream/network stack.
	loadDocument func(string) (streams.Document, error)
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
	service.ruleService = factory.Rule()
	service.userService = factory.User()
	service.host = factory.Host()

	// Resolve reaction targets through the App client, which caches the loaded document so the
	// validation fetch and the later author-audience lookup share a single round-trip.
	service.loadDocument = func(url string) (streams.Document, error) {
		return service.activityStreamService.AppClient().Load(url)
	}
}

// Close stops any background processes controlled by this service
func (service *Response) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

// collection returns the Response collection for the provided database session
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
	options := service.reactionAudience(&user, response, originalActivity, service.objectAuthorURL(response))

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

// ObjectID returns the unique ID of the provided Response. Implements the ModelService interface.
func (service *Response) ObjectID(object data.Object) primitive.ObjectID {

	if response, ok := object.(*model.Response); ok {
		return response.ResponseID
	}

	return primitive.NilObjectID
}

// ObjectQuery returns every Response that matches the provided criteria. Implements the ModelService interface.
func (service *Response) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

// ObjectLoad retrieves a single Response as a data.Object. Implements the ModelService interface.
func (service *Response) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewResponse()
	err := service.Load(session, criteria, &result)
	return &result, err
}

// ObjectSave adds or updates a Response in the database. Implements the ModelService interface.
func (service *Response) ObjectSave(session data.Session, object data.Object, note string) error {

	if response, ok := object.(*model.Response); ok {
		return service.Save(session, response, note)
	}
	return derp.Internal("service.Response.ObjectSave", "Invalid object type", object)
}

// ObjectDelete marks a Response as deleted. Implements the ModelService interface.
func (service *Response) ObjectDelete(session data.Session, object data.Object, note string) error {
	if response, ok := object.(*model.Response); ok {
		return service.Delete(session, response, note)
	}
	return derp.Internal("service.Response.ObjectDelete", "Invalid object type", object)
}

// ObjectUserCan reports whether the provided Authorization may run an action on a Response. Implements the ModelService interface.
func (service *Response) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.Unauthorized("service.Response", "Not Authorized")
}

// Schema returns the rosetta schema that describes a Response
func (service *Response) Schema() schema.Schema {
	return schema.New(model.ResponseSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

// QueryByUserAndDate returns one page of a User's Responses of the provided type, newest first
func (service *Response) QueryByUserAndDate(session data.Session, userID primitive.ObjectID, responseType string, maxDate int64, pageSize int) ([]model.Response, error) {

	criteria := exp.Equal("userId", userID).AndEqual("type", responseType).And(exp.LessThan("createDate", maxDate))
	options := []option.Option{option.SortDesc("createDate"), option.MaxRows(int64(pageSize))}

	return service.Query(session, criteria, options...)
}

// QueryByObjectAndDate returns one page of the Responses to an object, newest first
func (service *Response) QueryByObjectAndDate(session data.Session, objectID string, responseType string, maxDate int64, pageSize int) ([]model.Response, error) {

	criteria := exp.Equal("object", objectID).AndEqual("type", responseType).And(exp.LessThan("createDate", maxDate))
	options := []option.Option{option.SortDesc("createDate"), option.MaxRows(int64(pageSize))}

	return service.Query(session, criteria, options...)
}

// LoadByID retrieves a single Response using its unique ID
func (service *Response) LoadByID(session data.Session, userID primitive.ObjectID, responseID primitive.ObjectID, response *model.Response) error {
	criteria := exp.Equal("userId", userID).AndEqual("_id", responseID)
	return service.Load(session, criteria, response)
}

// RangeByUserID returns an iterator over every Response belonging to the provided User
func (service *Response) RangeByUserID(session data.Session, userID primitive.ObjectID, options ...option.Option) (iter.Seq[model.Response], error) {

	criteria := exp.Equal("userId", userID)

	return service.Range(session, criteria, options...)
}

// QueryByUserAndObject returns every Response that the provided User has made to an object
func (service *Response) QueryByUserAndObject(session data.Session, userID primitive.ObjectID, object string, options ...option.Option) ([]model.Response, error) {

	criteria := exp.Equal("userId", userID).
		AndEqual("object", object)

	return service.Query(session, criteria, options...)
}

// LoadByUserAndObject retrieves the Response of the provided type that a User made to an object
func (service *Response) LoadByUserAndObject(session data.Session, userID primitive.ObjectID, object string, responseType string, response *model.Response) error {

	criteria := exp.Equal("userId", userID).
		AndEqual("object", object).
		AndEqual("type", responseType)

	return service.Load(session, criteria, response)
}

// LoadByActorAndObject retrieves the Response of the provided type that a remote actor made to an object
func (service *Response) LoadByActorAndObject(session data.Session, actor string, object string, responseType string, response *model.Response) error {

	criteria := exp.Equal("actor", actor).
		AndEqual("object", object).
		AndEqual("type", responseType)

	return service.Load(session, criteria, response)
}

// CountByContent tallies the Responses to an object, grouped by their content
func (service *Response) CountByContent(session data.Session, objectID string) (mapof.Int, error) {
	collection := service.collection(session)
	return queries.CountResponsesByContent(collection, objectID)
}

/******************************************
 * Custom Behaviors
 ******************************************/

// DeleteByUserID marks every Response belonging to the provided User as deleted
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

// SetResponse is the preferred way of creating/updating a Response.  It guarantees one active
// reaction per (User, Object) within a group of contradictory types -- removing any reaction that
// the new one displaces, and publishing an UNDO for each -- and it is idempotent, so repeating an
// identical reaction changes nothing and publishes nothing.
func (service *Response) SetResponse(session data.Session, user *model.User, url string, responseType string, content string) error {

	const location = "service.Response.SetResponse"

	// RULE: validate the reaction target BEFORE reading or writing anything.  It must be a
	// well-formed http(s) URL that resolves to a real object -- a reaction to a malformed,
	// nonexistent, or unreachable URL is rejected here rather than stored and (over federation)
	// broadcast to an arbitrary target.  The loaded document is reused below for the delivery
	// audience, so the target is fetched only once.
	object, err := service.validateReactionTarget(url)

	if err != nil {
		return derp.Wrap(err, location, "Invalid reaction target", url)
	}

	// RULE: R11 -- reacting to content from an actor this User has blocked is refused. Every
	// reaction funnel (web, Mastodon API, intents) passes through here, so all of them inherit
	// it. The author check mirrors the newsfeed walk: attributedTo, falling back to actor.
	author := object.AttributedTo().ID()

	if author == "" {
		author = object.ActorID()
	}

	disposition, err := service.ruleService.DispositionForKeys(session, user.UserID, model.ActorMatchKeys(author), time.Now().Unix())

	if err != nil {
		return derp.Wrap(err, location, "Checking rules before reacting", author)
	}

	if disposition.IsBlocked() {
		return derp.Forbidden(location, "Cannot react to content from a blocked account", author)
	}

	// Find every existing reaction that this new one displaces
	displaced, err := service.conflictingResponses(session, user.UserID, url, responseType)

	if err != nil {
		return derp.Wrap(err, location, "Loading conflicting Responses", user.UserID, url, responseType)
	}

	// RULE: Repeating an identical reaction is a no-op.  Callers are stateless setters (a
	// double-click, a resubmitted form, or a retried API call all arrive as "make this true"),
	// so re-Saving would duplicate the Response and spray pointless Undo/redo churn at the author.
	if responseIsUnchanged(displaced, responseType, content) {
		return nil
	}

	// Remove the displaced reactions.  Each Delete publishes its own Undo activity.
	for _, displacedResponse := range displaced {
		if err := service.Delete(session, &displacedResponse, "Displaced by a new Response"); err != nil {
			return derp.Wrap(err, location, "Deleting displaced Response", displacedResponse)
		}
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

		// A lost creation race trips the unique (userId, object, type) index, which data-mongo
		// reports as a Conflict.  The winner recorded the very reaction we were asked to set, so
		// the caller's desired end state already holds and re-inserting would duplicate it.
		// See queries/sync/response.go for the index.
		if derp.IsConflict(err) {
			return nil
		}

		return derp.Wrap(err, location, "Saving response", response)
	}

	// Build the outgoing activity map, then apply the per-type audience: Like/Dislike deliver
	// author-only (via the returned WithRecipients option); Announce is stamped Public + cc and
	// keeps the default follower fan-out (D7b / resolved Q3).  The author is read from the object
	// already loaded during validation, so no second fetch is needed here.
	activityMap := response.GetJSONLD()
	options := service.reactionAudience(user, &response, activityMap, object.AttributedTo().ID())

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

// conflictingResponses returns every existing Response by this User on this Object whose type
// cannot coexist with the provided type.  The result includes any Response of the provided type,
// so a repeated reaction is displaced by the new one instead of duplicating it.
func (service *Response) conflictingResponses(session data.Session, userID primitive.ObjectID, url string, responseType string) ([]model.Response, error) {

	const location = "service.Response.conflictingResponses"

	responses, err := service.QueryByUserAndObject(session, userID, url)

	if err != nil {
		return nil, derp.Wrap(err, location, "Querying Responses", userID, url)
	}

	conflictingTypes := model.ConflictingResponseTypes(responseType)
	result := make([]model.Response, 0, len(responses))

	for _, response := range responses {
		if slices.Contains(conflictingTypes, response.Type) {
			result = append(result, response)
		}
	}

	return result, nil
}

// responseIsUnchanged returns TRUE if the existing Responses are exactly the reaction being
// requested, meaning there is nothing to add, remove, or publish.
func responseIsUnchanged(existing []model.Response, responseType string, content string) bool {

	// Nothing to repeat (zero), or a contradiction to resolve (two or more, which only
	// pre-fix data can produce).  Either way the caller's reaction still has work to do.
	if len(existing) != 1 {
		return false
	}

	// A different type is a reaction being switched, not repeated (e.g. Dislike -> Like)
	if existing[0].Type != responseType {
		return false
	}

	// Same type, so only the content can still differ (e.g. changing an emoji)
	return existing[0].Content == content
}

// validateReactionTarget verifies that a reaction's target URL is safe to react to and returns the
// loaded object for reuse. It rejects (a) anything that is not a well-formed http(s) URL and (b)
// anything that does not resolve to a real object. A successful load also warms the document cache
// for the delivery-audience lookup that follows.
func (service *Response) validateReactionTarget(url string) (streams.Document, error) {

	const location = "service.Response.validateReactionTarget"

	// RULE: the target must be a syntactically valid http(s) URL. This is a cheap, network-free
	// gate that also keeps the client below from being pointed at non-http schemes.
	if uri.NotValidURL(url) {
		return streams.NilDocument(), derp.BadRequest(location, "Reaction target must be a valid http(s) URL", url)
	}

	// RULE: the target must resolve to a real object. A successful load proves the object exists
	// (and is reachable); a failure rejects reactions to nonexistent or unreachable URLs.
	object, err := service.loadDocument(url)

	if err != nil {
		return streams.NilDocument(), derp.Wrap(err, location, "Reaction target does not resolve to a known object", url)
	}

	return object, nil
}

// objectAuthorURL resolves the AUTHOR ACTOR (attributedTo) of the object this Response reacted to.
// Only actors have inboxes (D7a), so the author — not the object URL — is the deliverable target of
// a reaction. Returns "" if the object cannot be loaded or has no attributedTo.
func (service *Response) objectAuthorURL(response *model.Response) string {

	object, err := service.loadDocument(response.Object)

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
// for its followers-collection URL. `authorURL` is the reacted-to object's author (attributedTo),
// resolved by the caller -- "" when it could not be determined. When the author cannot be resolved,
// Like/Dislike fall back to an empty recipient set (suppressing fan-out) rather than accidentally
// broadcasting.
func (service *Response) reactionAudience(user *model.User, response *model.Response, activity mapof.Any, authorURL string) []PublishOption {

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
