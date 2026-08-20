package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/form"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Group is a named set of Users, used to grant permissions in bulk
type Group struct {
	GroupID     primitive.ObjectID `bson:"_id"`         // Unique identifier assigned by the database
	Token       string             `bson:"token"`       // Uniqe token chosen by the administrator
	Label       string             `bson:"label"`       // Human-readable label for this group.
	Description string             `bson:"description"` // Human-readable description of this Group
	Icon        string             `bson:"icon"`        // Icon for this Group

	journal.Journal `json:"-" bson:",inline"`
}

// NewGroup returns a fully initialized, empty Group
func NewGroup() Group {
	return Group{
		GroupID: primitive.NewObjectID(),
	}
}

// GroupFields returns the database fields required to populate a Group
func GroupFields() []string {
	return []string{"_id", "label", "description", "icon"}
}

// Fields returns the database fields required to populate a Group
func (userSummary Group) Fields() []string {
	return GroupFields()
}

/******************************************
 * data.Object Interface
 ******************************************/

// ID returns the primary key of this Group, as a string
func (group *Group) ID() string {
	return group.GroupID.Hex()
}

/******************************************
 * Mock Activity Vocabulary
 ******************************************/

// Name returns the human-readable label of this Group
func (group Group) Name() string {
	return group.Label
}

// Summary returns an abbreviated copy of this Group
func (group Group) Summary() string {
	return group.Description
}

/******************************************
 * AccessLister Interface
 ******************************************/

// State returns the current state of this Group.
// It is part of the AccessLister interface
func (group *Group) State() string {
	return "default"
}

// IsAuthor returns TRUE if the provided UserID the author of this Group
// It is part of the AccessLister interface
func (group *Group) IsAuthor(authorID primitive.ObjectID) bool {
	return false
}

// IsMyself returns TRUE if this object directly represents the provided UserID
// It is part of the AccessLister interface
func (group *Group) IsMyself(userID primitive.ObjectID) bool {
	return false
}

// RolesToGroupIDs returns a slice of Group IDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (group *Group) RolesToGroupIDs(roleIDs ...string) Permissions {
	return defaultRolesToGroupIDs(primitive.NilObjectID, roleIDs...)
}

// RolesToPrivilegeIDs returns a slice of Privileges that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (group *Group) RolesToPrivilegeIDs(roleIDs ...string) Permissions {
	return NewPermissions()
}

/******************************************
 * Other Data Accessors
 ******************************************/

// IconWithDefault returns this Group's icon, falling back to the generic "people" icon
func (group Group) IconWithDefault() string {
	if group.Icon == "" {
		return "people"
	}
	return group.Icon
}

// LookupCode returns this Group as a form.LookupCode, so it can be listed in a picker
func (group Group) LookupCode() form.LookupCode {
	return form.LookupCode{
		Value:       group.GroupID.Hex(),
		Label:       group.Label,
		Description: group.Description,
		Icon:        group.IconWithDefault(),
	}
}
