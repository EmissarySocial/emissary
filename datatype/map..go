package datatype

import "github.com/benpate/convert"

// Map implements some quality of life extensions to a standard map[string]interface{}
type Map map[string]interface{}

// AsString returns a named option as a string type.
func (m Map) AsString(name string) string {
	return convert.StringDefault(m[name], "")
}

// AsBool returns a named option as a bool type.
func (m Map) AsBool(name string) bool {
	return convert.BoolDefault(m[name], false)
}

// AsInt returns a named option as an int type.
func (m Map) AsInt(name string) int {
	return convert.IntDefault(m[name], 0)
}

// AsInterface returns a named option without any conversion.  You get what you get.
func (m Map) AsInterface(name string) interface{} {
	return m[name]
}
