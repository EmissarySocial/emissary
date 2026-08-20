package datetime

import (
	"strconv"
	"strings"
	"time"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/schema"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
)

// DateTime wraps a time.Time so that its date, time, and timezone can be read and written separately by forms and schemas
type DateTime struct {
	time.Time
}

// New returns a zero-value DateTime
func New() DateTime {
	return DateTime{time.Time{}}
}

// Schema returns the rosetta schema that describes a DateTime
func Schema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"date":     schema.String{},
			"time":     schema.String{},
			"datetime": schema.String{},
			"timezone": schema.String{},
			"unix":     schema.Integer{BitSize: 64},
		},
	}
}

/******************************************
 * Getters
 ******************************************/

// GetValue returns this DateTime as an RFC-3339 string
func (dt DateTime) GetValue() any {
	return dt.String()
}

// String returns this DateTime in RFC-3339 format, or an empty string if it is zero
func (dt DateTime) String() string {

	if dt.IsZero() {
		return ""
	}

	return dt.Format(time.RFC3339)
}

// GetStringOK implements the schema.StringGetter interface for the "date", "datetime", "time", and "timezone" properties
func (dt DateTime) GetStringOK(property string) (string, bool) {

	switch property {

	case "date":

		if dt.IsZero() {
			return "", true
		}
		return dt.Format("2006-01-02"), true

	case "datetime":
		if dt.IsZero() {
			return "", true
		}
		return dt.Format("2006-01-02T15:04"), true

	case "time":

		result := dt.Format("15:04")

		if result == "00:00" {
			return "", true
		}

		return result, true

	case "timezone":
		return dt.Location().String(), true
	}

	return "", false
}

// GetInt64OK implements the schema.Int64Getter interface for the "unix" property
func (dt DateTime) GetInt64OK(property string) (int64, bool) {

	switch property {

	case "unix":
		return int64(dt.Unix()), true
	}

	return 0, false
}

// Timezone returns the abbreviated name of this DateTime's timezone
func (dt DateTime) Timezone() string {
	result, _ := dt.Zone()
	return result
}

/******************************************
 * Setters
 ******************************************/

// SetString implements the schema.StringSetter interface for the "date", "datetime", "time", and "timezone" properties
func (dt *DateTime) SetString(property string, value string) bool {

	switch property {

	case "date":

		// Special case to CLEAR the date
		if value == "" {
			dt.Time = time.Date(0, 1, 1, dt.Hour(), dt.Minute(), dt.Second(), dt.Nanosecond(), dt.Location())
			return true
		}

		// Otherwise, SET the date
		if newValue, err := time.Parse("2006-01-02", value); err == nil {
			dt.Time = time.Date(newValue.Year(), newValue.Month(), newValue.Day(), dt.Hour(), dt.Minute(), dt.Second(), dt.Nanosecond(), dt.Location())
			return true
		}

	case "time":

		// Special case to CLEAR the time
		if value == "" {
			dt.Time = time.Date(dt.Year(), dt.Month(), dt.Day(), 0, 0, 0, 0, dt.Location())
			return true
		}

		// Otherwise, SET the time
		hours, minutes, seconds := splitTime(value)

		dt.Time = time.Date(dt.Year(), dt.Month(), dt.Day(), hours, minutes, seconds, 0, dt.Location())
		return true

	case "datetime":

		if newValue, err := time.Parse("2006-01-02T15:04", value); err == nil {
			dt.Time = newValue
			return true
		}

	case "timezone":
		return dt.SetTimezone(value) == nil
	}

	return false
}

// SetInt64 implements the schema.Int64Setter interface for the "unix" property
func (dt *DateTime) SetInt64(property string, value int64) bool {

	switch property {

	case "unix":
		dt.Time = time.Unix(value, 0)
		return true
	}

	return false
}

// SetDatetime replaces the entire value with the provided time
func (dt *DateTime) SetDatetime(value time.Time) error {
	dt.Time = value
	return nil
}

// SetDate replaces the calendar date, leaving the clock time unchanged
func (dt *DateTime) SetDate(value time.Time) error {
	dt.Time = time.Date(value.Year(), value.Month(), value.Day(), dt.Hour(), dt.Minute(), dt.Second(), dt.Nanosecond(), value.Location())
	return nil
}

// SetTime replaces the clock time, leaving the calendar date unchanged
func (dt *DateTime) SetTime(value time.Time) error {
	dt.Time = time.Date(dt.Year(), dt.Month(), dt.Day(), value.Hour(), value.Minute(), value.Second(), value.Nanosecond(), dt.Location())
	return nil
}

// SetTimezone moves this DateTime into the named IANA timezone, defaulting to UTC when the name is empty
func (dt *DateTime) SetTimezone(timezone string) error {

	const location = "datetime.SetTimezone"

	var newLocation *time.Location

	if timezone == "" {
		newLocation = time.UTC
	} else {

		var err error

		newLocation, err = time.LoadLocation(timezone)

		if err != nil {
			return derp.Wrap(err, location, "Setting timezone", timezone)
		}
	}

	dt.Time = time.Date(dt.Year(), dt.Month(), dt.Day(), dt.Hour(), dt.Minute(), dt.Second(), dt.Nanosecond(), newLocation)

	return nil
}

/******************************************
 * Conversion Methods
 ******************************************/

// ToTime returns the underlying time.Time
func (dt DateTime) ToTime() time.Time {
	return dt.Time
}

// DateOnly returns this DateTime truncated to the start of its day
func (dt DateTime) DateOnly() time.Time {
	return dt.Truncate(24 * time.Hour)
}

// TimeOnly returns this DateTime's clock time, on the zero date
func (dt DateTime) TimeOnly() time.Time {
	return time.Date(0, 1, 1, dt.Hour(), dt.Minute(), dt.Second(), dt.Nanosecond(), dt.Location())
}

// IsMidnight returns TRUE if this DateTime's clock time is exactly 00:00:00
func (dt DateTime) IsMidnight() bool {
	if dt.Hour() > 0 {
		return false
	}

	if dt.Minute() > 0 {
		return false
	}

	if dt.Second() > 0 {
		return false
	}

	return true
}

// NotMidnight returns TRUE if this DateTime has a clock time other than midnight
func (dt DateTime) NotMidnight() bool {
	return !dt.IsMidnight()
}

// NotZero returns TRUE if this DateTime has been populated
func (dt DateTime) NotZero() bool {
	return !dt.IsZero()
}

// MissingTimezone returns TRUE if this DateTime has a value, but no named timezone
func (dt DateTime) MissingTimezone() bool {
	if dt.IsZero() {
		return false
	}

	return dt.Timezone() == ""
}

// splitTime parses an "HH:MM:SS" string into its numeric parts, defaulting each missing part to zero
func splitTime(value string) (hours int, minutes int, seconds int) {

	if timeParts := strings.Split(value, ":"); len(timeParts) > 0 {

		hours, _ = strconv.Atoi(timeParts[0])

		if len(timeParts) > 1 {
			minutes, _ = strconv.Atoi(timeParts[1])

			if len(timeParts) > 2 {
				seconds, _ = strconv.Atoi(timeParts[2])
			}
		}
	}

	return hours, minutes, seconds
}

/******************************************
 * Marshalling/Unmarshalling
 ******************************************/

// MarshalJSON implements the json.Marshaler interface
func (dt DateTime) MarshalJSON() ([]byte, error) {
	return dt.Time.MarshalJSON()
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (dt *DateTime) UnmarshalJSON(data []byte) error {
	return dt.Time.UnmarshalJSON(data)
}

// MarshalText implements the encoding.TextMarshaler interface
func (dt DateTime) MarshalText() ([]byte, error) {
	return dt.Time.MarshalText()
}

// UnmarshalText implements the encoding.TextUnmarshaler interface
func (dt *DateTime) UnmarshalText(data []byte) error {
	return dt.Time.UnmarshalText(data)
}

// MarshalBinary implements the encoding.BinaryMarshaler interface
func (dt DateTime) MarshalBinary() ([]byte, error) {
	return dt.Time.MarshalBinary()
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface
func (dt *DateTime) UnmarshalBinary(data []byte) error {
	return dt.Time.UnmarshalBinary(data)
}

// MarshalBSONValue implements the bson.ValueMarshaler interface
func (dt DateTime) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bson.MarshalValue(dt.Time)
}

// UnmarshalBSONValue implements the bson.ValueUnmarshaler interface
func (dt *DateTime) UnmarshalBSONValue(t bsontype.Type, data []byte) error {
	err := bson.UnmarshalValue(t, data, &dt.Time)

	return err
}
