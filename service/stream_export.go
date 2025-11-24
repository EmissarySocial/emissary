package service

import (
	"encoding/json"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (service *Stream) ExportCollection(session data.Session, userID primitive.ObjectID) ([]model.IDOnly, error) {
	criteria := exp.Equal("navigationId", "profile").AndEqual("parentIds", userID)
	return service.QueryIDOnly(session, criteria, option.SortAsc("createDate"))
}

func (service *Stream) ExportDocument(session data.Session, userID primitive.ObjectID, streamID primitive.ObjectID) (string, error) {

	const location = "service.Stream.ExportDocument"

	// Load the Stream
	stream := model.NewStream()
	if err := service.LoadByID(session, streamID, &stream); err != nil {
		return "", derp.Wrap(err, location, "Unable to load Stream")
	}

	// RULE: Verify that the Stream is owned by the provided User
	if stream.ParentIDs.NotContains(userID) {
		return "", derp.NotFound(location, "Stream is not in User's profile", "userID: "+userID.Hex(), "streamID: "+streamID.Hex())
	}

	// Marshal the stream as JSON
	result, err := json.Marshal(stream)

	if err != nil {
		return "", derp.Wrap(err, location, "Unable to marshal Stream", stream)
	}

	// Success
	return string(result), nil
}
