package model

// FollowerTypeApplication represents the domain service actor
const FollowerTypeApplication = "Application"

// FollowerTypeSearch represents a Follower that is following a Search Query
const FollowerTypeSearch = "Search"

// FollowerTypeSearchDomain represents a Follower that is following a Global Domain Query
const FollowerTypeSearchDomain = "SearchDomain"

// FollowerTypeStream represents a Follower that is following a Stream
const FollowerTypeStream = "Stream"

// FollowerTypeUser represents a Follower that is following a User
const FollowerTypeUser = "User"

const ActorTypeApplication = "Application"

const ActorTypeSearchDomain = "SearchDomain"

const ActorTypeSearchQuery = "Search"

const ActorTypeStream = "Stream"

const ActorTypeUser = "User"

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

// FollowerStateDeleted represents an inacti e Follower record that has been
// deleted.  The canonical value for this is still the `DeleteDate` field,
// but this value is also used for convenience.
const FollowerStateDeleted = "DELETED"
