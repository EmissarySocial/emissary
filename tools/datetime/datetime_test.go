package datetime

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

// TestDateTime verifies that every schema property round-trips through Set and Get
func TestDateTime(t *testing.T) {

	test := func(path string, value any) {

		dt := New()
		s := schema.New(Schema())

		err := s.Set(&dt, path, value)
		require.Nil(t, err)

		result, err := s.Get(&dt, path)
		require.Nil(t, err)
		require.Equal(t, value, result)
	}

	test("date", "2021-01-02")
	test("time", "15:04")
	test("datetime", "2021-01-02T15:04")
	test("timezone", "UTC")
	test("unix", int64(1609542240))
}

// TestDateTime_JSON verifies that a DateTime survives a JSON marshal/unmarshal round-trip
func TestDateTime_JSON(t *testing.T) {

	test := func(value DateTime) {

		result, err := json.Marshal(value)
		require.Nil(t, err)

		var newValue DateTime
		err = json.Unmarshal(result, &newValue)
		require.Nil(t, err)
		require.Equal(t, value, newValue)
	}

	test(DateTime{Time: time.Now().UTC()})
	test(DateTime{Time: time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)})
}

/******************************************
 * BSON format tests
 *
 * DateTime has no plain MarshalBSON to fall back on, so if it
 * stops satisfying the *Value interfaces the driver silently
 * encodes it with the default struct codec instead.
 ******************************************/

// DateTime must satisfy both *Value BSON interfaces. Losing either one changes
// the shape this type writes to the database.
var (
	_ bson.ValueMarshaler   = DateTime{}
	_ bson.ValueUnmarshaler = &DateTime{}
)

// TestDateTime_FieldFormatLock pins the bytes a DateTime writes as a struct
// field. A DateTime is a BSON datetime, not a document.
func TestDateTime_FieldFormatLock(t *testing.T) {

	// Marshalled as a FIELD, not at the top level: a top-level Marshal takes the
	// ValueMarshaler path directly and never consults the codec registry, so it
	// cannot observe this class of bug.
	data, err := bson.Marshal(struct {
		StartDate DateTime `bson:"startDate"`
	}{
		StartDate: DateTime{Time: time.Date(2023, 11, 14, 22, 13, 20, 0, time.UTC)},
	})

	require.Nil(t, err)
	require.Equal(t,
		`{"startDate": {"$date":{"$numberLong":"1700000000000"}}}`,
		bson.Raw(data).String(),
		"format changed -- a DateTime field must stay a BSON datetime")
}

// TestDateTime_FieldRoundTrip confirms a DateTime field decodes back to the
// value it was stored from, at the millisecond precision BSON preserves.
func TestDateTime_FieldRoundTrip(t *testing.T) {

	type wrapper struct {
		StartDate DateTime `bson:"startDate"`
	}

	original := wrapper{StartDate: DateTime{Time: time.Date(2023, 11, 14, 22, 13, 20, 0, time.UTC)}}

	data, err := bson.Marshal(original)
	require.Nil(t, err)

	result := wrapper{}
	require.Nil(t, bson.Unmarshal(data, &result))
	require.True(t, original.StartDate.Time.Equal(result.StartDate.Time),
		"want %s, got %s", original.StartDate.Time, result.StartDate.Time)
}
