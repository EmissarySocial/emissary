package model

// Role is used in a map[sring]Role within each Template.  Role IDs are used to
// identify what actions a User can take on a Stream (given the user's Groups and the Stream's Template)
type Role struct {
	RoleID       string `json:"roleId"      bson:"roleId"`      // Unique ID for this role
	Label        string `json:"label"       bson:"label"`       // Short, human-friendly label used to select this role in UX
	Description  string `json:"description" bson:"description"` // Medium-length, human-friendly description that gives more details about this role
	IsPrivileged bool   `json:"privileged"  bson:"privileged"`  // Whether this role can be purchased by a guest or assigned to a Circle member
}
