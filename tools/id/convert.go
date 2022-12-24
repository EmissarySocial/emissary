package id

import (
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func Convert(value any) (primitive.ObjectID, error) {
	if value == nil {
		return primitive.NilObjectID, nil
	}

	switch v := value.(type) {
	case primitive.ObjectID:
		return v, nil

	case string:
		return primitive.ObjectIDFromHex(v)

	default:
		return primitive.NilObjectID, derp.NewInternalError("id.Convert", "Invalid Type", value)
	}
}

func ToBytes(value primitive.ObjectID) []byte {
	return value[:]
}

func FromBytes(value []byte) primitive.ObjectID {

	if len(value) == 12 {
		array := (*[12]byte)(value)
		return primitive.ObjectID(*array)
	}

	return primitive.NilObjectID
}
