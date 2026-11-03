package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
)

// Step interface is used here to bind together the structs in this package
type Step interface {

	// Name returns the name of the step, which is used in debugging.
	Name() string

	// RequiredModel returns the name of the model object that MUST be present in the Template.
	// If this value is not empty, then the Template MUST use this model object.
	RequiredModel() string

	// RequiredStates returns a slice of states that must be defined in any Template that uses this Step.
	RequiredStates() []string

	// RequiredRoles returns a slice of roles that must be defined in any Template that uses this Step
	RequiredRoles() []string
}

// ModelRequirer interface wraps the "RequireModel" method, which specifies that a step can ONLY
// be used with a specific type of model object. (like: "stream", "follower", "following", etc.)
type ModelRequirer interface {
	RequireModel() string
}

// TemplateRoleRequirer interface wraps the "RequiredTemplateRoles" method, which specifies that a
// Step can ONLY be used in a Template that declares one of the named template roles (like "admin").
// This is a narrower restriction than RequiredModel, because several Templates can build the same
// model object while playing very different roles in the system.  The interface is optional: Steps
// that work in any Template simply do not implement it.
type TemplateRoleRequirer interface {
	RequiredTemplateRoles() []string
}

// FormGetter interface is implemented by steps that render a form, so that the
// form can be validated against the Template schema when the Template is loaded.
type FormGetter interface {
	GetForm() form.Element
}

// New uses an Step object to create a new action.  Every template the step declares
// is parsed with funcMap, the helper functions of the Template that holds the step.
func New(stepInfo mapof.Any, funcMap template.FuncMap) (Step, error) {

	// Populate the action with the data from
	switch stepInfo["do"] { // NOSONAR - there is no clean way to handle this without a big switch statement.

	// STEPS THAT WORK ON ALL MODEL OBJECTS

	case "add":
		return NewAddModelObject(stepInfo, funcMap)

	case "add-event":
		return NewAddEvent(stepInfo)

	case "add-stream":
		return NewAddStream(stepInfo, funcMap)

	case "as-confirmation":
		return NewAsConfirmation(stepInfo)

	case "as-modal":
		return NewAsModal(stepInfo, funcMap)

	case "as-tooltip":
		return NewAsTooltip(stepInfo, funcMap)

	case "cache-url":
		return NewCacheURL(stepInfo)

	case "delete":
		return NewDelete(stepInfo, funcMap)

	case "delete-archive":
		return NewDeleteArchive(stepInfo)

	case "delete-attachments":
		return NewDeleteAttachments(stepInfo)

	case "dump":
		return NewDump(stepInfo, funcMap)

	case "edit":
		return NewEditModelObject(stepInfo, funcMap)

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
		return NewForwardTo(stepInfo, funcMap)

	case "get-archive":
		return NewGetArchive(stepInfo)

	case "halt":
		return NewHalt(stepInfo)

	case "if":
		return NewIfCondition(stepInfo, funcMap)

	case "include":
		return NewInclude(stepInfo)

	case "inline-error":
		return NewInlineError(stepInfo, funcMap)

	case "inline-save-button":
		return NewInlineSaveButton(stepInfo, funcMap)

	case "inline-success":
		return NewInlineSuccess(stepInfo, funcMap)

	case "make-archive":
		return NewMakeArchive(stepInfo)

	case "mark-folder-read":
		return NewMarkFolderRead(stepInfo)

	case "mark-notifications-read":
		return NewMarkNotificationsRead(stepInfo)

	case "process-content":
		return NewProcessContent(stepInfo)

	case "process-tags":
		return NewProcessTags(stepInfo)

	case "promote-draft":
		return NewStreamPromoteDraft(stepInfo)

	case "redirect-to":
		return NewRedirectTo(stepInfo, funcMap)

	case "refresh-page":
		return NewRefreshPage(stepInfo)

	case "reload-page":
		return NewReloadPage(stepInfo)

	case "remove-event":
		return NewRemoveEvent(stepInfo)

	case "require-password":
		return NewRequirePassword(stepInfo, funcMap)

	case "save":
		return NewSave(stepInfo, funcMap)

	case "save-and-publish":
		return NewSaveAndPublish(stepInfo)

	case "schedule-delete":
		return NewScheduleDelete(stepInfo, funcMap)

	case "search-index":
		return NewSearchIndex(stepInfo, funcMap)

	case "read-form":
		return NewReadForm(stepInfo)

	case "send-email":
		return NewSendEmail(stepInfo, funcMap)

	case "set-args":
		return NewSetRenderData(stepInfo, funcMap)

	case "set-circle-sharing":
		return NewSetCircleSharing(stepInfo)

	case "set-data":
		return NewSetData(stepInfo, funcMap)

	case "set-header":
		return NewSetHeader(stepInfo, funcMap)

	case "set-password":
		return NewSetPassword(stepInfo)

	case "set-privileges":
		return NewSetPrivileges(stepInfo)

	case "set-query-param":
		return NewSetQueryParam(stepInfo, funcMap)

	case "set-response":
		return NewSetResponse(stepInfo)

	case "set-sharing":
		return NewSetSharing(stepInfo)

	case "set-simple-sharing":
		return NewSetSimpleSharing(stepInfo)

	case "set-state":
		return NewSetState(stepInfo)

	case "set-thumbnail":
		return NewSetThumbnail(stepInfo)

	case "share-reply-policy":
		return NewShareReplyPolicy(stepInfo)

	case "sleep":
		return NewSleep(stepInfo)

	case "sort":
		return NewSort(stepInfo)

	case "sort-attachments":
		return NewSortAttachments(stepInfo)

	case "sort-widgets":
		return NewSortWidgets(stepInfo)

	case "startup-complete":
		return NewStartupComplete(stepInfo)

	case "startup-create-streams":
		return NewStartupCreateStreams(stepInfo)

	case "startup-save-task":
		return NewStartupSaveTask(stepInfo)

	case "trigger-event":
		return NewTriggerEvent(stepInfo, funcMap)

	case "unpublish":
		return NewUnPublish(stepInfo)

	case "upload-attachments":
		return NewUploadAttachments(stepInfo)

	case "view-attachment":
		return NewViewAttachment(stepInfo)

	case "view-css":
		return NewViewCSS(stepInfo)

	case "view-feed":
		return NewViewFeed(stepInfo)

	case "view-html":
		return NewViewHTML(stepInfo)

	case "view-json":
		return NewViewJSON(stepInfo, funcMap)

	case "with-annotation":
		return NewWithAnnotation(stepInfo, funcMap)

	case "with-attachment":
		return NewWithAttachment(stepInfo, funcMap)

	case "with-children":
		return NewWithChildren(stepInfo, funcMap)

	case "with-circle":
		return NewWithCircle(stepInfo, funcMap)

	case "with-draft":
		return NewWithDraft(stepInfo, funcMap)

	case "with-folder":
		return NewWithFolder(stepInfo, funcMap)

	case "with-following":
		return NewWithFollowing(stepInfo, funcMap)

	case "with-follower":
		return NewWithFollower(stepInfo, funcMap)

	case "with-import":
		return NewWithImport(stepInfo, funcMap)

	case "with-keypackage":
		return NewWithKeyPackage(stepInfo, funcMap)

	case "with-merchant-account":
		return NewWithMerchantAccount(stepInfo, funcMap)

	case "with-user-connection":
		return NewWithUserConnection(stepInfo, funcMap)

	case "with-message":
		return NewWithMessage(stepInfo, funcMap)

	case "with-next-sibling":
		return NewWithNextSibling(stepInfo, funcMap)

	case "with-notification":
		return NewWithNotification(stepInfo, funcMap)

	case "with-oauth-token":
		return NewWithOAuthToken(stepInfo, funcMap)

	case "with-parent":
		return NewWithParent(stepInfo, funcMap)

	case "with-prev-sibling":
		return NewWithPrevSibling(stepInfo, funcMap)

	case "with-privilege":
		return NewWithPrivilege(stepInfo, funcMap)

	case "with-response":
		return NewWithResponse(stepInfo, funcMap)

	case "with-rule":
		return NewWithRule(stepInfo, funcMap)

	case "with-stream-source":
		return NewWithStreamSource(stepInfo, funcMap)
	}

	// Fall through means we have an unrecognized action
	return nil, derp.Internal("model.step.New", "Unrecognized step type", stepInfo.GetString("do"), stepInfo)
}

// Pipeline is an ordered list of Steps.  Steps are built with their Template's helper
// functions, so a Pipeline is decoded by the type that holds it (see NewPipeline).
type Pipeline []Step

// NewPipeline parses a series of build steps into a new array, using funcMap for every template
func NewPipeline[T ~map[string]any](stepInfo []T, funcMap template.FuncMap) ([]Step, error) {

	const location = "model.step.NewPipeline"

	result := make([]Step, len(stepInfo))

	for index := range stepInfo {

		if step, err := New(mapof.Any(stepInfo[index]), funcMap); err != nil {
			return result, derp.Wrap(err, location, "Parsing step", stepInfo)
		} else {
			result[index] = step
		}
	}

	return result, nil
}
