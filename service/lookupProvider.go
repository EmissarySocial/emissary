package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/dataset"
	"github.com/benpate/form"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LookupProvider struct {
	folderService       *Folder
	groupService        *Group
	registrationService *Registration
	themeService        *Theme
	userID              primitive.ObjectID
}

func NewLookupProvider(folderService *Folder, groupService *Group, registrationService *Registration, themeService *Theme, userID primitive.ObjectID) LookupProvider {
	return LookupProvider{
		themeService:        themeService,
		registrationService: registrationService,
		groupService:        groupService,
		folderService:       folderService,
		userID:              userID,
	}
}

func (service LookupProvider) Group(path string) form.LookupGroup {

	switch path {

	case "following-behaviors":
		return form.NewReadOnlyLookupGroup(
			form.LookupCode{Value: "POSTS+REPLIES", Label: "Posts and Replies"},
			form.LookupCode{Value: "POSTS", Label: "Posts Only (ignore replies)"},
		)

	case "following-rule-actions":
		return form.NewReadOnlyLookupGroup(
			form.LookupCode{Value: "IGNORE", Label: "Do not import rules from this source (display messages normally)"},
			form.LookupCode{Value: "LABEL", Label: "LABEL posts that are blocked by this source"},
			form.LookupCode{Value: "MUTE", Label: "MUTE senders who are blocked by this source (one-way block)"},
			form.LookupCode{Value: "BLOCK", Label: "BLOCK senders and prevent followers who are blocked by this source (two-way block)"},
		)

	case "rule-actions":
		return form.NewReadOnlyLookupGroup(
			form.LookupCode{Value: "LABEL", Label: "LABEL posts that match this rule"},
			form.LookupCode{Value: "MUTE", Label: "MUTE senders but do not prevent followers (one-way block)"},
			form.LookupCode{Value: "BLOCK", Label: "BLOCK senders and prevent followers (two-way block)"},
		)

	case "rule-types":
		return form.NewReadOnlyLookupGroup(
			form.LookupCode{Label: "Filter by Person", Value: model.RuleTypeActor},
			form.LookupCode{Label: "Filter by Domain", Value: model.RuleTypeDomain},
			form.LookupCode{Label: "Filter by Tags & Keywords", Value: model.RuleTypeContent},
		)

	case "folders":
		return NewFolderLookupProvider(service.folderService, service.userID)

	case "folder-icons":
		return form.NewReadOnlyLookupGroup(dataset.Icons()...)

	case "groups":
		return NewGroupLookupProvider(service.groupService)

	case "reaction-icons":
		return form.NewReadOnlyLookupGroup(
			form.LookupCode{Label: "Love", Group: "Like", Value: "❤️"},
			form.LookupCode{Label: "Like", Group: "Like", Value: "👍"},
			form.LookupCode{Label: "Dislike", Group: "Dislike", Value: "👎"},
			form.LookupCode{Label: "Smile", Group: "Like", Value: "😀"},
			form.LookupCode{Label: "Laugh", Group: "Like", Value: "🤣"},
			form.LookupCode{Label: "Frown", Group: "Dislike", Value: "🙁"},
			form.LookupCode{Label: "Emphasize", Group: "Like", Value: "‼️", Icon: ""},
			form.LookupCode{Label: "Celebrate", Group: "Like", Value: "🎉"},
			form.LookupCode{Label: "Question", Group: "Like", Value: "❓"},
			form.LookupCode{Label: "Crown", Group: "Like", Value: "👑"},
			form.LookupCode{Label: "Fire", Group: "Like", Value: "🔥"},
		)

	case "sharing":
		return form.NewReadOnlyLookupGroup(
			form.LookupCode{Value: "anonymous", Label: "Everyone (including anonymous visitors)"},
			form.LookupCode{Value: "authenticated", Label: "Authenticated People Only"},
			form.LookupCode{Value: "private", Label: "Only Selected Groups"},
		)

	case "themes":
		return NewThemeLookupProvider(service.themeService)

	case "signup-templates":
		return form.ReadOnlyLookupGroup(service.registrationService.List())

	default:
		return form.NewReadOnlyLookupGroup()
	}
}
