package datetime

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

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
