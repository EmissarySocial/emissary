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

// FollowerStateDeleted represents an inacti e Follower record that has been
// deleted.  The canonical value for this is still the `DeleteDate` field,
// but this value is also used for convenience.
const FollowerStateDeleted = "DELETED"

// FollowerStateImportPending represents a Follower who has followed
// this User on a previous server, and whose record has been Imported.
// This Follower will not receive notifications until the Import is
// finalized with a `Move` announcement, at which point it will be up
// to the Follower's server to re-follow the newly imported account.
const FollowerStateImportPending = "IMPORT-PENDING"

// FollowerStatePending represents an inactive Follower who has yet
// to confirm their subscription status (e.g. via email confirmation)
const FollowerStatePending = "PENDING"
