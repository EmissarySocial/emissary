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

type DateTime struct {
	time.Time
}

func New() DateTime {
	return DateTime{time.Time{}}
}

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

func (dt DateTime) GetValue() any {
	return dt.String()
}

func (dt DateTime) String() string {

	if dt.IsZero() {
		return ""
	}

	return dt.Format(time.RFC3339)
}

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

func (dt DateTime) GetInt64OK(property string) (int64, bool) {

	switch property {

	case "unix":
		return int64(dt.Unix()), true
	}

	return 0, false
}

func (dt DateTime) Timezone() string {
	result, _ := dt.Time.Zone()
	return result
}

/******************************************
 * Setters
 ******************************************/

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

func (dt *DateTime) SetInt64(property string, value int64) bool {

	switch property {

	case "unix":
		dt.Time = time.Unix(value, 0)
		return true
	}

	return false
}

func (dt *DateTime) SetDatetime(value time.Time) error {
	dt.Time = value
	return nil
}

func (dt *DateTime) SetDate(value time.Time) error {
	dt.Time = time.Date(value.Year(), value.Month(), value.Day(), dt.Hour(), dt.Minute(), dt.Second(), dt.Nanosecond(), value.Location())
	return nil
}

func (dt *DateTime) SetTime(value time.Time) error {
	dt.Time = time.Date(dt.Year(), dt.Month(), dt.Day(), value.Hour(), value.Minute(), value.Second(), value.Nanosecond(), dt.Location())
	return nil
}

func (dt *DateTime) SetTimezone(timezone string) error {

	const location = "datetime.SetTimezone"

	var newLocation *time.Location

	if timezone == "" {
		newLocation = time.UTC
	} else {

		var err error

		newLocation, err = time.LoadLocation(timezone)

		if err != nil {
			return derp.Wrap(err, location, "Unable to set timezone", timezone)
		}
	}

	dt.Time = time.Date(dt.Year(), dt.Month(), dt.Day(), dt.Hour(), dt.Minute(), dt.Second(), dt.Nanosecond(), newLocation)

	return nil
}

/******************************************
 * Conversion Methods
 ******************************************/

func (dt DateTime) ToTime() time.Time {
	return dt.Time
}

func (dt DateTime) DateOnly() time.Time {
	return dt.Truncate(24 * time.Hour)
}

func (dt DateTime) TimeOnly() time.Time {
	return time.Date(0, 1, 1, dt.Hour(), dt.Minute(), dt.Second(), dt.Nanosecond(), dt.Location())
}

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

func (dt DateTime) NotMidnight() bool {
	return !dt.IsMidnight()
}
func (dt DateTime) NotZero() bool {
	return !dt.IsZero()
}

func (dt DateTime) MissingTimezone() bool {
	if dt.IsZero() {
		return false
	}

	return dt.Timezone() == ""
}

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

func (dt DateTime) MarshalJSON() ([]byte, error) {
	return dt.Time.MarshalJSON()
}

func (dt *DateTime) UnmarshalJSON(data []byte) error {
	return dt.Time.UnmarshalJSON(data)
}

func (dt DateTime) MarshalText() ([]byte, error) {
	return dt.Time.MarshalText()
}

func (dt *DateTime) UnmarshalText(data []byte) error {
	return dt.Time.UnmarshalText(data)
}

func (dt DateTime) MarshalBinary() ([]byte, error) {
	return dt.Time.MarshalBinary()
}

func (dt *DateTime) UnmarshalBinary(data []byte) error {
	return dt.Time.UnmarshalBinary(data)
}

func (dt DateTime) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bson.MarshalValue(dt.Time)
}

func (dt *DateTime) UnmarshalBSONValue(t bsontype.Type, data []byte) error {
	err := bson.UnmarshalValue(t, data, &dt.Time)

	return err
}
