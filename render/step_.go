package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model/step"
)

type Step interface {
	Get(Renderer, io.Writer) error
	Post(Renderer) error
	UseGlobalWrapper() bool
	// isWrapped() bool // Returns true if this step can be wrapped by the global frame.
}

// ExecutableStep uses an Step object to create a new action
func ExecutableStep(stepInfo step.Step) Step {

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

	case step.EditFeatures:
		return StepEditFeatures(s)

	case step.EditProperties:
		return StepEditProperties(s)

	case step.EditModelObject:
		return StepEditModelObject(s)

	case step.Form:
		return StepForm(s)

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

	case step.SetData:
		return StepSetData(s)

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

	case step.ViewHTML:
		return StepViewHTML(s)

	case step.ViewRSS:
		return StepViewRSS(s)

	case step.WebSub:
		return StepWebSub(s)

	case step.WithChildren:
		return StepWithChildren(s)

	case step.WithDraft:
		return StepWithDraft(s)

	case step.WithFolder:
		return StepWithFolder(s)

	case step.WithParent:
		return StepWithParent(s)

	case step.WithPrevSibling:
		return StepWithPrevSibling(s)

	case step.WithNextSibling:
		return StepWithNextSibling(s)
	}

	return StepError{Original: stepInfo}
}
