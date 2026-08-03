package service

import (
	"iter"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/sender"
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
	inboxService    *Inbox
	activityService *ActivityStream
	locator         *Locator
	getSendLocator  func(data.Session) SendLocator
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
	service.inboxService = factory.Inbox()
	service.activityService = factory.ActivityStream()
	service.locator = factory.Locator()
	service.queue = factory.Queue()
	service.host = factory.Host()
	service.getSendLocator = factory.SendLocator
}

// Close stops any background processes controlled by this service
func (service *Outbox2) Close() {
	// No background processes to stop
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *Outbox2) collection(session data.Session) data.Collection {
	return session.Collection("Outbox2")
}

// New creates a newly initialized Outbox that is ready to use
func (service *Outbox2) New() model.OutboxItem {
	return model.NewOutboxItem()
}

// Count returns the number of records that match the provided criteria
func (service *Outbox2) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns a slice containing all of the Activities that match the provided criteria
func (service *Outbox2) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.OutboxItem, error) {
	result := make([]model.OutboxItem, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)

	return result, err
}

// Range returns a Go 1.23 RangeFunc that iterates over the Activity records that match the provided criteria
func (service *Outbox2) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.OutboxItem], error) {

	iter, err := service.collection(session).Iterator(notDeleted(criteria), options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Outbox2.Range", "Creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewOutboxItem), nil
}

// Load retrieves an Outbox from the database
func (service *Outbox2) Load(session data.Session, criteria exp.Expression, result *model.OutboxItem) error {

	if err := service.collection(session).Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Outbox2.Load", "Loading Outbox activity", criteria)
	}

	return nil
}

// Save adds/updates an Outbox in the database
func (service *Outbox2) Save(session data.Session, item *model.OutboxItem, note string) error {

	const location = "service.Outbox2.Save"

	if item.IsNew() {

		// Calculate the ActivityURL for this message and the user who sent it.
		item.URL = service.locator.ActivityURL(item.ActorType, item.ActorID, item.ActivityID)

		// Calculate the list of unique recipients
		item.CalcRecipients()

		// If the actor is also a recipient, then we can put it straight into their inbox
		if actorURL := service.locator.UserURL(item.ActorID); item.Recipients.Contains(actorURL) {

			asActivity := streams.NewDocument(item.Activity)

			inboxActivity := model.NewInboxActivity()
			inboxActivity.ActivityID = asActivity.ID()
			inboxActivity.UserID = item.ActorID
			inboxActivity.ActorID = asActivity.ActorID()
			inboxActivity.Context = asActivity.Context()
			inboxActivity.ActivityType = asActivity.Type()
			inboxActivity.ObjectType = asActivity.Object().Type()
			inboxActivity.ObjectID = asActivity.Object().ID()
			inboxActivity.MediaType = asActivity.Object().MediaType()
			inboxActivity.ReceivedDate = time.Now().UnixMilli()
			inboxActivity.RawActivity = item.Activity
			inboxActivity.IsPublic = asActivity.IsPublic()
			inboxActivity.PublishedDate = asActivity.Published().UnixMilli()

			if err := service.inboxService.Save(session, &inboxActivity, "Saved directly from outbox"); err != nil {
				return derp.Wrap(err, location, "Saving item to inbox", inboxActivity)
			}
		}

		// Send ActivityPub notifications to recipient(s) POST-COMMIT. Enqueuing the
		// fan-out as a task (released only after this transaction commits) keeps the
		// signed HTTP sends off the request's open transaction. The old synchronous
		// sender.Send delivered to local inboxes before this txn committed, so a
		// receiver's own (gated) task could 404 on rows not yet visible on its
		// separate majority-read session. The Outbox:SendToAllRecipients consumer
		// rebuilds the Sender with AllowPrivateIPs threaded via WithSender
		// (consumer/wrappers.go). See POST-COMMIT-TASKS-DESIGN.md / POST-COMMIT-FEDERATION.md F0.
		postcommit.Publish(session, service.queue, sender.OutboxSendToAllRecipients, item.Activity)
	}

	// Save the value to the database
	if err := service.collection(session).Save(item, note); err != nil {
		return derp.Wrap(err, location, "Saving Outbox activity", item, note)
	}

	return nil
}

// Send delivers an ActivityPub activity through the sender pipeline WITHOUT writing an
// outbox record. Use it for transient, idempotent activities (profile Updates and the like)
// that don't belong in an actor's outbox collection. The activity must carry its own actor
// and addressing (to/cc); delivery is enqueued post-commit, so a rolled-back transaction
// sends nothing. See PROFILE-UPDATE-FEDERATION.md D-1.
func (service *Outbox2) Send(session data.Session, activity mapof.Any) error {

	const location = "service.Outbox2.Send"

	// RULE: The activity must identify its actor -- the send consumer signs with this actor's key
	if streams.NewDocument(activity).ActorID() == "" {
		return derp.Internal(location, "Activity must include an actor", activity)
	}

	// Enqueue delivery to all addressed recipients (released only after the transaction commits)
	postcommit.Publish(session, service.queue, sender.OutboxSendToAllRecipients, activity)

	// boom.
	return nil
}

// Delete removes an Outbox from the database (virtual delete)
func (service *Outbox2) Delete(session data.Session, item *model.OutboxItem, note string) error {

	const location = "service.Outbox2.Delete"

	// Delete the message from the outbox
	criteria := exp.Equal("_id", item.ActivityID)

	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Deleting Outbox activity", item, note)
	}

	return nil
}

/******************************************
 * Generic Data Methods
 ******************************************/

func (service *Outbox2) Schema() schema.Schema {
	return schema.New(model.OutboxItemSchema())
}

/******************************************
 * Custom Query Methods
 ******************************************/

// RangeByUser returns a Go 1.23 RangeFunc that iterates over the Activity records for a specific User
func (service *Outbox2) RangeByUser(session data.Session, userID primitive.ObjectID, options ...option.Option) (iter.Seq[model.OutboxItem], error) {
	return service.RangeByActor(session, model.ActorTypeUser, userID, options...)
}

// RangeByStream returns a Go 1.23 RangeFunc that iterates over the Activity records for a specific Stream / Content Actor
func (service *Outbox2) RangeByStream(session data.Session, streamID primitive.ObjectID, options ...option.Option) (iter.Seq[model.OutboxItem], error) {
	return service.RangeByActor(session, model.ActorTypeStream, streamID, options...)
}

// RangeBySearchQuery returns a Go 1.23 RangeFunc that iterates over the Activity records for a specific SearchQuery
func (service *Outbox2) RangeBySearchQuery(session data.Session, searchQueryID primitive.ObjectID, options ...option.Option) (iter.Seq[model.OutboxItem], error) {
	return service.RangeByActor(session, model.ActorTypeSearchQuery, searchQueryID, options...)
}

// RangeBySearchDomain returns a Go 1.23 RangeFunc that iterates over the Activity records for the gloabl @search actor
func (service *Outbox2) RangeBySearchDomain(session data.Session, options ...option.Option) (iter.Seq[model.OutboxItem], error) {
	return service.RangeByActor(session, model.ActorTypeSearchDomain, primitive.NilObjectID, options...)
}

// RangeByApplication returns a Go 1.23 RangeFunc that iterates over the Activity records for the gloabl @application actor
func (service *Outbox2) RangeByApplication(session data.Session, options ...option.Option) (iter.Seq[model.OutboxItem], error) {
	return service.RangeByActor(session, model.ActorTypeApplication, primitive.NilObjectID, options...)
}

// RangeByActor returns a Go 1.23 RangeFunc that iterates over the Activity records for a specific parent (actorType, actorID)
func (service *Outbox2) RangeByActor(session data.Session, actorType string, actorID primitive.ObjectID, options ...option.Option) (iter.Seq[model.OutboxItem], error) {
	criteria := exp.Equal("actorType", actorType).
		AndEqual("actorId", actorID)

	return service.Range(session, criteria, options...)
}

func (service *Outbox2) LoadByID(session data.Session, actorType string, actorID primitive.ObjectID, activityID primitive.ObjectID, item *model.OutboxItem) error {
	criteria := exp.Equal("_id", activityID).
		AndEqual("actorId", actorID).
		AndEqual("actorType", actorType)

	return service.Load(session, criteria, item)
}

func (service *Outbox2) DeleteByActor(session data.Session, actorType string, actorID primitive.ObjectID) error {

	const location = "service.Outbox2.DeleteByParent"

	// Get all messages in this Outbox
	rangeFunc, err := service.RangeByActor(session, actorType, actorID)

	if err != nil {
		return derp.Wrap(err, location, "Querying Outbox activities", actorType, actorID)
	}

	for message := range rangeFunc {
		if err := service.Delete(session, &message, "Deleted"); err != nil {
			derp.Report(derp.Wrap(err, location, "Deleting Outbox activity", message))
		}
	}

	return nil
}
