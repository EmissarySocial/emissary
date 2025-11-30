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

func (service *Inbox) ExportCollection(session data.Session, userID primitive.ObjectID) ([]model.IDOnly, error) {
	criteria := exp.Equal("userId", userID)
	return service.QueryIDOnly(session, criteria, option.SortAsc("createDate"))
}

func (service *Inbox) ExportDocument(session data.Session, userID primitive.ObjectID, messageID primitive.ObjectID) (string, error) {

	const location = "service.Inbox.ExportDocument"

	// Load the Inbox
	message := model.NewMessage()
	if err := service.LoadByID(session, userID, messageID, &message); err != nil {
		return "", derp.Wrap(err, location, "Unable to load Inbox")
	}

	// Marshal the message as JSON
	result, err := json.Marshal(message)

	if err != nil {
		return "", derp.Wrap(err, location, "Unable to marshal Inbox", message)
	}

	// Success
	return string(result), nil
}
