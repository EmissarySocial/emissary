package datetime

import (
	"strconv"
	"strings"
	"time"

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

		if location, err := time.LoadLocation(value); err == nil {
			dt.Time = time.Date(dt.Year(), dt.Month(), dt.Day(), dt.Hour(), dt.Minute(), dt.Second(), dt.Nanosecond(), location)
			return true
		}
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

func (dt DateTime) ToTime() time.Time {
	return dt.Time
}

func (dt DateTime) DateOnly() time.Time {
	return dt.Time.Truncate(24 * time.Hour)
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

func splitTime(value string) (hours int, minutes int, seconds int) {

	timeParts := strings.Split(value, ":")

	if len(timeParts) > 0 {

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

func (dt *DateTime) UnmarshalBSONValue(bsonType bsontype.Type, data []byte) error {
	return bson.UnmarshalValue(bson.TypeDateTime, data, &dt.Time)
}
