package model

import (
	"time"

	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/benpate/data/journal"
	"github.com/benpate/domain"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/toot/object"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a person or machine account that can own pages and sections.
type User struct {
	UserID          primitive.ObjectID                 `json:"userId"          bson:"_id"`                  // Unique identifier for this user.
	MapIDs          mapof.String                       `json:"mapIds"          bson:"mapIds"`               // Map of IDs for this user on other web services.
	GroupIDs        sliceof.Object[primitive.ObjectID] `json:"groupIds"        bson:"groupIds"`             // Slice of IDs for the groups that this user belongs to.
	IconID          primitive.ObjectID                 `json:"iconId"          bson:"iconId"`               // AttachmentID of this user's avatar/icon image.
	ImageID         primitive.ObjectID                 `json:"imageId"         bson:"imageId"`              // AttachmentID of this user's banner image.
	DisplayName     string                             `json:"displayName"     bson:"displayName"`          // Name to be displayed for this user
	StatusMessage   string                             `json:"statusMessage"   bson:"statusMessage"`        // Status summary for this user
	Location        string                             `json:"location"        bson:"location"`             // Human-friendly description of this user's physical location.
	ProfileURL      string                             `json:"profileUrl"      bson:"profileUrl"`           // Fully Qualified profile URL for this user (including domain name)
	EmailAddress    string                             `json:"emailAddress"    bson:"emailAddress"`         // Email address for this user
	Username        string                             `json:"username"        bson:"username"`             // This is the primary public identifier for the user.
	Password        string                             `json:"-"               bson:"password"`             // This password should be encrypted with BCrypt.
	Locale          string                             `json:"locale"          bson:"locale"`               // Language code for this user's preferred language.
	SignupNote      string                             `json:"signupNote"      bson:"signupNote,omitempty"` // Note that was included when this user signed up.
	StateID         string                             `json:"stateId"         bson:"stateId"`              // State ID for this user
	InboxTemplate   string                             `json:"inboxTemplate"   bson:"inboxTemplate"`        // Template for the user's inbox
	OutboxTemplate  string                             `json:"outboxTemplate"  bson:"outboxTemplate"`       // Template for the user's outbox
	NoteTemplate    string                             `json:"noteTemplate"    bson:"noteTemplate"`         // Template for generically created notes
	Hashtags        sliceof.String                     `json:"hashtags"        bson:"hashtags"`             // Slice of tags that can be used to categorize this user.
	Links           sliceof.Object[PersonLink]         `json:"links"           bson:"links"`                // Slice of links to profiles on other web services.
	PasswordReset   PasswordReset                      `json:"-"               bson:"passwordReset"`        // Most recent password reset information.
	Data            mapof.String                       `json:"data"            bson:"data"`                 // Custom profile data that can be stored with this User.
	journal.Journal `json:"-" bson:",inline"`

	FollowerCount  int  `json:"followerCount"   bson:"followerCount"`  // Number of followers for this user
	FollowingCount int  `json:"followingCount"  bson:"followingCount"` // Number of actors that this user is following
	RuleCount      int  `json:"ruleCount"       bson:"ruleCount"`      // Number of rules (blocks) that this user has implemented
	IsOwner        bool `json:"isOwner"         bson:"isOwner"`        // If TRUE, then this user is a website owner with FULL privileges.
	IsPublic       bool `json:"isPublic"        bson:"isPublic"`       // If TRUE, then this user's profile is publicly visible
	IsIndexable    bool `json:"isIndexable"     bson:"isIndexable"`    // If TRUE, then this user's profile can be indexed by search engines.
}

// NewUser returns a fully initialized User object.
func NewUser() User {
	return User{
		UserID:   primitive.NewObjectID(),
		MapIDs:   mapof.NewString(),
		GroupIDs: sliceof.NewObject[primitive.ObjectID](),
		Links:    sliceof.NewObject[PersonLink](),
		Data:     mapof.NewString(),
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

func (user User) PersonLink() PersonLink {
	return PersonLink{
		UserID:       user.UserID,
		Name:         user.DisplayName,
		ProfileURL:   user.ProfileURL,
		InboxURL:     user.ActivityPubInboxURL(),
		EmailAddress: user.EmailAddress,
		IconURL:      user.ActivityPubIconURL(),
	}
}

// Summary generates a lightweight summary of this user record.
func (user User) Summary() UserSummary {
	return UserSummary{
		UserID:       user.UserID,
		DisplayName:  user.DisplayName,
		Username:     user.Username,
		EmailAddress: user.EmailAddress,
		IconID:       user.IconID,
		ProfileURL:   user.ProfileURL,
	}
}

/******************************************
 * Group Interface
 ******************************************/

func (user *User) IsGroupMember(groupIDs ...primitive.ObjectID) bool {

	for _, groupID := range groupIDs {
		for _, existingID := range user.GroupIDs {
			if existingID == groupID {
				return true
			}
		}
	}
	return false
}

// AddGroup adds a new group to this user's list of groups, avoiding duplicates
func (user *User) AddGroup(groupID primitive.ObjectID) {

	for _, existingID := range user.GroupIDs {
		if existingID == groupID {
			return
		}
	}

	user.GroupIDs = append(user.GroupIDs, groupID)
}

// RemoveGroup removes a group from this user's list of groups
func (user *User) RemoveGroup(groupID primitive.ObjectID) {

	for index, existingID := range user.GroupIDs {
		if existingID == groupID {
			user.GroupIDs = append(user.GroupIDs[:index], user.GroupIDs[index+1:]...)
			return
		}
	}
}

/******************************************
 * Steranko Interfaces
 ******************************************/

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

/******************************************
 * StateSetter Methods
 ******************************************/

func (user *User) SetState(stateID string) {
	user.StateID = stateID
}

/******************************************
 * AccessLister Interface
 ******************************************/

// State returns the current state of this User.
// It is part of the AccessLister interface
func (user *User) State() string {
	// return user.StateID
	return "default" // This is a hack to maybe make this work.
}

// IsAuthor returns TRUE if the provided UserID the author of this User
// It is part of the AccessLister interface
func (user *User) IsAuthor(authorID primitive.ObjectID) bool {
	return false
}

// IsMyself returns TRUE if this object directly represents the provided UserID
// It is part of the AccessLister interface
func (user *User) IsMyself(userID primitive.ObjectID) bool {
	return userID == user.UserID
}

// RolesToGroupIDs returns a slice of Group IDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (user *User) RolesToGroupIDs(roleIDs ...string) id.Slice {
	return id.NewSlice()
}

// RolesToPrivileges returns a slice of Privileges that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (user *User) RolesToPrivileges(roleIDs ...string) sliceof.String {
	return sliceof.NewString()
}

/******************************************
 * ActivityPub Interfaces
 ******************************************/

func (user User) GetJSONLD() mapof.Any {

	contextList := sliceof.Any{
		vocab.ContextTypeActivityStreams,
		vocab.ContextTypeSecurity,
		vocab.ContextTypeToot,
	}

	result := mapof.Any{
		vocab.AtContext:                 contextList,
		vocab.PropertyID:                user.ActivityPubURL(),
		vocab.PropertyType:              vocab.ActorTypePerson,
		vocab.PropertyURL:               user.Host() + "/@" + user.Username,
		vocab.PropertyName:              user.DisplayName,
		vocab.PropertyPreferredUsername: user.Username,
		vocab.PropertyTootDiscoverable:  true,
		vocab.PropertyTootIndexable:     user.IsIndexable,
		vocab.PropertyInbox:             user.ActivityPubInboxURL(),
		vocab.PropertyOutbox:            user.ActivityPubOutboxURL(),
		vocab.PropertyFollowing:         user.ActivityPubFollowingURL(),
		vocab.PropertyFollowers:         user.ActivityPubFollowersURL(),
		vocab.PropertyLiked:             user.ActivityPubLikedURL(),
		vocab.PropertyFeatured:          user.ActivityPubFeaturedURL(),
	}

	if user.StatusMessage != "" {
		result[vocab.PropertySummary] = user.StatusMessage
	}

	if iconURL := user.ActivityPubIconURL(); iconURL != "" {
		result[vocab.PropertyIcon] = mapof.Any{
			vocab.PropertyType:      vocab.ObjectTypeImage,
			vocab.PropertyMediaType: "image/webp",
			vocab.PropertyURL:       user.ActivityPubIconURL(),
		}
	}

	if imageURL := user.ActivityPubImageURL(); imageURL != "" {
		result[vocab.PropertyImage] = mapof.Any{
			vocab.PropertyType:      vocab.ObjectTypeImage,
			vocab.PropertyMediaType: "image/webp",
			vocab.PropertyURL:       user.ActivityPubImageURL(),
		}
	}

	return result
}

func (user *User) ActivityPubURL() string {
	return user.ProfileURL
}

func (user *User) ActivityPubIconURL() string {

	if user.IconID.IsZero() {
		return ""
	}
	return user.ProfileURL + "/attachments/" + user.IconID.Hex()
}

func (user *User) ActivityPubImageURL() string {

	if user.ImageID.IsZero() {
		return ""
	}
	return user.ProfileURL + "/attachments/" + user.ImageID.Hex()
}

func (user *User) ActivityPubBlockedURL() string {
	if user.ProfileURL == "" {
		return ""
	}

	return user.ProfileURL + "/pub/blocked"
}

func (user *User) ActivityPubInboxURL() string {
	if user.ProfileURL == "" {
		return ""
	}

	return user.ProfileURL + "/pub/inbox"
}

func (user *User) ActivityPubFollowersURL() string {
	if user.ProfileURL == "" {
		return ""
	}

	return user.ProfileURL + "/pub/followers"
}

func (user *User) ActivityPubFollowingURL() string {
	if user.ProfileURL == "" {
		return ""
	}

	return user.ProfileURL + "/pub/following"
}

func (user *User) ActivityPubLikedURL() string {
	if user.ProfileURL == "" {
		return ""
	}

	return user.ProfileURL + "/pub/liked"
}

func (user *User) ActivityPubFeaturedURL() string {
	if user.ProfileURL == "" {
		return ""
	}

	return user.ProfileURL + "/pub/featured"
}

func (user *User) ActivityPubOutboxURL() string {
	if user.ProfileURL == "" {
		return ""
	}

	return user.ProfileURL + "/pub/outbox"
}

func (user *User) ActivityPubPublicKeyURL() string {
	if user.ProfileURL == "" {
		return ""
	}

	return user.ProfileURL + "#main-key"
}

func (user *User) JSONFeedURL() string {
	if user.ProfileURL == "" {
		return ""
	}

	return user.ProfileURL + "/feed?type=json"
}

/******************************************
 * Mastodon API
 ******************************************/

func (user User) Toot() object.Account {
	return object.Account{
		ID:       user.ActivityPubURL(),
		Username: user.Username,
		// Acct: user.WebFingerAccount,
		DisplayName:  user.DisplayName,
		Note:         user.StatusMessage,
		Avatar:       user.ActivityPubIconURL(),
		Header:       user.ActivityPubImageURL(),
		Discoverable: user.IsPublic,
		CreatedAt:    time.Unix(user.CreateDate, 0).Format(time.RFC3339),
	}
}

func (user User) GetRank() int64 {
	return user.CreateDate
}

/******************************************
 * Webhook Interface
 ******************************************/

// GetWebhookData returns the data for this
// User that will be sent to a webhook
func (user User) GetWebhookData() mapof.Any {
	return mapof.Any{
		"userId":     user.UserID.Hex(),
		"name":       user.DisplayName,
		"email":      user.EmailAddress,
		"username":   user.Username,
		"url":        user.ProfileURL,
		"iconUrl":    user.ActivityPubIconURL(),
		"imageUrl":   user.ActivityPubImageURL(),
		"createDate": user.CreateDate,
		"updateDate": user.UpdateDate,
		"deleteDate": user.DeleteDate,
	}
}

/******************************************
 * Activity Intent Data
 ******************************************/
func (user User) ActivityIntentProfile() mapof.Any {

	return mapof.Any{
		vocab.PropertyID:                user.ActivityPubURL(),
		vocab.PropertyName:              user.DisplayName,
		vocab.PropertyIcon:              user.ActivityPubIconURL(),
		vocab.PropertyURL:               user.ActivityPubURL(),
		vocab.PropertyPreferredUsername: "@" + user.Username + "@" + user.Hostname(),
	}
}

func (user User) Host() string {

	hostname := user.Hostname()

	return domain.Protocol(hostname) + hostname
}

func (user User) Hostname() string {

	return domain.NameOnly(user.ProfileURL)
}
