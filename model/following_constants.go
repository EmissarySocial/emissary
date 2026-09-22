package model

/******************************************
 * Following Methods
 ******************************************/

// FollowingMethodActivityPub represents the ActivityPub subscription
// https://www.w3.org/TR/activitypub/
const FollowingMethodActivityPub = "ACTIVITYPUB"

// FollowingMethodPoll represents a subscription that must be polled for updates
const FollowingMethodPoll = "POLL"

/******************************************
 * Following Statuses
 ******************************************/

// FollowingStatusNew represents a new following that has not yet been polled
const FollowingStatusNew = "NEW"

// FollowingStatusLoading represents a following that is being loaded for the first time
const FollowingStatusLoading = "LOADING"

// FollowingStatusImportPending represents a following that has been imported from a remote server,
// but the import has not been finalized.  This is a placeholder record until the user
// finalized the migration with a "Move" announcement.  At that point, the server will send
// a "Follow" request to the remote server
const FollowingStatusImportPending = "IMPORT-PENDING"

// FollowingStatusSuccess represents a following that has successfully loaded
const FollowingStatusSuccess = "SUCCESS"

// FollowingStatusFailure represents a following that has failed to load
const FollowingStatusFailure = "FAILURE"

// FollowingStatusBlocked represents a following whose actor the user has blocked (R8).  It has
// sent its Undo/Follow and is never polled; only an explicit re-follow resumes it.
const FollowingStatusBlocked = "BLOCKED"

// FollowingStatusPaused represents a following that has failed for so long (30+ days without a
// successful retrieval, and at least 5 consecutive failures) that polling has backed off to
// once every 90 days.  It is still a failure, and a successful poll clears it.
const FollowingStatusPaused = "PAUSED"

// FollowingStatusGone represents a following whose actor answered 410 Gone: the remote server
// has stated the account is deleted.  It is never polled again; only an explicit re-save
// reconnects it.
const FollowingStatusGone = "GONE"

/******************************************
 * Following Behaviors
 ******************************************/

// FollowingBehaviorPostsAndReplies declares that all messages (both Posts and Replies) should be imported from a followed account
const FollowingBehaviorPostsAndReplies = "POSTS+REPLIES"

// FollowingBehaviorPosts declares that only Posts (not Replies) should be imported from a followed account
const FollowingBehaviorPosts = "POSTS"
