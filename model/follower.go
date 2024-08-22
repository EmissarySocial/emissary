package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Follower struct {
	FollowerID primitive.ObjectID `json:"followerId" bson:"_id"`        // Unique identifier for this Follower
	ParentType string             `json:"type"       bson:"type"`       // Type of record being followed (e.g. "User", "Stream")
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
 * RoleStateEnumerator Interface
 ******************************************/

// State returns the current state of this object.
// For users, there is no state, so it returns ""
func (follower Follower) State() string {
	return ""
}

// Roles returns a list of all roles that match the provided authorization.
// Since Follower records should only be accessible by the follower owner, this
// function only returns MagicRoleMyself if applicable.
func (follower Follower) Roles(authorization *Authorization) []string {

	// Everyone matches "Anonymous"
	result := []string{MagicRoleAnonymous}

	// If the user is authenticated, then they match "Authenticated"
	if authorization.IsAuthenticated() {
		result = append(result, MagicRoleAuthenticated)
	}

	// If the user is the owner of this Follower, then they match "Myself"
	if authorization.UserID == follower.ParentID {
		result = append(result, MagicRoleMyself)
	}

	// Success?
	return result
}

func (follower Follower) GetJSONLD() mapof.Any {

	return mapof.Any{
		vocab.PropertyID:   follower.Actor.ProfileURL,
		vocab.PropertyName: follower.Actor.Name,
	}
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
