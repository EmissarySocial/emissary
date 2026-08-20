package service

import (
	"net/http"
	"slices"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/dataset"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/rosetta/slice"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// LookupProvider resolves the named lookup groups that forms use to populate their pickers
type LookupProvider struct {
	factory *Factory
	request *http.Request
	session data.Session
	userID  primitive.ObjectID
}

// NewLookupProvider returns a fully initialized LookupProvider for the provided User and request
func NewLookupProvider(factory *Factory, request *http.Request, session data.Session, userID primitive.ObjectID) LookupProvider {
	return LookupProvider{
		factory: factory,
		request: request,
		session: session,
		userID:  userID,
	}
}

// Group returns the named LookupGroup. Implements the form.LookupProvider interface.
func (service LookupProvider) Group(path string) form.LookupGroup {

	switch path {

	case "circles":
		return NewCircleLookupProvider(service.session, service.factory.Circle(), service.userID)

	case "circle-icons":
		return form.NewReadOnlyLookupGroup(dataset.Icons()...)

	case "folders":
		return NewFolderLookupProvider(service.session, service.factory.Folder(), service.userID)

	case "folder-icons":
		return form.NewReadOnlyLookupGroup(dataset.Icons()...)

	case "following-behaviors":
		return form.NewReadOnlyLookupGroup(
			form.LookupCode{Value: "POSTS+REPLIES", Label: "Posts and Replies"},
			form.LookupCode{Value: "POSTS", Label: "Posts Only (ignore replies)"},
		)

	case "geocode-tiles":
		result := form.NewReadOnlyLookupGroup(dataset.GeocodeTiles()...)
		return result

	case "group-icons":
		return form.NewReadOnlyLookupGroup(dataset.Icons()...)

	case "groups":
		return NewGroupLookupProvider(service.session, service.factory.Group())

	case "inbox-templates":
		return form.ReadOnlyLookupGroup(service.factory.Template().ListByTemplateRole("user-inbox"))

	case "merchantAccounts":
		return service.getMerchantAccounts()

	case "merchantAccounts-all-products":
		return service.getMerchantAccountsAllProducts()

	case "notification-channels":
		return form.NewReadOnlyLookupGroup(
			form.LookupCode{Value: model.NotificationChannelDirectMessage, Label: "Direct Messages", Description: "Someone sends you a private message.", Icon: "envelope"},
			form.LookupCode{Value: model.NotificationChannelReply, Label: "Replies to my posts", Description: "Someone replies to one of your posts.", Icon: "reply"},
			form.LookupCode{Value: model.NotificationChannelMentionFollowing, Label: "Mentions from people I follow", Description: "Someone you follow tags you in a public post.", Icon: "chat"},
			form.LookupCode{Value: model.NotificationChannelMentionNotFollowing, Label: "Mentions from people I don't follow", Description: "Someone you don't follow tags you in a public post.", Icon: "chat"},
			form.LookupCode{Value: model.NotificationChannelFollow, Label: "New Followers", Description: "Someone starts following you.", Icon: "person-add"},
			form.LookupCode{Value: model.NotificationChannelReaction, Label: "Boosts and Likes", Description: "Someone boosts or likes one of your posts.", Icon: "heart"},
		)

	case "outbox-templates":
		return form.ReadOnlyLookupGroup(service.factory.Template().ListByTemplateRole("user-outbox"))

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

	case "actor-rule-buttons":
		return form.NewReadOnlyLookupGroup(
			form.LookupCode{Value: "", Icon: "check-circle", Label: "Allow", Description: "All interactions are allowed: You can see posts from this person, and they can see your posts."},
			form.LookupCode{Value: "LABEL", Icon: "tag", Label: "Label", Description: "All interactions are allowed, but a content warning is added to this person's posts."},
			form.LookupCode{Value: "MUTE", Icon: "mic-mute", Label: "Mute", Description: "This person's posts are hidden, but your posts will still appear in their newsfeed. (one-way block)"},
			form.LookupCode{Value: "BLOCK", Icon: "ban", Label: "Block", Description: "This person's posts are hidden, and your posts will not appear in their newsfeed. (two-way block)"},
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
			form.LookupCode{Label: "Filter by Tag", Value: model.RuleTypeTag},
		)

	case "rule-reasons":

		return form.NewReadOnlyLookupGroup(
			form.LookupCode{Value: "account-takeover", Label: "Account Takeover"},
			form.LookupCode{Value: "apt", Label: "Advanced Persistent Threat"},
			form.LookupCode{Value: "astroturfing", Label: "Astroturfing"},
			form.LookupCode{Value: "brigading", Label: "Brigading"},
			form.LookupCode{Value: "catfishing", Label: "Catfishing"},
			form.LookupCode{Value: "cib", Label: "Coordinated Inauthentic Behaviour"},
			form.LookupCode{Value: "content-and-conduct-related-risk", Label: "Content- and Conduct-Related Risk"},
			form.LookupCode{Value: "copyright-infringement", Label: "Copyright Infringement"},
			form.LookupCode{Value: "counterfeit", Label: "Counterfeit"},
			form.LookupCode{Value: "cross-platform-abuse", Label: "Cross-Platform Abuse"},
			form.LookupCode{Value: "csam", Label: "Child Sexual Abuse Material"},
			form.LookupCode{Value: "csea", Label: "Child Sexual Exploitation and Abuse"},
			form.LookupCode{Value: "defamation", Label: "Defamation"},
			form.LookupCode{Value: "dehumanisation", Label: "Dehumanisation"},
			form.LookupCode{Value: "disinformation", Label: "Disinformation"},
			form.LookupCode{Value: "doxxing", Label: "Doxxing"},
			form.LookupCode{Value: "explicit-content", Label: "Explicit Content"},
			form.LookupCode{Value: "farming", Label: "Farming"},
			form.LookupCode{Value: "glorification-of-violence", Label: "Glorification of Violence"},
			form.LookupCode{Value: "hate-speech", Label: "Hate Speech"},
			form.LookupCode{Value: "impersonation", Label: "Impersonation"},
			form.LookupCode{Value: "incitement", Label: "Incitement"},
			form.LookupCode{Value: "misinformation", Label: "Misinformation"},
			form.LookupCode{Value: "ncii", Label: "Non-Consensual Intimate Imagery"},
			form.LookupCode{Value: "online-harassment", Label: "Online Harassment"},
			form.LookupCode{Value: "phishing", Label: "Phishing"},
			form.LookupCode{Value: "service-abuse", Label: "Service Abuse"},
			form.LookupCode{Value: "sock-puppet", Label: "Sock Puppet"},
			form.LookupCode{Value: "sextortion", Label: "Sextortion"},
			form.LookupCode{Value: "spam", Label: "Spam"},
			form.LookupCode{Value: "synthetic-media", Label: "Synthetic Media"},
			form.LookupCode{Value: "troll", Label: "Troll"},
			form.LookupCode{Value: "tvec", Label: "Terrorist and Violent Extremist Content"},
			form.LookupCode{Value: "violent-threat", Label: "Violent Threat"},
		)

	case "searchTag-states":
		return form.NewReadOnlyLookupGroup(
			form.LookupCode{Value: "2", Label: "Featured", Description: "Features this tag on search pages."},
			form.LookupCode{Value: "1", Label: "Allowed", Description: "Users can search for this tag."},
			form.LookupCode{Value: "0", Label: "Waiting", Description: "Has not yet been categorized."},
			form.LookupCode{Value: "-1", Label: "Blocked", Description: "Users cannot see this tag at all."},
		)

	case "searchTag-groups":
		return form.ReadOnlyLookupGroup(service.factory.SearchTag().ListGroups(service.session))

	case "sharing":
		return form.NewReadOnlyLookupGroup(
			form.LookupCode{Value: "anonymous", Label: "Everyone (including anonymous visitors)"},
			form.LookupCode{Value: "authenticated", Label: "Authenticated People Only"},
			form.LookupCode{Value: "private", Label: "Only Selected Groups"},
		)

	case "signup-templates":
		return form.ReadOnlyLookupGroup(service.factory.Registration().List())

	case "streams-with-products":
		return service.getSubscribableStreams()

	case "syndication-targets":
		domain := service.factory.Domain().Get()
		return form.NewReadOnlyLookupGroup(domain.Syndication...)

	case "themes":
		return NewThemeLookupProvider(service.factory.Theme())

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
	if template, err := service.factory.Template().Load(templateName); err == nil {

		// second element is the name of the dataset
		datasetName := tail.First()

		if dataset, exists := template.Datasets[datasetName]; exists {
			return dataset // UwU
		}
	}

	// Fall through means one or more of the above tests failed.
	// We couldn't find the template or dataset, so just return an empty group.
	derp.Report(derp.Internal("service.LookupProvider.Group", "Could not find template or dataset named '"+path+"'"))
	return form.NewReadOnlyLookupGroup()
}

/******************************************
 * Custom Queries
 ******************************************/

// getSubscribableStreams returns all streams that have subscribe-able content
func (service *LookupProvider) getSubscribableStreams() form.LookupGroup {

	const location = "service.LookupProvider.getSubscribableStreams"

	// Query all streams in the User's outbox that are subscribe-able
	streams, err := service.factory.Stream().QuerySubscribable(service.session, service.userID)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Loading streams with products"))
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
	result, err := service.factory.MerchantAccount().QueryByUser(service.session, service.userID)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Loading merchant accounts"))
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

	_, products, err := service.factory.Product().SyncRemoteProducts(service.session, service.userID)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Loading remote products for user", service.userID.Hex()))
		return form.NewReadOnlyLookupGroup()
	}

	result := mapProductsToLookupCodes(products...)

	// Sort the results by label
	slices.SortFunc(result, form.SortLookupCodeByGroupThenLabel)

	// Everything is cool when you're part of a team.
	return form.NewReadOnlyLookupGroup(result...)
}
