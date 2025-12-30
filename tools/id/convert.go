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
		return primitive.NilObjectID, derp.Internal("id.Convert", "Invalid Type", value)
	}
}

func ConvertSlice(original []string) ([]primitive.ObjectID, error) {

	result := make([]primitive.ObjectID, 0, len(original))

	for index, value := range original {

		objectID, err := primitive.ObjectIDFromHex(value)

		if err != nil {
			return nil, derp.Wrap(err, "id.ConvertSlice", "Error converting string to ObjectID", value, index)
		}

		result = append(result, objectID)
	}

	return result, nil
}
