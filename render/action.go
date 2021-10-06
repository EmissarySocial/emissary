package render

import (
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/steranko"
)

// Action configures an individual action function that will be executed when a stream transitions from one state to another.
type Action interface {
	Get(Renderer) (string, error)
	Post(*steranko.Context, *model.Stream) error
	UserCan(*model.Stream, *model.Authorization) bool
}

// NewAction locates and populates the action.Action for a specific template and actionID
func NewAction(factory Factory, stream *model.Stream, authorization *model.Authorization, actionID string) (Action, error) {

	// Try to find the action based on the stream and actionID
	templateService := factory.Template()
	actionConfig, err := templateService.ActionConfig(stream.TemplateID, actionID)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.render.NewAction", "Could not create action", stream, actionID)
	}

	// Enforce user permissions here.
	if !actionConfig.UserCan(stream, authorization) {
		return nil, derp.New(derp.CodeForbiddenError, "ghost.render.NewAction", "Forbidden", stream, authorization)
	}

	return NewActionFromConfig(factory, actionConfig)
}

// NewActionFromConfig uses an ActionConfig object to create a new action
func NewActionFromConfig(factory Factory, actionConfig model.ActionConfig) (Action, error) {

	// Populate the action with the data from
	switch actionConfig.Method {

	case "create-stream":
		return NewAction_CreateStream(factory, actionConfig), nil

	case "create-top-stream":
		return NewAction_CreateTopStream(factory, actionConfig), nil

	case "delete-stream":
		return NewAction_DeleteStream(factory, actionConfig), nil

	case "delete-draft":
		return NewAction_DeleteDraft(factory, actionConfig), nil

	case "publish-draft":
		return NewAction_PublishDraft(factory, actionConfig), nil

	case "update-draft":
		return NewAction_UpdateDraft(factory, actionConfig), nil

	case "update-data":
		return NewAction_UpdateData(factory, actionConfig), nil

	case "update-state":
		return NewAction_UpdateState(factory, actionConfig), nil

	case "view-stream":
		return NewAction_ViewStream(factory, actionConfig), nil
	}

	// Fall through means we have an unrecognized action
	return nil, derp.New(derp.CodeInternalError, "ghost.remder.NewAction", "Invalid action configuration", actionConfig)
}
