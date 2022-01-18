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

	// STEPS THAT WORK ON ALL MODEL OBJECTS

	case "add":
		return NewStepAddModelObject(factory.FormLibrary(), stepInfo), nil

	case "edit":
		return NewStepEditModelObject(factory.FormLibrary(), stepInfo), nil

	case "delete":
		return NewStepDelete(stepInfo), nil

	case "save":
		return NewStepSave(stepInfo), nil

	case "form-html":
		return NewStepForm(factory.FormLibrary(), stepInfo), nil

	case "set-data":
		return NewStepSetData(stepInfo), nil

	case "set-thumbnail":
		return NewStepStreamThumbnail(factory.Attachment(), stepInfo), nil

	case "set-publishdate":
		return NewStepSetPublishDate(stepInfo), nil

	case "set-simple-sharing":
		return NewStepSetSimpleSharing(factory.FormLibrary(), stepInfo), nil

	case "set-state":
		return NewStepStreamState(stepInfo), nil

	case "sort":
		return NewStepSort(stepInfo), nil

	case "view-html":
		return NewStepViewHTML(stepInfo), nil

	// STREAM-SPECIFIC STEPS

	case "add-child":
		return NewStepAddChildStream(factory.Template(), factory.Stream(), stepInfo), nil

	case "add-top-level":
		return NewStepAddTopStream(factory.Template(), factory.Stream(), stepInfo), nil

	case "edit-content":
		return NewStepEditContent(factory.ContentLibrary(), stepInfo), nil

	// DRAFTS

	case "promote-draft":
		return NewStepStreamPromoteDraft(factory.StreamDraft(), stepInfo), nil

	case "with-draft":
		return NewStepWithDraft(factory.Stream(), stepInfo), nil

	// ATTACHMENTS

	case "upload-attachments":
		return NewStepUploadAttachment(factory.Stream(), factory.Attachment(), factory.MediaServer(), stepInfo), nil

	// SERVER-SIDE CONTROL LOGIC

	case "with-children":
		return NewStepWithChildren(factory.Stream(), stepInfo), nil

	case "with-parent":
		return NewStepWithParent(factory.Stream(), stepInfo), nil

	case "if":
		return NewStepIfCondition(factory, stepInfo), nil

	// CLIENT-SIDE CONTROLS

	case "as-modal":
		return NewStepAsModal(stepInfo), nil

	case "as-confirmation":
		return NewStepAsConfirmation(stepInfo), nil

	case "forward-to":
		return NewStepForwardTo(stepInfo), nil

	case "trigger-event":
		return NewStepTriggerEvent(stepInfo), nil

	case "refresh-page":
		return NewStepRefreshPage(stepInfo), nil

	}

	// Fall through means we have an unrecognized action
	return nil, derp.New(derp.CodeInternalError, "whisper.factory.RenderStep", "Unrecognized action configuration", stepInfo)
}
