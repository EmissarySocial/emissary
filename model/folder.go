package model

import (
	"strconv"

	"github.com/benpate/data/journal"
	"github.com/benpate/form"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Folder represents a custom folder that organizes incoming messages
type Folder struct {
	FolderID primitive.ObjectID `json:"folderId" bson:"_id"`      // Unique ID for this folder
	UserID   primitive.ObjectID `json:"userId"   bson:"userId"`   // ID of the User who owns this folder
	Label    string             `json:"label"    bson:"label"`    // Label of the folder
	Icon     string             `json:"icon"     bson:"icon"`     // Icon of the folder
	Layout   string             `json:"layout"   bson:"layout"`   // Layout type of the folder
	Group    int                `json:"group"    bson:"group"`    // Group number of the folder (starting with 1)
	Rank     int                `json:"rank"     bson:"rank"`     // Sort order of the folder
	ReadDate int64              `json:"readDate" bson:"readDate"` // PublishDate of the most recently read message in this folder

	journal.Journal `json:"-" bson:"journal"`
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

// ReadDateSeonds returns the ReadDate as a number of seconds since the Unix Epoch.
// Since this value is stored as milliseconds, we divide by 1000 to get the seconds.
func (folder Folder) ReadDateSeconds() int64 {
	return folder.ReadDate / 1000
}

// ReadDateString returns the ReadDate as a string
func (folder Folder) ReadDateString() string {
	return strconv.FormatInt(folder.ReadDate, 10)
}

func (folder Folder) LookupCode() form.LookupCode {
	return form.LookupCode{
		Value: folder.FolderID.Hex(),
		Label: folder.Label,
	}
}

/******************************************
 * RoleStateEnumerator Interface
 ******************************************/

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
