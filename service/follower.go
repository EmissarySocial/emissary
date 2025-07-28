package service

import (
	"iter"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/domain"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/sherlock"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Follower defines a service that tracks the (possibly external) accounts that are followers of an internal User

type Follower struct {
	collection      data.Collection
	userService     *User
	ruleService     *Rule
	streamService   *Stream
	domainEmail     *DomainEmail
	activityService *ActivityStream
	queue           *queue.Queue // The server-wide queue for background tasks
	host            string       // The HOST for this domain (protocol + hostname)
}

// NewFollower returns a fully initialized Follower service
func NewFollower() Follower {
	return Follower{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Follower) Refresh(collection data.Collection, userService *User, streamService *Stream, ruleService *Rule, domainEmail *DomainEmail, activityService *ActivityStream, queue *queue.Queue, host string) {
	service.collection = collection
	service.userService = userService
	service.streamService = streamService
	service.ruleService = ruleService
	service.domainEmail = domainEmail
	service.activityService = activityService
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

func (service *Follower) Query(criteria exp.Expression, options ...option.Option) ([]model.Follower, error) {
	result := make([]model.Follower, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)
	return result, err
}

// Count returns the number of records that match the provided criteria
func (service *Follower) Count(criteria exp.Expression) (int64, error) {
	return service.collection.Count(notDeleted(criteria))
}

// List returns an iterator containing all of the Followers who match the provided criteria
func (service *Follower) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the Follower records that match the provided criteria
func (service *Follower) Range(criteria exp.Expression, options ...option.Option) iter.Seq[model.Follower] {

	return func(yield func(model.Follower) bool) {

		// Retrieve the Followers from the database
		followers, err := service.List(criteria, options...)

		// Soft fail.  Report, but do not crash.
		if err != nil {
			derp.Report(derp.Wrap(err, "service.Follower.Range", "Error creating iterator", criteria))
			return
		}

		defer followers.Close()

		// Yield each follower to the caller one-by-one
		for follower := model.NewFollower(); followers.Next(&follower); follower = model.NewFollower() {
			if !yield(follower) {
				return
			}
		}
	}
}

// Load retrieves an Follower from the database
func (service *Follower) Load(criteria exp.Expression, follower *model.Follower) error {

	if err := service.collection.Load(notDeleted(criteria), follower); err != nil {
		return derp.Wrap(err, "service.Follower.Load", "Error loading Follower", criteria)
	}

	return nil
}

// Save adds/updates an Follower in the database
func (service *Follower) Save(follower *model.Follower, note string) error {

	// Validate the value before saving
	if err := service.Schema().Validate(follower); err != nil {
		return derp.Wrap(err, "service.Follower.Save", "Error validating Follower", follower)
	}

	// Save the follower to the database
	if err := service.collection.Save(follower, note); err != nil {
		return derp.Wrap(err, "service.Follower.Save", "Error saving Follower", follower, note)
	}

	// Recalculate the follower count for this user
	go service.userService.CalcFollowerCount(follower.ParentID)

	return nil
}

// Delete removes an Follower from the database (virtual delete)
func (service *Follower) Delete(follower *model.Follower, note string) error {

	const location = "service.Follower.Delete"

	// Mark the Follower as deleted
	follower.StateID = model.FollowerStateDeleted

	// Delete this Follower
	if err := service.collection.Delete(follower, note); err != nil {
		return derp.Wrap(err, location, "Error deleting Follower", follower, note)
	}

	// Maybe delete the SearchQuery if it's no longer needed
	if follower.ParentType == model.FollowerTypeSearch {

		task := queue.NewTask(
			"DeleteEmptySearchQuery",
			mapof.Any{
				"host":          domain.NameOnly(service.host),
				"searchQueryID": follower.ParentID,
			},
			queue.WithPriority(200),
		)

		if err := service.queue.Publish(task); err != nil {
			return derp.Wrap(err, location, "Error publishing cleanup task", task)
		}
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

func (service *Follower) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *Follower) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewFollower()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Follower) ObjectSave(object data.Object, comment string) error {
	if follower, ok := object.(*model.Follower); ok {
		return service.Save(follower, comment)
	}
	return derp.InternalError("service.Follower.ObjectSave", "Invalid Object Type", object)
}

func (service *Follower) ObjectDelete(object data.Object, comment string) error {
	if follower, ok := object.(*model.Follower); ok {
		return service.Delete(follower, comment)
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

func (service *Follower) CountByParent(parentType string, parentID primitive.ObjectID) (int64, error) {
	criteria := exp.Equal("type", parentType).AndEqual("parentId", parentID)
	return service.Count(criteria)
}

func (service *Follower) LoadOrCreate(parentID primitive.ObjectID, actorID string) (model.Follower, error) {

	result := model.NewFollower()

	err := service.LoadByActor(parentID, actorID, &result)

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
	return result, derp.Wrap(err, "service.Follower.LoadOrCreate", "Error loading Follower", parentID, actorID)
}

// LoadByToken loads a follower using either the FollowerID (if an ObjectID is passed) or the Actor's ProfileURL
func (service *Follower) LoadByToken(parentID primitive.ObjectID, token string, follower *model.Follower) error {

	// If the token is an ObjectID then load the follower by FollowerID
	if followerID, err := primitive.ObjectIDFromHex(token); err == nil {
		criteria := exp.Equal("_id", followerID).AndEqual("parentId", parentID)
		return service.Load(criteria, follower)
	}

	// Otherwise, load the Follower by the Actor's ProfileURL
	criteria := exp.Equal("parentId", parentID).AndEqual("actor.profileUrl", token)
	return service.Load(criteria, follower)
}

// LoadBySecret loads a follower based on the FollowerID.  It confirms that the secret value matches
func (service *Follower) LoadBySecret(followerID primitive.ObjectID, secret string, follower *model.Follower) error {

	const location = "service.Follower.LoadBySecret"

	// RULE: The secret must not be empty.
	if secret == "" {
		return derp.ForbiddenError(location, "Secret cannot be empty", followerID)
	}

	// Load the Follower using the FollowerID
	criteria := exp.Equal("_id", followerID)
	if err := service.Load(criteria, follower); err != nil {
		return derp.Wrap(err, location, "Error loading follower", followerID)
	}

	// Verify that the secret matches
	if follower.Data.GetString("secret") != secret {
		return derp.ForbiddenError(location, "Invalid secret", followerID)
	}

	// Success
	return nil
}

// LoadByActor retrieves an Follower from the database by parentID and actorID
func (service *Follower) LoadByActor(parentID primitive.ObjectID, actorID string, follower *model.Follower) error {

	// RULE: The parentID must not be zero
	if parentID.IsZero() {
		return derp.ValidationError("ParentID cannot be zero", parentID)
	}

	// RULE: The actorID must not be empty
	if actorID == "" {
		return derp.ValidationError("ActorID cannot be empty", actorID)
	}

	criteria := exp.Equal("parentId", parentID).AndEqual("actor.profileUrl", actorID)
	return service.Load(criteria, follower)
}

// QueryByParent returns an slice containing all of the Followers of specific parentID
func (service *Follower) QueryByParent(parentType string, parentID primitive.ObjectID, options ...option.Option) ([]model.Follower, error) {

	criteria := exp.Equal("type", parentType).
		AndEqual("parentId", parentID)

	return service.Query(criteria, options...)
}

// RangeByUserID returns an iterator containing all of the Followers of a specific User
func (service *Follower) RangeByUserID(userID primitive.ObjectID) iter.Seq[model.Follower] {
	return service.Range(
		exp.Equal("parentId", userID).
			AndEqual("type", model.FollowerTypeUser),
	)
}

// RangeActivityPubByUserID returns an iterator containing all of the Followers of a specific User
func (service *Follower) RangeActivityPubByType(followerType string, userID primitive.ObjectID) iter.Seq[model.Follower] {
	return service.Range(
		exp.Equal("parentId", userID).
			AndEqual("type", followerType).
			AndEqual("method", model.FollowerMethodActivityPub),
	)
}

// RangeBySearch returns an iterator containing all of the Followers of a specific SearchQuery
func (service *Follower) RangeBySearch(searchQueryID primitive.ObjectID) iter.Seq[model.Follower] {
	return service.Range(
		exp.Equal("parentId", searchQueryID).
			AndEqual("type", model.FollowerTypeSearch),
	)
}

// RangeBySearch returns an iterator containing all of the Followers of a specific SearchQuery
func (service *Follower) RangeByGlobalSearch() iter.Seq[model.Follower] {

	// Special case for Domain search queries.
	return service.Range(
		exp.Equal("parentId", primitive.NilObjectID).
			AndEqual("type", model.FollowerTypeSearchDomain),
	)
}

// DeleteByUserID removes all Followers of a specific User
func (service *Follower) DeleteByUserID(userID primitive.ObjectID, comment string) error {

	const location = "service.Follower.DeleteByUserID"

	for follower := range service.RangeByUserID(userID) {

		if err := service.Delete(&follower, comment); err != nil {
			return derp.Wrap(err, location, "Error deleting follower", follower)
		}
	}

	return nil
}

// RangeFollowers returns a rangeFunc containing all of the Followers of specific parentID
func (service *Follower) RangeFollowers(parentType string, parentID primitive.ObjectID) iter.Seq[model.Follower] {

	return service.Range(
		exp.Equal("parentId", parentID).
			AndEqual("type", parentType),
	)
}

func (service *Follower) QueryByParentAndDate(parentType string, parentID primitive.ObjectID, method string, maxCreateDate int64, pageSize int) ([]model.Follower, error) {

	criteria := exp.Equal("type", parentType).
		AndEqual("parentId", parentID).
		AndEqual("method", method).
		AndLessThan("createDate", maxCreateDate)

	return service.Query(criteria, option.SortDesc("createDate"), option.MaxRows(int64(pageSize)))
}

/******************************************
 * WebSub Queries
 ******************************************/

// LoadByWebSub retrieves a follower based on the parentID and callback
func (service *Follower) LoadByWebSub(objectType string, parentID primitive.ObjectID, callback string, result *model.Follower) error {

	criteria := exp.
		Equal("type", objectType).
		AndEqual("parentId", parentID).
		AndEqual("method", model.FollowerMethodWebSub).
		AndEqual("actor.inboxUrl", callback)

	return service.Load(criteria, result)
}

// LoadOrCreateByWebSub finds a follower based on the parentID and callback.  If no follower is found, a new record is created.
func (service *Follower) LoadOrCreateByWebSub(objectType string, parentID primitive.ObjectID, callback string) (model.Follower, error) {

	// Try to load the Follower from the database
	result := model.NewFollower()
	err := service.LoadByWebSub(objectType, parentID, callback, &result)

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
	return result, derp.Wrap(err, "service.Follower.LoadByWebSub", "Error loading follower", parentID, callback)
}

func (service *Follower) LoadParentActor(follower *model.Follower) (model.PersonLink, error) {

	switch follower.ParentType {

	case model.FollowerTypeUser:

		user := model.NewUser()
		if err := service.userService.LoadByID(follower.ParentID, &user); err != nil {
			return model.PersonLink{}, derp.Wrap(err, "service.Follower.LoadParentActor", "Error loading parent user", follower)
		}

		return user.PersonLink(), nil

	case model.FollowerTypeStream:

		stream := model.NewStream()
		if err := service.streamService.LoadByID(follower.ParentID, &stream); err != nil {
			return model.PersonLink{}, derp.Wrap(err, "service.Follower.LoadParentActor", "Error loading parent stream", follower)
		}

		return stream.ActorLink(), nil

	}

	return model.PersonLink{}, derp.InternalError("service.Follower.LoadParentActor", "Invalid parentType", follower)
}

/******************************************
 * ActivityPub Queries
 ******************************************/

// IsActivityPubFollower searches
func (service *Follower) IsActivityPubFollower(parentType string, parentID primitive.ObjectID, followerURL string) bool {
	result := model.NewFollower()
	err := service.LoadByActivityPubFollower(parentType, parentID, followerURL, &result)
	return err == nil
}

// ListActivityPub returns an iterator containing all of the Followers of specific parentID
func (service *Follower) ListActivityPub(parentID primitive.ObjectID, options ...option.Option) (data.Iterator, error) {

	criteria := exp.
		Equal("parentId", parentID).
		AndEqual("method", model.FollowerMethodActivityPub)

	return service.List(criteria, options...)
}

func (service *Follower) NewActivityPubFollower(parentType string, parentID primitive.ObjectID, actor streams.Document, follower *model.Follower) error {

	// Try to find an existing follower record
	if err := service.LoadByActor(parentID, actor.ID(), follower); err != nil {
		if !derp.IsNotFound(err) {
			return derp.Wrap(err, "handler.activityPub_HandleRequest_Follow", "Error loading existing follower", actor)
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
	if err := service.Save(follower, "New Follower via ActivityPub"); err != nil {
		return derp.Wrap(err, "handler.activityPub_HandleRequest_Follow", "Error saving new follower", follower)
	}

	// Salút!
	return nil
}

func (service *Follower) LoadByActivityPubFollower(parentType string, parentID primitive.ObjectID, followerURL string, follower *model.Follower) error {

	criteria := exp.
		Equal("type", parentType).
		AndEqual("parentId", parentID).
		AndEqual("method", model.FollowerMethodActivityPub).
		AndEqual("actor.profileUrl", followerURL)

	return service.Load(criteria, follower)
}

// RemoteActor returns the ActivityStream document for a remote Actor for a specific Follower
func (service *Follower) RemoteActor(follower *model.Follower) (streams.Document, error) {

	// RULE: Guarantee that the Follower is using ActivityPub for updates
	if follower.Method != model.FollowerMethodActivityPub {
		return streams.NilDocument(), derp.InternalError("service.Follower.RemoteActor", "Follower must use ActivityPub method", follower)
	}

	// Return the remote Actor's profile document
	return service.activityService.Load(follower.Actor.ProfileURL, sherlock.AsActor())
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

func (service *Follower) LoadPendingEmailFollower(followerID primitive.ObjectID, secret string, follower *model.Follower) error {

	criteria := exp.
		Equal("_id", followerID).
		AndEqual("method", model.FollowerMethodEmail).
		AndEqual("stateId", model.FollowerStatePending).
		AndEqual("data.secret", secret)

	return service.Load(criteria, follower)
}

/******************************************
 * Email Methods
 ******************************************/

// SendFollowConfirmation sends an email to an email-type follower with a link to confirm their subscription.
// Subscriptions are not "ACTIVE" until confirmed.
func (service *Follower) SendFollowConfirmation(follower *model.Follower) error {

	// RULE: This method only applies to EMAIL-type Followers
	if follower.Method != model.FollowerMethodEmail {
		return derp.InternalError("service.Follower.SendFollowConfirmation", "Follower must use Email method", follower)
	}

	actor, err := service.LoadParentActor(follower)

	if err != nil {
		return derp.Wrap(err, "service.Follower.SendFollowConfirmation", "Error loading parent actor", follower)
	}

	if err := service.domainEmail.SendFollowerConfirmation(actor, follower); err != nil {
		return derp.Wrap(err, "service.Follower.SendFollowConfirmation", "Error sending follow confirmation email", follower)
	}

	return nil
}
