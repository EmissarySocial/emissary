package realtime

import "go.mongodb.org/mongo-driver/bson/primitive"

// Message represents a realtime update to be sent to the client via Server-Sent Events (SSE)
type Message struct {
	ObjectID primitive.ObjectID
	Topic    int
	Event    string
	Data     string
}

// NewMessage_ChildUpdated creates a new SSE message sent when a Stream's child has been updated
func NewMessage_ChildUpdated(objectID primitive.ObjectID) Message {
	return Message{
		ObjectID: objectID,
		Topic:    TopicChildUpdated,
		Event:    objectID.Hex(),
		Data:     "child updated",
	}
}

// NewMessage_FollowingUpdated creates a new SSE message sent when a User's Following record has been updated
func NewMessage_FollowingUpdated(objectID primitive.ObjectID) Message {
	return Message{
		ObjectID: objectID,
		Topic:    TopicFollowingUpdated,
		Event:    objectID.Hex(),
		Data:     "following updated",
	}
}

// NewMessage_ImportProgress creates a new SSE message sent to report progress during an import operation
func NewMessage_ImportProgress(objectID primitive.ObjectID) Message {
	return Message{
		ObjectID: objectID,
		Topic:    TopicImportProgress,
		Event:    objectID.Hex(),
		Data:     "import progress",
	}
}

// NewMessage_InboxActivity creates a new SSE message sent when there is new activity in a User's Inbox
func NewMessage_InboxActivity(objectID primitive.ObjectID, data string) Message {
	return Message{
		ObjectID: objectID,
		Topic:    TopicInboxActivity,
		Event:    "", // Default "message" -type event
		Data:     data,
	}
}

// NewMessage_InboxActivity_DirectMessage creates a new SSE message sent when a User receives a direct message
func NewMessage_InboxActivity_DirectMessage(objectID primitive.ObjectID, data string) Message {
	return Message{
		ObjectID: objectID,
		Topic:    TopicInboxActivity_DirectMessage,
		Event:    "", // Default "message" -type event
		Data:     data,
	}
}

// NewMessage_InboxActivity_DirectMessage_MLS creates a new SSE message sent when a User receives an MLS-encoded direct message
func NewMessage_InboxActivity_DirectMessage_MLS(objectID primitive.ObjectID, data string) Message {
	return Message{
		ObjectID: objectID,
		Topic:    TopicInboxActivity_DirectMessage_MLS,
		Event:    "", // Default "message" -type event
		Data:     data,
	}
}

// NewMessage_NewReplies creates a new SSE message sent when a Stream receives a new reply
func NewMessage_NewReplies(objectID primitive.ObjectID) Message {
	return Message{
		ObjectID: objectID,
		Topic:    TopicNewReplies,
		Event:    objectID.Hex(),
		Data:     "new replies",
	}
}

// NewMessage_Updated creates a new SSE message sent when a User or Stream that has been updated
func NewMessage_Updated(objectID primitive.ObjectID) Message {
	return Message{
		ObjectID: objectID,
		Topic:    TopicUpdated,
		Event:    objectID.Hex(),
		Data:     "updated",
	}
}
