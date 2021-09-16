package null

import (
	"strconv"

	"github.com/benpate/derp"
)

// Float provides a nullable float64
type Float struct {
	value   float64
	present bool
}

// NewFloat returns a fully populated, nullable float64
func NewFloat(value float64) Float {
	return Float{
		value:   value,
		present: true,
	}
}

// Float returns the actual value of this object
func (f Float) Float() float64 {
	return f.value
}

// String returns a string representation of this value
func (f Float) String() string {

	if f.present {
		return strconv.FormatFloat(f.value, 'f', -2, 64)
	}

	return ""
}

// Interface returns the float64 value (if present) or NIL
func (f Float) Interface() interface{} {

	if f.present == false {
		return nil
	}

	return f.value
}

// Set applies a new value to the nullable item
func (f *Float) Set(value float64) {
	f.value = value
	f.present = true
}

// Unset removes the value from this item, and sets it to null
func (f *Float) Unset() {
	f.value = 0
	f.present = false
}

// IsNull returns TRUE if this value is null
func (f Float) IsNull() bool {
	return f.present == false
}

// IsPresent returns TRUE if this value is present
func (f Float) IsPresent() bool {
	return f.present
}

// MarshalJSON implements the json.Marshaller interface
func (f Float) MarshalJSON() ([]byte, error) {

	if f.present {
		return []byte(f.String()), nil
	}

	return []byte("null"), nil
}

// UnmarshalJSON implements the json.Unmarshaller interface
func (f *Float) UnmarshalJSON(value []byte) error {

	valueStr := string(value)

	// Allow null values to be null
	if (valueStr == "") || (valueStr == "null") {
		f.Unset()
		return nil
	}

	// Try to convert the value to an integer
	result, err := strconv.ParseFloat(valueStr, 64)

	if err == nil {
		f.Set(result)
		return nil
	}

	// Fall through means error
	return derp.Wrap(err, "null.Float.UnmarshalJSON", "Invalid float value", valueStr)
}
