package model

// FollowerTypeStream represents a Follower that is following a Stream
const FollowerTypeStream = "Stream"

// FollowerTypeUser represents a Follower that is following a User
const FollowerTypeUser = "User"

// FollowerMethodActivityPub represents a Follower subscription that
// receives real-time updates via ActivityPub
// https://www.w3.org/TR/activitypub/
const FollowerMethodActivityPub = "ACTIVITYPUB"

// FollowerMethodEmail represents a Follower subscription that
// received real-time updates via email
const FollowerMethodEmail = "EMAIL"

// FollowerMethodWebSub represents a Follower subscription that
// receives real-time updates via WebSub
// https://websub.rocks
const FollowerMethodWebSub = "WEBSUB"

// FollowerStateActive represents an active Follower who is currently
// receiving updates from the Stream
const FollowerStateActive = "ACTIVE"

// FollowerStatePending represents an inactive Follower who has yet
// to confirm their subscription status (e.g. via email confirmation)
const FollowerStatePending = "PENDING"
