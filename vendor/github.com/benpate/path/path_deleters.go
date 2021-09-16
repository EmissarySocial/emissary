package path

import "github.com/benpate/derp"

func deleteSliceOfString(path Path, object []string) error {
	return derp.New(500, "path.deleteSliceOfString", "Unimplemented")
}

func deleteSliceOfInt(path Path, object []int) error {
	return derp.New(500, "path.deleteSliceOfString", "Unimplemented")
}

func deleteSliceOfDeleter(path Path, object []Deleter) error {
	return derp.New(500, "path.deleteSliceOfString", "Unimplemented")
}

func deleteSliceOfInterface(path Path, object []interface{}) error {
	return derp.New(500, "path.deleteSliceOfString", "Unimplemented")
}

func deleteMapOfString(path Path, object map[string]string) error {

	head, tail := path.Split()

	if tail.IsEmpty() {
		delete(object, head)
		return nil
	}

	return tail.Delete(object[head])
}

func deleteMapOfInterface(path Path, object map[string]interface{}) error {

	head, tail := path.Split()

	if tail.IsEmpty() {
		delete(object, head)
		return nil
	}

	return tail.Delete(object[head])
}
