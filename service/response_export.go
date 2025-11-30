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

func (service *Response) ExportCollection(session data.Session, userID primitive.ObjectID) ([]model.IDOnly, error) {
	criteria := exp.Equal("userId", userID)
	return service.QueryIDOnly(session, criteria, option.SortAsc("createDate"))
}

func (service *Response) ExportDocument(session data.Session, userID primitive.ObjectID, responseID primitive.ObjectID) (string, error) {

	const location = "service.Response.ExportDocument"

	// Load the Response
	response := model.NewResponse()
	if err := service.LoadByID(session, userID, responseID, &response); err != nil {
		return "", derp.Wrap(err, location, "Unable to load Response")
	}

	// Marshal the response as JSON
	result, err := json.Marshal(response)

	if err != nil {
		return "", derp.Wrap(err, location, "Unable to marshal Response", response)
	}

	// Success
	return string(result), nil
}
