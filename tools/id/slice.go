package id

import (
	"sort"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Converts a value into a slice of ObjectIDs
func SliceOfID(value any) []primitive.ObjectID {

	switch v := value.(type) {

	case []primitive.ObjectID:
		return v

	case []string:
		result := make([]primitive.ObjectID, len(v))
		for index := range v {
			result[index] = ID(v[index])
		}
		return result
	}

	return make([]primitive.ObjectID, 0)
}

func SliceOfString(value []primitive.ObjectID) []string {
	result := make([]string, len(value))

	for index := range value {
		result[index] = value[index].Hex()
	}

	return result
}

func Sort(value []primitive.ObjectID) []primitive.ObjectID {

	sort.Slice(value, func(i int, j int) bool {
		return (value[i].Hex() < value[j].Hex())
	})

	return value
}
