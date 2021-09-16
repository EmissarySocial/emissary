package null

import "github.com/benpate/derp"

// Bool provides a nullable bool
type Bool struct {
	value   bool
	present bool
}

// NewBool returns a fully populated, nullable bool
func NewBool(value bool) Bool {
	return Bool{
		value:   value,
		present: true,
	}
}

// Bool returns the actual value of this object
func (b Bool) Bool() bool {
	return b.value
}

func (b Bool) String() string {

	if b.present {

		if b.value {
			return "true"
		}
		return "false"
	}
	return ""
}

// Interface returns the boolean value (if present) or NIL
func (b Bool) Interface() interface{} {

	if b.present == false {
		return nil
	}

	return b.value
}

// Set applies a new value to the nullable item
func (b *Bool) Set(value bool) {
	b.value = value
	b.present = true
}

// Unset removes the value from this item, and sets it to null
func (b *Bool) Unset() {
	b.value = false
	b.present = false
}

// IsNull returns TRUE if this value is null
func (b Bool) IsNull() bool {
	return b.present == false
}

// IsPresent returns TRUE if this value is present
func (b Bool) IsPresent() bool {
	return b.present
}

// MarshalJSON implements the json.Marshaller interface
func (b Bool) MarshalJSON() ([]byte, error) {

	if b.present {
		if b.value {
			return []byte("true"), nil
		}
		return []byte("false"), nil
	}
	return []byte("null"), nil
}

// UnmarshalJSON implements the json.Unmarshaller interface
func (b *Bool) UnmarshalJSON(value []byte) error {

	valueStr := string(value)

	switch valueStr {
	case "true":
		b.Set(true)
		return nil
	case "false":
		b.Set(false)
		return nil
	case "null":
		b.Unset()
		return nil
	}

	return derp.New(500, "null.Bool.UnmarshalJSON", "Invalid boolean value", valueStr)
}
