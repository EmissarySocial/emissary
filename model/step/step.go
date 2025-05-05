// Package Step encapsulates the DATA required for each pipeline step in the builder.
// This package does not contain any building functions (that's in /build) but these
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

// TypeRequirer interface wraps the "RequireType" method, which specifies that a step can ONLY
// be used with a specific type of builder (like: "template", "registration", "widget", "email", etc.)
type TypeRequirer interface {
	RequireType() string
}

// ModelRequirer interface wraps the "RequireModel" method, which specifies that a step can ONLY
// be used with a specific type of model object. (like: "stream", "follower", "following", etc.)
type ModelRequirer interface {
	RequireModel() string
}

// New uses an Step object to create a new action
func New(stepInfo mapof.Any) (Step, error) {

	// Populate the action with the data from
	switch stepInfo["do"] {

	// STEPS THAT WORK ON ALL MODEL OBJECTS

	case "add":
		return NewAddModelObject(stepInfo)

	case "add-stream":
		return NewAddStream(stepInfo)

	case "as-confirmation":
		return NewAsConfirmation(stepInfo)

	case "as-modal":
		return NewAsModal(stepInfo)

	case "as-tooltip":
		return NewAsTooltip(stepInfo)

	case "cache-url":
		return NewCacheURL(stepInfo)

	case "delete":
		return NewDelete(stepInfo)

	case "delete-archive":
		return NewDeleteArchive(stepInfo)

	case "delete-attachments":
		return NewDeleteAttachments(stepInfo)

	case "dump":
		return NewDump(stepInfo)

	case "edit":
		return NewEditModelObject(stepInfo)

	case "edit-connection":
		return NewEditConnection(stepInfo)

	case "edit-content":
		return NewEditContent(stepInfo)

	case "edit-registration":
		return NewEditRegistration(stepInfo)

	case "edit-table":
		return NewTableEditor(stepInfo)

	case "edit-template":
		return NewEditTemplate(stepInfo)

	case "edit-widget":
		return NewEditWidget(stepInfo)

	case "forward-to":
		return NewForwardTo(stepInfo)

	case "get-archive":
		return NewGetArchive(stepInfo)

	case "halt":
		return NewHalt(stepInfo)

	case "if":
		return NewIfCondition(stepInfo)

	case "include":
		return NewDo(stepInfo)

	case "inline-error":
		return NewInlineError(stepInfo)

	case "inline-save-button":
		return NewInlineSaveButton(stepInfo)

	case "inline-success":
		return NewInlineSuccess(stepInfo)

	case "make-archive":
		return NewMakeArchive(stepInfo)

	case "process-content":
		return NewProcessContent(stepInfo)

	case "process-tags":
		return NewProcessTags(stepInfo)

	case "promote-draft":
		return NewStreamPromoteDraft(stepInfo)

	case "redirect-to":
		return NewRedirectTo(stepInfo)

	case "refresh-page":
		return NewRefreshPage(stepInfo)

	case "reload-page":
		return NewReloadPage(stepInfo)

	case "remove-event":
		return NewRemoveEvent(stepInfo)

	case "save":
		return NewSave(stepInfo)

	case "save-and-publish":
		return NewSaveAndPublish(stepInfo)

	case "search-index":
		return NewSearchIndex(stepInfo)

	case "send-email":
		return NewSendEmail(stepInfo)

	case "set-args":
		return NewSetRenderData(stepInfo)

	case "set-data":
		return NewSetData(stepInfo)

	case "set-header":
		return NewSetHeader(stepInfo)

	case "set-password":
		return NewSetPassword(stepInfo)

	case "set-query-param":
		return NewSetQueryParam(stepInfo)

	case "set-response":
		return NewSetResponse(stepInfo)

	case "set-simple-sharing":
		return NewSetSimpleSharing(stepInfo)

	case "set-state":
		return NewSetState(stepInfo)

	case "set-products":
		return NewSetProducts(stepInfo)

	case "set-thumbnail":
		return NewSetThumbnail(stepInfo)

	case "sleep":
		return NewSleep(stepInfo)

	case "sort":
		return NewSort(stepInfo)

	case "sort-attachments":
		return NewSortAttachments(stepInfo)

	case "sort-widgets":
		return NewSortWidgets(stepInfo)

	case "trigger-event":
		return NewTriggerEvent(stepInfo)

	case "unpublish":
		return NewUnPublish(stepInfo)

	case "upload-attachments":
		return NewUploadAttachments(stepInfo)

	case "view-css":
		return NewViewCSS(stepInfo)

	case "view-feed":
		return NewViewFeed(stepInfo)

	case "view-html":
		return NewViewHTML(stepInfo)

	case "websub":
		return NewWebSub(stepInfo)

	case "with-attachment":
		return NewWithAttachment(stepInfo)

	case "with-children":
		return NewWithChildren(stepInfo)

	case "with-draft":
		return NewWithDraft(stepInfo)

	case "with-folder":
		return NewWithFolder(stepInfo)

	case "with-following":
		return NewWithFollowing(stepInfo)

	case "with-follower":
		return NewWithFollower(stepInfo)

	case "with-merchant-account":
		return NewWithMerchantAccount(stepInfo)

	case "with-message":
		return NewWithMessage(stepInfo)

	case "with-next-sibling":
		return NewWithNextSibling(stepInfo)

	case "with-parent":
		return NewWithParent(stepInfo)

	case "with-prev-sibling":
		return NewWithPrevSibling(stepInfo)

	case "with-response":
		return NewWithResponse(stepInfo)

	case "with-rule":
		return NewWithRule(stepInfo)

	case "with-product":
		return NewWithProduct(stepInfo)

	}

	// Fall through means we have an unrecognized action
	return nil, derp.NewInternalError("model.step.New", "Unrecognized step type", stepInfo)
}

// NewPipeline parses a series of build steps into a new array
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
