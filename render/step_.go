package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model/step"
)

type Step interface {
	Get(Renderer, io.Writer) error
	Post(Renderer, io.Writer) error
	UseGlobalWrapper() bool // TODO: should have a better solution for using/skipping the global wrapper than this function
}

// ExecutableStep uses an Step object to create a new action
func ExecutableStep(stepInfo step.Step) Step {

	// These actual action steps require access to more of the application than is available in the model package.
	// Model objects are only concerned with the data that they contain, and not with the actual rendering of that data.
	// So, we convert the model object into a "render step" that CAN access the rendering context.

	switch s := stepInfo.(type) {

	case step.AddChildEmbed:
		return StepAddChildEmbed(s)

	case step.AddChildStream:
		return StepAddChildStream(s)

	case step.AddModelObject:
		return StepAddModelObject(s)

	case step.AddSiblingStream:
		return StepAddSiblingStream(s)

	case step.AddTopStream:
		return StepAddTopStream(s)

	case step.AsConfirmation:
		return StepAsConfirmation(s)

	case step.AsModal:
		return StepAsModal(s)

	case step.Delete:
		return StepDelete(s)

	case step.DeleteAttachments:
		return StepDeleteAttachments(s)

	case step.EditConnection:
		return StepEditConnection(s)

	case step.EditContent:
		return StepEditContent(s)

	case step.EditModelObject:
		return StepEditModelObject(s)

	case step.EditProperties:
		return StepEditProperties(s)

	case step.EditWidget:
		return StepEditWidget(s)

	case step.ForwardTo:
		return StepForwardTo(s)

	case step.IfCondition:
		return StepIfCondition(s)

	case step.Publish:
		return StepPublish(s)

	case step.RedirectTo:
		return StepRedirectTo(s)

	case step.ReloadPage:
		return StepReloadPage(s)

	case step.RefreshPage:
		return StepRefreshPage(s)

	case step.Save:
		return StepSave(s)

	case step.ServerRedirect:
		return StepServerRedirect(s)

	case step.SetData:
		return StepSetData(s)

	case step.SetHeader:
		return StepSetHeader(s)

	case step.SetQueryParam:
		return StepSetQueryParam(s)

	case step.SetSimpleSharing:
		return StepSetSimpleSharing(s)

	case step.SetState:
		return StepSetState(s)

	case step.SetThumbnail:
		return StepSetThumbnail(s)

	case step.Sort:
		return StepSort(s)

	case step.SortAttachments:
		return StepSortAttachments(s)

	case step.SortWidgets:
		return StepSortWidgets(s)

	case step.StreamPromoteDraft:
		return StepStreamPromoteDraft(s)

	case step.StripeComplete:
		return StepStripeComplete(s)

	case step.StripeCheckout:
		return StepStripeCheckout(s)

	case step.StripeProduct:
		return StepStripeProduct(s)

	case step.StripeSetup:
		return StepStripeSetup(s)

	case step.TableEditor:
		return StepTableEditor(s)

	case step.TriggerEvent:
		return StepTriggerEvent(s)

	case step.UnPublish:
		return StepUnPublish(s)

	case step.UploadAttachment:
		return StepUploadAttachment(s)

	case step.ViewActivityPub:
		return StepViewActivityPub(s)

	case step.ViewHTML:
		return StepViewHTML(s)

	case step.ViewFeed:
		return StepViewFeed(s)

	case step.WebSub:
		return StepWebSub(s)

	case step.WithBlock:
		return StepWithBlock(s)

	case step.WithChildren:
		return StepWithChildren(s)

	case step.WithDraft:
		return StepWithDraft(s)

	case step.WithFolder:
		return StepWithFolder(s)

	case step.WithFollower:
		return StepWithFollower(s)

	case step.WithFollowing:
		return StepWithFollowing(s)

	case step.WithMessage:
		return StepWithMessage(s)

	case step.WithParent:
		return StepWithParent(s)

	case step.WithPrevSibling:
		return StepWithPrevSibling(s)

	case step.WithNextSibling:
		return StepWithNextSibling(s)
	}

	return StepError{Original: stepInfo}
}
