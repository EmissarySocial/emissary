package datatype

import "go.mongodb.org/mongo-driver/bson/primitive"

func ConvertObjectIDs(data []primitive.ObjectID) []string {

	result := make([]string, len(data))

	for index := range data {
		result[index] = data[index].Hex()
	}

	return result
}
