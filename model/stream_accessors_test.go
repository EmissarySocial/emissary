package model

import (
	"math"
	"testing"
	"time"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

// TestStream_SetDateTime pins that a form's date-time string sets startDate and endDate, and that an
// empty string clears one, so an optional end date can be left blank (qwerty-event's form).
func TestStream_SetDateTime(t *testing.T) {

	stream := NewStream()

	require.True(t, stream.SetString("startDate", "2026-10-15T19:00"))
	require.Equal(t, time.Date(2026, 10, 15, 19, 0, 0, 0, time.UTC), stream.StartDate.Time)

	require.True(t, stream.SetString("endDate", "2026-10-15T21:30"))
	require.False(t, stream.EndDate.IsZero())

	require.True(t, stream.SetString("endDate", ""))
	require.True(t, stream.EndDate.IsZero(), "an empty value clears the date")

	require.False(t, stream.SetString("startDate", "not a date"))
	require.Equal(t, time.Date(2026, 10, 15, 19, 0, 0, 0, time.UTC), stream.StartDate.Time, "a bad value leaves the date alone")
}

// TestStream_DatesAreSiteRelative pins BUG-283: a typed time, and a Timeline's date-only value, read back exactly as typed
func TestStream_DatesAreSiteRelative(t *testing.T) {

	stream := NewStream()

	// A Calendar's datetime widget posts a clock reading with no zone
	require.True(t, stream.SetString("startDate", "2026-10-20T19:00"))
	value, ok := stream.StartDate.GetStringOK("datetime")
	require.True(t, ok)
	require.Equal(t, "2026-10-20T19:00", value)

	// A Timeline's date widget posts a date only, which is stored at midnight and so shows no time
	require.True(t, stream.SetString("endDate", "2026-10-22"))
	require.Equal(t, "2026-10-22", stream.EndDate.Format("2006-01-02"))
	require.Equal(t, "00:00", stream.EndDate.Format("15:04"))
}

// TestStream_PublishDateTime_Get pins the virtual "publishDateTime" reading of PublishDate, in UTC
func TestStream_PublishDateTime_Get(t *testing.T) {

	reads := func(publishDate int64, expected string) {
		t.Helper()
		stream := NewStream()
		stream.PublishDate = publishDate
		value, ok := stream.GetStringOK("publishDateTime")
		require.True(t, ok)
		require.Equal(t, expected, value, "publishDate %d", publishDate)
	}

	reads(0, "")
	reads(-1, "")
	reads(math.MaxInt64, "")
	reads(math.MaxInt64-73, "")
	reads(maxPublishDateTime, "9999-12-31T23:59:59Z")
	reads(maxPublishDateTime+1, "")
	reads(1, "1970-01-01T00:00:01Z")
	reads(time.Date(2026, 10, 15, 19, 0, 42, 0, time.UTC).Unix(), "2026-10-15T19:00:42Z")
}

// TestStream_PublishDateTime_Set pins how a posted date-time string writes PublishDate
func TestStream_PublishDateTime_Set(t *testing.T) {

	original := time.Date(2026, 10, 15, 19, 0, 42, 0, time.UTC).Unix()

	sets := func(value string, expectOK bool, expected int64) {
		t.Helper()
		stream := NewStream()
		stream.PublishDate = original
		require.Equal(t, expectOK, stream.SetString("publishDateTime", value), "value %q", value)
		require.Equal(t, expected, stream.PublishDate, "value %q", value)
	}

	// A new minute replaces the date, read as UTC
	sets("2026-11-03T08:15", true, time.Date(2026, 11, 3, 8, 15, 0, 0, time.UTC).Unix())
	sets("2026-11-03", true, time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC).Unix())
	sets("2026-11-03T08:15:00-07:00", true, time.Date(2026, 11, 3, 15, 15, 0, 0, time.UTC).Unix())

	// The same minute keeps the stored seconds, so saving a form never shifts the date
	sets("2026-10-15T19:00", true, original)
	sets("2026-10-15T19:00:42Z", true, original)

	// An empty value changes nothing, so clearing the field never unpublishes
	sets("", true, original)

	// A value that is not a time, or not after the epoch, is refused and changes nothing
	sets("not a date", false, original)
	sets("1969-12-31T23:59", false, original)
	sets("1970-01-01T00:00", false, original)
	sets("\xff", false, original)
	sets("+10000-01-01T00:00:00Z", false, original)
}

// TestStream_PublishDateTime_Unpublished pins that a Stream with no PublishDate takes any valid time
func TestStream_PublishDateTime_Unpublished(t *testing.T) {

	stream := NewStream()
	stream.PublishDate = 0

	require.True(t, stream.SetString("publishDateTime", "1970-01-01T00:00:30Z"))
	require.Equal(t, int64(30), stream.PublishDate)
}

// TestStream_PublishDateTime_Schema pins that the schema reaches the virtual field, and that a
// read followed by a write leaves PublishDate unchanged, as an edit form does on every save
func TestStream_PublishDateTime_Schema(t *testing.T) {

	s := schema.New(StreamSchema())

	for _, publishDate := range []int64{0, 1, 59, 61, time.Date(2026, 10, 15, 19, 0, 42, 0, time.UTC).Unix(), math.MaxInt64} {

		stream := NewStream()
		stream.PublishDate = publishDate

		value, err := s.Get(&stream, "publishDateTime")
		require.Nil(t, err)
		require.Nil(t, s.Set(&stream, "publishDateTime", value))
		require.Equal(t, publishDate, stream.PublishDate, "publishDate %d", publishDate)
	}

	stream := NewStream()
	require.Nil(t, s.Set(&stream, "publishDateTime", "2026-11-03T08:15"))
	require.Equal(t, time.Date(2026, 11, 3, 8, 15, 0, 0, time.UTC).Unix(), stream.PublishDate)
}

// FuzzStream_PublishDateTime checks that any posted string either sets a moment after the epoch
// or leaves PublishDate alone, and that reading the result back writes the same value
func FuzzStream_PublishDateTime(f *testing.F) {

	f.Add("2026-11-03T08:15", int64(0))
	f.Add("", int64(1793739265))
	f.Add("not a date", int64(1793739265))
	f.Add("1969-12-31T23:59", int64(0))
	f.Add("9999-12-31T23:59:59Z", int64(math.MaxInt64))

	f.Fuzz(func(t *testing.T, value string, publishDate int64) {

		stream := NewStream()
		stream.PublishDate = publishDate

		if !stream.SetString("publishDateTime", value) {
			require.Equal(t, publishDate, stream.PublishDate, "a refused value changes nothing")
			return
		}

		if stream.PublishDate != publishDate {
			require.Positive(t, stream.PublishDate)
		}

		before := stream.PublishDate
		readBack, ok := stream.GetStringOK("publishDateTime")
		require.True(t, ok)
		require.True(t, stream.SetString("publishDateTime", readBack))
		require.Equal(t, before, stream.PublishDate, "read back %q", readBack)
	})
}
