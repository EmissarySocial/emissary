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

// NotificationTypeDirect identifies a Notification created because someone sent the
// recipient a PRIVATE message, and outranks REPLY and MENTION in the classification ladder
const NotificationTypeDirect = "DIRECT"

// Subtype is a per-Type discriminant.  Its VOCABULARY DEPENDS ON Type: DIRECT carries the message's
// codec (MLS / PLAINTEXT); every other type carries the recipient's follow-state for the actor at
// receipt time (FOLLOWING / NOT_FOLLOWING).  Read Subtype only after switching on Type.

// NotificationSubtypeFollowing marks a Notification whose actor was followed by the recipient at receipt time
const NotificationSubtypeFollowing = "FOLLOWING"

// NotificationSubtypeNotFollowing marks a Notification whose actor was NOT followed by the recipient at receipt time
const NotificationSubtypeNotFollowing = "NOT_FOLLOWING"

// NotificationSubtypeMLS marks a DIRECT Notification whose message is MLS ciphertext
// ("message/mls"), which the server can point at but never render
const NotificationSubtypeMLS = "MLS"

// NotificationSubtypePlaintext marks a DIRECT Notification whose message is readable (non-MLS)
// content.  Private, but not encrypted end-to-end.
const NotificationSubtypePlaintext = "PLAINTEXT"

// NotificationChannelDirectMessage enables notifications for private messages, and is
// NOT split by follow-state the way the mention channels are
const NotificationChannelDirectMessage = "DIRECT_MESSAGE"

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

// AllNotificationChannels returns every notification channel a User may enable,
// in display order
func AllNotificationChannels() sliceof.String {

	// This is the source of the User schema's enum, so a channel missing here cannot be
	// saved through a form even though the constant exists.
	return sliceof.String{
		NotificationChannelDirectMessage,
		NotificationChannelMentionFollowing,
		NotificationChannelMentionNotFollowing,
		NotificationChannelReply,
		NotificationChannelFollow,
		NotificationChannelReaction,
	}
}

// DefaultNotificationChannels returns the channels enabled for new Users
func DefaultNotificationChannels() sliceof.String {

	// Conversational events (direct messages, mentions, replies) notify by default;
	// ambient events (followers, reactions) are opt-in. An EMPTY slice is a valid,
	// deliberate state meaning "everything off".
	return sliceof.String{
		NotificationChannelDirectMessage,
		NotificationChannelMentionFollowing,
		NotificationChannelMentionNotFollowing,
		NotificationChannelReply,
	}
}
