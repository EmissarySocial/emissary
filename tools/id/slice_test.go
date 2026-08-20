package id

import (
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestSchema verifies that a Slice can be read and written through the rosetta schema
func TestSchema(t *testing.T) {
	s := schema.New(SliceSchema())
	value := NewSlice()

	require.Nil(t, s.Set(&value, "0", "123456123456123456123456"))
	result, err := s.Get(&value, "0")
	require.Nil(t, err)
	require.Equal(t, "123456123456123456123456", result)

}

// TestSort verifies that Sort orders ObjectIDs ascending by hex value
func TestSort(t *testing.T) {

	id0, _ := primitive.ObjectIDFromHex("000000000000000000000000")
	id1, _ := primitive.ObjectIDFromHex("000000000000000000000001")
	id2, _ := primitive.ObjectIDFromHex("000000000000000000000002")
	id3, _ := primitive.ObjectIDFromHex("000000000000000000000003")
	id4, _ := primitive.ObjectIDFromHex("000000000000000000000004")

	slice := []primitive.ObjectID{id4, id2, id3, id0, id1}

	Sort(slice)

	require.Equal(t, id0, slice[0])
	require.Equal(t, id1, slice[1])
	require.Equal(t, id2, slice[2])
	require.Equal(t, id3, slice[3])
	require.Equal(t, id4, slice[4])
}

// TestSetValue confirms that SetValue accepts every shape that rosetta and the
// form package hand it -- including the *sliceof.String wrapper that multi-value
// widgets (multiselect, check-button-group) produce.
func TestSetValue(t *testing.T) {

	id2, _ := primitive.ObjectIDFromHex("000000000000000000000002")
	id3, _ := primitive.ObjectIDFromHex("000000000000000000000003")

	{ // Native slice of ObjectIDs
		value := NewSlice()
		require.Nil(t, value.SetValue([]primitive.ObjectID{id2, id3}))
		require.Equal(t, Slice{id2, id3}, value)
	}

	{ // Another id.Slice
		value := NewSlice()
		require.Nil(t, value.SetValue(Slice{id2}))
		require.Equal(t, Slice{id2}, value)
	}

	{ // Single ObjectID
		value := NewSlice()
		require.Nil(t, value.SetValue(id3))
		require.Equal(t, Slice{id3}, value)
	}

	{ // Slice of hex strings
		value := NewSlice()
		require.Nil(t, value.SetValue([]string{"000000000000000000000002", "000000000000000000000003"}))
		require.Equal(t, Slice{id2, id3}, value)
	}

	{ // Single hex string
		value := NewSlice()
		require.Nil(t, value.SetValue("000000000000000000000002"))
		require.Equal(t, Slice{id2}, value)
	}

	{ // *sliceof.String -- the shape posted by multiselect widgets
		posted := sliceof.String{"000000000000000000000002", "000000000000000000000003"}
		value := NewSlice()
		require.Nil(t, value.SetValue(&posted))
		require.Equal(t, Slice{id2, id3}, value)
	}

	{ // Empty *sliceof.String clears the slice (un-checking every option)
		posted := sliceof.String{}
		value := Slice{id2, id3}
		require.Nil(t, value.SetValue(&posted))
		require.Zero(t, value.Length())
	}

	{ // NIL clears the slice
		value := Slice{id2, id3}
		require.Nil(t, value.SetValue(nil))
		require.Zero(t, value.Length())
	}

	{ // Empty strings are dropped, not converted into zero ObjectIDs
		value := NewSlice()
		require.Nil(t, value.SetValue([]string{"", "000000000000000000000002", ""}))
		require.Equal(t, Slice{id2}, value)
	}

	{ // Invalid hex values are reported as errors
		value := NewSlice()
		require.Error(t, value.SetValue([]string{"not-an-objectId"}))
	}

	{ // Values that are not string-like are reported as errors
		value := NewSlice()
		require.Error(t, value.SetValue(struct{}{}))
	}
}
