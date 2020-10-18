package render

import (
	"context"
	"testing"

	"github.com/benpate/data/expression"
	"github.com/benpate/data/mongodb"
	"github.com/benpate/ghost/model"
	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestDatabase(t *testing.T) {

	value := model.Stream{}

	server, err := mongodb.New("mongodb+srv://sandbox:MLcnoRwsgzqtfdKDbUuEQqP7WwQhtPTNyUHfhQtDLV@cluster0.wfvvk.mongodb.net/ghost?retryWrites=true&w=majority", "ghost")
	require.Nil(t, err)

	session, err := server.Session(context.TODO())

	collection := session.Collection("Stream")
 	streamID, err := primitive.ObjectIDFromHex("5f84e964e49c4c226eb51097")
//	streamID, err := primitive.ObjectIDFromHex("000000000000000000000000")
	require.Nil(t, err)

	iterator, err := collection.List(expression.Equal("parentId", streamID).And("journal.deleteDate", "=", 0))

	require.Nil(t, err)

	require.True(t, iterator.Next(&value))

	spew.Dump(value)

	require.Nil(t, err)
}