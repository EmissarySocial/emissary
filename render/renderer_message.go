package render

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/steranko"
)

type Message struct {
	Model
}

func NewMessage(factory Factory, ctx *steranko.Context, inboxService *service.Inbox, message *model.Message, template model.Template, actionID string) (Message, error) {
	model, err := NewModel(factory, ctx, inboxService, message, template, actionID)

	if err != nil {
		return Message{}, err
	}

	return Message{
		Model: model,
	}, nil

}

func (w Message) templateRole() string {
	return "inbox"
}
