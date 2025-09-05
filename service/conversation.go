package service

import (
	"iter"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Conversation defines a service that can send and receive conversation data
type Conversation struct {
}

// NewConversation returns a fully initialized Conversation service
func NewConversation() Conversation {
	return Conversation{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service Conversation) Refresh() {
}

// Close stops any background processes controlled by this service
func (service Conversation) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service Conversation) collection(session data.Session) data.Collection {
	return session.Collection("Conversation")
}

// Count returns the number of Conversations that match the provided criteria
func (service Conversation) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns a slice containing all of the Conversations that match the provided criteria
func (service Conversation) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.Conversation, error) {
	result := make([]model.Conversation, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// List returns an iterator containing all of the Conversations that match the provided criteria
func (service Conversation) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Range returns an iterator containing all of the Users who match the provided criteria
func (service Conversation) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.Conversation], error) {

	iter, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.User.Range", "Error creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewConversation), nil
}

// Load retrieves an Conversation from the database
func (service Conversation) Load(session data.Session, criteria exp.Expression, conversation *model.Conversation) error {

	if err := service.collection(session).Load(notDeleted(criteria), conversation); err != nil {
		return derp.Wrap(err, "service.Conversation.Load", "Error loading Conversation", criteria)
	}

	return nil
}

// Save adds/updates an Conversation in the database
func (service Conversation) Save(session data.Session, conversation *model.Conversation, note string) error {

	const location = "service.Conversation.Save"

	// Validate the value before saving
	if err := service.Schema().Validate(conversation); err != nil {
		return derp.Wrap(err, location, "Error validating Conversation", conversation)
	}

	// Save the value to the database
	if err := service.collection(session).Save(conversation, note); err != nil {
		return derp.Wrap(err, location, "Error saving Conversation", conversation, note)
	}

	return nil
}

// Delete removes an Conversation from the database (hard delete)
func (service Conversation) Delete(session data.Session, conversation *model.Conversation, note string) error {

	const location = "service.Conversation.Delete"

	// Delete this Conversation
	if err := service.collection(session).HardDelete(exp.Equal("_id", conversation.ConversationID)); err != nil {
		return derp.Wrap(err, location, "Unable to delete Conversation", conversation)
	}

	return nil
}

/******************************************
 * Generic Data Methods
******************************************/

// ObjectType returns the type of object that this service manages
func (service Conversation) ObjectType() string {
	return "Conversation"
}

// New returns a fully initialized model.Conversation as a data.Object.
func (service Conversation) ObjectNew() data.Object {
	result := model.NewConversation()
	return &result
}

func (service Conversation) ObjectID(object data.Object) primitive.ObjectID {

	if conversation, ok := object.(*model.Conversation); ok {
		return conversation.ConversationID
	}

	return primitive.NilObjectID
}

func (service Conversation) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

func (service Conversation) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewConversation()
	err := service.Load(session, criteria, &result)
	return &result, err
}

func (service Conversation) ObjectSave(session data.Session, object data.Object, note string) error {

	if conversation, ok := object.(*model.Conversation); ok {
		return service.Save(session, conversation, note)
	}
	return derp.InternalError("service.Conversation.ObjectSave", "Invalid object type", object)
}

func (service Conversation) ObjectDelete(session data.Session, object data.Object, note string) error {
	if conversation, ok := object.(*model.Conversation); ok {
		return service.Delete(session, conversation, note)
	}
	return derp.InternalError("service.Conversation.ObjectDelete", "Invalid object type", object)
}

func (service Conversation) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.UnauthorizedError("service.Conversation", "Not Authorized")
}

func (service Conversation) Schema() schema.Schema {
	return schema.New(model.ConversationSchema())
}

/******************************************
 * Common Queries
 ******************************************/

func (service Conversation) LoadByID(session data.Session, userID, conversationID primitive.ObjectID, conversation *model.Conversation) error {
	criteria := exp.Equal("userId", userID).AndEqual("_id", conversationID)
	return service.Load(session, criteria, conversation)
}
