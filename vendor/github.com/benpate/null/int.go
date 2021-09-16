package null

import (
	"strconv"

	"github.com/benpate/derp"
)

// Int provides a nullable bool
type Int struct {
	value   int
	present bool
}

// NewInt returns a fully populated, nullable bool
func NewInt(value int) Int {
	return Int{
		value:   value,
		present: true,
	}
}

// Int returns the actual value of this object
func (i Int) Int() int {
	return i.value
}

// String returns a string representation of this value
func (i Int) String() string {

	if i.present {
		return strconv.Itoa(i.value)
	}

	return ""
}

// Set applies a new value to the nullable item
func (i *Int) Set(value int) {
	i.value = value
	i.present = true
}

// Unset removes the value from this item, and sets it to null
func (i *Int) Unset() {
	i.value = 0
	i.present = false
}

// IsNull returns TRUE if this value is null
func (i Int) IsNull() bool {
	return i.present == false
}

// Interface returns the int value (if present) or NIL
func (i Int) Interface() interface{} {

	if i.present == false {
		return nil
	}

	return i.value
}

// IsPresent returns TRUE if this value is present
func (i Int) IsPresent() bool {
	return i.present
}

// MarshalJSON implements the json.Marshaller interface
func (i Int) MarshalJSON() ([]byte, error) {

	if i.present {
		return []byte(strconv.Itoa(i.value)), nil
	}

	return []byte("null"), nil
}

// UnmarshalJSON implements the json.Unmarshaller interface
func (i *Int) UnmarshalJSON(value []byte) error {

	valueStr := string(value)

	// Allow null values to be null
	if (valueStr == "") || (valueStr == "null") {
		i.Unset()
		return nil
	}

	// Try to convert the value to an integer
	result, err := strconv.Atoi(valueStr)

	if err == nil {
		i.Set(result)
		return nil
	}

	// Fall through means error
	return derp.Wrap(err, "null.Int.UnmarshalJSON", "Invalid int value", valueStr)
}
