// Package Step encapsulates the DATA required for each pipeline step in the renderer.
// This package does not contain any rendering functions (that's in /render) but these
// objects know how to parse and "compile" raw data into the arguments required to execute
// each step.
package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// Step interface is used here to bind together the structs in this package
type Step interface {
	// AmStep is a NO-OP.  It is only used to validate that a struct contains "step-like" data.
	AmStep()
}

// New uses an Step object to create a new action
func New(stepInfo mapof.Any) (Step, error) {

	// Populate the action with the data from
	switch stepInfo["step"] {

	// STEPS THAT WORK ON ALL MODEL OBJECTS

	case "add":
		return NewAddModelObject(stepInfo)

	case "add-stream":
		return NewAddStream(stepInfo)

	case "edit":
		return NewEditModelObject(stepInfo)

	case "edit-table":
		return NewTableEditor(stepInfo)

	case "delete":
		return NewDelete(stepInfo)

	case "save":
		return NewSave(stepInfo)

	case "set-data":
		return NewSetData(stepInfo)

	case "set-header":
		return NewSetHeader(stepInfo)

	case "set-render-data":
		return NewSetRenderData(stepInfo)

	case "set-response":
		return NewSetResponse(stepInfo)

	case "set-query-param":
		return NewSetQueryParam(stepInfo)

	case "set-thumbnail":
		return NewSetThumbnail(stepInfo)

	case "set-simple-sharing":
		return NewSetSimpleSharing(stepInfo)

	case "set-state":
		return NewSetState(stepInfo)

	case "sort":
		return NewSort(stepInfo)

	case "view-html":
		return NewViewHTML(stepInfo)

	case "view-json":
		return NewViewJSONLD(stepInfo)

	// STREAM-SPECIFIC STEPS

	case "edit-content":
		return NewEditContent(stepInfo)

	case "edit-properties":
		return NewEditProperties(stepInfo)

	case "edit-widget":
		return NewEditWidget(stepInfo)

	case "sort-widgets":
		return NewSortWidgets(stepInfo)

	case "view-feed":
		return NewViewFeed(stepInfo)

	// SOCIAL / ACTIVITYPUB STEPS

	case "publish":
		return NewPublish(stepInfo)

	case "unpublish":
		return NewUnPublish(stepInfo)

	// DRAFTS

	case "promote-draft":
		return NewStreamPromoteDraft(stepInfo)

	case "with-draft":
		return NewWithDraft(stepInfo)

	// ATTACHMENTS

	case "delete-attachments":
		return NewDeleteAttachments(stepInfo)

	case "upload-attachments":
		return NewUploadAttachment(stepInfo)

	case "sort-attachments":
		return NewSortAttachments(stepInfo)

	// CLIENT-SIDE CONTROLS

	case "as-modal":
		return NewAsModal(stepInfo)

	case "as-tooltip":
		return NewAsTooltip(stepInfo)

	case "as-confirmation":
		return NewAsConfirmation(stepInfo)

	case "forward-to":
		return NewForwardTo(stepInfo)

	case "trigger-event":
		return NewTriggerEvent(stepInfo)

	case "refresh-page":
		return NewRefreshPage(stepInfo)

	case "reload-page":
		return NewReloadPage(stepInfo)

	// SERVER-SIDE CONTROL LOGIC

	case "halt":
		return NewHalt(stepInfo)

	case "if":
		return NewIfCondition(stepInfo)

	case "redirect-to":
		return NewRedirectTo(stepInfo)

	case "server-redirect":
		return NewServerRedirect(stepInfo)

	case "with-block":
		return NewWithBlock(stepInfo)

	case "with-children":
		return NewWithChildren(stepInfo)

	case "with-folder":
		return NewWithFolder(stepInfo)

	case "with-following":
		return NewWithFollowing(stepInfo)

	case "with-follower":
		return NewWithFollower(stepInfo)

	case "with-message":
		return NewWithMessage(stepInfo)

	case "with-next-sibling":
		return NewWithNextSibling(stepInfo)

	case "with-prev-sibling":
		return NewWithPrevSibling(stepInfo)

	case "with-parent":
		return NewWithParent(stepInfo)

	case "with-response":
		return NewWithResponse(stepInfo)

	// EXTERNAL CONNECTIONS

	case "edit-connection":
		return NewEditConnection(stepInfo)

	case "websub":
		return NewWebSub(stepInfo)
	}

	// Fall through means we have an unrecognized action
	return nil, derp.NewInternalError("model.step.New", "Unrecognized step type", stepInfo["step"], stepInfo)
}

// NewPipeline parses a series of render steps into a new array
func NewPipeline[T ~map[string]any](stepInfo []T) ([]Step, error) {

	const location = "model.step.NewPipeline"

	result := make([]Step, len(stepInfo))

	for index := range stepInfo {

		if step, err := New(mapof.Any(stepInfo[index])); err != nil {
			return result, derp.Wrap(err, location, "Error parsing step", stepInfo)
		} else {
			result[index] = step
		}
	}

	return result, nil
}
