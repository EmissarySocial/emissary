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
