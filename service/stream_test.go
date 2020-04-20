package service

import (
	"context"
	"testing"

	"github.com/benpate/data/expression"
	"github.com/benpate/data/mockdb"
	"github.com/benpate/data/option"
	"github.com/benpate/ghost/model"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestStream_ReadWrite(t *testing.T) {

	service := getTestStreamService()

	stream1 := service.New()
	stream1.StreamID = primitive.NewObjectID()
	stream1.Label = "My New Stream"
	stream1.Token = "my-new-stream"

	if err := service.Save(stream1, "This is the first record I'm going to save."); err != nil {
		t.Error(err)
		return
	}

	stream2, err := service.Load(expression.New("token", "=", "my-new-stream"))

	if err != nil {
		t.Error(err)
		return
	}

	assert.Equal(t, stream1.StreamID, stream2.StreamID)
	assert.Equal(t, "My New Stream", stream2.Label)
	assert.Equal(t, "my-new-stream", stream2.Token)
}

func TestStream_List(t *testing.T) {

	service := getTestStreamService()
	stream := service.New()

	it, err := service.List(nil, option.SortDesc("token"))

	assert.Nil(t, err)

	assert.True(t, it.Next(stream))
	assert.Equal(t, "3-my-third-stream", stream.Token)

	assert.True(t, it.Next(stream))
	assert.Equal(t, "2-my-second-stream", stream.Token)

	assert.True(t, it.Next(stream))
	assert.Equal(t, "1-my-first-stream", stream.Token)

	assert.False(t, it.Next(stream))
	assert.False(t, it.Next(stream))
	assert.False(t, it.Next(stream))
	assert.False(t, it.Next(stream))
}

func getTestStreamService() Stream {

	// Create service
	datasource := mockdb.New()
	factory := NewFactoryMaker(datasource).Factory(context.TODO())
	service := factory.Stream()

	// Initial data to load
	data := []*model.Stream{
		{
			StreamID: primitive.NewObjectID(),
			Label:    "My First Stream",
			Token:    "1-my-first-stream",
		},
		{
			StreamID: primitive.NewObjectID(),
			Label:    "My Second Stream",
			Token:    "2-my-second-stream",
		},
		{
			StreamID: primitive.NewObjectID(),
			Label:    "My Third Stream",
			Token:    "3-my-third-stream",
		},
	}

	// Populate datasource
	for _, record := range data {
		if err := service.Save(record, "comment"); err != nil {
			panic(err)
		}
	}

	return service
}
