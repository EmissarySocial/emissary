package model

// MessageStatusNew labels a message that has been received but not yet read by its owner
const MessageStatusUnread = "UNREAD"

// MessageStatusRead labels a message that has been read by its owner. If additional
// replies are recieved for this message, it will be re-displayed in their inbox.
const MessageStatusRead = "READ"

// MessageStatusMuted labels a message that has been read by its owner and marked as "Muted".
// If additional replies are received for this messages, it will NOT be re-displayed
// in their inbox.
const MessageStatusMuted = "MUTED"

// MessageStatusNewReplies labels a message that has been read by its owner, and is now being
// re-displayed in their inbox because new replies have been received.
const MessageStatusNewReplies = "NEW-REPLIES"
