package service

import (
	"iter"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/collection"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/ranges"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MLSMessage manages all interactions with the MLSMessage collection
type MLSMessage struct {
	host string
}

// NewMLSMessage returns a fully populated MLSMessage service
func NewMLSMessage() MLSMessage {
	return MLSMessage{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *MLSMessage) Refresh(factory *Factory) {
	service.host = factory.Host()
}

// Close stops any background processes controlled by this service
func (service *MLSMessage) Close() {

}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *MLSMessage) collection(session data.Session) data.Collection {
	return session.Collection("MLSMessage")
}

// Count returns the number of records that match the provided criteria
func (service *MLSMessage) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns a slice of MLSMessages that match the provided criteria
func (service *MLSMessage) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.MLSMessage, error) {
	result := make([]model.MLSMessage, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// List returns an iterator containing all of the MLSMessages who match the provided criteria
func (service *MLSMessage) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the MLSMessages that match the provided criteria
func (service *MLSMessage) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.MLSMessage], error) {

	iter, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.MLSMessage.Range", "Unable to create iterator", criteria)
	}

	return RangeFunc(iter, model.NewMLSMessage), nil
}

// Load retrieves an MLSMessage from the database
func (service *MLSMessage) Load(session data.Session, criteria exp.Expression, result *model.MLSMessage) error {

	const location = "service.MLSMessage.Load"

	if err := service.collection(session).Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, location, "Unable to load MLSMessage", criteria)
	}

	return nil
}

// Save adds/updates an MLSMessage in the database
func (service *MLSMessage) Save(session data.Session, message *model.MLSMessage, note string) error {

	const location = "service.MLSMessage.Save"

	// Set the ActivityPub URL for this message
	message.ActivityPubURL = service.host + "/@" + message.UserID.Hex() + "/pub/mls/messages/" + message.ID()

	// Validate the value before saving
	if err := service.Schema().Validate(message); err != nil {
		return derp.Wrap(err, location, "Unable to validate MLSMessage", message)
	}

	// Save the value to the database
	if err := service.collection(session).Save(message, note); err != nil {
		return derp.Wrap(err, location, "Unable to save MLSMessage", message, note)
	}

	return nil
}

// Delete removes an MLSMessage from the database (virtual delete)
func (service *MLSMessage) Delete(session data.Session, message *model.MLSMessage, note string) error {

	const location = "service.MLSMessage.Delete"

	if err := service.collection(session).Delete(message, note); err != nil {
		return derp.Wrap(err, location, "Unable to delete MLSMessage", message, note)
	}

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

func (service *MLSMessage) Schema() schema.Schema {
	return schema.New(model.MLSMessageSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *MLSMessage) LoadByToken(session data.Session, userID primitive.ObjectID, token string, result *model.MLSMessage) error {

	const location = "service.MLSMessage.LoadByToken"

	messageID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.Wrap(err, location, "Invalid MLSMessage ID", "token", token)
	}

	return service.LoadByID(session, userID, messageID, result)
}

func (service *MLSMessage) LoadByID(session data.Session, userID primitive.ObjectID, messageID primitive.ObjectID, result *model.MLSMessage) error {
	criteria := exp.Equal("_id", messageID).AndEqual("userId", userID)
	return service.Load(session, criteria, result)
}

// CountByUser returns the number of MLSMessages that belong to a user
func (service *MLSMessage) CountByUser(session data.Session, userID primitive.ObjectID) (int64, error) {
	return service.Count(session, exp.Equal("userId", userID))
}

// RangeByUser returns a Go 1.23 RangeFunc that iterates over the MLSMessages that belong to a user (in natural chronological order)
func (service *MLSMessage) RangeByUser(session data.Session, userID primitive.ObjectID, startAfter string) (iter.Seq[model.MLSMessage], error) {

	// Build the base criteria
	var criteria exp.Expression = exp.Equal("userId", userID)

	// Add the "startAfter" criteria (if applicable)
	if startAfterID := service.parseMessageID(startAfter, userID); !startAfterID.IsZero() {
		criteria = criteria.AndGreaterThan("_id", startAfterID)
	}

	// Return the filtered range
	return service.Range(session, criteria, option.SortAsc("_id"))
}

/******************************************
 * Collection Interface
 ******************************************/

// CollectionID returns the identifier function for this collection
func (service *MLSMessage) CollectionID(userID primitive.ObjectID) collection.IdentifierFunc {
	return func() string {
		return "https://emissary.social/@" + userID.Hex() + "/pub/mls/messages"
	}
}

// CollectionCount returns the counter function for this collection
func (service *MLSMessage) CollectionCount(session data.Session, userID primitive.ObjectID) collection.CounterFunc {
	return func() (int64, error) {
		return service.CountByUser(session, userID)
	}
}

// CollectionIterator returns the iterator function for this collection
func (service *MLSMessage) CollectionIterator(session data.Session, userID primitive.ObjectID) collection.IteratorFunc {

	const location = "service.MLSMessage.CollectionIterator"

	return func(startAfter string) (iter.Seq[mapof.Any], error) {

		result, err := service.RangeByUser(session, userID, startAfter)

		if err != nil {
			return nil, derp.Wrap(err, location, "Unable to create iterator", "userID", userID.Hex())
		}

		return ranges.Map(result, func(item model.MLSMessage) mapof.Any {
			return item.GetJSONLD()
		}), nil
	}
}

// parseMessageID extracts the messageID from the provided URL
func (service *MLSMessage) parseMessageID(url string, userID primitive.ObjectID) primitive.ObjectID {

	if url == "" {
		return primitive.NilObjectID
	}

	if url == "START" {
		return primitive.NilObjectID
	}

	prefix := service.host + "/@" + userID.Hex() + "/pub/mls/messages/"

	if !strings.HasPrefix(url, prefix) {
		return primitive.NilObjectID
	}

	messageIDHex := strings.TrimPrefix(url, prefix)
	messageID, err := primitive.ObjectIDFromHex(messageIDHex)

	if err != nil {
		return primitive.NilObjectID
	}

	return messageID
}
