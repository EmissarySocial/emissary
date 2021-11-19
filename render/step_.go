package render

import (
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
)

type Step interface {
	Get(*Renderer) error
	Post(*Renderer) error
}

// NewStep uses an ActionConfig object to create a new action
func NewStep(factory Factory, config datatype.Map) (Step, error) {

	// Populate the action with the data from
	switch config["method"] {

	case "create-stream":
		return NewCreateStream(factory, config), nil

	case "create-top-stream":
		return NewCreateTopStream(factory, config), nil

	case "delete-stream":
		return NewDeleteStream(factory, config), nil

	case "delete-draft":
		return NewDeleteDraft(factory, config), nil

	case "publish-draft":
		return NewPublishDraft(factory, config), nil

	case "update-draft":
		return NewUpdateDraft(factory, config), nil

	case "update-data":
		return NewUpdateData(factory, config), nil

	case "update-state":
		return NewUpdateState(factory, config), nil

	case "view-stream":
		return NewViewStream(factory, config), nil
	}

	// Fall through means we have an unrecognized action
	return nil, derp.New(derp.CodeInternalError, "ghost.render.NewStep", "Invalid action configuration", config)
}
