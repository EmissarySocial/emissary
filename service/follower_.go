package service

import (
	"iter"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	dt "github.com/benpate/domain"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/sherlock"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Follower defines a service that tracks the (possibly external) accounts that are followers of an internal User

type Follower struct {
	factory           *Factory
	domainEmail       *DomainEmail
	importItemService *ImportItem
	ruleService       *Rule
	streamService     *Stream
	userService       *User
	queue             *queue.Queue // The server-wide queue for background tasks
	host              string       // The HOST for this domain (protocol + hostname)
}

// NewFollower returns a fully initialized Follower service
func NewFollower(factory *Factory) Follower {
	return Follower{
		factory: factory,
	}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Follower) Refresh(domainEmail *DomainEmail, importItemService *ImportItem, ruleService *Rule, streamService *Stream, userService *User, queue *queue.Queue, host string) {
	service.domainEmail = domainEmail
	service.importItemService = importItemService
	service.ruleService = ruleService
	service.streamService = streamService
	service.userService = userService
	service.queue = queue
	service.host = host
}

// Close stops any background processes controlled by this service
func (service *Follower) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *Follower) collection(session data.Session) data.Collection {
	return session.Collection("Follower")
}

func (service *Follower) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.Follower, error) {
	result := make([]model.Follower, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// Count returns the number of records that match the provided criteria
func (service *Follower) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// List returns an iterator containing all of the Followers who match the provided criteria
func (service *Follower) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the Follower records that match the provided criteria
func (service *Follower) Range(session data.Session, criteria exp.Expression, options ...option.Option) iter.Seq[model.Follower] {

	return func(yield func(model.Follower) bool) {

		// Retrieve the Followers from the database
		followers, err := service.List(session, criteria, options...)

		// Soft fail.  Report, but do not crash.
		if err != nil {
			derp.Report(derp.Wrap(err, "service.Follower.Range", "Unable to create iterator", criteria))
			return
		}

		defer derp.ReportFunc(followers.Close)

		// Yield each follower to the caller one-by-one
		for follower := model.NewFollower(); followers.Next(&follower); follower = model.NewFollower() {
			if !yield(follower) {
				return
			}
		}
	}
}

// Load retrieves an Follower from the database
func (service *Follower) Load(session data.Session, criteria exp.Expression, follower *model.Follower) error {

	if err := service.collection(session).Load(notDeleted(criteria), follower); err != nil {
		return derp.Wrap(err, "service.Follower.Load", "Unable to load Follower", criteria)
	}

	return nil
}

// Save adds/updates an Follower in the database
func (service *Follower) Save(session data.Session, follower *model.Follower, note string) error {

	const location = "service.Follower.Save"

	// Validate the value before saving
	if err := service.Schema().Validate(follower); err != nil {
		return derp.Wrap(err, location, "Invalid Follower record", follower)
	}

	// Save the follower to the database
	if err := service.collection(session).Save(follower, note); err != nil {
		return derp.Wrap(err, location, "Unable to save Follower", follower, note)
	}

	// Recalculate the follower count for this user
	if err := service.userService.CalcFollowerCount(session, follower.ParentID); err != nil {
		return derp.Wrap(err, location, "Unable to re-calculate follower count", follower)
	}

	return nil
}

// Delete removes an Follower from the database (virtual delete)
func (service *Follower) Delete(session data.Session, follower *model.Follower, note string) error {

	const location = "service.Follower.Delete"

	// Mark the Follower as deleted
	follower.StateID = model.FollowerStateDeleted

	// Delete this Follower
	if err := service.collection(session).Delete(follower, note); err != nil {
		return derp.Wrap(err, location, "Unable to delete Follower", follower, note)
	}

	// Maybe delete the SearchQuery if it's no longer needed
	if follower.ParentType == model.FollowerTypeSearch {

		service.queue.NewTask(
			"DeleteEmptySearchQuery",
			mapof.Any{
				"host":          dt.NameOnly(service.host),
				"searchQueryID": follower.ParentID,
			},
		)
	}

	return nil
}

/******************************************
 * Special Case Methods
 ******************************************/

// QueryIDOnly returns a slice of IDOnly records that match the provided criteria
func (service *Follower) QueryIDOnly(session data.Session, criteria exp.Expression, options ...option.Option) (sliceof.Object[model.IDOnly], error) {
	result := make([]model.IDOnly, 0)
	options = append(options, option.Fields("_id"))
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// HardDeleteByID removes a specific Follower record, without applying any additional business rules
func (service *Follower) HardDeleteByID(session data.Session, userID primitive.ObjectID, followerID primitive.ObjectID) error {

	const location = "service.Follower.HardDeleteByID"

	criteria := exp.Equal("userId", userID).AndEqual("_id", followerID)

	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Unable to delete Follower", "userID: "+userID.Hex(), "followerID: "+followerID.Hex())
	}

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Follower) ObjectType() string {
	return "Follower"
}

// New returns a fully initialized model.Follower as a data.Object.
func (service *Follower) ObjectNew() data.Object {
	result := model.NewFollower()
	return &result
}

func (service *Follower) ObjectID(object data.Object) primitive.ObjectID {

	if mention, ok := object.(*model.Follower); ok {
		return mention.FollowerID
	}

	return primitive.NilObjectID
}

func (service *Follower) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

func (service *Follower) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewFollower()
	err := service.Load(session, criteria, &result)
	return &result, err
}

func (service *Follower) ObjectSave(session data.Session, object data.Object, comment string) error {
	if follower, ok := object.(*model.Follower); ok {
		return service.Save(session, follower, comment)
	}
	return derp.InternalError("service.Follower.ObjectSave", "Invalid Object Type", object)
}

func (service *Follower) ObjectDelete(session data.Session, object data.Object, comment string) error {
	if follower, ok := object.(*model.Follower); ok {
		return service.Delete(session, follower, comment)
	}
	return derp.InternalError("service.Follower.ObjectDelete", "Invalid Object Type", object)
}

func (service *Follower) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.UnauthorizedError("service.Follower", "Not Authorized")
}

func (service *Follower) Schema() schema.Schema {
	return schema.New(model.FollowerSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *Follower) CountByParent(session data.Session, parentType string, parentID primitive.ObjectID) (int64, error) {
	criteria := exp.Equal("type", parentType).AndEqual("parentId", parentID)
	return service.Count(session, criteria)
}

func (service *Follower) LoadOrCreate(session data.Session, parentID primitive.ObjectID, actorID string) (model.Follower, error) {

	result := model.NewFollower()

	err := service.LoadByActor(session, parentID, actorID, &result)

	// No error means the record was found
	if err == nil {
		return result, nil
	}

	// NotFound error means we should create a new record
	if derp.IsNotFound(err) {
		result.ParentID = parentID
		result.Actor.ProfileURL = actorID
		return result, nil
	}

	// Other error is bad.  Return the error
	return result, derp.Wrap(err, "service.Follower.LoadOrCreate", "Unable to load Follower", parentID, actorID)
}

// LoadByID loads a follower using the FollowerID
func (service *Follower) LoadByID(session data.Session, parentID primitive.ObjectID, followerID primitive.ObjectID, follower *model.Follower) error {

	// If the token is an ObjectID then load the follower by FollowerID
	criteria := exp.Equal("_id", followerID).AndEqual("parentId", parentID)
	return service.Load(session, criteria, follower)
}

// LoadByToken loads a follower using either the FollowerID (if an ObjectID is passed) or the Actor's ProfileURL
func (service *Follower) LoadByToken(session data.Session, parentID primitive.ObjectID, token string, follower *model.Follower) error {

	// If the token is an ObjectID then load the follower by FollowerID
	if followerID, err := primitive.ObjectIDFromHex(token); err == nil {
		return service.LoadByID(session, parentID, followerID, follower)
	}

	// Otherwise, load the Follower by the Actor's ProfileURL
	criteria := exp.Equal("parentId", parentID).AndEqual("actor.profileUrl", token)
	return service.Load(session, criteria, follower)
}

// LoadBySecret loads a follower based on the FollowerID.  It confirms that the secret value matches
func (service *Follower) LoadBySecret(session data.Session, followerID primitive.ObjectID, secret string, follower *model.Follower) error {

	const location = "service.Follower.LoadBySecret"

	// RULE: The secret must not be empty.
	if secret == "" {
		return derp.ForbiddenError(location, "Secret cannot be empty", followerID)
	}

	// Load the Follower using the FollowerID
	criteria := exp.Equal("_id", followerID)
	if err := service.Load(session, criteria, follower); err != nil {
		return derp.Wrap(err, location, "Unable to load follower", followerID)
	}

	// Verify that the secret matches
	if follower.Data.GetString("secret") != secret {
		return derp.ForbiddenError(location, "Invalid secret", followerID)
	}

	// Success
	return nil
}

// LoadByActor retrieves an Follower from the database by parentID and actorID
func (service *Follower) LoadByActor(session data.Session, parentID primitive.ObjectID, actorID string, follower *model.Follower) error {

	// RULE: Allow parentID to be zero.  This means it's the "@search" actor

	// RULE: The actorID must not be empty
	if actorID == "" {
		return derp.ValidationError("ActorID cannot be empty", actorID)
	}

	criteria := exp.Equal("parentId", parentID).AndEqual("actor.profileUrl", actorID)
	return service.Load(session, criteria, follower)
}

// QueryByParent returns an slice containing all of the Followers of specific parentID
func (service *Follower) QueryByParent(session data.Session, parentType string, parentID primitive.ObjectID, options ...option.Option) ([]model.Follower, error) {

	criteria := exp.Equal("type", parentType).
		AndEqual("parentId", parentID)

	return service.Query(session, criteria, options...)
}

// RangeByUserID returns an iterator containing all of the Followers of a specific User
func (service *Follower) RangeByUserID(session data.Session, userID primitive.ObjectID) iter.Seq[model.Follower] {
	return service.Range(
		session,
		exp.Equal("parentId", userID).
			AndEqual("type", model.FollowerTypeUser),
	)
}

// RangeActivityPubByUserID returns an iterator containing all of the Followers of a specific User
func (service *Follower) RangeActivityPubByType(session data.Session, followerType string, userID primitive.ObjectID) iter.Seq[model.Follower] {
	return service.Range(
		session,
		exp.Equal("parentId", userID).
			AndEqual("type", followerType).
			AndEqual("method", model.FollowerMethodActivityPub),
	)
}

// RangeBySearch returns an iterator containing all of the Followers of a specific SearchQuery
func (service *Follower) RangeBySearch(session data.Session, searchQueryID primitive.ObjectID) iter.Seq[model.Follower] {
	return service.Range(
		session,
		exp.Equal("parentId", searchQueryID).
			AndEqual("type", model.FollowerTypeSearch),
	)
}

// RangeBySearch returns an iterator containing all of the Followers of a specific SearchQuery
func (service *Follower) RangeByGlobalSearch(session data.Session) iter.Seq[model.Follower] {

	// Special case for Domain search queries.
	return service.Range(
		session,
		exp.Equal("parentId", primitive.NilObjectID).
			AndEqual("type", model.FollowerTypeSearchDomain),
	)
}

// DeleteByUserID removes all Followers of a specific User
func (service *Follower) DeleteByUserID(session data.Session, userID primitive.ObjectID, comment string) error {

	const location = "service.Follower.DeleteByUserID"

	for follower := range service.RangeByUserID(session, userID) {

		if err := service.Delete(session, &follower, comment); err != nil {
			return derp.Wrap(err, location, "Error deleting follower", follower)
		}
	}

	return nil
}

// RangeFollowers returns a rangeFunc containing all of the Followers of specific parentID
func (service *Follower) RangeFollowers(session data.Session, parentType string, parentID primitive.ObjectID) iter.Seq[model.Follower] {

	return service.Range(
		session,
		exp.Equal("parentId", parentID).
			AndEqual("type", parentType),
	)
}

func (service *Follower) QueryByParentAndDate(session data.Session, parentType string, parentID primitive.ObjectID, method string, maxCreateDate int64, pageSize int) ([]model.Follower, error) {

	criteria := exp.Equal("type", parentType).
		AndEqual("parentId", parentID).
		AndEqual("method", method).
		AndLessThan("createDate", maxCreateDate)

	return service.Query(session, criteria, option.SortDesc("createDate"), option.MaxRows(int64(pageSize)))
}

/******************************************
 * WebSub Queries
 ******************************************/

// LoadByWebSub retrieves a follower based on the parentID and callback
func (service *Follower) LoadByWebSub(session data.Session, objectType string, parentID primitive.ObjectID, callback string, result *model.Follower) error {

	criteria := exp.
		Equal("type", objectType).
		AndEqual("parentId", parentID).
		AndEqual("method", model.FollowerMethodWebSub).
		AndEqual("actor.inboxUrl", callback)

	return service.Load(session, criteria, result)
}

// LoadOrCreateByWebSub finds a follower based on the parentID and callback.  If no follower is found, a new record is created.
func (service *Follower) LoadOrCreateByWebSub(session data.Session, objectType string, parentID primitive.ObjectID, callback string) (model.Follower, error) {

	// Try to load the Follower from the database
	result := model.NewFollower()
	err := service.LoadByWebSub(session, objectType, parentID, callback, &result)

	// If EXISTS, then we've found it.
	if err == nil {
		return result, nil
	}

	// If NOT EXISTS, then create a new one
	if derp.IsNotFound(err) {
		result.ParentID = parentID
		result.ParentType = objectType
		result.Method = model.FollowerMethodWebSub
		result.Actor.InboxURL = callback
		return result, nil
	}

	// If REAL ERROR, then derp
	return result, derp.Wrap(err, "service.Follower.LoadByWebSub", "Unable to load follower", parentID, callback)
}

func (service *Follower) LoadParentActor(session data.Session, follower *model.Follower) (model.PersonLink, error) {

	switch follower.ParentType {

	case model.FollowerTypeUser:

		user := model.NewUser()
		if err := service.userService.LoadByID(session, follower.ParentID, &user); err != nil {
			return model.PersonLink{}, derp.Wrap(err, "service.Follower.LoadParentActor", "Unable to load parent user", follower)
		}

		return user.PersonLink(), nil

	case model.FollowerTypeStream:

		stream := model.NewStream()
		if err := service.streamService.LoadByID(session, follower.ParentID, &stream); err != nil {
			return model.PersonLink{}, derp.Wrap(err, "service.Follower.LoadParentActor", "Unable to load parent stream", follower)
		}

		return stream.ActorLink(), nil

	}

	return model.PersonLink{}, derp.InternalError("service.Follower.LoadParentActor", "Invalid parentType", follower)
}

/******************************************
 * ActivityPub Queries
 ******************************************/

// IsActivityPubFollower searches
func (service *Follower) IsActivityPubFollower(session data.Session, parentType string, parentID primitive.ObjectID, followerURL string) bool {
	result := model.NewFollower()
	err := service.LoadByActivityPubFollower(session, parentType, parentID, followerURL, &result)
	return err == nil
}

// ListActivityPub returns an iterator containing all of the Followers of specific parentID
func (service *Follower) ListActivityPub(session data.Session, parentID primitive.ObjectID, options ...option.Option) (data.Iterator, error) {

	criteria := exp.
		Equal("parentId", parentID).
		AndEqual("method", model.FollowerMethodActivityPub)

	return service.List(session, criteria, options...)
}

func (service *Follower) NewActivityPubFollower(session data.Session, parentType string, parentID primitive.ObjectID, actor streams.Document, follower *model.Follower) error {

	const location = "service.Follower.NewActivityPubFollower"

	// Try to find an existing follower record
	if err := service.LoadByActor(session, parentID, actor.ID(), follower); err != nil {
		if !derp.IsNotFound(err) {
			return derp.Wrap(err, location, "Unable to load existing follower", actor)
		}
	}

	// Set/Update follower data from the activity
	follower.Method = model.FollowerMethodActivityPub
	follower.ParentType = parentType
	follower.ParentID = parentID
	follower.StateID = model.FollowerStateActive

	follower.Actor = model.PersonLink{
		ProfileURL:   actor.ID(),
		Name:         actor.Name(),
		Username:     actor.UsernameOrID(),
		IconURL:      actor.IconOrImage().URL(),
		InboxURL:     actor.Get("inbox").String(),
		EmailAddress: actor.Get("email").String(),
	}

	// Try to save the new follower to the database
	if err := service.Save(session, follower, "New Follower via ActivityPub"); err != nil {
		return derp.Wrap(err, location, "Unable to save new follower", follower)
	}

	// Salút!
	return nil
}

func (service *Follower) LoadByActivityPubFollower(session data.Session, parentType string, parentID primitive.ObjectID, followerURL string, follower *model.Follower) error {

	criteria := exp.
		Equal("type", parentType).
		AndEqual("parentId", parentID).
		AndEqual("method", model.FollowerMethodActivityPub).
		AndEqual("actor.profileUrl", followerURL)

	return service.Load(session, criteria, follower)
}

// RemoteActor returns the ActivityStream document for a remote Actor for a specific Follower
func (service *Follower) RemoteActor(session data.Session, follower *model.Follower) (streams.Document, error) {

	// RULE: Guarantee that the Follower is using ActivityPub for updates
	if follower.Method != model.FollowerMethodActivityPub {
		return streams.NilDocument(), derp.InternalError("service.Follower.RemoteActor", "Follower must use ActivityPub method", follower)
	}

	activityService := service.factory.ActivityStream(follower.ParentType, follower.ParentID)

	// Return the remote Actor's profile document
	return activityService.Client().Load(follower.Actor.ProfileURL, sherlock.AsActor())
}

/******************************************
 * ActivityPub Methods
 ******************************************/

func (service *Follower) ActivityPubID(follower *model.Follower) string {
	return service.host + "/@" + follower.ParentID.Hex() + "/pub/follower/" + follower.FollowerID.Hex()
}

func (service *Follower) ActivityPubObjectID(follower *model.Follower) string {
	return service.host + "/@" + follower.ParentID.Hex()
}

func (service *Follower) AsJSONLD(follower *model.Follower) mapof.Any {

	return mapof.Any{
		"@context": vocab.ContextTypeActivityStreams,
		"id":       service.ActivityPubID(follower),
		"type":     vocab.ActivityTypeFollow,
		"actor":    follower.Actor.ProfileURL,
		"object":   service.ActivityPubObjectID(follower),
	}
}

/******************************************
 * Email Queries
 ******************************************/

func (service *Follower) LoadPendingEmailFollower(session data.Session, followerID primitive.ObjectID, secret string, follower *model.Follower) error {

	criteria := exp.
		Equal("_id", followerID).
		AndEqual("method", model.FollowerMethodEmail).
		AndEqual("stateId", model.FollowerStatePending).
		AndEqual("data.secret", secret)

	return service.Load(session, criteria, follower)
}

/******************************************
 * Email Methods
 ******************************************/

// SendFollowConfirmation sends an email to an email-type follower with a link to confirm their subscription.
// Subscriptions are not "ACTIVE" until confirmed.
func (service *Follower) SendFollowConfirmation(session data.Session, follower *model.Follower) error {

	// RULE: This method only applies to EMAIL-type Followers
	if follower.Method != model.FollowerMethodEmail {
		return derp.InternalError("service.Follower.SendFollowConfirmation", "Follower must use Email method", follower)
	}

	actor, err := service.LoadParentActor(session, follower)

	if err != nil {
		return derp.Wrap(err, "service.Follower.SendFollowConfirmation", "Unable to load parent actor", follower)
	}

	if err := service.domainEmail.SendFollowerConfirmation(actor, follower); err != nil {
		return derp.Wrap(err, "service.Follower.SendFollowConfirmation", "Unable to send follow confirmation email", follower)
	}

	return nil
}
