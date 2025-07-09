package service

import (
	"net/http"
	"slices"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/dataset"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/rosetta/slice"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LookupProvider struct {
	circleService          *Circle
	domainService          *Domain
	folderService          *Folder
	groupService           *Group
	merchantAccountService *MerchantAccount
	productService         *Product
	registrationService    *Registration
	searchTagService       *SearchTag
	streamService          *Stream
	templateService        *Template
	themeService           *Theme
	request                *http.Request
	userID                 primitive.ObjectID
}

func NewLookupProvider(circleService *Circle, domainService *Domain, folderService *Folder, groupService *Group, merchantAccountService *MerchantAccount, productService *Product, registrationService *Registration, searchTagService *SearchTag, streamService *Stream, templateService *Template, themeService *Theme, request *http.Request, userID primitive.ObjectID) LookupProvider {
	return LookupProvider{
		circleService:          circleService,
		domainService:          domainService,
		folderService:          folderService,
		groupService:           groupService,
		merchantAccountService: merchantAccountService,
		productService:         productService,
		registrationService:    registrationService,
		searchTagService:       searchTagService,
		streamService:          streamService,
		templateService:        templateService,
		themeService:           themeService,
		request:                request,
		userID:                 userID,
	}
}

func (service LookupProvider) Group(path string) form.LookupGroup {

	switch path {

	case "circles":
		return NewCircleLookupProvider(service.circleService, service.userID)

	case "circle-icons":
		return form.NewReadOnlyLookupGroup(dataset.Icons()...)

	case "folders":
		return NewFolderLookupProvider(service.folderService, service.userID)

	case "folder-icons":
		return form.NewReadOnlyLookupGroup(dataset.Icons()...)

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

	case "group-icons":
		return form.NewReadOnlyLookupGroup(dataset.Icons()...)

	case "groups":
		return NewGroupLookupProvider(service.groupService)

	case "inbox-templates":
		return form.ReadOnlyLookupGroup(service.templateService.ListByTemplateRole("user-inbox"))

	case "merchantAccounts":
		return service.getMerchantAccounts()

	case "merchantAccounts-all-products":
		return service.getMerchantAccountsAllProducts()

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

	case "searchTag-states":
		return form.NewReadOnlyLookupGroup(
			form.LookupCode{Value: "2", Label: "Featured", Description: "Features this tag on search pages."},
			form.LookupCode{Value: "1", Label: "Allowed", Description: "Users can search for this tag."},
			form.LookupCode{Value: "0", Label: "Waiting", Description: "Has not yet been categorized."},
			form.LookupCode{Value: "-1", Label: "Blocked", Description: "Users cannot see this tag at all."},
		)

	case "searchTag-groups":
		return form.ReadOnlyLookupGroup(service.searchTagService.ListGroups())

	case "sharing":
		return form.NewReadOnlyLookupGroup(
			form.LookupCode{Value: "anonymous", Label: "Everyone (including anonymous visitors)"},
			form.LookupCode{Value: "authenticated", Label: "Authenticated People Only"},
			form.LookupCode{Value: "private", Label: "Only Selected Groups"},
		)

	case "signup-templates":
		return form.ReadOnlyLookupGroup(service.registrationService.List())

	case "streams-with-products":
		return service.getSubscribableStreams()

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
	derp.Report(derp.InternalError("service.LookupProvider.Group", "Could not find template or dataset named '"+path+"'"))
	return form.NewReadOnlyLookupGroup()
}

/******************************************
 * Custom Queries
 ******************************************/

// getSubscribableStreams returns all streams that have subscribe-able content
func (service *LookupProvider) getSubscribableStreams() form.LookupGroup {

	// Query all streams in the User's outbox that are subscribe-able
	streams, err := service.streamService.QuerySubscribable(service.userID)

	if err != nil {
		derp.Report(derp.Wrap(err, "service.LookupProvider.getSubscribableStreams", "Error loading streams with products"))
		return form.NewReadOnlyLookupGroup()
	}

	// Convert results into a LookupGroup
	lookupCodes := slice.Map(streams, func(streamSummary model.StreamSummary) form.LookupCode {
		return form.LookupCode{
			Group: streamSummary.TemplateID,
			Value: streamSummary.StreamID(),
			Label: streamSummary.Label,
		}
	})

	// Subbesss!!
	return form.NewReadOnlyLookupGroup(lookupCodes...)
}

// getMerchantAccounts returns all merchant accounts for the current user
func (service *LookupProvider) getMerchantAccounts() form.LookupGroup {

	const location = "service.LookupProvider.getMerchantAccounts"

	// Load the Merchant Accounts for this User
	result, err := service.merchantAccountService.QueryByUser(service.userID)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Error loading merchant accounts"))
		return form.NewReadOnlyLookupGroup()
	}

	lookupCodes := slice.Map(result, func(merchantAccount model.MerchantAccount) form.LookupCode {
		return merchantAccount.LookupCode()
	})

	// Success?!?!?
	return form.NewReadOnlyLookupGroup(lookupCodes...)
}

// getMerchantAccountsAllProducts returns all products defined by the selected merchant account
func (service *LookupProvider) getMerchantAccountsAllProducts() form.LookupGroup {

	const location = "service.LookupProvider.getMerchantAccountsAllProducts"

	_, products, err := service.productService.SyncRemoteProducts(service.userID)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Error loading remote products for user", service.userID.Hex()))
		return form.NewReadOnlyLookupGroup()
	}

	result := mapProductsToLookupCodes(products...)

	// Sort the results by label
	slices.SortFunc(result, form.SortLookupCodeByGroupThenLabel)

	// Everything is cool when you're part of a team.
	return form.NewReadOnlyLookupGroup(result...)
}
