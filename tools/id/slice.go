package id

import (
	"sort"

	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Slice []primitive.ObjectID

func NewSlice() Slice {
	return make(Slice, 0)
}

func SliceSchema() schema.Element {
	return schema.Array{
		Items: schema.String{Format: "objectId"},
	}
}

/******************************************
 * Schema Getter/Setter Interfaces
 ******************************************/

func (slice Slice) Length() int {
	if slice == nil {
		return 0
	}
	return len(slice)
}

func (slice Slice) GetStringOK(name string) (string, bool) {

	if index, ok := schema.Index(name, slice.Length()); ok {
		return (slice)[index].Hex(), true
	}

	return "", false
}

func (slice *Slice) SetStringOK(name string, value string) bool {

	if objectID, err := primitive.ObjectIDFromHex(value); err == nil {

		if index, ok := schema.Index(name); ok {

			for index >= slice.Length() {
				(*slice) = append(*slice, primitive.NilObjectID)
			}

			(*slice)[index] = objectID
			return true
		}
	}

	return false
}

// Converts a value into a slice of ObjectIDs
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
