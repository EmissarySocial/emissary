package id

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

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
