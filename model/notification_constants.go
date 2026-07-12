package model

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
