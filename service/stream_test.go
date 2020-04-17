package service

import (
	"testing"

	"github.com/benpate/data"
	"github.com/benpate/data/expression"
	"github.com/benpate/data/mock"
	"github.com/benpate/ghost/model"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestStream1(t *testing.T) {

	datasource := mock.New()

	factory := NewFactory(datasource)

	service := factory.Stream()
	stream1 := service.New()

	stream1.StreamID = primitive.NewObjectID()
	stream1.Label = "My First Stream"
	stream1.Token = "my-first-stream"

	if err := service.Save(stream1, "This is the first record I'm going to save."); err != nil {
		t.Error(err)
		return
	}

	stream2, err := service.Load(expression.New("token", "=", "my-first-stream"))

	if err != nil {
		t.Error(err)
		return
	}

	assert.Equal(t, stream1.StreamID, stream2.StreamID)
	assert.Equal(t, "My First Stream", stream2.Label)
	assert.Equal(t, "my-first-stream", stream2.Token)
}

func TestStream2(t *testing.T) {

	service := getTestStreamService()

	service.Load(expression.New())
	datasource := mock.New()

	factory := NewFactory(datasource)

	service := factory.Stream()

	stream1 := service.New()
	stream1.StreamID = primitive.NewObjectID()
	stream1.Label = "My First Stream"
	stream1.Token = "my-first-stream"

	if err := service.Save(stream1, "This is the first record I'm going to save."); err != nil {
		t.Error(err)
		return
	}

	stream2 := service.New()
	stream2.StreamID = primitive.NewObjectID()
	stream2.Label = "My Second Stream"
	stream2.Token = "my-second-stream"

	if err := service.Save(stream2, "This is the second record I'm going to save."); err != nil {
		t.Error(err)
		return
	}

	stream3 := service.New()
	stream3.StreamID = primitive.NewObjectID()
	stream3.Label = "My Third Stream"
	stream3.Token = "my-third-stream"

	if err := service.Save(stream3, "This is the third record I'm going to save."); err != nil {
		t.Error(err)
		return
	}

	criteria := data.NewExpression()

	service.List(criteria)
}

func getTestStreamService(t *testing.T) Stream {

	data := []model.Stream{
		model.Stream{
			Label: "My First Stream",
			Token: "my-first-stream",
		},
		model.Stream{
			Label: "My Second Stream",
			Token: "my-second-stream",
		},
		model.Stream{
			Label: "My Third Stream",
			Token: "my-third-stream",
		},
	}

	datasource := mock.New()
	factory := NewFactory(datasource)
	service := factory.Stream()

	for _, record := range data {
		if err := service.Save(record); err != nil {
			t.Error(err)
		}
	}

	return service
}
