package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model/step"
)

type Step interface {
	Get(Builder, io.Writer) PipelineBehavior
	Post(Builder, io.Writer) PipelineBehavior
}

// ExecutableStep uses an Step object to create a new action
func ExecutableStep(stepInfo step.Step) Step {

	// These actual action steps require access to more of the application than is available in the model package.
	// Model objects are only concerned with the data that they contain, and not with the actual building of that data.
	// So, we convert the model object into a "build step" that CAN access the building context.

	switch s := stepInfo.(type) {

	case step.AddModelObject:
		return StepAddModelObject(s)

	case step.AddStream:
		return StepAddStream(s)

	case step.AsConfirmation:
		return StepAsConfirmation(s)

	case step.AsModal:
		return StepAsModal(s)

	case step.AsTooltip:
		return StepAsTooltip(s)

	case step.CacheURL:
		return StepCacheURL(s)

	case step.Delete:
		return StepDelete(s)

	case step.DeleteArchive:
		return StepDeleteArchive(s)

	case step.DeleteAttachments:
		return StepDeleteAttachments(s)

	case step.Dump:
		return StepDump(s)

	case step.EditConnection:
		return StepEditConnection(s)

	case step.EditContent:
		return StepEditContent(s)

	case step.EditModelObject:
		return StepEditModelObject(s)

	case step.EditRegistration:
		return StepEditRegistration(s)

	case step.EditTemplate:
		return StepEditTemplate(s)

	case step.EditWidget:
		return StepEditWidget(s)

	case step.ForwardTo:
		return StepForwardTo(s)

	case step.GetArchive:
		return StepGetArchive(s)

	case step.Halt:
		return StepHalt(s)

	case step.IfCondition:
		return StepIfCondition(s)

	case step.Include:
		return StepInclude(s)

	case step.InlineError:
		return StepInlineError(s)

	case step.InlineSaveButton:
		return StepInlineSaveButton(s)

	case step.InlineSuccess:
		return StepInlineSuccess(s)

	case step.MakeArchive:
		return StepMakeArchive(s)

	case step.ProcessContent:
		return StepProcessContent(s)

	case step.ProcessTags:
		return StepProcessTags(s)

	case step.RedirectTo:
		return StepRedirectTo(s)

	case step.ReloadPage:
		return StepReloadPage(s)

	case step.RefreshPage:
		return StepRefreshPage(s)

	case step.RemoveEvent:
		return StepRemoveEvent(s)

	case step.Save:
		return StepSave(s)

	case step.SaveAndPublish:
		return StepSaveAndPublish(s)

	case step.SearchIndex:
		return StepSearchIndex(s)

	case step.SendEmail:
		return StepSendEmail(s)

	case step.SetCircleSharing:
		return StepSetCircleSharing(s)

	case step.SetData:
		return StepSetData(s)

	case step.SetHeader:
		return StepSetHeader(s)

	case step.SetPassword:
		return StepSetPassword(s)

	case step.SetPrivileges:
		return StepSetPrivileges(s)

	case step.SetQueryParam:
		return StepSetQueryParam(s)

	case step.SetResponse:
		return StepSetResponse(s)

	case step.SetRenderData:
		return StepSetRenderData(s)

	case step.SetSimpleSharing:
		return StepSetSimpleSharing(s)

	case step.SetState:
		return StepSetState(s)

	case step.SetThumbnail:
		return StepSetThumbnail(s)

	case step.Sleep:
		return StepSleep(s)

	case step.Sort:
		return StepSort(s)

	case step.SortAttachments:
		return StepSortAttachments(s)

	case step.SortWidgets:
		return StepSortWidgets(s)

	case step.StreamPromoteDraft:
		return StepStreamPromoteDraft(s)

	case step.TableEditor:
		return StepTableEditor(s)

	case step.TriggerEvent:
		return StepTriggerEvent(s)

	case step.UnPublish:
		return StepUnPublish(s)

	case step.UploadAttachments:
		return StepUploadAttachments(s)

	case step.ViewAttachment:
		return StepViewAttachment(s)

	case step.ViewCSS:
		return StepViewCSS(s)

	case step.ViewFeed:
		return StepViewFeed(s)

	case step.ViewHTML:
		return StepViewHTML(s)

	case step.WebSub:
		return StepWebSub(s)

	case step.WithAttachment:
		return StepWithAttachment(s)

	case step.WithChildren:
		return StepWithChildren(s)

	case step.WithCircle:
		return StepWithCircle(s)

	case step.WithDraft:
		return StepWithDraft(s)

	case step.WithFolder:
		return StepWithFolder(s)

	case step.WithFollower:
		return StepWithFollower(s)

	case step.WithFollowing:
		return StepWithFollowing(s)

	case step.WithMerchantAccount:
		return StepWithMerchantAccount(s)

	case step.WithMessage:
		return StepWithMessage(s)

	case step.WithNextSibling:
		return StepWithNextSibling(s)

	case step.WithParent:
		return StepWithParent(s)

	case step.WithPrevSibling:
		return StepWithPrevSibling(s)

	case step.WithPrivilege:
		return StepWithPrivilege(s)

	case step.WithResponse:
		return StepWithResponse(s)

	case step.WithRule:
		return StepWithRule(s)

	}

	return StepError{Original: stepInfo}
}
