package id

import (
	"sort"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SliceOfID coerces a value into a Slice of ObjectIDs
func SliceOfID(value any) Slice {

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

// SliceOfString converts a slice of ObjectIDs into a slice of hex strings
func SliceOfString(value []primitive.ObjectID) []string {
	result := make([]string, len(value))

	for index := range value {
		result[index] = value[index].Hex()
	}

	return result
}

// Sort orders a slice of ObjectIDs in place, ascending by hex value, and returns it
func Sort(value []primitive.ObjectID) []primitive.ObjectID {

	sort.Slice(value, func(i int, j int) bool {
		return (value[i].Hex() < value[j].Hex())
	})

	return value
}
