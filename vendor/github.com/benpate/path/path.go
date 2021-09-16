package path

import (
	"strconv"
	"strings"

	"github.com/benpate/derp"
)

// Path is a reference to a value within another data object.
type Path []string

// New creates a new Path object
func New(value string) Path {

	if value == "" {
		return Path([]string{})
	}

	return Path(strings.Split(value, "."))
}

// Get tries to return the value of the object at the provided path.
func Get(object interface{}, path string) (interface{}, error) {
	return New(path).Get(object)
}

// Set tries to set the value of the object at the provided path.
func Set(object interface{}, path string, value interface{}) error {
	return New(path).Set(object, value)
}

func Delete(object interface{}, path string) error {
	return New(path).Delete(object)
}

// Get tries to return the value of the object at this path.
func (path Path) Get(object interface{}) (interface{}, error) {

	// If the path is empty, then we have reached our goal.  Return the value of this object
	if path.IsEmpty() {
		return object, nil
	}

	// Next steps depend on the type of object we're working with.
	switch obj := object.(type) {

	case Getter:
		return obj.GetPath(path)

	case []string:
		return getSliceOfString(path, obj)

	case []int:
		return getSliceOfInt(path, obj)

	case []interface{}:
		return getSliceOfInterface(path, obj)

	case []Getter:
		return getSliceOfGetter(path, obj)

	case map[string]string:
		return getMapOfString(path, obj)

	case map[string]interface{}:
		return getMapOfInterface(path, obj)

	}

	return nil, derp.New(500, "path.Path.Get", "Object does not support 'Getter' interface", object)
}

// Set tries to return the value of the object at this path.
func (path Path) Set(object interface{}, value interface{}) error {

	switch obj := object.(type) {

	case Setter:
		return obj.SetPath(path, value)

	case []string:
		return setSliceOfString(path, obj, value)

	case []int:
		return setSliceOfInt(path, obj, value)

	case []interface{}:
		return setSliceOfInterface(path, obj, value)

	case []Setter:
		return setSliceOfSetter(path, obj, value)

	case map[string]string:
		return setMapOfString(path, obj, value)

	case map[string]interface{}:
		return setMapOfInterface(path, obj, value)

	}

	return derp.New(500, "path.Path.Set", "Object does not support 'Setter' interface", object)
}

// Delete tries to remove a value from ths object at this path
func (path Path) Delete(object interface{}) error {

	switch obj := object.(type) {

	case Deleter:
		return obj.DeletePath(path)

	case []string:
		return deleteSliceOfString(path, obj)

	case []int:
		return deleteSliceOfInt(path, obj)

	case []interface{}:
		return deleteSliceOfInterface(path, obj)

	case []Deleter:
		return deleteSliceOfDeleter(path, obj)

	case map[string]string:
		return deleteMapOfString(path, obj)

	case map[string]interface{}:
		return deleteMapOfInterface(path, obj)
	}

	return derp.New(500, "path.Path.Delete", "Unable to delete from this type of record.")
}

// IsEmpty returns TRUE if this path does not contain any tokens
func (path Path) IsEmpty() bool {
	return len(path) == 0
}

// HasTail returns TRUE if this path has one or more items in its tail.
func (path Path) HasTail() bool {
	return len(path) > 1
}

// IsTailEmpty returns TRUE if this path has one or more items in its tail.
func (path Path) IsTailEmpty() bool {
	return len(path) <= 1
}

// Head returns the first token in the path.
func (path Path) Head() string {
	return path[0]
}

// Tail returns a slice of all tokens *after the first token*
func (path Path) Tail() Path {
	return path[1:]
}

// Split returns two values, the Head and the Tail of the current path
func (path Path) Split() (string, Path) {
	switch len(path) {
	case 0:
		return "", New("")
	case 1:
		return path[0], New("")
	default:
		return path[0], path[1:]
	}
}

// String implements the Stringer interface, and converts the path into a readable string
func (path Path) String() string {
	return strings.Join(path, ".")
}

// Push returns a new path with a new value appended to the beginning of the path.
func (path Path) Push(value string) Path {
	return append([]string{value}, path...)
}

// Index is useful for vetting array indices.  It attempts to convert the Head() token int
// an integer, and then check that the integer is within the designated array bounds (is greater than zero,
// and less than the maximum value provided to the function).
//
// It returns the array index and an error
func (path Path) Index(maximum int) (int, error) {

	result, err := strconv.Atoi(path.Head())

	if err != nil {
		return 0, derp.Wrap(err, "path.Index", "Index must be an integer", path, maximum)
	}

	if result < 0 {
		return 0, derp.New(500, "path.Index", "Index cannot be negative", path, maximum)
	}

	if (maximum != -1) && (result >= maximum) {
		return 0, derp.New(500, "path.Index", "Index out of bounds", path, maximum)
	}

	// Fall through means that this is a valid array index
	return result, nil
}
