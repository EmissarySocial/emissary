package queries

type Counter struct {
	Count int `bson:"count"`
}

type GroupedCounter struct {
	Group string `bson:"_id"`
	Count int    `bson:"count"`
}
