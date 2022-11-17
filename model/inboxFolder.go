package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type InboxFolder struct {
	InboxFolderID primitive.ObjectID `path:"inboxFolderId" json:"inboxFolderId" bson:"_id"`    // Unique ID for this folder
	UserID        primitive.ObjectID `path:"userId"        json:"userId"        bson:"userId"` // ID of the User who owns this folder
	Label         string             `path:"label"         json:"label"         bson:"label"`  // Label of the folder
	Rank          int                `path:"rank"          json:"rank"          bson:"rank"`   // Sort order of the folder

	journal.Journal `json:"-" bson:"journal"`
}

func NewInboxFolder() InboxFolder {
	return InboxFolder{
		InboxFolderID: primitive.NewObjectID(),
	}
}

func InboxFolderSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"inboxFolderId": schema.String{Format: "objectId"},
			"userId":        schema.String{Format: "objectId"},
			"label":         schema.String{MaxLength: 100},
			"rank":          schema.Integer{},
		},
	}
}

/*******************************************
 * data.Object Interface
 *******************************************/

func (folder InboxFolder) ID() string {
	return folder.InboxFolderID.Hex()
}

func (folder *InboxFolder) GetObjectID(name string) (primitive.ObjectID, error) {
	switch name {
	case "inboxFolderId":
		return folder.InboxFolderID, nil
	case "userId":
		return folder.UserID, nil
	}
	return primitive.NilObjectID, derp.NewInternalError("model.InboxFolder.GetObjectID", "Invalid property", name)
}

func (folder *InboxFolder) GetString(name string) (string, error) {
	switch name {
	case "label":
		return folder.Label, nil
	}
	return "", derp.NewInternalError("model.InboxFolder.GetString", "Invalid property", name)
}

func (folder *InboxFolder) GetInt(name string) (int, error) {
	switch name {
	case "rank":
		return folder.Rank, nil
	}
	return 0, derp.NewInternalError("model.InboxFolder.GetInt", "Invalid property", name)
}

func (folder *InboxFolder) GetInt64(name string) (int64, error) {
	return 0, derp.NewInternalError("model.InboxFolder.GetInt64", "Invalid property", name)
}

func (folder *InboxFolder) GetBool(name string) (bool, error) {
	return false, derp.NewInternalError("model.InboxFolder.GetBool", "Invalid property", name)
}

/*******************************************
 * RoleStateEnumerator Interface
 *******************************************/

// State returns the current state of this object.
// For users, there is no state, so it returns ""
func (folder InboxFolder) State() string {
	return ""
}

// Roles returns a list of all roles that match the provided authorization
func (folder InboxFolder) Roles(authorization *Authorization) []string {

	// Everyone has "anonymous" access
	result := []string{MagicRoleAnonymous}

	if authorization == nil {
		return result
	}

	if authorization.UserID == primitive.NilObjectID {
		return result
	}

	// Owners are hard-coded to do everything, so no other roles need to be returned.
	if authorization.DomainOwner {
		return []string{MagicRoleOwner}
	}

	// If we know who you are, then you're "Authenticated"
	result = append(result, MagicRoleAuthenticated)

	// Users sometimes have special permissions over their own records.
	if authorization.UserID == folder.UserID {
		result = append(result, MagicRoleMyself)
	}

	// TODO: special roles for follower/following...

	return result
}
