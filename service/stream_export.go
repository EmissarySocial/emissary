package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/mapof"
	"github.com/go-viper/mapstructure/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (service *Stream) ExportCollection(session data.Session, userID primitive.ObjectID) ([]model.IDOnly, error) {
	criteria := exp.Equal("navigationId", "profile").AndEqual("parentIds", userID)
	return service.QueryIDOnly(session, criteria, option.SortAsc("createDate"))
}

func (service *Stream) ExportRecord(session data.Session, userID primitive.ObjectID, streamID primitive.ObjectID) (mapof.Any, error) {

	const location = "service.Stream.ExportRecord"

	stream := model.NewStream()
	if err := service.LoadByID(session, streamID, &stream); err != nil {
		return nil, derp.Wrap(err, location, "Unable to load Stream")
	}

	if stream.ParentIDs.NotContains(userID) {
		return nil, derp.NotFound(location, "Stream is not in User's profile", "userID: "+userID.Hex(), "streamID: "+streamID.Hex())
	}

	result := make(map[string]any)
	if err := mapstructure.Decode(stream, &result); err != nil {
		return nil, derp.Wrap(err, location, "Unable to encode Stream as a Map")
	}

	return result, nil
}
