package queries

// Counter holds the result of a MongoDB count aggregation
type Counter struct {
	Count int `bson:"count"`
}

// GroupedCounter holds one row of a MongoDB "group and count" aggregation
type GroupedCounter struct {
	Group string `bson:"_id"`
	Count int    `bson:"count"`
}
