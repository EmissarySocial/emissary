package model

// InboxMessageStateReceived represents an inbox message that has been received, but has not yet passed through any block filters.
// "Received" messages are not visible to users
const InboxMessageStateReceived = "RECEIVED"

// InboxMessageStateVisible represents an inbox message that has been received and passed through all block filters.
// "Visible" messages are visible to users in their designated inbox folders.
const InboxMessageStateVisible = "VISIBLE"

// InboxMessageStateMuted represents an inbox message that has been muted by the user's block settings.
// "Muted" messages are visible to users in a special folder, but are not visible in the main inbox.
const InboxMessageStateMuted = "MUTED"

// InboxMessageStateBlocked represents an inbox message that has been blocked by the user's block settings.
// "Blocked" messages should not be saved to the database, and should be discarded by the inbox service without
// any feedback to the originating server.
const InboxMessageStateBlocked = "BLOCKED"
