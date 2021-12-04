package mongodb

import (
	dataOption "github.com/benpate/data/option"
	"go.mongodb.org/mongo-driver/bson"
	mongoOptions "go.mongodb.org/mongo-driver/mongo/options"
)

func convertOptions(options ...dataOption.Option) *mongoOptions.FindOptions {

	if len(options) == 0 {
		return nil
	}

	result := mongoOptions.Find()

	for _, option := range options {

		switch o := option.(type) {
		case dataOption.FirstRowConfig:
			result.SetLimit(1)

		case dataOption.MaxRowsConfig:
			result.SetLimit(int64(o))

		case dataOption.SortConfig:
			result.SetSort(bson.D{{o.FieldName, sortDirection(o.Direction)}})
		}

	}

	return result
}

func sortDirection(direction string) int {
	if direction == dataOption.SortDirectionDescending {
		return -1
	}

	return 1
}
