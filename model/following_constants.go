package model

// FollowMethodActivityPub represents the ActivityPub subscription
// https://www.w3.org/TR/activitypub/
const FollowMethodActivityPub = "ACTIVITYPUB"

// FollowMethodPoll represents a subscription that must be polled for updates
const FollowMethodPoll = "POLL"

// FollowMethodWebSub represents a WebSub subscription
// https://websub.rocks
const FollowMethodWebSub = "WEBSUB"

const FollowMethodRssCloud = "RSSCLOUD"

// FollowingStatusNew represents a new following that has not yet been polled
const FollowingStatusNew = "NEW"

// FollowingStatusLoading represents a following that is being loaded for the first time
const FollowingStatusLoading = "LOADING"

// FollowingStatusSuccess represents a following that has successfully loaded
const FollowingStatusSuccess = "SUCCESS"

// FollowingStatusFailure represents a following that has failed to load
const FollowingStatusFailure = "FAILURE"

// FollowingBlockIgnore declares that Blocks published by a followed account should be ignored
const FollowingBlockActionIgnore = "IGNORE"

// FollowingBehaviorPostsAndReplies declares that all messages (both Posts and Replies) should be imported from a followed account
const FollowingBehaviorPostsAndReplies = "POSTS+REPLIES"

// FollowingBehaviorPosts declares that only Posts (not Replies) should be imported from a followed account
const FollowingBehaviorPosts = "POSTS"
