package model

import "github.com/benpate/rosetta/sliceof"

// NotificationTypeMention identifies a Notification created because the recipient was tagged (Mention) in an inbound activity
const NotificationTypeMention = "MENTION"

// NotificationTypeReply identifies a Notification created because someone replied to the recipient's content
const NotificationTypeReply = "REPLY"

// NotificationTypeLike identifies a Notification created because someone Liked the recipient's content
const NotificationTypeLike = "LIKE"

// NotificationTypeDislike identifies a Notification created because someone Disliked the recipient's content
const NotificationTypeDislike = "DISLIKE"

// NotificationTypeAnnounce identifies a Notification created because someone Announced (boosted) the recipient's content
const NotificationTypeAnnounce = "ANNOUNCE"

// NotificationTypeFollow identifies a Notification created because someone began following the recipient
const NotificationTypeFollow = "FOLLOW"

// NotificationSubtypeFollowing marks a Notification whose actor was followed by the recipient at receipt time
const NotificationSubtypeFollowing = "FOLLOWING"

// NotificationSubtypeNotFollowing marks a Notification whose actor was NOT followed by the recipient at receipt time
const NotificationSubtypeNotFollowing = "NOT_FOLLOWING"

// NotificationChannelMentionFollowing enables notifications for mentions by people the recipient follows
const NotificationChannelMentionFollowing = "MENTION_FOLLOWING"

// NotificationChannelMentionNotFollowing enables notifications for mentions by people the recipient does not follow
const NotificationChannelMentionNotFollowing = "MENTION_NOT_FOLLOWING"

// NotificationChannelReply enables notifications for replies to the recipient's posts
const NotificationChannelReply = "REPLY"

// NotificationChannelFollow enables notifications for new followers
const NotificationChannelFollow = "FOLLOW"

// NotificationChannelReaction enables notifications for likes, dislikes, and boosts
const NotificationChannelReaction = "REACTION"

// DefaultNotificationChannels returns the channels enabled for new Users: conversational
// events (mentions and replies) notify by default; ambient events (followers, reactions)
// are opt-in.  An EMPTY slice is a valid, deliberate state meaning "everything off".
func DefaultNotificationChannels() sliceof.String {
	return sliceof.String{
		NotificationChannelMentionFollowing,
		NotificationChannelMentionNotFollowing,
		NotificationChannelReply,
	}
}
