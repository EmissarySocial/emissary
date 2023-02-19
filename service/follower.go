package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Follower defines a service that tracks the (possibly external) accounts that are followers of an internal User

type Follower struct {
	collection  data.Collection
	userService *User
	host        string
}

// NewFollower returns a fully initialized Follower service
func NewFollower(collection data.Collection, userService *User, host string) Follower {
	service := Follower{
		userService: userService,
		host:        host,
	}

	service.Refresh(collection)
	return service
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Follower) Refresh(collection data.Collection) {
	service.collection = collection
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

// List returns an iterator containing all of the Followers who match the provided criteria
func (service *Follower) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(notDeleted(criteria), options...)
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

	// Clean the value before saving
	if err := service.Schema().Clean(follower); err != nil {
		return derp.Wrap(err, "service.Follower.Save", "Error cleaning Follower", follower)
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

	// Delete this Follower
	if err := service.collection.Delete(follower, note); err != nil {
		return derp.Wrap(err, "service.Follower.Delete", "Error deleting Follower", follower, note)
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

// New returns a fully initialized model.Group as a data.Object.
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

func (service *Follower) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
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
	return derp.NewInternalError("service.Follower.ObjectSave", "Invalid Object Type", object)
}

func (service *Follower) ObjectDelete(object data.Object, comment string) error {
	if follower, ok := object.(*model.Follower); ok {
		return service.Delete(follower, comment)
	}
	return derp.NewInternalError("service.Follower.ObjectDelete", "Invalid Object Type", object)
}

func (service *Follower) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.NewUnauthorizedError("service.Follower", "Not Authorized")
}

func (service *Follower) Schema() schema.Schema {
	return schema.New(model.FollowerSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *Follower) LoadOrCreate(parentID primitive.ObjectID, actorID string) (model.Follower, error) {

	result := model.NewFollower()

	err := service.LoadByActor(parentID, actorID, &result)

	// No error means the record was found
	if err == nil {
		return result, nil
	}

	// NotFound error means we should create a new record
	if derp.NotFound(err) {
		result.ParentID = parentID
		result.Actor.ProfileURL = actorID
		return result, nil
	}

	// Other error is bad.  Return the error
	return result, derp.Wrap(err, "service.Follower.LoadOrCreate", "Error loading Follower", parentID, actorID)
}

func (service *Follower) LoadByToken(parentID primitive.ObjectID, token string, follower *model.Follower) error {
	followerID, err := primitive.ObjectIDFromHex(token)

	if err == nil {
		return derp.Wrap(err, "service.Follower.LoadByToken", "Error converting token to ObjectID", token)
	}

	criteria := exp.Equal("_id", followerID).AndEqual("parentId", parentID)
	return service.Load(criteria, follower)
}

func (service *Follower) LoadByActor(parentID primitive.ObjectID, actorID string, follower *model.Follower) error {

	criteria := exp.Equal("parentId", parentID).AndEqual("actor.profileUrl", actorID)
	return service.Load(criteria, follower)
}

func (service *Follower) QueryAllURLs(criteria exp.Expression) ([]string, error) {

	result := make([]string, 0)
	fields := option.Fields("actor.profileUrl")

	if err := service.collection.Query(&result, notDeleted(criteria), fields); err != nil {
		return result, derp.Wrap(err, "service.Follower.QueryFollowerURLs", "Error querying follower URLs", criteria)
	}

	return result, nil
}

/******************************************
 * WebSub Queries
 ******************************************/

// ListWebSub returns an iterator containing all of the Followers of specific parentID
func (service *Follower) ListWebSub(parentID primitive.ObjectID, options ...option.Option) (data.Iterator, error) {

	criteria := exp.
		Equal("parentId", parentID).
		AndEqual("method", model.FollowMethodWebSub)

	return service.List(criteria, options...)
}

// LoadByWebSub retrieves a follower based on the parentID and callback
func (service *Follower) LoadByWebSub(objectType string, parentID primitive.ObjectID, callback string, result *model.Follower) error {

	criteria := exp.
		Equal("type", objectType).
		AndEqual("parentId", parentID).
		AndEqual("method", model.FollowMethodWebSub).
		AndEqual("actor.inboxUrl", callback)

	return service.Load(criteria, result)
}

// LoadByWebSubUnique finds a follower based on the parentID and callback.  If no follower is found, a new record is created.
func (service *Follower) LoadByWebSubUnique(objectType string, parentID primitive.ObjectID, callback string) (model.Follower, error) {

	// Try to load the Follower from the database
	result := model.NewFollower()
	err := service.LoadByWebSub(objectType, parentID, callback, &result)

	// If EXISTS, then we've found it.
	if err == nil {
		return result, nil
	}

	// If NOT EXISTS, then create a new one
	if derp.NotFound(err) {
		result.ParentID = parentID
		result.Type = objectType
		result.Method = model.FollowMethodWebSub
		result.Actor.InboxURL = callback
		return result, nil
	}

	// If REAL ERROR, then derp
	return result, derp.Wrap(err, "service.Follower.LoadByWebSub", "Error loading follower", parentID, callback)
}

/******************************************
 * ActivityPub Queries
 ******************************************/

// ListActivityPub returns an iterator containing all of the Followers of specific parentID
func (service *Follower) ListActivityPub(parentID primitive.ObjectID, options ...option.Option) (data.Iterator, error) {

	criteria := exp.
		Equal("parentId", parentID).
		AndEqual("method", model.FollowMethodActivityPub)

	return service.List(criteria, options...)
}

func (service *Follower) NewActivityPubFollower(user *model.User, actor streams.Document) error {

	// Create a new Follower record
	follower := model.NewFollower()

	// Try to find an existing follower record
	if err := service.LoadByActor(user.UserID, actor.ID(), &follower); err != nil {
		if !derp.NotFound(err) {
			return derp.Wrap(err, "handler.activityPub_HandleRequest_Follow", "Error loading existing follower", actor)
		}
	}

	// Set/Update follower data from the activity
	follower.Type = model.FollowerTypeUser
	follower.Method = model.FollowMethodActivityPub
	follower.ParentID = user.UserID

	follower.Actor = model.PersonLink{
		ProfileURL:   actor.ID(),
		Name:         actor.Name(),
		ImageURL:     first.String(actor.IconURL(), actor.ImageURL()),
		InboxURL:     actor.Get("inbox").AsString(),
		EmailAddress: actor.Get("email").AsString(),
	}

	// Try to save the new follower to the database
	if err := service.Save(&follower, "New Follower via ActivityPub"); err != nil {
		return derp.Wrap(err, "handler.activityPub_HandleRequest_Follow", "Error saving new follower", follower)
	}

	// Salút!
	return nil
}

func (service *Follower) LoadByActivityPubFollower(parentID primitive.ObjectID, followerURL string, follower *model.Follower) error {

	criteria := exp.
		Equal("parentId", parentID).
		AndEqual("method", model.FollowMethodActivityPub).
		AndEqual("actor.profileUrl", followerURL)

	return service.Load(criteria, follower)
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
