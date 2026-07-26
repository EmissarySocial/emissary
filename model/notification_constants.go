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

// NotificationTypeDirect identifies a Notification created because someone sent the recipient a
// PRIVATE message: a non-public activity addressed to them by name.  It outranks REPLY and MENTION
// in the classification ladder (see service.Notification.notifyFromCreateOrUpdate), because a
// direct message belongs to the Conversations app no matter what else it also is.
const NotificationTypeDirect = "DIRECT"

// Subtype is a per-Type discriminant.  Its VOCABULARY DEPENDS ON Type: DIRECT carries the message's
// codec (MLS / PLAINTEXT); every other type carries the recipient's follow-state for the actor at
// receipt time (FOLLOWING / NOT_FOLLOWING).  Read Subtype only after switching on Type.

// NotificationSubtypeFollowing marks a Notification whose actor was followed by the recipient at receipt time
const NotificationSubtypeFollowing = "FOLLOWING"

// NotificationSubtypeNotFollowing marks a Notification whose actor was NOT followed by the recipient at receipt time
const NotificationSubtypeNotFollowing = "NOT_FOLLOWING"

// NotificationSubtypeMLS marks a DIRECT Notification whose message is MLS ciphertext (media type
// "message/mls").  Only the Conversations app holds the group's ratchet state, so the server can
// never render this message -- it can only point at it.
const NotificationSubtypeMLS = "MLS"

// NotificationSubtypePlaintext marks a DIRECT Notification whose message is readable (non-MLS)
// content.  Private, but not encrypted end-to-end.
const NotificationSubtypePlaintext = "PLAINTEXT"

// NotificationChannelDirectMessage enables notifications for private messages sent to the recipient.
// Unlike the mention channels, it is NOT split by follow-state: a DIRECT Notification's Subtype
// carries the message codec (MLS/PLAINTEXT), so the recipient's follow-state is not recorded on the
// record and Channels() -- a pure method -- cannot look it up.
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

// DefaultNotificationChannels returns the channels enabled for new Users: conversational
// events (direct messages, mentions, and replies) notify by default; ambient events (followers,
// reactions) are opt-in.  An EMPTY slice is a valid, deliberate state meaning "everything off".
func DefaultNotificationChannels() sliceof.String {
	return sliceof.String{
		NotificationChannelDirectMessage,
		NotificationChannelMentionFollowing,
		NotificationChannelMentionNotFollowing,
		NotificationChannelReply,
	}
}
