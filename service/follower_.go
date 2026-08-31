package service

import (
	"iter"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/sherlock"
	"github.com/benpate/turbine/queue"
	"github.com/benpate/uri"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Follower defines a service that tracks the (possibly external) accounts that are followers of an internal User
type Follower struct {
	activityService   *ActivityStream
	domainEmail       *DomainEmail
	importItemService *ImportItem
	ruleService       *Rule
	streamService     *Stream
	userService       *User
	queue             *queue.Queue // The server-wide queue for background tasks
	host              string       // The HOST for this domain (protocol + hostname)
}

// NewFollower returns a fully initialized Follower service
func NewFollower() Follower {
	return Follower{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Follower) Refresh(factory *Factory) {
	service.activityService = factory.ActivityStream()
	service.domainEmail = factory.Email()
	service.importItemService = factory.ImportItem()
	service.ruleService = factory.Rule()
	service.streamService = factory.Stream()
	service.userService = factory.User()
	service.queue = factory.Queue()
	service.host = factory.Host()
}

/******************************************
 * Common Data Methods
 ******************************************/

// collection returns the mongo collection where Followers are stored
func (service *Follower) collection(session data.Session) data.Collection {
	return session.Collection("Follower")
}

// Query returns a slice containing all of the Followers who match the provided criteria
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
			derp.Report(derp.Wrap(err, "service.Follower.Range", "Creating iterator", criteria))
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
		return derp.Wrap(err, "service.Follower.Load", "Loading Follower", criteria)
	}

	return nil
}

// Save adds/updates an Follower in the database
func (service *Follower) Save(session data.Session, follower *model.Follower, note string) error {

	const location = "service.Follower.Save"

	// Validate the value before saving
	if _, err := service.Schema().Validate(follower); err != nil {
		return derp.Wrap(err, location, "Invalid Follower record", follower)
	}

	// Save the follower to the database
	if err := service.collection(session).Save(follower, note); err != nil {
		return derp.Wrap(err, location, "Saving Follower", follower, note)
	}

	// Recalculate the follower count for this user
	if err := service.userService.CalcFollowerCount(session, follower.ParentID); err != nil {
		return derp.Wrap(err, location, "Re-calculating follower count", follower)
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
		return derp.Wrap(err, location, "Deleting Follower", follower, note)
	}

	// Recalculate the follower count for this user.  This mirrors Save (above): without it the
	// denormalized count is monotonic, and it is now published to the network as `totalItems` on
	// the followers collection.  Every deletion path -- all four Undo(Follow) handlers, plus
	// DeleteByUserID -- funnels through here, so this is the only place it needs to happen.
	if err := service.userService.CalcFollowerCount(session, follower.ParentID); err != nil {
		return derp.Wrap(err, location, "Re-calculating follower count", follower)
	}

	// Maybe delete the SearchQuery if it's no longer needed
	if follower.ParentType == model.FollowerTypeSearch {

		postcommit.Publish(
			session,
			service.queue,
			"DeleteEmptySearchQuery",
			mapof.Any{
				"hostname":      uri.Hostname(service.host),
				"searchQueryID": follower.ParentID,
			},
		)
	}

	return nil
}

// Pause marks this Follower as paused by a block rule (R8): it stays out of every delivery
// fan-out until the block is deleted and the restore pass reactivates it. Saved directly
// (like the DELETED path) because PAUSED is server-set only and deliberately absent from the
// user-facing schema enum, so Save's validation would refuse it.
func (service *Follower) Pause(session data.Session, follower *model.Follower) error {

	const location = "service.Follower.Pause"

	// A Follower that is already paused has nothing more to pause
	if follower.StateID == model.FollowerStatePaused {
		return nil
	}

	follower.StateID = model.FollowerStatePaused

	if err := service.collection(session).Save(follower, "Paused by block rule"); err != nil {
		return derp.Wrap(err, location, "Saving Follower", follower)
	}

	return nil
}

// Reactivate returns a paused Follower to ACTIVE. It is the restore pass's write half: called
// only after the remaining rules have been re-evaluated and no block covers this actor anymore.
func (service *Follower) Reactivate(session data.Session, follower *model.Follower) error {

	const location = "service.Follower.Reactivate"

	follower.StateID = model.FollowerStateActive

	if err := service.collection(session).Save(follower, "Reactivated after block removed"); err != nil {
		return derp.Wrap(err, location, "Saving Follower", follower)
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
		return derp.Wrap(err, location, "Deleting Follower", "userID: "+userID.Hex(), "followerID: "+followerID.Hex())
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

// ObjectNew returns a fully initialized model.Follower as a data.Object.
func (service *Follower) ObjectNew() data.Object {
	result := model.NewFollower()
	return &result
}

// ObjectID returns the primary key of the provided Follower object
func (service *Follower) ObjectID(object data.Object) primitive.ObjectID {

	if mention, ok := object.(*model.Follower); ok {
		return mention.FollowerID
	}

	return primitive.NilObjectID
}

// ObjectQuery populates the result value with all Followers who match the provided criteria
func (service *Follower) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

// ObjectLoad retrieves a single Follower who matches the provided criteria, as a data.Object
func (service *Follower) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewFollower()
	err := service.Load(session, criteria, &result)
	return &result, err
}

// ObjectSave saves the provided Follower object to the database
func (service *Follower) ObjectSave(session data.Session, object data.Object, comment string) error {
	if follower, ok := object.(*model.Follower); ok {
		return service.Save(session, follower, comment)
	}
	return derp.Internal("service.Follower.ObjectSave", "Invalid Object Type", object)
}

// ObjectDelete removes the provided Follower object from the database (virtual delete)
func (service *Follower) ObjectDelete(session data.Session, object data.Object, comment string) error {
	if follower, ok := object.(*model.Follower); ok {
		return service.Delete(session, follower, comment)
	}
	return derp.Internal("service.Follower.ObjectDelete", "Invalid Object Type", object)
}

// ObjectUserCan always returns Unauthorized: Followers are never edited through the generic data.Object path
func (service *Follower) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.Unauthorized("service.Follower", "Not Authorized")
}

// Schema returns the validating schema for Follower objects
func (service *Follower) Schema() schema.Schema {
	return schema.New(model.FollowerSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

// CountByParent counts the Followers of a single User or Stream
func (service *Follower) CountByParent(session data.Session, parentType string, parentID primitive.ObjectID) (int64, error) {
	criteria := exp.Equal("type", parentType).AndEqual("parentId", parentID)
	return service.Count(session, criteria)
}

// LoadOrCreate returns the Follower record for an Actor, creating an unsaved one if none exists yet
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
	return result, derp.Wrap(err, "service.Follower.LoadOrCreate", "Loading Follower", parentID, actorID)
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

// LoadBySecret loads an email Follower using the unlisted secret from their confirmation or
// unsubscribe link.  It is the only path that an anonymous visitor can use to reach a Follower
// record, so it carries the whole authorization for those two actions.
func (service *Follower) LoadBySecret(session data.Session, followerID primitive.ObjectID, secret string, follower *model.Follower) error {

	const location = "service.Follower.LoadBySecret"

	// RULE: The secret must not be empty.
	if secret == "" {
		return derp.Forbidden(location, "Secret cannot be empty", followerID)
	}

	// RULE: Only EMAIL Followers can be reached by secret.  The email flow is the only one that
	// issues a secret, so the method belongs in the query rather than in an assumption about who
	// is holding the link.
	criteria := exp.
		Equal("_id", followerID).
		AndEqual("method", model.FollowerMethodEmail)

	if err := service.Load(session, criteria, follower); err != nil {
		return derp.Wrap(err, location, "Loading follower", followerID)
	}

	// Verify that the secret matches
	if follower.Data.GetString("secret") != secret {
		return derp.Forbidden(location, "Invalid secret", followerID)
	}

	// Success
	return nil
}

// LoadByActor retrieves an Follower from the database by parentID and actorID
func (service *Follower) LoadByActor(session data.Session, parentID primitive.ObjectID, actorID string, follower *model.Follower) error {

	// RULE: Allow parentID to be zero.  This means it's the "@search" actor

	// RULE: The actorID must not be empty
	if actorID == "" {
		return derp.Validation("ActorID cannot be empty", actorID)
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

// RangeActivityPubByType returns an iterator containing all of the ActivityPub Followers of a specific parent
func (service *Follower) RangeActivityPubByType(session data.Session, followerType string, userID primitive.ObjectID) iter.Seq[model.Follower] {

	// RULE: Followers paused by a block rule are excluded from delivery fan-out (R8)
	return service.Range(
		session,
		exp.Equal("parentId", userID).
			AndEqual("type", followerType).
			AndEqual("method", model.FollowerMethodActivityPub).
			AndNotEqual("stateId", model.FollowerStatePaused),
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

// RangeByGlobalSearch returns an iterator containing all of the Followers of the domain-wide search Actor
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
			return derp.Wrap(err, location, "Deleting follower", follower)
		}
	}

	return nil
}

// RangeFollowers returns a rangeFunc containing all of the Followers of specific parentID
func (service *Follower) RangeFollowers(session data.Session, parentType string, parentID primitive.ObjectID) iter.Seq[model.Follower] {

	// RULE: Followers paused by a block rule are excluded from delivery fan-out (R8). Only
	// PAUSED is excluded -- other states keep their existing delivery behavior.
	return service.Range(
		session,
		exp.Equal("parentId", parentID).
			AndEqual("type", parentType).
			AndNotEqual("stateId", model.FollowerStatePaused),
	)
}

// RangePausedByUserID returns an iterator containing every PAUSED Follower of the provided User.
// This is the restore pass's source: deleting a block re-evaluates exactly these rows (R8).
func (service *Follower) RangePausedByUserID(session data.Session, userID primitive.ObjectID) iter.Seq[model.Follower] {
	return service.Range(
		session,
		exp.Equal("parentId", userID).
			AndEqual("type", model.FollowerTypeUser).
			AndEqual("stateId", model.FollowerStatePaused),
	)
}

// QueryByParentAndDate returns one page of a parent's Followers of a single method, newest first
func (service *Follower) QueryByParentAndDate(session data.Session, parentType string, parentID primitive.ObjectID, method string, maxCreateDate int64, pageSize int) ([]model.Follower, error) {

	criteria := exp.Equal("type", parentType).
		AndEqual("parentId", parentID).
		AndEqual("method", method).
		AndLessThan("createDate", maxCreateDate)

	return service.Query(session, criteria, option.SortDesc("createDate"), option.MaxRows(int64(pageSize)))
}

// LoadParentActor returns the PersonLink of the User or Stream that a Follower follows
func (service *Follower) LoadParentActor(session data.Session, follower *model.Follower) (model.PersonLink, error) {

	const location = "service.Follower.LoadParentActor"

	switch follower.ParentType {

	case model.FollowerTypeUser:

		user := model.NewUser()
		if err := service.userService.LoadByID(session, follower.ParentID, &user); err != nil {
			return model.PersonLink{}, derp.Wrap(err, location, "Loading parent user", follower)
		}

		return user.PersonLink(), nil

	case model.FollowerTypeStream:

		stream := model.NewStream()
		if err := service.streamService.LoadByID(session, follower.ParentID, &stream); err != nil {
			return model.PersonLink{}, derp.Wrap(err, location, "Loading parent stream", follower)
		}

		return stream.ActorLink(), nil

	}

	return model.PersonLink{}, derp.Internal(location, "Invalid parentType", follower)
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

// NewActivityPubFollower creates (or refreshes) an active Follower record from a remote Actor document
func (service *Follower) NewActivityPubFollower(session data.Session, parentType string, parentID primitive.ObjectID, actor streams.Document, follower *model.Follower) error {

	const location = "service.Follower.NewActivityPubFollower"

	// Try to find an existing follower record
	if err := service.LoadByActor(session, parentID, actor.ID(), follower); err != nil {
		if !derp.IsNotFound(err) {
			return derp.Wrap(err, location, "Loading existing follower", actor)
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
	if err := service.Save(session, follower, "via ActivityPub"); err != nil {
		return derp.Wrap(err, location, "Saving new follower", follower)
	}

	// Salút!
	return nil
}

// LoadByActivityPubFollower loads the ActivityPub Follower of a parent, identified by their profile URL
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
		return streams.NilDocument(), derp.Internal("service.Follower.RemoteActor", "Follower must use ActivityPub method", follower)
	}

	// Return the remote Actor's profile document
	return service.activityService.Client(follower.ParentType, follower.ParentID).Load(follower.Actor.ProfileURL, sherlock.AsActor())
}

/******************************************
 * ActivityPub Methods
 ******************************************/

// ActivityPubID returns the URL that identifies this Follow activity to ActivityPub
func (service *Follower) ActivityPubID(follower *model.Follower) string {
	return service.host + "/@" + follower.ParentID.Hex() + "/pub/follower/" + follower.FollowerID.Hex()
}

// ActivityPubObjectID returns the URL of the Actor that this Follower follows
func (service *Follower) ActivityPubObjectID(follower *model.Follower) string {
	return service.host + "/@" + follower.ParentID.Hex()
}

// AsJSONLD returns a Follower as an ActivityStreams Follow activity
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

// LoadPendingEmailFollower loads an unconfirmed email Follower, and verifies their confirmation secret
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
		return derp.Internal("service.Follower.SendFollowConfirmation", "Follower must use Email method", follower)
	}

	actor, err := service.LoadParentActor(session, follower)

	if err != nil {
		return derp.Wrap(err, "service.Follower.SendFollowConfirmation", "Loading parent actor", follower)
	}

	if err := service.domainEmail.SendFollowerConfirmation(actor, follower); err != nil {
		return derp.Wrap(err, "service.Follower.SendFollowConfirmation", "Sending follow confirmation email", follower)
	}

	return nil
}
