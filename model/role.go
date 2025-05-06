package model

// Role is used in a map[sring]Role within each Template.  Role IDs are used to
// identify what actions a User can take on a Stream (given the user's Groups and the Stream's Template)
type Role struct {
	RoleID      string `bson:"roleId"`      // Unique ID for this role
	Label       string `bson:"label"`       // Short, human-friendly label used to select this role in UX
	Description string `bson:"description"` // Medium-length, human-friendly description that gives more details about this role
	Purchasable bool   `bson:"purchasable"` // Whether this role can be purchased by a guest
}
