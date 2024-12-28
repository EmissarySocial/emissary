package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/dataset"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/list"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LookupProvider struct {
	domainService       *Domain
	folderService       *Folder
	groupService        *Group
	registrationService *Registration
	templateService     *Template
	themeService        *Theme
	userID              primitive.ObjectID
}

func NewLookupProvider(domainService *Domain, folderService *Folder, groupService *Group, registrationService *Registration, templateService *Template, themeService *Theme, userID primitive.ObjectID) LookupProvider {
	return LookupProvider{
		domainService:       domainService,
		themeService:        themeService,
		templateService:     templateService,
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

	case "inbox-templates":
		return form.ReadOnlyLookupGroup(service.templateService.ListByTemplateRole("user-inbox"))

	case "outbox-templates":
		return form.ReadOnlyLookupGroup(service.templateService.ListByTemplateRole("user-outbox"))

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

	case "searchTag-states":
		return form.NewReadOnlyLookupGroup(
			form.LookupCode{Value: "1", Label: "ALLOWED - users can search for this tag"},
			form.LookupCode{Value: "0", Label: "WAITING - has not yet been categorized."},
			form.LookupCode{Value: "-1", Label: "BLOCKED - users cannot search for this tag"},
		)

	case "sharing":
		return form.NewReadOnlyLookupGroup(
			form.LookupCode{Value: "anonymous", Label: "Everyone (including anonymous visitors)"},
			form.LookupCode{Value: "authenticated", Label: "Authenticated People Only"},
			form.LookupCode{Value: "private", Label: "Only Selected Groups"},
		)

	case "signup-templates":
		return form.ReadOnlyLookupGroup(service.registrationService.List())

	case "syndication-targets":
		domain := service.domainService.Get()
		return form.NewReadOnlyLookupGroup(domain.Syndication...)

	case "themes":
		return NewThemeLookupProvider(service.themeService)

	case "webhook-types":
		return form.NewReadOnlyLookupGroup(
			form.LookupCode{Label: "stream:create", Description: "Occurs when a Stream is first created", Value: "stream:create"},
			form.LookupCode{Label: "stream:update", Description: "Occurs when a Stream is updated", Value: "stream:update"},
			form.LookupCode{Label: "stream:delete", Description: "Occurs when a Stream is deleted", Value: "stream:delete"},
			form.LookupCode{Label: "stream:publish", Description: "Occurs when a Stream is published", Value: "stream:publish"},
			form.LookupCode{Label: "stream:publish:undo", Description: "Occurs when a Stream is unpublished", Value: "stream:publish:undo"},
			form.LookupCode{Label: "user:create", Description: "Occurs when a User is first created", Value: "user:create"},
			form.LookupCode{Label: "user:update", Description: "Occurs when a User is updated", Value: "user:update"},
			form.LookupCode{Label: "user:delete", Description: "Occurs when a User is deleted", Value: "user:delete"},
		)
	}

	// If we've fallen through to here, then look for a template-based dataset
	p := list.ByDot(path)

	// first value is the template name.  If this matches a known template, then continue
	templateName, tail := p.Split()
	if template, err := service.templateService.Load(templateName); err == nil {

		// second element is the name of the dataset
		datasetName := tail.First()

		if dataset, exists := template.Datasets[datasetName]; exists {
			return dataset // UwU
		}
	}

	// Fall through means one or more of the above tests failed.
	// We couldn't find the template or dataset, so just return an empty group.
	return form.NewReadOnlyLookupGroup()
}
