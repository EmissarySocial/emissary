package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/whisperverse/whisperverse/config"
	"github.com/whisperverse/whisperverse/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestService(t *testing.T) {

	factory, err := NewFactory(config.Domain{
		ConnectString: "mongodb://127.0.0.1/whisper",
		DatabaseName:  "whisper",
	})

	require.Nil(t, err)

	streamService := factory.Stream()

	streamID, err := primitive.ObjectIDFromHex("5f84e964e49c4c226eb51097")
	require.Nil(t, err)

	it, err := streamService.ListByParent(streamID)

	require.Nil(t, err)

	stream := model.Stream{}

	for it.Next(&stream) {
	}
}
