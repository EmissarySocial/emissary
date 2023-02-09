package model

import (
	"time"

	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/benpate/data/journal"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/golang-jwt/jwt/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a person or machine account that can own pages and sections.
type User struct {
	UserID         primitive.ObjectID         `json:"userId"          bson:"_id"`            // Unique identifier for this user.
	GroupIDs       id.Slice                   `json:"groupIds"        bson:"groupIds"`       // Slice of IDs for the groups that this user belongs to.
	ImageID        primitive.ObjectID         `json:"imageId"         bson:"imageId"`        // AttachmentID of this user's avatar image.
	DisplayName    string                     `json:"displayName"     bson:"displayName"`    // Name to be displayed for this user
	StatusMessage  string                     `json:"statusMessage"   bson:"statusMessage"`  // Status summary for this user
	Location       string                     `json:"location"        bson:"location"`       // Human-friendly description of this user's physical location.
	Links          sliceof.Object[PersonLink] `json:"links"           bson:"links"`          // Slice of links to profiles on other web services.
	ProfileURL     string                     `json:"profileUrl"      bson:"profileUrl"`     // Fully Qualified profile URL for this user (including domain name)
	EmailAddress   string                     `json:"emailAddress"    bson:"emailAddress"`   // Email address for this user
	Username       string                     `json:"username"        bson:"username"`       // This is the primary public identifier for the user.
	Password       string                     `json:"-"               bson:"password"`       // This password should be encrypted with BCrypt.
	FollowerCount  int                        `json:"followerCount"   bson:"followerCount"`  // Number of followers for this user
	FollowingCount int                        `json:"followingCount"  bson:"followingCount"` // Number of users that this user is following
	BlockCount     int                        `json:"blockCount"      bson:"blockCount"`     // Number of users that this user is following
	IsOwner        bool                       `json:"isOwner"         bson:"isOwner"`        // If TRUE, then this user is a website owner with FULL privileges.
	PasswordReset  PasswordReset              `                       bson:"passwordReset"`  // Most recent password reset information.

	journal.Journal `json:"journal" bson:"journal"`
}

// NewUser returns a fully initialized User object.
func NewUser() User {
	return User{
		UserID:   primitive.NewObjectID(),
		GroupIDs: make([]primitive.ObjectID, 0),
		Links:    make([]PersonLink, 0),
	}
}

/******************************************
 * data.Object Interface
 ******************************************/

// ID returns the primary key for this record
func (user *User) ID() string {
	return user.UserID.Hex()
}

/******************************************
 * Conversion Methods
 ******************************************/

func (user *User) PersonLink() PersonLink {
	return PersonLink{
		InternalID:   user.UserID,
		Name:         user.DisplayName,
		EmailAddress: user.Username,
		ProfileURL:   user.ProfileURL,
		ImageURL:     user.ActivityPubAvatarURL(),
	}
}

// Summary generates a lightweight summary of this user record.
func (user *User) Summary() UserSummary {
	return UserSummary{
		UserID:      user.UserID,
		DisplayName: user.DisplayName,
		Username:    user.Username,
		ImageURL:    user.ActivityPubAvatarURL(),
		ProfileURL:  user.ProfileURL,
	}
}

// Copy returns a duplicate copy of this User
// NOTE: This must NOT be a pointer receiver, so that a true COPY
// of this record is returned.
func (user User) Copy() User {
	return user
}

func (user *User) GetPath(path string) (any, error) {

	switch path {

	case "imageId":
		return user.ImageID, nil

	case "groupIds":
		return id.SliceOfString(user.GroupIDs), nil

	case "displayName":
		return user.DisplayName, nil

	case "username":
		return user.Username, nil

	}

	return nil, derp.NewBadRequestError("model.User.SetPath", "Invalid Path", path)
}

func (user *User) SetPath(path string, value any) error {

	switch path {

	case "imageId":
		user.ImageID, _ = id.Convert(value)

	case "groupIds":
		user.GroupIDs = id.SliceOfID(value)

	case "displayName":
		user.DisplayName = convert.String(value)

	case "username":
		user.Username = convert.String(value)

	default:
		return derp.NewBadRequestError("model.User.SetPath", "Invalid Path", path, value)
	}

	return nil
}

/******************************
 Steranko Interfaces
*******************************/

// GetUsername returns the username for this User.  A part of the "steranko.User" interface.
func (user *User) GetUsername() string {
	return user.Username
}

// GetPassword returns the (encrypted) passsword for this User.  A part of the "steranko.User" interface.
func (user *User) GetPassword() string {
	return user.Password
}

// SetUsername updates the username for this User.  A part of the "steranko.User" interface.
func (user *User) SetUsername(username string) {
	user.Username = username
}

// SetPassword updates the password for this User.  A part of the "steranko.User" interface.
func (user *User) SetPassword(password string) {
	user.Password = password
}

// Claims returns all access privileges given to this user.  A part of the "steranko.User" interface.
func (user *User) Claims() jwt.Claims {

	result := Authorization{
		UserID:      user.UserID,
		GroupIDs:    user.GroupIDs,
		DomainOwner: user.IsOwner,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),                   // Current create date.  (Used by Steranko to refresh tokens)
			ExpiresAt: jwt.NewNumericDate(time.Now().AddDate(10, 0, 0)), // Expires ten years from now (but re-validated sooner by Steranko)
		},
	}

	return result
}

/******************************************
 * RoleStateEnumerator Interface
 ******************************************/

// State returns the current state of this object.
// For users, there is no state, so it returns ""
func (user *User) State() string {
	return ""
}

// Roles returns a list of all roles that match the provided authorization
func (user *User) Roles(authorization *Authorization) []string {

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
	if authorization.UserID == user.UserID {
		result = append(result, MagicRoleMyself)
	}

	// TODO: LOW: Add special roles for follower/following?

	return result
}

/******************************************
 * ActivityPub Interfaces
 ******************************************/

func (user User) GetJSONLD() mapof.Any {

	return mapof.Any{
		"@context":          sliceof.String{"https://www.w3.org/ns/activitystreams", "https://w3id.org/security/v1"},
		"id":                user.ActivityPubURL(),
		"type":              "Person",
		"url":               user.ProfileURL,
		"name":              user.DisplayName,
		"preferredUsername": user.Username,
		"summary":           user.StatusMessage,
		"icon":              user.ActivityPubAvatarURL(),
		"inbox":             user.ActivityPubInboxURL(),
		"outbox":            user.ActivityPubOutboxURL(),
		"following":         user.ActivityPubFollowingURL(),
		"followers":         user.ActivityPubFollowersURL(),
		"liked":             user.ActivityPubLikedURL(),
		"blocked":           user.ActivityPubBlockedURL(),
		"publicKey":         user.ActivityPubPublicKeyURL(),
	}
}

func (user *User) ActivityPubProfileURL() string {
	return user.ProfileURL
}

func (user *User) ActivityPubURL() string {
	return user.ProfileURL
}

func (user *User) ActivityPubAvatarURL() string {
	if user.ImageID.IsZero() {
		return ""
	}
	return user.ProfileURL + "/avatar"
}

func (user *User) ActivityPubBlockedURL() string {
	return user.ProfileURL + "/pub/blocked"
}

func (user *User) ActivityPubInboxURL() string {
	return user.ProfileURL + "/pub/inbox"
}

func (user *User) ActivityPubOutboxURL() string {
	return user.ProfileURL + "/pub/outbox"
}

func (user *User) ActivityPubOutbox_NewItemURL() string {
	return user.ProfileURL + "/pub/outbox/" + primitive.NewObjectID().Hex()
}

func (user *User) ActivityPubFollowersURL() string {
	return user.ProfileURL + "/pub/followers"
}

func (user *User) ActivityPubFollowingURL() string {
	return user.ProfileURL + "/pub/following"
}

func (user *User) ActivityPubLikedURL() string {
	return user.ProfileURL + "/pub/liked"
}

func (user *User) ActivityPubPublicKeyURL() string {
	return user.ProfileURL + "/pub/key"
}

func (user *User) JSONFeedURL() string {
	return user.ProfileURL + "/feed?type=json"
}
