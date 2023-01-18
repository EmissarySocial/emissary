package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/maps"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const FollowerTypeStream = "Stream"

const FollowerTypeUser = "User"

type Follower struct {
	FollowerID primitive.ObjectID `json:"followerId" bson:"_id"`        // Unique identifier for this Follower
	ParentID   primitive.ObjectID `json:"parentId"   bson:"parentId"`   // Unique identifier for the Stream that is being followed (including user's outboxes)
	Type       string             `json:"type"       bson:"type"`       // Type of record being followed (e.g. "User", "Stream")
	Method     string             `json:"method"     bson:"method"`     // Method of follower (e.g. "POLL", "WEBSUB", "RSS-CLOUD", "ACTIVITYPUB")
	Format     string             `json:"format"     bson:"format"`     // Format of the data being followed (e.g. "JSON", "XML", "ATOM", "RSS")
	Actor      PersonLink         `json:"actor"      bson:"actor"`      // Person who is follower the User
	Data       maps.Map           `json:"data"       bson:"data"`       // Additional data about this Follower that depends on the follow method
	ExpireDate int64              `json:"expireDate" bson:"expireDate"` // Unix timestamp (in seconds) when this follower will be automatically purged.

	journal.Journal `json:"journal" bson:"journal"`
}

func NewFollower() Follower {
	return Follower{
		FollowerID: primitive.NewObjectID(),
		Data:       make(maps.Map),
	}
}

func FollowerSchema() schema.Element {

	return schema.Object{
		Properties: schema.ElementMap{
			"followerId": schema.String{Format: "objectId"},
			"parentId":   schema.String{Format: "objectId"},
			"type":       schema.String{Enum: []string{FollowerTypeStream, FollowerTypeUser}},
			"method":     schema.String{Enum: []string{FollowMethodPoll, FollowMethodWebSub, FollowMethodActivityPub}},
			"format":     schema.String{Enum: []string{MimeTypeActivityPub, MimeTypeAtom, MimeTypeHTML, MimeTypeJSONFeed, MimeTypeRSS, MimeTypeXML}},
			"actor":      PersonLinkSchema(),
			"data":       schema.Object{Wildcard: schema.String{MaxLength: 256}},
			"expireDate": schema.Integer{BitSize: 64},
		},
	}
}

/*******************************************
 * data.Object Interface
 *******************************************/

func (follower *Follower) ID() string {
	return follower.FollowerID.Hex()
}

/*******************************************
 * RoleStateEnumerator Interface
 *******************************************/

// State returns the current state of this object.
// For users, there is no state, so it returns ""
func (follower Follower) State() string {
	return ""
}

// Roles returns a list of all roles that match the provided authorization.
// Since Follower records should only be accessible by the follower owner, this
// function only returns MagicRoleMyself if applicable.  Others (like Anonymous
// and Authenticated) should never be allowed on an Follower record, so they
// are not returned.
func (follower Follower) Roles(authorization *Authorization) []string {

	// Folders are private, so only MagicRoleMyself is allowed
	if authorization.UserID == follower.ParentID {
		return []string{MagicRoleMyself}
	}

	// Intentionally NOT allowing MagicRoleAnonymous, MagicRoleAuthenticated, or MagicRoleOwner
	return []string{}
}
