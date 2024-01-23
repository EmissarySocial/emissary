package model

// MessageStateNew labels a message that has been received but not yet read by its owner
const MessageStateUnread = "UNREAD"

// MessageStateRead labels a message that has been read by its owner. If additional
// replies are recieved for this message, it will be re-displayed in their inbox.
const MessageStateRead = "READ"

// MessageStateMuted labels a message that has been read by its owner and marked as "Muted".
// If additional replies are received for this messages, it will NOT be re-displayed
// in their inbox.
const MessageStateMuted = "MUTED"

// MessageStateNewReplies labels a message that has been read by its owner, and is now being
// re-displayed in their inbox because new replies have been received.
const MessageStateNewReplies = "NEW-REPLIES"
