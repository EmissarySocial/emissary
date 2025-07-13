package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/form"
	"github.com/benpate/toot/object"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Folder represents a custom folder that organizes incoming messages
type Folder struct {
	FolderID    primitive.ObjectID `json:"folderId"    bson:"_id"`         // Unique ID for this folder
	UserID      primitive.ObjectID `json:"userId"      bson:"userId"`      // ID of the User who owns this folder
	Label       string             `json:"label"       bson:"label"`       // Label of the folder
	Icon        string             `json:"icon"        bson:"icon"`        // Icon of the folder
	Layout      string             `json:"layout"      bson:"layout"`      // Layout type of the folder
	Group       int                `json:"group"       bson:"group"`       // Group number of the folder (starting with 1)
	Rank        int                `json:"rank"        bson:"rank"`        // Sort order of the folder
	UnreadCount int                `json:"unreadCount" bson:"unreadCount"` // Number of unread messages in this folder

	journal.Journal `json:"-" bson:",inline"`
}

// NewFolder returns a fully initialized Folder object
func NewFolder() Folder {
	return Folder{
		FolderID: primitive.NewObjectID(),
		Icon:     "folder",
	}
}

/******************************************
 * data.Object Interface
 ******************************************/

func (folder Folder) ID() string {
	return folder.FolderID.Hex()
}

/******************************************
 * Other Data Accessors
 ******************************************/

func (folder Folder) LookupCode() form.LookupCode {
	return form.LookupCode{
		Value: folder.FolderID.Hex(),
		Label: folder.Label,
	}
}

/******************************************
 * AccessLister Interface
 ******************************************/

// State returns the current state of this Folder.
// It is part of the AccessLister interface
func (folder *Folder) State() string {
	return "default"
}

// IsAuthor returns TRUE if the provided UserID the author of this Folder
// It is part of the AccessLister interface
func (folder *Folder) IsAuthor(authorID primitive.ObjectID) bool {
	return false
}

// IsMyself returns TRUE if this object directly represents the provided UserID
// It is part of the AccessLister interface
func (folder *Folder) IsMyself(userID primitive.ObjectID) bool {
	return !userID.IsZero() && userID == folder.UserID
}

// RolesToGroupIDs returns a slice of Group IDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (folder *Folder) RolesToGroupIDs(roleIDs ...string) Permissions {
	return defaultRolesToGroupIDs(folder.UserID, roleIDs...)
}

// RolesToGroupIDs returns a slice of Group IDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (folder *Folder) RolesToPrivilegeIDs(roleIDs ...string) Permissions {
	return NewPermissions()
}

/******************************************
 * Mastodon API
 ******************************************/

func (folder Folder) Toot() object.List {
	return object.List{
		ID:            folder.FolderID.Hex(),
		Title:         folder.Label,
		RepliesPolicy: object.ListRepliesPolicyFollowed,
	}
}

func (folder Folder) GetRank() int64 {
	return int64(folder.Rank)
}
