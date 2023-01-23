package id

import (
	"github.com/benpate/derp"
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

func (slice *Slice) SetString(name string, value string) bool {

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

func (slice *Slice) SetValue(value any) error {

	switch typed := value.(type) {

	case []primitive.ObjectID:
		*slice = typed
		return nil

	case []string:
		*slice = make([]primitive.ObjectID, len(typed))
		for index := range typed {
			item, _ := primitive.ObjectIDFromHex(typed[index])
			(*slice)[index] = item
		}
		return nil

	default:
		return derp.NewBadRequestError("id.Slice.SetValue", "Unable to convert value to Slice", value)
	}
}
