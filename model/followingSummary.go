package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type FollowingSummary struct {
	FollowingID primitive.ObjectID `bson:"_id"`
	URL         string             `bson:"url"`
	Label       string             `bson:"label"`
	Status      string             `bson:"status"`
	LastPolled  int64              `bson:"lastPolled"`
	NextPoll    int64              `bson:"nextPoll"`
}

// FollowingSummaryFields returns a slice of all BSON field names for a FollowingSummary
func FollowingSummaryFields() []string {
	return []string{"_id", "url", "label", "status", "lastPolled", "nextPoll"}
}

func (followingSummary FollowingSummary) Fields() []string {
	return FollowingSummaryFields()
}
