package model

// FollowingMethodActivityPub represents the ActivityPub subscription
// https://www.w3.org/TR/activitypub/
const FollowingMethodActivityPub = "ACTIVITYPUB"

// FollowingMethodPoll represents a subscription that must be polled for updates
const FollowingMethodPoll = "POLL"

// FollowingStatusNew represents a new following that has not yet been polled
const FollowingStatusNew = "NEW"

// FollowingStatusLoading represents a following that is being loaded for the first time
const FollowingStatusLoading = "LOADING"

// FollowingStatusImportPending represents a following that has been imported from a remote server,
// but the import has not been finalized.  This is a placeholder record until the user
// finalized the migration with a "Move" announcement.  At that point, the server will send
// a "Follow" request to the
const FollowingStatusImportPending = "IMPORT-PENDING"

// FollowingStatusSuccess represents a following that has successfully loaded
const FollowingStatusSuccess = "SUCCESS"

// FollowingStatusFailure represents a following that has failed to load
const FollowingStatusFailure = "FAILURE"

// FollowingStatusPaused represents a following that is paused by a BLOCK rule -- and exactly
// that (R8). A paused Following has sent its Undo/Follow and is excluded from polling; it is
// never auto-resumed (deleting the block offers re-follow, it does not re-follow for you).
const FollowingStatusPaused = "PAUSED"

// FollowingBehaviorPostsAndReplies declares that all messages (both Posts and Replies) should be imported from a followed account
const FollowingBehaviorPostsAndReplies = "POSTS+REPLIES"

// FollowingBehaviorPosts declares that only Posts (not Replies) should be imported from a followed account
const FollowingBehaviorPosts = "POSTS"
