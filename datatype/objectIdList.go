package datatype

import (
	"github.com/benpate/convert"
	"github.com/benpate/derp"
	"github.com/benpate/path"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ObjectIDList []primitive.ObjectID

func NewObjectIDList() ObjectIDList {
	return make(ObjectIDList, 0)
}

func ParseObjectIDList(value interface{}) (ObjectIDList, error) {

	switch values := value.(type) {

	case ObjectIDList:
		return values, nil

	case []primitive.ObjectID:
		return ObjectIDList(values), nil

	case []interface{}:
		result := make([]primitive.ObjectID, len(values))
		for index, item := range values {

			switch item := item.(type) {
			case primitive.ObjectID:
				result[index] = item
			case string:
				objectID, err := primitive.ObjectIDFromHex(item)
				if err != nil {
					return nil, derp.Wrap(err, "whisper.datatype.ParseObjectIDList", "Invalid item in array", item)
				}
				result[index] = objectID

			default:
				return nil, derp.New(500, "whisper.datatype.ParseObjectIDList", "Invalid item in array", index, item)
			}
		}
		return result, nil
	}
	return nil, derp.New(500, "whisper.datatype.ParseObjectIDList", "Invalid data type", value)
}

func (objectIDList ObjectIDList) GetPath(p path.Path) (interface{}, error) {

	if index, ok := convert.IntOk(p.Head(), 0); ok {
		if index < len(objectIDList) {
			return objectIDList[index], nil
		}

		return nil, derp.New(500, "whisper.datatype.ObjectIDList.GetPath", "Index out of bounds", index)
	}

	return nil, derp.New(500, "whisper.datatype.ObjectIDList.GetPath", "Invalid Index", p.Head())
}

func (objectIDList *ObjectIDList) SetPath(p path.Path, value interface{}) error {

	if index, ok := convert.IntOk(p.Head(), 0); ok {

		objectID, err := primitive.ObjectIDFromHex(convert.String(value))

		if err != nil {
			return derp.New(500, "whisper.datatype.ObjectIDList.SetPath", "Invalid Value", value)
		}

		for index < len(*objectIDList) {
			*objectIDList = append(*objectIDList, primitive.NewObjectID())
		}

		(*objectIDList)[index] = objectID
		return nil
	}

	return derp.New(500, "whisper.datatype.ObjectIDList.SetPath", "Invalid Index", p.Head())
}

func (objectIDList *ObjectIDList) DeletePath(p path.Path) error {

	if index, ok := convert.IntOk(p.Head(), 0); ok {

		if index < len(*objectIDList) {
			*objectIDList = append((*objectIDList)[:index], (*objectIDList)[index+1:]...)
			return nil
		}

		return derp.New(500, "whisper.datatype.ObjectIDList.DeletePath", "Index Out of Bounds", p.Head())
	}

	return derp.New(500, "whisper.datatype.ObjectIDList.DeletePath", "Invalid Index", p.Head())
}
