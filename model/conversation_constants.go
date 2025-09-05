package model

// ConversationStateUnread represents a conversation that has unread messages
const ConversationStateUnread = "UNREAD"

// ConversationStateRead represents a conversation that has no unread messages
const ConversationStateRead = "READ"

// ConversationStateArchived represents a conversation that has been archived
// and will be hidden until new unread messages are received.
const ConversationStateArchived = "ARCHIVED"
