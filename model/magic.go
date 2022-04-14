package model

import "go.mongodb.org/mongo-driver/bson/primitive"

// MagicGroupIDAnonymous refers to every client on the Internet, whether signed in or not.
// Go won't let us make constant arrays, but consider this variable to be immutable.
var MagicGroupIDAnonymous primitive.ObjectID

// MaicGroupIDAuthenticated refers to every user who has been signed in, regardless of other permissions,
// but does not include Anonymous users who are not signed in.
// Go won't let us make constant arrays, but consider this variable to be immutable.
var MagicGroupIDAuthenticated primitive.ObjectID

func init() {
	MagicGroupIDAnonymous = primitive.ObjectID{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	MagicGroupIDAuthenticated = primitive.ObjectID{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
}
