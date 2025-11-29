package realtime

import "go.mongodb.org/mongo-driver/bson/primitive"

type Message struct {
	ObjectID primitive.ObjectID
	Topic    int
}

func NewMessage_Updated(objectID primitive.ObjectID) Message {
	return Message{
		ObjectID: objectID,
		Topic:    TopicUpdated,
	}
}

func NewMessage_ChildUpdated(objectID primitive.ObjectID) Message {
	return Message{
		ObjectID: objectID,
		Topic:    TopicChildUpdated,
	}
}

func NewMessage_NewReplies(objectID primitive.ObjectID) Message {
	return Message{
		ObjectID: objectID,
		Topic:    TopicNewReplies,
	}
}

func NewMessage_ImportProgress(objectID primitive.ObjectID) Message {
	return Message{
		ObjectID: objectID,
		Topic:    TopicImportProgress,
	}
}
