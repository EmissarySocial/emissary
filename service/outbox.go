package service

import (
	"sync"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queue"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Outbox manages all Outbox records for a User.  This includes Outbox and Outbox
type Outbox struct {
	collection             data.Collection
	activityStreamsService *ActivityStreams
	streamService          *Stream
	followerService        *Follower
	userService            *User
	counter                int
	lock                   *sync.Mutex
	queue                  *queue.Queue
}

// NewOutbox returns a fully populated Outbox service
func NewOutbox() Outbox {
	return Outbox{
		lock: &sync.Mutex{},
	}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Outbox) Refresh(collection data.Collection, streamService *Stream, activityStreamsService *ActivityStreams, followerService *Follower, userService *User, queue *queue.Queue) {
	service.collection = collection
	service.streamService = streamService
	service.activityStreamsService = activityStreamsService
	service.followerService = followerService
	service.userService = userService
	service.queue = queue
}

// Close stops any background processes controlled by this service
func (service *Outbox) Close() {
}

/******************************************
 * Common Data Methods
 ******************************************/

// New creates a newly initialized Outbox that is ready to use
func (service *Outbox) New() model.OutboxMessage {
	return model.NewOutboxMessage()
}

// Query returns a slice containing all of the Activities that match the provided criteria
func (service *Outbox) Query(criteria exp.Expression, options ...option.Option) ([]model.OutboxMessage, error) {
	result := make([]model.OutboxMessage, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)

	return result, err
}

// List returns an iterator containing all of the Activities that match the provided criteria
func (service *Outbox) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Load retrieves an Outbox from the database
func (service *Outbox) Load(criteria exp.Expression, result *model.OutboxMessage) error {

	if err := service.collection.Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Outbox", "Error loading Outbox Message", criteria)
	}

	return nil
}

// Save adds/updates an Outbox in the database
func (service *Outbox) Save(outboxMessage *model.OutboxMessage, note string) error {

	// Calculate the rank for this outboxMessage, using the number of outboxMessages with an identical PublishDate
	if err := service.CalculateRank(outboxMessage); err != nil {
		return derp.Wrap(err, "service.Outbox.Save", "Error calculating rank", outboxMessage)
	}

	// Save the value to the database
	if err := service.collection.Save(outboxMessage, note); err != nil {
		return derp.Wrap(err, "service.Outbox", "Error saving Outbox", outboxMessage, note)
	}

	// If this message has a valid URL, then try cache it into the activitystream service.
	// nolint:errcheck
	go service.activityStreamsService.LoadDocument(outboxMessage.URL, mapof.NewAny())

	return nil
}

// Delete removes an Outbox from the database (virtual delete)
func (service *Outbox) Delete(outboxMessage *model.OutboxMessage, note string) error {

	// Delete Outbox record last.
	criteria := exp.Equal("_id", outboxMessage.OutboxMessageID)

	if err := service.collection.HardDelete(criteria); err != nil {
		return derp.Wrap(err, "service.Outbox", "Error deleting Outbox", outboxMessage, note)
	}

	if err := service.activityStreamsService.DeleteDocumentByURL(outboxMessage.URL); err != nil {
		return derp.Wrap(err, "service.Outbox", "Error deleting ActivityStream", outboxMessage, note)
	}

	return nil
}

// DeleteMany removes all child streams from the provided stream (virtual delete)
func (service *Outbox) DeleteMany(criteria exp.Expression, note string) error {

	it, err := service.List(criteria)

	if err != nil {
		return derp.Wrap(err, "service.Message.Delete", "Error listing streams to delete", criteria)
	}

	outboxMessage := model.NewOutboxMessage()

	for it.Next(&outboxMessage) {
		if err := service.Delete(&outboxMessage, note); err != nil {
			return derp.Wrap(err, "service.Message.Delete", "Error deleting outboxMessage", outboxMessage)
		}
		outboxMessage = model.NewOutboxMessage()
	}

	return nil
}

/******************************************
 * Custom Query Methods
 ******************************************/

func (service *Outbox) LoadOrCreate(userID primitive.ObjectID, url string) (model.OutboxMessage, error) {

	result := model.NewOutboxMessage()

	err := service.LoadByURL(userID, url, &result)

	if err == nil {
		return result, nil
	}

	if derp.NotFound(err) {
		result.UserID = userID
		result.URL = url
		return result, nil
	}

	return result, derp.Wrap(err, "service.Outbox", "Error loading Outbox", userID, url)
}

func (service *Outbox) QueryByUserID(userID primitive.ObjectID, criteria exp.Expression, options ...option.Option) ([]model.OutboxMessage, error) {
	criteria = exp.Equal("userId", userID).And(criteria)
	return service.Query(criteria, options...)
}

func (service *Outbox) QueryByUserAndDate(userID primitive.ObjectID, maxDate int64, maxRows int) ([]model.OutboxMessageSummary, error) {

	criteria := exp.Equal("userId", userID).
		And(exp.LessThan("createDate", maxDate))

	options := []option.Option{
		option.Fields(model.OutboxMessageSummaryFields()...),
		option.SortDesc("createDate"),
		option.MaxRows(int64(maxRows)),
	}

	result := make([]model.OutboxMessageSummary, 0, maxRows)

	if err := service.collection.Query(&result, criteria, options...); err != nil {
		return nil, derp.Wrap(err, "service.Outbox", "Error querying outbox", userID, maxDate)
	}

	return result, nil
}

func (service *Outbox) LoadByURL(userID primitive.ObjectID, url string, result *model.OutboxMessage) error {
	criteria := exp.Equal("userId", userID).
		AndEqual("url", url)

	return service.Load(criteria, result)
}

/******************************************
 * Custom Behaviors
 ******************************************/

func (service *Outbox) CalculateRank(outboxMessage *model.OutboxMessage) error {

	counter := service.getNextCounter()

	outboxMessage.Rank = (outboxMessage.CreateDate * 1000) + counter
	return nil
}

// getNextCounter safely increments the service counter (MOD 1000)
func (service *Outbox) getNextCounter() int64 {
	service.lock.Lock()
	defer service.lock.Unlock()

	service.counter = (service.counter + 1) % 1000
	return int64(service.counter)
}
