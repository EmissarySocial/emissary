package domain

import (
	"github.com/benpate/ghost/action"
	"github.com/benpate/ghost/model"
)

func (factory *Factory) ParseAction(config *model.ActionConfig) action.Action {

	switch config.Method {

	case "create-stream":
		return action.NewAction_CreateStream(config, factory.Stream())

	case "create-top-stream":
		return action.NewAction_CreateTopStream(config, factory.Stream())

	case "delete-stream":
		return action.NewAction_DeleteStream(config, factory.Stream())

	case "publish-content":
		return action.NewAction_PublishContent(config, factory.Stream())

	case "update-content":
		return action.NewAction_UpdateContent(config, factory.Stream())

	case "update-data":
		return action.NewAction_UpdateData(config, factory.Template(), factory.Stream(), factory.FormLibrary())

	case "update-state":
		return action.NewAction_UpdateState(config, factory.Template(), factory.Stream(), factory.FormLibrary())

	// case "view-stream":
	default:
		return action.NewAction_ViewStream(config, factory.Layout())
	}

}
