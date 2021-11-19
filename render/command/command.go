package command

import (
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/steranko"
)

type Command interface {
	Get(interface{}) (string, error)
	Post(*steranko.Context, *model.Stream) error
}

// New uses an ActionConfig object to create a new action
func New(factory Factory, command datatype.Map) (Command, error) {

	// Populate the action with the data from
	switch command["method"] {

	case "create-stream":
		return NewCreateStream(factory, command), nil

	case "create-top-stream":
		return NewCreateTopStream(factory, command), nil

	case "delete-stream":
		return NewDeleteStream(factory, command), nil

	case "delete-draft":
		return NewDeleteDraft(factory, command), nil

	case "publish-draft":
		return NewPublishDraft(factory, command), nil

	case "update-draft":
		return NewUpdateDraft(factory, command), nil

	case "update-data":
		return NewUpdateData(factory, command), nil

	case "update-state":
		return NewUpdateState(factory, command), nil

	case "view-stream":
		return NewViewStream(factory, command), nil
	}

	// Fall through means we have an unrecognized action
	return nil, derp.New(derp.CodeInternalError, "ghost.remder.NewAction", "Invalid action configuration", command)
}
