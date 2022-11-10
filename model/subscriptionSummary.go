package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type SubscriptionSummary struct {
	SubscriptionID primitive.ObjectID `bson:"_id"`
	URL            string             `bson:"url"`
	Label          string             `bson:"label"`
	Status         string             `bson:"status"`
	LastPolled     int64              `bson:"lastPolled"`
	NextPoll       int64              `bson:"nextPoll"`
}
