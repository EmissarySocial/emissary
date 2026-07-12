package model

// FolderListSectionConversations indicates that the highlighted section
// of the inbox UX is the Conversations view, which contains unencrypted
// "direct messages", and end-to-end-encrypted "private messages".
const FolderListSectionConversations = "CONVERSATIONS"

// FolderListSectionFolder indicates that the highlighted section
// of the inbox UX is a folder - either a specific folder from the
// list, or the generic "All Folders" view.
const FolderListSectionFolder = "FOLDER"

// FolderListSectionNotifications indicates that the highlighted section
// of the inbox UX is the Notifications view (mentions, replies, likes,
// follows), which is a pinned section separate from the folder list.
const FolderListSectionNotifications = "NOTIFICATIONS"
