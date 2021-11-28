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
	switch stepInfo["step"] {

	// STREAMS
	case "new-child":
		return NewStepCreateChild(factory.Stream(), stepInfo), nil

	case "new-sibling":
		return NewStepCreateSibling(factory.Stream(), stepInfo), nil

	case "delete":
		return NewStepStreamDelete(factory.Stream(), stepInfo), nil

	case "form-html":
		return NewStepForm(factory.Template(), factory.FormLibrary(), stepInfo), nil

	case "view-html":
		return NewStepStreamHTML(stepInfo), nil

	case "save":
		return NewStepStreamSave(factory.Stream(), stepInfo), nil

	case "set-data":
		return NewStepStreamData(factory.Template(), factory.Stream(), factory.FormLibrary(), stepInfo), nil

	case "set-defaults":
		return NewStepStreamDefaults(stepInfo), nil

	case "set-sharing":
		return NewStepStreamShare(stepInfo), nil

	case "set-state":
		return NewStepStreamState(stepInfo), nil

	// DRAFTS
	case "edit-draft":
		return NewStepStreamDraftEdit(factory.StreamDraft(), stepInfo), nil

	case "delete-draft":
		return NewStepStreamDraftDelete(factory.StreamDraft(), stepInfo), nil

	case "publish-draft":
		return NewStepStreamDraftPublish(factory.Stream(), factory.StreamDraft(), stepInfo), nil

	// FOLDERS
	case "new-folder":
		return NewStepTopFolderCreate(factory.Stream(), stepInfo), nil

	case "edit-folder":
		return NewStepTopFolderEdit(factory.Stream(), stepInfo), nil

	case "delete-folder":
		return NewStepTopFolderDelete(factory.Stream(), stepInfo), nil

	// CONTROL LOGIC
	case "for-each-child":
		return NewStepForChildren(factory.Stream(), stepInfo), nil

	case "if":
		return NewStepIfCondition(stepInfo), nil

	}

	// Fall through means we have an unrecognized action
	return nil, derp.New(derp.CodeInternalError, "ghost.factory.RenderStep", "Unrecognized action configuration", stepInfo)
}
