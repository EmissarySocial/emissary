package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Folder represents a custom folder that organizes incoming messages
type Folder struct {
	FolderID primitive.ObjectID `json:"folderId" bson:"_id"`    // Unique ID for this folder
	UserID   primitive.ObjectID `json:"userId"   bson:"userId"` // ID of the User who owns this folder
	Label    string             `json:"label"    bson:"label"`  // Label of the folder
	Rank     int                `json:"rank"     bson:"rank"`   // Sort order of the folder

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
