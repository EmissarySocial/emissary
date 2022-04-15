package model

import "go.mongodb.org/mongo-driver/bson/primitive"

// MagicGroupIDAnonymous refers to a user who has not been signed in.
// Every user on the Internet is given this group, whether signed in or not.
// Go won't let us make constant arrays, but consider this variable to be immutable.
var MagicGroupIDAnonymous primitive.ObjectID

// MaicGroupIDAuthenticated refers to every user who has been signed in, regardless of other permissions,
// but does not include Anonymous users who are not signed in.
// Go won't let us make constant arrays, but consider this variable to be immutable.
var MagicGroupIDAuthenticated primitive.ObjectID

// MagicRoleAnonymous grants permissions to a user who has not been signed in.
const MagicRoleAnonymous = "anonymous"

// MagicRoleAuthenticated grants permissions to a user who has been signed in, regardless of any other group roles.
const MagicRoleAuthenticated = "authenticated"

// MagicRoleAuthor grants permissions to the user who originally created a stream
const MagicRoleAuthor = "author"

// MagicRoleMyself grants permissions to a user who is trying to access their own profile.
const MagicRoleMyself = "self"

// MagicRoleOwner grants full access to a user with database owner privileges
const MagicRoleOwner = "owner"

func init() {
	MagicGroupIDAnonymous = primitive.ObjectID{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	MagicGroupIDAuthenticated = primitive.ObjectID{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
}
