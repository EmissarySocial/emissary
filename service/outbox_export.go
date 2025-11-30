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

func (service *Outbox) ExportCollection(session data.Session, userID primitive.ObjectID) ([]model.IDOnly, error) {
	criteria := exp.Equal("userId", userID)
	return service.QueryIDOnly(session, criteria, option.SortAsc("createDate"))
}

func (service *Outbox) ExportDocument(session data.Session, userID primitive.ObjectID, outboxMessageID primitive.ObjectID) (string, error) {

	const location = "service.Outbox.ExportDocument"

	// Load the Outbox
	outboxMessage := model.NewOutboxMessage()
	if err := service.LoadByID(session, userID, outboxMessageID, &outboxMessage); err != nil {
		return "", derp.Wrap(err, location, "Unable to load Outbox")
	}

	// Marshal the outboxMessage as JSON
	result, err := json.Marshal(outboxMessage)

	if err != nil {
		return "", derp.Wrap(err, location, "Unable to marshal Outbox", outboxMessage)
	}

	// Success
	return string(result), nil
}
