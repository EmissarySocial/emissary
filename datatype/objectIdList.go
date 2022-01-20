package datatype

import (
	"github.com/benpate/convert"
	"github.com/benpate/derp"
	"github.com/benpate/list"
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

func (objectIDList ObjectIDList) GetPath(name string) (interface{}, error) {

	head, tail := list.Split(name, ".")

	if tail != "" {
		return nil, derp.NewInternalError("whisper.datatype.ObjectIDlist.GetPath", "Invalid path", name)
	}

	index, err := path.Index(head, len(objectIDList))

	if err != nil {
		return nil, derp.Wrap(err, "whisper.datatype.ObjectIDlist.GetPath", "Bad index", name)
	}

	return objectIDList[index], nil
}

func (objectIDList *ObjectIDList) SetPath(name string, value interface{}) error {

	head, tail := list.Split(name, ".")

	if tail != "" {
		return derp.NewInternalError("whisper.datatype.ObjectIDlist.GetPath", "Invalid path", name)
	}

	index, err := path.Index(head, len(*objectIDList))

	if err != nil {
		return derp.Wrap(err, "whisper.datatype.ObjectIDlist.GetPath", "Bad index", name)
	}

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

func (objectIDList *ObjectIDList) DeletePath(name string) error {

	head, tail := list.Split(name, ".")

	if tail != "" {
		return derp.NewInternalError("whisper.datatype.ObjectIDlist.GetPath", "Invalid path", name)
	}

	index, err := path.Index(head, len(*objectIDList))

	if err != nil {
		return derp.Wrap(err, "whisper.datatype.ObjectIDlist.GetPath", "Bad index", name)
	}

	*objectIDList = append((*objectIDList)[:index], (*objectIDList)[index+1:]...)
	return nil
}
