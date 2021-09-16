package path

import (
	"github.com/benpate/derp"
)

/////////////////////////////
// Interface-based getters
/////////////////////////////

func getSliceOfString(path Path, value []string) (string, error) {

	index, err := path.Index(len(value))

	if err != nil {
		return "", err
	}

	if path.HasTail() {
		return "", derp.New(500, "path.Path.getSliceOfString", "Invalid path", path)
	}

	return value[index], nil
}

func getSliceOfInt(path Path, value []int) (int, error) {

	index, err := path.Index(len(value))

	if err != nil {
		return 0, err
	}

	if path.HasTail() {
		return 0, derp.New(500, "path.Path.getSliceOfString", "Invalid path", path)
	}

	return value[index], nil
}

func getSliceOfInterface(path Path, value []interface{}) (interface{}, error) {

	index, err := path.Index(len(value))

	if err != nil {
		return nil, err
	}

	return path.Tail().Get(value[index])
}

func getSliceOfGetter(path Path, value []Getter) (interface{}, error) {

	index, err := path.Index(len(value))

	if err != nil {
		return nil, err
	}

	return path.Tail().Get(value[index])
}

func getMapOfString(path Path, value map[string]string) (interface{}, error) {

	head, tail := path.Split()

	if tail.IsEmpty() {
		return value[head], nil
	}

	return nil, derp.New(500, "path.getMapOfString", "Invalid Path", path)
}

func getMapOfInterface(path Path, value map[string]interface{}) (interface{}, error) {

	head, tail := path.Split()

	if tail.IsEmpty() {
		return value[head], nil
	}

	return tail.Get(value[head])
}
