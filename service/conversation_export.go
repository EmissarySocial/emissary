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

func (service *Conversation) ExportCollection(session data.Session, userID primitive.ObjectID) ([]model.IDOnly, error) {
	criteria := exp.Equal("userId", userID)
	return service.QueryIDOnly(session, criteria, option.SortAsc("createDate"))
}

func (service *Conversation) ExportDocument(session data.Session, userID primitive.ObjectID, conversationID primitive.ObjectID) (string, error) {

	const location = "service.Conversation.ExportDocument"

	// Load the Conversation
	conversation := model.NewConversation()
	if err := service.LoadByID(session, userID, conversationID, &conversation); err != nil {
		return "", derp.Wrap(err, location, "Unable to load Conversation")
	}

	// Marshal the conversation as JSON
	result, err := json.Marshal(conversation)

	if err != nil {
		return "", derp.Wrap(err, location, "Unable to marshal Conversation", conversation)
	}

	// Success
	return string(result), nil
}
