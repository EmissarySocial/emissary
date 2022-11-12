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
	case step.AddChildStream:
		return StepAddChildStream(s)

	case step.AddModelObject:
		return StepAddModelObject(s)

	case step.AddOutboxItem:
		return StepAddOutboxItem(s)

	case step.AddSiblingStream:
		return StepAddSiblingStream(s)

	case step.AddSubscription:
		return StepAddSubscription(s)

	case step.AddTopStream:
		return StepAddTopStream(s)

	case step.AsConfirmation:
		return StepAsConfirmation(s)

	case step.AsModal:
		return StepAsModal(s)

	case step.Delete:
		return StepDelete(s)

	case step.DeleteAttachment:
		return StepDeleteAttachment(s)

	case step.DeleteOutboxItem:
		return StepDeleteOutboxItem(s)

	case step.DeleteSubscription:
		return StepDeleteSubscription(s)

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

	case step.EditSubscription:
		return StepEditSubscription(s)

	case step.Form:
		return StepForm(s)

	case step.ForwardTo:
		return StepForwardTo(s)

	case step.IfCondition:
		return StepIfCondition(s)

	case step.RedirectTo:
		return StepRedirectTo(s)

	case step.RefreshPage:
		return StepRefreshPage(s)

	case step.Save:
		return StepSave(s)

	case step.SendMentions:
		return StepSendMentions(s)

	case step.SetData:
		return StepSetData(s)

	case step.SetPublishDate:
		return StepSetPublishDate(s)

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

	case step.UploadAttachment:
		return StepUploadAttachment(s)

	case step.ViewHTML:
		return StepViewHTML(s)

	case step.ViewRSS:
		return StepViewRSS(s)

	case step.WithChildren:
		return StepWithChildren(s)

	case step.WithDraft:
		return StepWithDraft(s)

	case step.WithInboxFolder:
		return StepWithInboxFolder(s)

	case step.WithParent:
		return StepWithParent(s)

	case step.WithPrevSibling:
		return StepWithPrevSibling(s)

	case step.WithNextSibling:
		return StepWithNextSibling(s)
	}

	return StepError{Original: stepInfo}
}
