package schema

// Type enumerates all of the data types that can make up a schema
type Type string

// String implements the ubiquitous "Stringer" interface, so that these types can be represented as strings, if necessary
func (schemaType Type) String() string {
	return string(schemaType)
}

// TypeAny is the token used by JSON-Schema to designate that any kind of data
const TypeAny = Type("any")

// TypeArray is the token used by JSON-Schema to designate that a schema describes an array.
const TypeArray = Type("array")

// TypeBoolean is the token used by JSON-Schema to designate that a schema describes an boolean.
const TypeBoolean = Type("boolean")

// TypeInteger is the token used by JSON-Schema to designate that a schema describes an integer.
const TypeInteger = Type("integer")

// TypeNumber is the token used by JSON-Schema to designate that a schema describes an number.
const TypeNumber = Type("number")

// TypeObject is the token used by JSON-Schema to designate that a schema describes an object.
const TypeObject = Type("object")

// TypeString is the token used by JSON-Schema to designate that a schema describes an string.
const TypeString = Type("string")
