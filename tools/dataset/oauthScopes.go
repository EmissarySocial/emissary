package dataset

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"github.com/benpate/toot/scope"
)

// OAuthScopes returns every OAuth scope this server describes, in plain words, grouped
// and ordered as the consent page lists them
func OAuthScopes() []form.LookupCode {

	const (
		identity    = "Identity"
		read        = "Read Access"
		write       = "Write Access"
		moderation  = "Moderation"
		portability = "Data Portability"
	)

	return []form.LookupCode{

		// IndieAuth scopes.  Mastodon 4.3 added a "profile" scope of its own, with the same meaning
		{Value: model.OAuthUserTokenScopeProfile, Label: "Your Profile", Description: "See your name, profile address, and profile picture.", Icon: "person-vcard", Group: identity},
		{Value: model.OAuthUserTokenScopeEmail, Label: "Your Email Address", Description: "See the email address on your account.", Icon: "envelope", Group: identity},

		// Mastodon read scopes
		{Value: scope.Read, Label: "Read Everything", Description: "See everything in your account, including private posts, direct messages, lists, and notifications.", Icon: "eye", Group: read},
		{Value: scope.ReadAccounts, Label: "Read Profiles", Description: "See profiles, including your own preferences and the accounts and hashtags you feature.", Icon: "person", Group: read},
		{Value: scope.ReadBlocks, Label: "Read Blocks", Description: "See the accounts and domains you have blocked.", Icon: "slash-circle", Group: read},
		{Value: scope.ReadBookmarks, Label: "Read Bookmarks", Description: "See the posts you have bookmarked.", Icon: "bookmark", Group: read},
		{Value: scope.ReadCollections, Label: "Read Collections", Description: "See your collections of recommended accounts, and the collections that include you.", Icon: "view-stacked", Group: read},
		{Value: scope.ReadFavourites, Label: "Read Favorites", Description: "See the posts you have favorited.", Icon: "star", Group: read},
		{Value: scope.ReadFilters, Label: "Read Filters", Description: "See your keyword and post filters.", Icon: "funnel", Group: read},
		{Value: scope.ReadFollows, Label: "Read Follows", Description: "See who you follow, your follow requests, and the hashtags you follow.", Icon: "people", Group: read},
		{Value: scope.ReadLists, Label: "Read Lists", Description: "See your lists and the accounts on them.", Icon: "list-ul", Group: read},
		{Value: scope.ReadMutes, Label: "Read Mutes", Description: "See the accounts you have muted.", Icon: "volume-mute", Group: read},
		{Value: scope.ReadNotifications, Label: "Read Notifications", Description: "See your notifications.", Icon: "bell", Group: read},
		{Value: scope.ReadSearch, Label: "Search", Description: "Search on your behalf.", Icon: "search", Group: read},
		{Value: scope.ReadStatuses, Label: "Read Posts", Description: "See posts and timelines, including direct messages, polls, and where you stopped reading.", Icon: "chat-square-text", Group: read},
		{Value: scope.Push, Label: "Push Notifications", Description: "Receive your notifications as alerts on your devices.", Icon: "phone-vibrate", Group: read},

		// Mastodon write scopes
		{Value: scope.Write, Label: "Change Everything", Description: "Publish, edit, and delete posts, and change anything in your account.", Icon: "pencil", Group: write},
		{Value: scope.WriteAccounts, Label: "Edit Your Profile", Description: "Change your name, bio, pictures, and the accounts, hashtags, and posts you feature.", Icon: "person", Group: write},
		{Value: scope.WriteBlocks, Label: "Block and Unblock", Description: "Block and unblock accounts and domains.", Icon: "slash-circle", Group: write},
		{Value: scope.WriteBookmarks, Label: "Bookmark", Description: "Add and remove bookmarks.", Icon: "bookmark", Group: write},
		{Value: scope.WriteCollections, Label: "Manage Collections", Description: "Create, change, and delete your collections of recommended accounts.", Icon: "view-stacked", Group: write},
		{Value: scope.WriteConversations, Label: "Manage Conversations", Description: "Mark direct message conversations as read, and remove them.", Icon: "chat-dots", Group: write},
		{Value: scope.WriteFavourites, Label: "Favorite", Description: "Favorite and unfavorite posts.", Icon: "star", Group: write},
		{Value: scope.WriteFilters, Label: "Manage Filters", Description: "Create, change, and delete filters.", Icon: "funnel", Group: write},
		{Value: scope.WriteFollows, Label: "Follow and Unfollow", Description: "Follow and unfollow accounts and hashtags, and answer follow requests.", Icon: "people", Group: write},
		{Value: scope.WriteLists, Label: "Manage Lists", Description: "Create, change, and delete lists, and choose who is on them.", Icon: "list-ul", Group: write},
		{Value: scope.WriteMedia, Label: "Upload Media", Description: "Upload images, video, and audio.", Icon: "image", Group: write},
		{Value: scope.WriteMutes, Label: "Mute and Unmute", Description: "Mute and unmute accounts and conversations.", Icon: "volume-mute", Group: write},
		{Value: scope.WriteNotifications, Label: "Dismiss Notifications", Description: "Dismiss and clear your notifications.", Icon: "bell", Group: write},
		{Value: scope.WriteReports, Label: "Report", Description: "Report accounts and posts to moderators.", Icon: "exclamation-octagon", Group: write},
		{Value: scope.WriteStatuses, Label: "Publish Posts", Description: "Publish, edit, delete, and boost posts, and vote in polls.", Icon: "chat-square-text", Group: write},

		// Mastodon deprecated "follow" in 3.5, but many apps still ask for it.  toot defines no constant.
		{Value: "follow", Label: "Manage Relationships", Description: "See and change who you follow, block, and mute.", Icon: "people", Group: write},

		// Mastodon moderation scopes
		{Value: scope.AdminRead, Label: "Read All Moderation Data", Description: "See every moderation record on this server, including accounts, reports, and blocks.", Icon: "shield", Group: moderation},
		{Value: scope.AdminReadAccounts, Label: "Read Account Records", Description: "See moderation details for every account on this server, including email and IP addresses.", Icon: "person-badge", Group: moderation},
		{Value: scope.AdminReadReports, Label: "Read Reports", Description: "See the reports that people have filed.", Icon: "exclamation-octagon", Group: moderation},
		{Value: scope.AdminReadDomainAllows, Label: "Read Allowed Domains", Description: "See the domains this server federates with.", Icon: "globe2", Group: moderation},
		{Value: scope.AdminReadDomainBlocks, Label: "Read Blocked Domains", Description: "See the domains this server blocks.", Icon: "slash-circle", Group: moderation},
		{Value: scope.AdminReadIPBlocks, Label: "Read Blocked IP Addresses", Description: "See the IP address ranges this server blocks.", Icon: "hdd-network", Group: moderation},
		{Value: scope.AdminReadEmailDomainBlocks, Label: "Read Blocked Email Domains", Description: "See the email domains this server refuses at sign-up.", Icon: "envelope-slash", Group: moderation},
		{Value: scope.AdminReadCanonicalEmailBlocks, Label: "Read Blocked Email Addresses", Description: "See the email addresses this server refuses at sign-up.", Icon: "envelope-x", Group: moderation},
		{Value: scope.AdminWrite, Label: "Moderate Everything", Description: "Take any moderation action on this server.", Icon: "shield", Group: moderation},
		{Value: scope.AdminWriteAccounts, Label: "Moderate Accounts", Description: "Approve, limit, suspend, and delete accounts on this server.", Icon: "person-badge", Group: moderation},
		{Value: scope.AdminWriteReports, Label: "Resolve Reports", Description: "Assign, resolve, and reopen reports.", Icon: "exclamation-octagon", Group: moderation},
		{Value: scope.AdminWriteDomainAllows, Label: "Allow Domains", Description: "Add and remove the domains this server federates with.", Icon: "globe2", Group: moderation},
		{Value: scope.AdminWriteDomainBlocks, Label: "Block Domains", Description: "Block and unblock domains for everyone on this server.", Icon: "slash-circle", Group: moderation},
		{Value: scope.AdminWriteIPBlocks, Label: "Block IP Addresses", Description: "Block and unblock IP address ranges.", Icon: "hdd-network", Group: moderation},
		{Value: scope.AdminWriteEmailDomainBlocks, Label: "Block Email Domains", Description: "Block and unblock email domains at sign-up.", Icon: "envelope-slash", Group: moderation},
		{Value: scope.AdminWriteCanonicalEmailBlocks, Label: "Block Email Addresses", Description: "Block and unblock email addresses at sign-up.", Icon: "envelope-x", Group: moderation},

		// Emissary's data-portability export
		{Value: model.OAuthUserTokenScopeActivityPubPortability, Label: "Export All of Your Data", Description: "Copy your whole account, including private data, to another server. Approve this only if you are moving your account.", Icon: "box-arrow-right", Group: portability},
	}
}
