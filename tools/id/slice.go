package id

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/slice"
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

func (slice Slice) IsEmpty() bool {
	return slice.Length() == 0
}

func (slice Slice) NotEmpty() bool {
	return slice.Length() > 0
}

func (slice Slice) First() primitive.ObjectID {
	if slice.Length() == 0 {
		return primitive.NilObjectID
	}
	return slice[0]
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

	if value == nil {
		*slice = make([]primitive.ObjectID, 0)
		return nil
	}

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

	case primitive.ObjectID:
		return slice.SetValue([]primitive.ObjectID{typed})

	case string:
		return slice.SetValue([]string{typed})

	default:
		return derp.BadRequestError("id.Slice.SetValue", "Unable to convert value to Slice", value)
	}

}

// Append adds one or more elements to the end of the slice
func (x *Slice) Append(value ...primitive.ObjectID) {
	*x = append(*x, value...)
}

func (x Slice) Contains(value primitive.ObjectID) bool {
	return slice.Contains(x, value)
}

func (x Slice) NotContains(value primitive.ObjectID) bool {
	return !slice.Contains(x, value)
}

func (x Slice) ContainsAny(values ...primitive.ObjectID) bool {
	return slice.ContainsAny(x, values...)
}

// ContainsInterface returns TRUE if the provided generic value is contained in the slice.
func (x Slice) ContainsInterface(value any) bool {

	// Convert the value to a string
	if value, err := Convert(value); err == nil {
		return slice.Contains(x, value)
	}

	// If we can't convert the value to a string, then it is not contained in the slice
	return false
}

func (x Slice) SliceOfString() []string {

	// Convert the slice of ObjectIDs to a slice of strings
	result := make([]string, len(x))
	for index := range x {
		result[index] = x[index].Hex()
	}

	return result
}
