package handler

import (
	"github.com/EmissarySocial/emissary/handler/mastodon"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot"
)

func Mastodon(serverFactory *server.Factory) toot.API[model.Authorization] {

	return toot.API[model.Authorization]{
		Authorize: mastodon.Authorizer(serverFactory),

		// https://docs.joinmastodon.org/methods/accounts/
		PostAccount:                    mastodon.PostAccount(serverFactory),
		GetAccount_VerifyCredentials:   mastodon.GetAccount_VerifyCredentials(serverFactory),
		PatchAccount_UpdateCredentials: mastodon.PatchAccount_UpdateCredentials(serverFactory),
		GetAccount:                     mastodon.GetAccount(serverFactory),
		GetAccount_Statuses:            mastodon.GetAccount_Statuses(serverFactory),
		GetAccount_Followers:           mastodon.GetAccount_Followers(serverFactory),
		GetAccount_Following:           mastodon.GetAccount_Following(serverFactory),
		GetAccount_FeaturedTags:        mastodon.GetAccount_FeaturedTags(serverFactory),
		PostAccount_Follow:             mastodon.PostAccount_Follow(serverFactory),
		PostAccount_Unfollow:           mastodon.PostAccount_Unfollow(serverFactory),
		PostAccount_Block:              mastodon.PostAccount_Block(serverFactory),
		PostAccount_Unblock:            mastodon.PostAccount_Unblock(serverFactory),
		PostAccount_Mute:               mastodon.PostAccount_Mute(serverFactory),
		PostAccount_Unmute:             mastodon.PostAccount_Unmute(serverFactory),
		PostAccount_Pin:                mastodon.PostAccount_Pin(serverFactory),
		PostAccount_Unpin:              mastodon.PostAccount_Unpin(serverFactory),
		PostAccount_Note:               mastodon.PostAccount_Note(serverFactory),
		GetAccount_Relationships:       mastodon.GetAccount_Relationships(serverFactory),
		GetAccount_FamiliarFollowers:   mastodon.GetAccount_FamiliarFollowers(serverFactory),
		GetAccount_Search:              mastodon.GetAccount_Search(serverFactory),
		GetAccount_Lookup:              mastodon.GetAccount_Lookup(serverFactory),

		// https://docs.joinmastodon.org/methods/announcements/
		GetAnnouncements:            mastodon.GetAnnouncements(serverFactory),
		PostAnnouncement_Dismiss:    mastodon.PostAnnouncement_Dismiss(serverFactory),
		PutAnnouncement_Reaction:    mastodon.PutAnnouncement_Reaction(serverFactory),
		DeleteAnnouncement_Reaction: mastodon.DeleteAnnouncement_Reaction(serverFactory),

		// https://docs.joinmastodon.org/methods/apps/
		PostApplication:                  mastodon.PostApplication(serverFactory),
		GetApplication_VerifyCredentials: mastodon.GetApplication_VerifyCredentials(serverFactory),

		// https://docs.joinmastodon.org/methods/blocks/
		GetBlocks: mastodon.GetBlocks(serverFactory),

		// https://docs.joinmastodon.org/methods/bookmarks/
		GetBookmarks: mastodon.GetBookmarks(serverFactory),

		// https://docs.joinmastodon.org/methods/conversations/
		GetConversations:     mastodon.GetConversations(serverFactory),
		DeleteConversation:   mastodon.DeleteConversation(serverFactory),
		PostConversationRead: mastodon.PostConversationRead(serverFactory),

		// https://docs.joinmastodon.org/methods/custom_emojis/
		GetCustomEmojis: mastodon.GetCustomEmojis(serverFactory),

		// https://docs.joinmastodon.org/methods/directory/
		GetDirectory: mastodon.GetDirectory(serverFactory),

		// https://docs.joinmastodon.org/methods/domain_blocks/
		GetDomainBlocks:   mastodon.GetDomainBlocks(serverFactory),
		PostDomainBlock:   mastodon.PostDomainBlock(serverFactory),
		DeleteDomainBlock: mastodon.DeleteDomainBlock(serverFactory),

		// https://docs.joinmastodon.org/methods/emails/
		PostEmailConfirmation: mastodon.PostEmailConfirmation(serverFactory),

		// https://docs.joinmastodon.org/methods/endorsements/
		GetEndorsements: mastodon.GetEndorsements(serverFactory),

		// https://docs.joinmastodon.org/methods/favourites/
		GetFavourites: mastodon.GetFavourites(serverFactory),

		// https://docs.joinmastodon.org/methods/featured_tags/
		GetFeaturedTags:             mastodon.GetFeaturedTags(serverFactory),
		PostFeaturedTag:             mastodon.PostFeaturedTag(serverFactory),
		DeleteFeaturedTag:           mastodon.DeleteFeaturedTag(serverFactory),
		GetFeaturedTags_Suggestions: mastodon.GetFeaturedTags_Suggestions(serverFactory),

		// https://docs.joinmastodon.org/methods/filters/
		GetFilters:           mastodon.GetFilters(serverFactory),
		GetFilter:            mastodon.GetFilter(serverFactory),
		PostFilter:           mastodon.PostFilter(serverFactory),
		PutFilter:            mastodon.PutFilter(serverFactory),
		DeleteFilter:         mastodon.DeleteFilter(serverFactory),
		GetFilter_Keywords:   mastodon.GetFilter_Keywords(serverFactory),
		PostFilter_Keyword:   mastodon.PostFilter_Keyword(serverFactory),
		GetFilter_Keyword:    mastodon.GetFilter_Keyword(serverFactory),
		PutFilter_Keyword:    mastodon.PutFilter_Keyword(serverFactory),
		DeleteFilter_Keyword: mastodon.DeleteFilter_Keyword(serverFactory),
		GetFilter_Statuses:   mastodon.GetFilter_Statuses(serverFactory),
		PostFilter_Status:    mastodon.PostFilter_Status(serverFactory),
		GetFilter_Status:     mastodon.GetFilter_Status(serverFactory),
		DeleteFilter_Status:  mastodon.DeleteFilter_Status(serverFactory),
		GetFilter_V1:         mastodon.GetFilter_V1(serverFactory),
		PostFilter_V1:        mastodon.PostFilter_V1(serverFactory),
		PutFilter_V1:         mastodon.PutFilter_V1(serverFactory),
		DeleteFilter_V1:      mastodon.DeleteFilter_V1(serverFactory),

		// https://docs.joinmastodon.org/methods/follow_requests/
		GetFollowRequests:           mastodon.GetFollowRequests(serverFactory),
		PostFollowRequest_Authorize: mastodon.PostFollowRequest_Authorize(serverFactory),
		PostFollowRequest_Reject:    mastodon.PostFollowRequest_Reject(serverFactory),

		// https://docs.joinmastodon.org/methods/followed_tags/
		GetFollowedTags: mastodon.GetFollowedTags(serverFactory),

		// https://docs.joinmastodon.org/methods/instance/
		GetInstance:                     mastodon.GetInstance(serverFactory),
		GetInstance_Peers:               mastodon.GetInstance_Peers(serverFactory),
		GetInstance_Activity:            mastodon.GetInstance_Activity(serverFactory),
		GetInstance_Rules:               mastodon.GetInstance_Rules(serverFactory),
		GetInstance_DomainBlocks:        mastodon.GetInstance_DomainBlocks(serverFactory),
		GetInstance_ExtendedDescription: mastodon.GetInstance_ExtendedDescription(serverFactory),

		// https://docs.joinmastodon.org/methods/lists/
		GetLists:            mastodon.GetLists(serverFactory),
		GetList:             mastodon.GetList(serverFactory),
		PostList:            mastodon.PostList(serverFactory),
		PutList:             mastodon.PutList(serverFactory),
		DeleteList:          mastodon.DeleteList(serverFactory),
		GetList_Accounts:    mastodon.GetList_Accounts(serverFactory),
		PostList_Accounts:   mastodon.PostList_Accounts(serverFactory),
		DeleteList_Accounts: mastodon.DeleteList_Accounts(serverFactory),

		// https://docs.joinmastodon.org/methods/markers/
		GetMarkers: mastodon.GetMarkers(serverFactory),
		PostMarker: mastodon.PostMarker(serverFactory),

		// https://docs.joinmastodon.org/methods/media/
		PostMedia: mastodon.PostMedia(serverFactory),

		// https://docs.joinmastodon.org/methods/mutes/
		GetMutes: mastodon.GetMutes(serverFactory),

		// https://docs.joinmastodon.org/methods/notifications/
		GetNotifications:         mastodon.GetNotifications(serverFactory),
		GetNotification:          mastodon.GetNotification(serverFactory),
		PostNotifications_Clear:  mastodon.PostNotifications_Clear(serverFactory),
		PostNotification_Dismiss: mastodon.PostNotification_Dismiss(serverFactory),

		// https://docs.joinmastodon.org/methods/oauth/
		GetOAuth_Authorize: mastodon.GetOAuth_Authorize(serverFactory),
		PostOAuth_Token:    mastodon.PostOAuth_Token(serverFactory),
		PostOAuth_Revoke:   mastodon.PostOAuth_Revoke(serverFactory),

		// https://docs.joinmastodon.org/methods/oembed/
		GetOEmbed: mastodon.GetOEmbed(serverFactory),

		// https://docs.joinmastodon.org/methods/polls/
		GetPoll:        mastodon.GetPoll(serverFactory),
		PostPoll_Votes: mastodon.PostPoll_Votes(serverFactory),

		// https://docs.joinmastodon.org/methods/preferences/
		GetPreferences: mastodon.GetPreferences(serverFactory),

		// https://docs.joinmastodon.org/methods/profile/
		DeleteProfile_Avatar: mastodon.DeleteProfile_Avatar(serverFactory),
		DeleteProfile_Header: mastodon.DeleteProfile_Header(serverFactory),

		// https://docs.joinmastodon.org/methods/reports/
		PostReport: mastodon.PostReport(serverFactory),

		// https://docs.joinmastodon.org/methods/scheduled_statuses/
		GetScheduledStatuses:  mastodon.GetScheduledStatuses(serverFactory),
		GetScheduledStatus:    mastodon.GetScheduledStatus(serverFactory),
		PutScheduledStatus:    mastodon.PutScheduledStatus(serverFactory),
		DeleteScheduledStatus: mastodon.DeleteScheduledStatus(serverFactory),

		// https://docs.joinmastodon.org/methods/search/
		GetSearch: mastodon.GetSearch(serverFactory),

		// https://docs.joinmastodon.org/methods/statuses/#create
		PostStatus:             mastodon.PostStatus(serverFactory),
		GetStatus:              mastodon.GetStatus(serverFactory),
		DeleteStatus:           mastodon.DeleteStatus(serverFactory),
		GetStatus_Context:      mastodon.GetStatus_Context(serverFactory),
		PostStatus_Translate:   mastodon.PostStatus_Translate(serverFactory),
		GetStatus_RebloggedBy:  mastodon.GetStatus_RebloggedBy(serverFactory),
		GetStatus_FavouritedBy: mastodon.GetStatus_FavouritedBy(serverFactory),
		PostStatus_Favourite:   mastodon.PostStatus_Favourite(serverFactory),
		PostStatus_Unfavourite: mastodon.PostStatus_Unfavourite(serverFactory),
		PostStatus_Reblog:      mastodon.PostStatus_Reblog(serverFactory),
		PostStatus_Unreblog:    mastodon.PostStatus_Unreblog(serverFactory),
		PostStatus_Bookmark:    mastodon.PostStatus_Bookmark(serverFactory),
		PostStatus_Unbookmark:  mastodon.PostStatus_Unbookmark(serverFactory),
		PostStatus_Mute:        mastodon.PostStatus_Mute(serverFactory),
		PostStatus_Unmute:      mastodon.PostStatus_Unmute(serverFactory),
		PostStatus_Pin:         mastodon.PostStatus_Pin(serverFactory),
		PostStatus_Unpin:       mastodon.PostStatus_Unpin(serverFactory),
		PutStatus:              mastodon.PutStatus(serverFactory),
		GetStatus_History:      mastodon.GetStatus_History(serverFactory),
		GetStatus_Source:       mastodon.GetStatus_Source(serverFactory),

		// https://docs.joinmastodon.org/methods/suggestions/
		GetSuggestions:   mastodon.GetSuggestions(serverFactory),
		DeleteSuggestion: mastodon.DeleteSuggestion(serverFactory),

		// https://docs.joinmastodon.org/methods/tags/
		GetTag:           mastodon.GetTag(serverFactory),
		PostTag_Follow:   mastodon.PostTag_Follow(serverFactory),
		PostTag_Unfollow: mastodon.PostTag_Unfollow(serverFactory),

		// https://docs.joinmastodon.org/methods/timelines/
		GetTimeline_Public:  mastodon.GetTimeline_Public(serverFactory),
		GetTimeline_Hashtag: mastodon.GetTimeline_Hashtag(serverFactory),
		GetTimeline_Home:    mastodon.GetTimeline_Home(serverFactory),
		GetTimeline_List:    mastodon.GetTimeline_List(serverFactory),

		// https://docs.joinmastodon.org/methods/trends/
		GetTrends:          mastodon.GetTrends(serverFactory),
		GetTrends_Statuses: mastodon.GetTrends_Statuses(serverFactory),
		GetTrends_Links:    mastodon.GetTrends_Links(serverFactory),
	}
}
