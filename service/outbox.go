package service

import (
	"sync"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Outbox manages all Outbox records for a User.  This includes Outbox and Outbox
type Outbox struct {
	collection      data.Collection
	activityService *ActivityStream
	streamService   *Stream
	followerService *Follower
	templateService *Template
	userService     *User
	lock            *sync.Mutex
	queue           queue.Queue
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
func (service *Outbox) Refresh(collection data.Collection, streamService *Stream, activityService *ActivityStream, followerService *Follower, templateService *Template, userService *User, queue queue.Queue) {
	service.collection = collection
	service.streamService = streamService
	service.activityService = activityService
	service.followerService = followerService
	service.templateService = templateService
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
		return derp.Wrap(err, "service.Outbox.Load", "Error loading Outbox Message", criteria)
	}

	return nil
}

// Save adds/updates an Outbox in the database
func (service *Outbox) Save(outboxMessage *model.OutboxMessage, note string) error {

	const location = "service.Outbox.Save"

	// Save the value to the database
	if err := service.collection.Save(outboxMessage, note); err != nil {
		return derp.Wrap(err, location, "Error saving Outbox", outboxMessage, note)
	}

	// If this message has a valid URL, then try cache it into the activitystream service.
	// nolint:errcheck
	go service.activityService.Load(outboxMessage.URL)

	return nil
}

// Delete removes an Outbox from the database (virtual delete)
func (service *Outbox) Delete(outboxMessage *model.OutboxMessage, note string) error {

	const location = "service.Outbox.Delete"

	// Delete the message from the outbox
	criteria := exp.Equal("_id", outboxMessage.OutboxMessageID)

	if err := service.collection.HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Error deleting Outbox", outboxMessage, note)
	}

	// Delete the document from the cache
	if err := service.activityService.Delete(outboxMessage.URL); err != nil {
		return derp.Wrap(err, location, "Error deleting ActivityStream", outboxMessage, note)
	}

	return nil
}

/******************************************
 * Custom Query Methods
 ******************************************/

func (service *Outbox) LoadOrCreate(parentType string, parentID primitive.ObjectID, url string) (model.OutboxMessage, error) {

	result := model.NewOutboxMessage()

	err := service.LoadByURL(parentType, parentID, url, &result)

	if err == nil {
		return result, nil
	}

	if derp.NotFound(err) {
		result.ParentID = parentID
		result.URL = url
		return result, nil
	}

	return result, derp.Wrap(err, "service.Outbox.LoadOrCreate", "Error loading Outbox", parentID, url)
}

func (service *Outbox) QueryByParentID(parentType string, parentID primitive.ObjectID, criteria exp.Expression, options ...option.Option) ([]model.OutboxMessage, error) {
	criteria = exp.Equal("parentType", parentType).
		AndEqual("parentId", parentID).
		And(criteria)

	return service.Query(criteria, options...)
}

func (service *Outbox) QueryByUserAndDate(parentType string, parentID primitive.ObjectID, maxDate int64, maxRows int) ([]model.OutboxMessageSummary, error) {

	criteria := exp.Equal("parentType", parentType).
		AndEqual("parentId", parentID).
		And(exp.LessThan("createDate", maxDate))

	options := []option.Option{
		option.Fields(model.OutboxMessageSummaryFields()...),
		option.SortDesc("createDate"),
		option.MaxRows(int64(maxRows)),
	}

	result := make([]model.OutboxMessageSummary, 0, maxRows)

	if err := service.collection.Query(&result, criteria, options...); err != nil {
		return nil, derp.Wrap(err, "service.Outbox.QueryByUserAndDate", "Error querying outbox", parentID, maxDate)
	}

	return result, nil
}

func (service *Outbox) LoadByURL(parentType string, parentID primitive.ObjectID, url string, result *model.OutboxMessage) error {

	criteria := exp.Equal("parentType", parentType).
		AndEqual("parentId", parentID).
		AndEqual("url", url)

	return service.Load(criteria, result)
}

func (service *Outbox) DeleteByURL(parentType string, parentID primitive.ObjectID, url string) error {

	const location = "service.Outbox.DeleteByURL"

	criteria := exp.Equal("parentType", parentType).
		AndEqual("parentId", parentID).
		AndEqual("url", url)

	if err := service.collection.HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Error deleting Outbox Message", url)
	}

	// Delete the document from the cache
	if err := service.activityService.Delete(url); err != nil {
		return derp.Wrap(err, location, "Error deleting ActivityStream", url)
	}

	return nil
}
