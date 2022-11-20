package model

import (
	"time"

	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/benpate/data/journal"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/schema"
	"github.com/golang-jwt/jwt/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a person or machine account that can own pages and sections.
type User struct {
	UserID          primitive.ObjectID   `path:"userId"       json:"userId"        bson:"_id"`           // Unique identifier for this user.
	GroupIDs        []primitive.ObjectID `path:"groupIds"     json:"groupIds"      bson:"groupIds"`      // Slice of IDs for the groups that this user belongs to.
	DisplayName     string               `path:"displayName"  json:"displayName"   bson:"displayName"`   // Name to be displayed for this user
	Description     string               `path:"description"  json:"description"   bson:"description"`   // Status summary for this user (used by ActivityPub)
	EmailAddress    string               `path:"emailAddress" json:"emailAddress" bson:"emailAddress"`   // Email address for this user
	Username        string               `path:"username"     json:"username"      bson:"username"`      // This is the primary public identifier for the user.
	Password        string               `path:"password"     json:"password"      bson:"password"`      // This password should be encrypted with BCrypt.
	IsOwner         bool                 `path:"isOwner"      json:"isOwner"       bson:"isOwner"`       // If TRUE, then this user is a website owner with FULL privileges.
	ProfileURL      string               `path:"profileUrl"   json:"profileUrl"    bson:"profileUrl"`    // URL for the primary profile URL for this user.
	ImageURL        string               `path:"imageUrl"     json:"imageUrl"      bson:"imageUrl"`      // Avatar image of this user.
	PasswordReset   PasswordReset        `                    json:"passwordReset" bson:"passwordReset"` // Most recent password reset information.
	journal.Journal `json:"journal" bson:"journal"`
}

// NewUser returns a fully initialized User object.
func NewUser() User {
	return User{
		UserID:   primitive.NewObjectID(),
		GroupIDs: make([]primitive.ObjectID, 0),
	}
}

func UserSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"userId":        schema.String{Format: "objectId"},
			"groupIds":      schema.Array{Items: schema.String{Format: "objectId"}},
			"displayName":   schema.String{MaxLength: 50},
			"description":   schema.String{MaxLength: 100},
			"emailAddress":  schema.String{Format: "email"},
			"username":      schema.String{MaxLength: 50, Required: true},
			"password":      schema.String{MaxLength: 255, Required: true},
			"isOwner":       schema.Boolean{},
			"profileUrl":    schema.String{Format: "url"},
			"imageUrl":      schema.String{Format: "url"},
			"passwordReset": PasswordResetSchema(),
		},
	}
}

/*******************************************
 * data.Object Interface
 *******************************************/

// ID returns the primary key for this record
func (user *User) ID() string {
	return user.UserID.Hex()
}

func (user *User) GetObjectID(name string) (primitive.ObjectID, error) {
	return primitive.NilObjectID, derp.NewInternalError("model.User.GetObjectID", "Invalid property", name)
}

func (user *User) GetString(name string) (string, error) {
	return "", derp.NewInternalError("model.User.GetString", "Invalid property", name)
}

func (user *User) GetInt(name string) (int, error) {
	return 0, derp.NewInternalError("model.User.GetInt", "Invalid property", name)
}

func (user *User) GetInt64(name string) (int64, error) {
	return 0, derp.NewInternalError("model.User.GetInt64", "Invalid property", name)
}

func (user *User) GetBool(name string) (bool, error) {
	return false, derp.NewInternalError("model.User.GetBool", "Invalid property", name)
}

/*******************************************
 * Conversion Methods
 *******************************************/

func (user *User) PersonLink(relation string) PersonLink {
	return PersonLink{
		Relation:     relation,
		InternalID:   user.UserID,
		Name:         user.DisplayName,
		EmailAddress: user.Username,
		ProfileURL:   user.ProfileURL,
		ImageURL:     user.ImageURL,
		UpdateDate:   time.Now().Unix(),
	}
}

// Summary generates a lightweight summary of this user record.
func (user *User) Summary() UserSummary {
	return UserSummary{
		UserID:      user.UserID,
		DisplayName: user.DisplayName,
		Username:    user.Username,
		ImageURL:    user.ImageURL,
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

	case "groupIds":
		return id.SliceOfString(user.GroupIDs), nil

	case "displayName":
		return user.DisplayName, nil

	case "username":
		return user.Username, nil

	case "imageUrl":
		return user.ImageURL, nil

	}

	return nil, derp.NewBadRequestError("model.User.SetPath", "Invalid Path", path)
}

func (user *User) SetPath(path string, value any) error {

	switch path {

	case "groupIds":
		user.GroupIDs = id.SliceOfID(value)

	case "displayName":
		user.DisplayName = convert.String(value)

	case "username":
		user.Username = convert.String(value)

	case "imageUrl":
		user.ImageURL = convert.String(value)

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

/*******************************************
 * RoleStateEnumerator Interface
 *******************************************/

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

	// TODO: special roles for follower/following...

	return result
}

/*******************************************
 * URLs
 *******************************************/

func (user *User) ActivityPubProfileURL(host string) string {
	return host + "/@" + user.UserID.Hex()
}

func (user *User) ActivityPubURL(host string) string {
	return host + "/@" + user.UserID.Hex() + "/pub"
}

func (user *User) ActivityPubAvatarURL(host string) string {
	return host + user.ImageURL
}

func (user *User) ActivityPubInboxURL(host string) string {
	return host + "/@" + user.UserID.Hex() + "/pub/inbox"
}

func (user *User) ActivityPubOutboxURL(host string) string {
	return host + "/@" + user.UserID.Hex() + "/pub/outbox"
}

func (user *User) ActivityPubFollowingURL(host string) string {
	return host + "/@" + user.UserID.Hex() + "/pub/following"
}

func (user *User) ActivityPubFollowersURL(host string) string {
	return host + "/@" + user.UserID.Hex() + "/pub/followers"
}

func (user *User) ActivityPubLikedURL(host string) string {
	return host + "/@" + user.UserID.Hex() + "/pub/liked"
}

func (user *User) ActivityPubPublicKeyURL(host string) string {
	return host + "/@" + user.UserID.Hex() + "/pub/key"
}

func (user *User) ActivityPubSubscribeRequestURL(host string) string {
	return host + "/@" + user.UserID.Hex() + "/pub/authorize" // TODO: WTF is this?? "http://ostatus.org/schema/1.0/subscribe"
}
