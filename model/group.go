package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/form"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Group struct {
	GroupID     primitive.ObjectID `bson:"_id"`         // Unique identifier assigned by the database
	Token       string             `bson:"token"`       // Uniqe token chosen by the administrator
	Label       string             `bson:"label"`       // Human-readable label for this group.
	Description string             `bson:"description"` // Human-readable description of this Group
	Icon        string             `bson:"icon"`        // Icon for this Group

	journal.Journal `json:"-" bson:",inline"`
}

func NewGroup() Group {
	return Group{
		GroupID: primitive.NewObjectID(),
	}
}

func GroupFields() []string {
	return []string{"_id", "label", "description", "icon"}
}

func (userSummary Group) Fields() []string {
	return GroupFields()
}

/******************************************
 * data.Object Interface
 ******************************************/

func (group *Group) ID() string {
	return group.GroupID.Hex()
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
	return NewPermissions()
}

// RolesToPrivilegeIDs returns a slice of Privileges that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (group *Group) RolesToPrivilegeIDs(roleIDs ...string) Permissions {
	return NewPermissions()
}

/******************************************
 * Other Data Accessors
 ******************************************/

func (group Group) IconWithDefault() string {
	if group.Icon == "" {
		return "people"
	}
	return group.Icon
}

func (group Group) LookupCode() form.LookupCode {
	return form.LookupCode{
		Value:       group.GroupID.Hex(),
		Label:       group.Label,
		Description: group.Description,
		Icon:        group.IconWithDefault(),
	}
}
