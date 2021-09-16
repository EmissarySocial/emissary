package path

import (
	"github.com/benpate/convert"
	"github.com/benpate/derp"
)

func setSliceOfString(path Path, object []string, value interface{}) error {

	index, err := path.Index(len(object))

	if err != nil {
		return err
	}

	if path.HasTail() {
		return derp.New(500, "path.Path.Set", "Invalid Path", path)
	}

	object[index] = convert.String(value)
	return nil
}

func setSliceOfInt(path Path, object []int, value interface{}) error {

	index, err := path.Index(len(object))

	if err != nil {
		return err
	}

	if path.HasTail() {
		return derp.New(500, "path.Path.Set", "Invalid Path", path)
	}

	object[index] = convert.Int(value)
	return nil
}

func setSliceOfInterface(path Path, object []interface{}, value interface{}) error {

	index, err := path.Index(len(object))

	if err != nil {
		return err
	}

	if path.IsTailEmpty() {
		object[index] = value
	}

	return path.Tail().Set(object[index], value)
}

func setSliceOfSetter(path Path, object []Setter, value interface{}) error {

	index, err := path.Index(len(object))

	if err != nil {
		return err
	}

	return path.Tail().Set(object[index], value)
}

func setMapOfInterface(path Path, object map[string]interface{}, value interface{}) error {

	if path.IsTailEmpty() {
		object[path.Head()] = value
		return nil
	}

	return derp.New(500, "path.Set", "Unimplemented")
}

func setMapOfString(path Path, object map[string]string, value interface{}) error {

	if path.IsTailEmpty() {
		object[path.Head()] = convert.String(value)
		return nil
	}

	return derp.New(500, "path.Set", "Unimplemented")
}
