package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MagicFolderIDOutbox identifies the unique id for the "Outbox" folder
const MagicFolderIDOutbox = "000000000000000000000000"

// MagicFolderIDInbox identifies the unique id for the "Inbox" folder
const MagicFolderIDInbox = "000000000000000000000001"

// Folder represents a custom folder that organizes incoming messages
type Folder struct {
	FolderID primitive.ObjectID `path:"folderId" json:"folderId" bson:"_id"`    // Unique ID for this folder
	UserID   primitive.ObjectID `path:"userId"   json:"userId"   bson:"userId"` // ID of the User who owns this folder
	Label    string             `path:"label"    json:"label"    bson:"label"`  // Label of the folder
	Rank     int                `path:"rank"     json:"rank"     bson:"rank"`   // Sort order of the folder

	journal.Journal `json:"-" bson:"journal"`
}

// NewFolder returns a fully initialized Folder object
func NewFolder() Folder {
	return Folder{
		FolderID: primitive.NewObjectID(),
	}
}

// FolderSchema returns a Rosetta Schema for the Folder object
func FolderSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"folderId": schema.String{Format: "objectId"},
			"userId":   schema.String{Format: "objectId"},
			"label":    schema.String{MaxLength: 100},
			"rank":     schema.Integer{},
		},
	}
}

/*******************************************
 * data.Object Interface
 *******************************************/

func (folder Folder) ID() string {
	return folder.FolderID.Hex()
}

func (folder *Folder) GetObjectID(name string) (primitive.ObjectID, error) {
	switch name {
	case "folderId":
		return folder.FolderID, nil
	case "userId":
		return folder.UserID, nil
	}
	return primitive.NilObjectID, derp.NewInternalError("model.Folder.GetObjectID", "Invalid property", name)
}

func (folder *Folder) GetString(name string) (string, error) {
	switch name {
	case "label":
		return folder.Label, nil
	}
	return "", derp.NewInternalError("model.Folder.GetString", "Invalid property", name)
}

func (folder *Folder) GetInt(name string) (int, error) {
	switch name {
	case "rank":
		return folder.Rank, nil
	}
	return 0, derp.NewInternalError("model.Folder.GetInt", "Invalid property", name)
}

func (folder *Folder) GetInt64(name string) (int64, error) {
	return 0, derp.NewInternalError("model.Folder.GetInt64", "Invalid property", name)
}

func (folder *Folder) GetBool(name string) (bool, error) {
	return false, derp.NewInternalError("model.Folder.GetBool", "Invalid property", name)
}

/*******************************************
 * Other Data Accessors
 *******************************************/

func (folder *Folder) LookupCode() form.LookupCode {
	return form.LookupCode{
		Value: folder.FolderID.Hex(),
		Label: folder.Label,
	}
}

/*******************************************
 * RoleStateEnumerator Interface
 *******************************************/

// State returns the current state of this object.
// For users, there is no state, so it returns ""
func (folder Folder) State() string {
	return ""
}

// Roles returns a list of all roles that match the provided authorization.
// Since Folders should only be accessible by the folder owner, this function only
// returns MagicRoleMyself if applicable.  Others (like Anonymous and Authenticated)
// should never be allowed on an inbox folder, so they are not returned.
func (folder Folder) Roles(authorization *Authorization) []string {

	// Folders are private, so only MagicRoleMyself is allowed
	if authorization.UserID == folder.UserID {
		return []string{MagicRoleMyself}
	}

	// Intentionally NOT allowing MagicRoleAnonymous, MagicRoleAuthenticated, or MagicRoleOwner
	return []string{}
}
