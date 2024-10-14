package consumer

import "go.mongodb.org/mongo-driver/bson/primitive"

func objectID(original string) primitive.ObjectID {
	result, _ := primitive.ObjectIDFromHex(original)
	return result
}
