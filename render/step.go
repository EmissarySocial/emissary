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

	case "draft-edit":
		return NewStepStreamDraftEdit(factory.StreamDraft(), stepInfo), nil

	case "draft-delete":
		return NewStepStreamDraftDelete(factory.StreamDraft(), stepInfo), nil

	case "draft-publish":
		return NewStepStreamDraftPublish(factory.Stream(), factory.StreamDraft(), stepInfo), nil

	case "form":
		return NewStepForm(factory.Template(), factory.FormLibrary(), stepInfo), nil

	case "stream-create":
		return NewStepStreamCreate(factory.Stream(), stepInfo), nil

	case "stream-data":
		return NewStepStreamData(factory.Template(), factory.Stream(), factory.FormLibrary(), stepInfo), nil

	case "stream-delete":
		return NewStepStreamDelete(factory.Stream(), stepInfo), nil

	case "stream-save":
		return NewStepStreamSave(factory.Stream(), stepInfo), nil

	case "stream-share":
		return NewStepStreamShare(stepInfo), nil

	case "stream-state":
		return NewStepStreamState(stepInfo), nil

	case "stream-html":
		return NewStepStreamHTML(stepInfo), nil

	case "top-folder-create":
		return NewStepTopFolderCreate(factory.Stream(), stepInfo), nil

	case "top-folder-edit":
		return NewStepTopFolderEdit(factory.Stream(), stepInfo), nil

	case "top-folder-delete":
		return NewStepTopFolderDelete(factory.Stream(), stepInfo), nil
	}

	// Fall through means we have an unrecognized action
	return nil, derp.New(derp.CodeInternalError, "ghost.factory.RenderStep", "Unrecognized action configuration", stepInfo)
}
