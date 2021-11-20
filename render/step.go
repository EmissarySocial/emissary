package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
)

type Step interface {
	Get(io.Writer, *Renderer) error
	Post(io.Writer, *Renderer) error
}

// NewStep uses an Step object to create a new action
func NewStep(factory Factory, stepInfo datatype.Map) (Step, error) {

	// Populate the action with the data from
	switch stepInfo["method"] {

	case "create-stream":
		return NewCreateStream(factory.Stream(), stepInfo), nil

	case "create-top-stream":
		return NewCreateTopStream(factory.Stream(), stepInfo), nil

	case "delete-stream":
		return NewDeleteStream(factory.Stream(), stepInfo), nil

	case "draft-edit":
		return NewDraftEdit(factory.StreamDraft(), stepInfo), nil

	case "draft-delete":
		return NewDraftDelete(factory.StreamDraft(), stepInfo), nil

	case "draft-publish":
		return NewDraftPublish(factory.Stream(), factory.StreamDraft(), stepInfo), nil

	case "update-data":
		return NewUpdateData(factory.Template(), factory.Stream(), factory.FormLibrary(), stepInfo), nil

	case "update-state":
		return NewUpdateState(factory.Template(), factory.Stream(), factory.FormLibrary(), stepInfo), nil

	case "view-stream":
		return NewViewStream(stepInfo), nil
	}

	// Fall through means we have an unrecognized action
	return nil, derp.New(derp.CodeInternalError, "ghost.factory.RenderStep", "Unrecognized action configuration", stepInfo)
}
