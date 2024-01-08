package model

// ActorSummary is a record returned by the ActivityStreams directory
type ActorSummary struct {
	ID       string `bson:"id"`
	Type     string `bson:"type"`
	Name     string `bson:"name"`
	Icon     string `bson:"icon"`
	Username string `bson:"preferredUsername"`
}
