package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
)

type Step interface {
	Get(Factory, Renderer, io.Writer) error
	Post(Factory, Renderer, io.Writer) error
	isWrapped() bool // Returns true if this step can be wrapped by the global frame.
}

// NewStep uses an Step object to create a new action
func NewStep(stepInfo datatype.Map) (Step, error) {

	// Populate the action with the data from
	switch stepInfo["step"] {

	// STEPS THAT WORK ON ALL MODEL OBJECTS

	case "add":
		return NewStepAddModelObject(stepInfo)

	case "edit":
		return NewStepEditModelObject(stepInfo)

	case "delete":
		return NewStepDelete(stepInfo)

	case "save":
		return NewStepSave(stepInfo)

	case "form-html":
		return NewStepForm(stepInfo)

	case "set-data":
		return NewStepSetData(stepInfo)

	case "set-thumbnail":
		return NewStepStreamThumbnail(stepInfo)

	case "set-publishdate":
		return NewStepSetPublishDate(stepInfo)

	case "set-simple-sharing":
		return NewStepSetSimpleSharing(stepInfo)

	case "set-state":
		return NewStepStreamState(stepInfo)

	case "sort":
		return NewStepSort(stepInfo)

	case "view-html":
		return NewStepViewHTML(stepInfo)

	// STREAM-SPECIFIC STEPS

	case "add-child":
		return NewStepAddChildStream(stepInfo)

	case "add-sibling":
		return NewStepAddSiblingStream(stepInfo)

	case "add-top-level":
		return NewStepAddTopStream(stepInfo)

	case "edit-content":
		return NewStepEditContent(stepInfo)

	case "view-rss":
		return NewStepViewRSS(stepInfo)

	// DRAFTS

	case "promote-draft":
		return NewStepStreamPromoteDraft(stepInfo)

	case "with-draft":
		return NewStepWithDraft(stepInfo)

	// ATTACHMENTS

	case "upload-attachments":
		return NewStepUploadAttachment(stepInfo)

	// SERVER-SIDE CONTROL LOGIC

	case "with-children":
		return NewStepWithChildren(stepInfo)

	case "with-parent":
		return NewStepWithParent(stepInfo)

	case "if":
		return NewStepIfCondition(stepInfo)

	// CLIENT-SIDE CONTROLS

	case "as-modal":
		return NewStepAsModal(stepInfo)

	case "as-confirmation":
		return NewStepAsConfirmation(stepInfo)

	case "forward-to":
		return NewStepForwardTo(stepInfo)

	case "trigger-event":
		return NewStepTriggerEvent(stepInfo)

	case "refresh-page":
		return NewStepRefreshPage(stepInfo)

	}

	// Fall through means we have an unrecognized action
	return nil, derp.New(derp.CodeInternalError, "whisper.factory.RenderStep", "Unrecognized action configuration", stepInfo)
}
