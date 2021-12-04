package datatype

import (
	"github.com/benpate/convert"
	"github.com/benpate/derp"
	"github.com/benpate/path"
)

// Map implements some quality of life extensions to a standard map[string]interface{}
type Map map[string]interface{}

// NewMap returns a fully initialized Map object.
func NewMap() Map {
	return Map(map[string]interface{}{})
}

// AsMapOfInterface returns the underlying map datastructure
func (m Map) AsMapOfInterface() map[string]interface{} {
	return map[string]interface{}(m)
}

// GetKeys returns all keys of the underlying map
func (m Map) GetKeys() []string {
	result := make([]string, len(m))

	index := 0
	for key := range m {
		result[index] = key
		index = index + 1
	}

	return result
}

// GetInterface returns a named option without any conversion.  You get what you get.
func (m Map) GetInterface(name string) interface{} {
	return m[name]
}

// GetString returns a named option as a string type.
func (m Map) GetString(name string) string {
	return convert.StringDefault(m[name], "")
}

// GetBool returns a named option as a bool type.
func (m Map) GetBool(name string) bool {
	return convert.BoolDefault(m[name], false)
}

// GetInt returns a named option as an int type.
func (m Map) GetInt(name string) int {
	return convert.IntDefault(m[name], 0)
}

// GetSliceOfString returns a named option as a slice of strings
func (m Map) GetSliceOfString(name string) []string {
	return convert.SliceOfString(m[name])
}

// GetSliceOfInt returns a named option as a slice of int values
func (m Map) GetSliceOfInt(name string) []int {
	return convert.SliceOfInt(m[name])
}

// GetSliceOfFloat returns a named option as a slice of float64 values
func (m Map) GetSliceOfFloat(name string) []float64 {
	return convert.SliceOfFloat(m[name])
}

// GetSliceOfMap returns a named option as a slice of datatype.Map objects.
func (m Map) GetSliceOfMap(name string) []Map {
	value := convert.SliceOfMap(m[name])
	result := make([]Map, len(value))

	for index := range value {
		result[index] = Map(value[index])
	}

	return result
}

func (m Map) GetPath(p path.Path) (interface{}, error) {

	if value, ok := m[p.Head()]; ok {
		return p.Tail().Get(value)
	}

	return nil, derp.New(500, "datatype.Map.GetPath", "Missing Key in Map", p)
}

func (m Map) SetPath(p path.Path, value interface{}) error {

	head, tail := p.Split()

	if tail.IsEmpty() {
		m[head] = value
		return nil
	}

	return tail.Set(m[head], value)
}
