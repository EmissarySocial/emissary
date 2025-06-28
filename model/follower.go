package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Follower struct {
	FollowerID primitive.ObjectID `json:"followerId" bson:"_id"`        // Unique identifier for this Follower
	ParentType string             `json:"type"       bson:"type"`       // Type of record being followed (e.g. "User", "Stream", or "Search")
	ParentID   primitive.ObjectID `json:"parentId"   bson:"parentId"`   // Unique identifier for the Stream that is being followed (including user's outboxes)
	StateID    string             `json:"stateId"    bson:"stateId"`    // Unique identifier for the State of this Follower ("ACTIVE", "PENDING")
	Method     string             `json:"method"     bson:"method"`     // Method of follower (e.g. "POLL", "WEBSUB", "RSS-CLOUD", "ACTIVITYPUB", "EMAIL")
	Format     string             `json:"format"     bson:"format"`     // Format of the data being followed (e.g. "ATOM", "HTML", "JSON", "RSS", "XML")
	Actor      PersonLink         `json:"actor"      bson:"actor"`      // Person who is follower the User
	Data       mapof.Any          `json:"data"       bson:"data"`       // Additional data about this Follower that depends on the follow method
	ExpireDate int64              `json:"expireDate" bson:"expireDate"` // Unix timestamp (in seconds) when this follower will be automatically purged.

	journal.Journal `json:"-" bson:",inline"`
}

func NewFollower() Follower {
	return Follower{
		FollowerID: primitive.NewObjectID(),
		Data:       make(mapof.Any),
		StateID:    FollowerStatePending,
	}
}

/******************************************
 * data.Object Interface
 ******************************************/

func (follower *Follower) ID() string {
	return follower.FollowerID.Hex()
}

/******************************************
 * AccessLister Interface
 ******************************************/

// State returns the current state of this Follower.
// It is part of the AccessLister interface
func (follower *Follower) State() string {
	return "default"
}

// IsAuthor returns TRUE if the provided UserID the author of this Follower
// It is part of the AccessLister interface
func (follower *Follower) IsAuthor(authorID primitive.ObjectID) bool {
	return false
}

// IsMyself returns TRUE if this object directly represents the provided UserID
// It is part of the AccessLister interface
func (follower *Follower) IsMyself(userID primitive.ObjectID) bool {
	if follower.ParentType == FollowerTypeUser {
		return follower.ParentID == userID
	}

	return false
}

// RolesToGroupIDs returns a slice of Group IDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (follower *Follower) RolesToGroupIDs(roleIDs ...string) Permissions {
	return NewPermissions()
}

// RolesToPrivilegeIDs returns a slice of Privileges that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (follower *Follower) RolesToPrivilegeIDs(roleIDs ...string) Permissions {
	return NewPermissions()
}

/******************************************
 * Other Calculations
 ******************************************/

// ParentURL returns the URL of the parent object that this Follower is following.
func (follower Follower) ParentURL(host string) string {

	if follower.ParentType == FollowerTypeUser {
		return host + "/@" + follower.ParentID.Hex()
	}

	return host + "/" + follower.ParentID.Hex()
}

// UnsubscribeLink returns a URL where an Email Follower can unsubscribe.
// It returns an empty string for all other follower types (ActivityPub, WebSub, etc.)
func (follower Follower) UnsubscribeLink(host string) string {

	if follower.Method == FollowerMethodEmail {
		return follower.ParentURL(host) + "/follower-unsubscribe?followerId=" + follower.FollowerID.Hex() + "&secret=" + follower.Data.GetString("secret")
	}

	return ""
}

/******************************************
 * ActivityPub Methods
 ******************************************/

func (follower Follower) GetJSONLD() mapof.Any {

	return mapof.Any{
		vocab.PropertyID:   follower.Actor.ProfileURL,
		vocab.PropertyName: follower.Actor.Name,
	}
}
