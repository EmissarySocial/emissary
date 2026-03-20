package model

// NewsItemStateUnread labels a message that has been received but not yet read by its owner
const NewsItemStateUnread = "UNREAD"

// NewsItemStateRead labels a message that has been read by its owner. If additional
// replies are recieved for this message, it will be re-displayed in their inbox.
const NewsItemStateRead = "READ"

// NewsItemStateMuted labels a message that has been read by its owner and marked as "Muted".
// If additional replies are received for this messages, it will NOT be re-displayed
// in their inbox.
const NewsItemStateMuted = "MUTED"

// NewsItemStateUnmuted is a magic state that is used to reset a message's MUTE status.
const NewsItemStateUnmuted = "UNMUTED"

// NewsItemStateNewReplies labels a message that has been read by its owner, and is now being
// re-displayed in their inbox because new replies have been received.
const NewsItemStateNewReplies = "NEW-REPLIES"
