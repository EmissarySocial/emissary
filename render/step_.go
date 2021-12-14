package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
)

type Step interface {
	Get(io.Writer, Renderer) error
	Post(io.Writer, Renderer) error
}

// NewStep uses an Step object to create a new action
func NewStep(factory Factory, stepInfo datatype.Map) (Step, error) {

	// Populate the action with the data from
	switch stepInfo["step"] {

	// STREAMS

	case "delete":
		return NewStepStreamDelete(factory.Stream(), factory.StreamDraft(), stepInfo), nil

	case "form-html":
		return NewStepForm(factory.Template(), factory.FormLibrary(), stepInfo), nil

	case "new-child":
		return NewStepNewChild(factory.Template(), factory.Stream(), stepInfo), nil

	case "save":
		return NewStepStreamSave(factory.Stream(), stepInfo), nil

	case "set-data":
		return NewStepSetData(factory.Template(), factory.Stream(), factory.FormLibrary(), stepInfo), nil

	case "set-thumbnail":
		return NewStepStreamThumbnail(factory.Attachment(), stepInfo), nil

	case "set-publishdate":
		return NewStepSetPublishDate(stepInfo), nil

	case "set-sharing":
		return NewStepStreamShare(stepInfo), nil

	case "set-state":
		return NewStepStreamState(stepInfo), nil

	case "view-html":
		return NewStepStreamHTML(stepInfo), nil

	// DRAFTS
	case "edit-draft":
		return NewStepStreamDraftEdit(factory.StreamDraft(), stepInfo), nil

	case "delete-draft":
		return NewStepStreamDraftDelete(factory.StreamDraft(), stepInfo), nil

	case "publish-draft":
		return NewStepStreamDraftPublish(factory.Stream(), factory.StreamDraft(), stepInfo), nil

	// ATTACHMENTS
	case "upload-attachments":
		return NewStepAttachmentUpload(factory.Stream(), factory.Attachment(), factory.MediaServer(), stepInfo), nil

	// CONTROL LOGIC
	case "with-children":
		return NewStepWithChildren(factory.Stream(), stepInfo), nil

	case "with-parent":
		return NewStepWithParent(factory.Stream(), stepInfo), nil

	case "if":
		return NewStepIfCondition(factory, stepInfo), nil

	case "forward-to":
		return NewStepForwardTo(stepInfo), nil

	case "trigger-event":
		return NewStepTriggerEvent(stepInfo), nil

	}

	// Fall through means we have an unrecognized action
	return nil, derp.New(derp.CodeInternalError, "ghost.factory.RenderStep", "Unrecognized action configuration", stepInfo)
}
