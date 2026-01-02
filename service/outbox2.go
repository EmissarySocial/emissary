package service

import (
	"iter"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Outbox2 is a new outbox service for Users, Streams, and Searches.
// It is being built alongside the existing Outbox service, which will be
// removed once this new service is fully functional.
type Outbox2 struct {
	followerService *Follower
	locatorService  *Locator
	queue           *queue.Queue
	host            string
}

// NewOutbox2 returns a fully populated Outbox2 service
func NewOutbox2() Outbox2 {
	return Outbox2{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Outbox2) Refresh(factory *Factory) {
	service.followerService = factory.Follower()
	service.locatorService = factory.Locator()
	service.queue = factory.Queue()
	service.host = factory.Hostname()
}

// Close stops any background processes controlled by this service
func (service *Outbox2) Close() {
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *Outbox2) collection(session data.Session) data.Collection {
	return session.Collection("Outbox2")
}

// New creates a newly initialized Outbox that is ready to use
func (service *Outbox2) New() model.Activity {
	return model.NewActivity()
}

// Count returns the number of records that match the provided criteria
func (service *Outbox2) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns a slice containing all of the Activities that match the provided criteria
func (service *Outbox2) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.Activity, error) {
	result := make([]model.Activity, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)

	return result, err
}

// Range returns a Go 1.23 RangeFunc that iterates over the Activity records that match the provided criteria
func (service *Outbox2) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.Activity], error) {

	iter, err := service.collection(session).Iterator(notDeleted(criteria), options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Outbox2.Range", "Unable to create iterator", criteria)
	}

	return RangeFunc(iter, model.NewActivity), nil
}

// Load retrieves an Outbox from the database
func (service *Outbox2) Load(session data.Session, criteria exp.Expression, result *model.Activity) error {

	if err := service.collection(session).Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Outbox2.Load", "Unable to load Outbox activity", criteria)
	}

	return nil
}

// Save adds/updates an Outbox in the database
func (service *Outbox2) Save(session data.Session, activity *model.Activity, note string) error {

	const location = "service.Outbox2.Save"

	// Calculate the ActivityURL for this message
	activity.URL = service.calcActivityURL(activity)

	// Calculate the list of unique recipients
	activity.CalcRecipients()

	// Save the value to the database
	if err := service.collection(session).Save(activity, note); err != nil {
		return derp.Wrap(err, location, "Unable to save Outbox activity", activity, note)
	}

	return nil
}

// Delete removes an Outbox from the database (virtual delete)
func (service *Outbox2) Delete(session data.Session, activity *model.Activity, note string) error {

	const location = "service.Outbox2.Delete"

	// Delete the message from the outbox
	criteria := exp.Equal("_id", activity.ActivityID)

	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Unable to delete Outbox activity", activity, note)
	}

	return nil
}

/******************************************
 * Generic Data Methods
 ******************************************/

func (service *Outbox2) Schema() schema.Schema {
	return schema.New(model.ActivitySchema())
}

/******************************************
 * Custom Query Methods
 ******************************************/

// RangeByUser returns a Go 1.23 RangeFunc that iterates over the Activity records for a specific User
func (service *Outbox2) RangeByUser(session data.Session, userID primitive.ObjectID, options ...option.Option) (iter.Seq[model.Activity], error) {
	return service.RangeByActor(session, model.ActorTypeUser, userID, options...)
}

// RangeByStream returns a Go 1.23 RangeFunc that iterates over the Activity records for a specific Stream / Content Actor
func (service *Outbox2) RangeByStream(session data.Session, streamID primitive.ObjectID, options ...option.Option) (iter.Seq[model.Activity], error) {
	return service.RangeByActor(session, model.ActorTypeStream, streamID, options...)
}

// RangeBySearchQuery returns a Go 1.23 RangeFunc that iterates over the Activity records for a specific SearchQuery
func (service *Outbox2) RangeBySearchQuery(session data.Session, searchQueryID primitive.ObjectID, options ...option.Option) (iter.Seq[model.Activity], error) {
	return service.RangeByActor(session, model.ActorTypeSearchQuery, searchQueryID, options...)
}

// RangeBySearchDomain returns a Go 1.23 RangeFunc that iterates over the Activity records for the gloabl @search actor
func (service *Outbox2) RangeBySearchDomain(session data.Session, options ...option.Option) (iter.Seq[model.Activity], error) {
	return service.RangeByActor(session, model.ActorTypeSearchDomain, primitive.NilObjectID, options...)
}

// RangeByApplication returns a Go 1.23 RangeFunc that iterates over the Activity records for the gloabl @application actor
func (service *Outbox2) RangeByApplication(session data.Session, options ...option.Option) (iter.Seq[model.Activity], error) {
	return service.RangeByActor(session, model.ActorTypeApplication, primitive.NilObjectID, options...)
}

// RangeByActor returns a Go 1.23 RangeFunc that iterates over the Activity records for a specific parent (actorType, actorID)
func (service *Outbox2) RangeByActor(session data.Session, actorType string, actorID primitive.ObjectID, options ...option.Option) (iter.Seq[model.Activity], error) {
	criteria := exp.Equal("actorType", actorType).
		AndEqual("actorId", actorID)

	return service.Range(session, criteria, options...)
}

func (service *Outbox2) LoadByID(session data.Session, actorType string, actorID primitive.ObjectID, activityID primitive.ObjectID, activity *model.Activity) error {
	criteria := exp.Equal("_id", activityID).
		AndEqual("actorId", actorID).
		AndEqual("actorType", actorType)

	return service.Load(session, criteria, activity)
}

func (service *Outbox2) DeleteByActor(session data.Session, actorType string, actorID primitive.ObjectID) error {

	const location = "service.Outbox2.DeleteByParent"

	// Get all messages in this Outbox
	rangeFunc, err := service.RangeByActor(session, actorType, actorID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to query Outbox activities", actorType, actorID)
	}

	for message := range rangeFunc {
		if err := service.Delete(session, &message, "Deleted"); err != nil {
			derp.Report(derp.Wrap(err, location, "Unable to delete Outbox activity", message))
		}
	}

	return nil
}

func (service *Outbox2) calcActivityURL(activity *model.Activity) string {

	switch activity.ActorType {

	case model.ActorTypeApplication:
		return service.host + "/@application/pub/outbox/" + activity.ActivityID.Hex()

	case model.ActorTypeSearchDomain:
		return service.host + "/@search/pub/outbox/" + activity.ActivityID.Hex()

	case model.ActorTypeSearchQuery:
		return service.host + "/@search_" + activity.ActorID.Hex() + "/pub/outbox/" + activity.ActivityID.Hex()

	case model.ActorTypeStream:
		return service.host + "/" + activity.ActorID.Hex() + "/pub/outbox/" + activity.ActivityID.Hex()

	case model.ActorTypeUser:
		return service.host + "/@" + activity.ActorID.Hex() + "/pub/outbox/" + activity.ActivityID.Hex()

	default:
		return ""
	}
}

/******************************************
 * Custom Actions
 ******************************************/

// AddUserActivity creates a new Outbox activity for the specified user
func (service *Outbox2) AddUserActivity(session data.Session, userID primitive.ObjectID, document streams.Document) error {

	const location = "service.Outbox2.AddUserActivity"

	activity := model.NewActivity()
	activity.ActorType = model.ActorTypeUser
	activity.ActorID = userID
	activity.Object = document.Map()

	// TODO: Rules here for public messages that we can map into streams

	if err := service.Save(session, &activity, "Created Outbox activity"); err != nil {
		return derp.Wrap(err, location, "Unable to save Outbox activity")
	}

	/* RESTORE THIS ONCE WE HAVE THE FACTORY SORTED

	// Get a service for the "Locator" interface
	sendLocator := context.factory.SendLocator(session)

	// Send ActivityPub notifications to participants
	sender := sender.New(sendLocator, service.queue.Queue())

	if err := sender.Send(activity.Map()); err != nil {
		return derp.Wrap(err, location, "Unable to send activity")
	}
	*/

	return nil
}

// Add creates a new Outbox activity for any type of actor actor
func (service *Outbox2) Add(session data.Session, actorType string, actorID primitive.ObjectID, object mapof.Any) error {

	activity := model.NewActivity()
	activity.ActorType = actorType
	activity.ActorID = actorID
	activity.Object = object

	return service.Save(session, &activity, "Created Outbox activity")
}
