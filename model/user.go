package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/EmissarySocial/emissary/tools/replace"
	"github.com/benpate/data/journal"
	"github.com/benpate/delta"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/toot/object"
	"github.com/benpate/uri"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a person or machine account that can own pages and sections.
type User struct {
	UserID               primitive.ObjectID         `bson:"_id"`                  // Unique identifier for this user.
	MapIDs               mapof.String               `bson:"mapIds"`               // Map of IDs for this user on other web services.
	GroupIDs             id.Slice                   `bson:"groupIds"`             // Slice of IDs for the groups that this user belongs to.
	IconID               primitive.ObjectID         `bson:"iconId"`               // AttachmentID of this user's avatar/icon image.
	ImageID              primitive.ObjectID         `bson:"imageId"`              // AttachmentID of this user's banner image.
	DisplayName          string                     `bson:"displayName"`          // Name to be displayed for this user
	StatusMessage        string                     `bson:"statusMessage"`        // Status summary for this user
	Location             string                     `bson:"location"`             // Human-friendly description of this user's physical location.
	ProfileURL           string                     `bson:"profileUrl"`           // Fully Qualified profile URL for this user (including domain name)
	EmailAddress         string                     `bson:"emailAddress"`         // Email address for this user
	Username             string                     `bson:"username"`             // This is the primary public identifier for the user.
	Password             string                     `bson:"password"`             // Hashed password. Only ever written via a PasswordHasher (see steranko.SetPassword); never contains plaintext.
	Locale               string                     `bson:"locale"`               // Language code for this user's preferred language.
	SignupNote           string                     `bson:"signupNote,omitempty"` // Note that was included when this user signed up.
	StateID              string                     `bson:"stateId"`              // State ID for this user
	InboxTemplate        string                     `bson:"inboxTemplate"`        // Template for the user's inbox
	OutboxTemplate       string                     `bson:"outboxTemplate"`       // Template for the user's outbox
	NoteTemplate         string                     `bson:"noteTemplate"`         // Template for generically created notes
	Hashtags             sliceof.String             `bson:"hashtags"`             // Slice of tags that can be used to categorize this user.
	TagURL               string                     `bson:"tagUrl"`               // URL prefix for hashtag links, denormalized from the outbox Template ("%23" + tag is appended).
	Links                sliceof.Object[PersonLink] `bson:"links"`                // Slice of links to profiles on other web services.
	NotificationChannels sliceof.String             `bson:"notificationChannels"` // Slice of ENABLED notification channel keys (see model.NotificationChannel* constants). Empty = all notifications off.
	PasswordReset        PasswordReset              `bson:"passwordReset"`        // Most recent password reset information.
	Data                 mapof.String               `bson:"data"`                 // Custom profile data that can be stored with this User.
	ProfileFingerprint   string                     `bson:"profileFingerprint"`   // Hash of the last-saved actor document (GetJSONLD). User.Save compares it to detect profile changes that must federate as an ActivityPub Update.
	MovedTo              string                     `bson:"movedTo,omitempty"`    // If present, this user has been moved to a new URL, and cannot sign in to this profile anymore.
	FollowerCount        int                        `bson:"followerCount"`        // Number of followers for this user
	FollowingCount       int                        `bson:"followingCount"`       // Number of actors that this user is following
	RuleCount            int                        `bson:"ruleCount"`            // Number of rules (blocks) that this user has implemented
	IsOwner              bool                       `bson:"isOwner"`              // If TRUE, then this user is a website owner with FULL privileges.
	IsPublic             bool                       `bson:"isPublic"`             // If TRUE, then this user's profile is publicly visible
	IsBridgeBluesky      delta.Bool                 `bson:"isBridgeBluesky"`      // If TRUE, then allow this user to be bridged to Bluesky
	IsIndexable          bool                       `bson:"isIndexable"`          // If TRUE, then this user's profile can be indexed by search engines.

	journal.Journal `json:"-" bson:",inline"`
}

// NewUser returns a fully initialized User object.
func NewUser() User {
	return User{
		UserID:               primitive.NewObjectID(),
		MapIDs:               mapof.NewString(),
		GroupIDs:             id.NewSlice(),
		Links:                sliceof.NewObject[PersonLink](),
		Data:                 mapof.NewString(),
		NotificationChannels: DefaultNotificationChannels(),
	}
}

/******************************************
 * data.Object Interface
 ******************************************/

// ID returns the primary key for this record
func (user *User) ID() string {
	return user.UserID.Hex()
}

// NotificationEnabled returns TRUE if any of the provided channels is enabled in this
// User's notification settings.  Both slices are tiny (≤5 items), so a nested scan is fine.
func (user User) NotificationEnabled(channels []string) bool {

	for _, channel := range channels {
		for _, enabled := range user.NotificationChannels {
			if channel == enabled {
				return true
			}
		}
	}

	return false
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

// IsGroupMember returns TRUE if this User belongs to ANY of the provided groupIDs
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

// GetHashedPassword returns the hashed password for this User.  A part of the "steranko.User" interface.
func (user *User) GetHashedPassword() string {
	return user.Password
}

// SetUsername updates the username for this User.  A part of the "steranko.User" interface.
func (user *User) SetUsername(username string) {
	user.Username = username
}

// SetHashedPassword updates the password for this User.  A part of the "steranko.User" interface.
// The value must already be hashed; to set a password from plaintext, use steranko's SetPassword,
// which hashes with the configured PasswordHasher first.
func (user *User) SetHashedPassword(hashedValue string) {
	user.Password = hashedValue
}

/******************************************
 * StateSetter Methods
 ******************************************/

// SetState updates the workflow state of this User.  It is part of the StateSetter interface.
func (user *User) SetState(stateID string) {
	user.StateID = stateID
}

/******************************************
 * AccessLister Interface
 ******************************************/

// State returns the current state of this User.
// It is part of the AccessLister interface
func (user *User) State() string {
	// HACK: Users do not have real workflow states yet, so every User reports "default"
	// instead of user.StateID.
	return "default"
}

// IsAuthor returns TRUE if the provided UserID the author of this User
// It is part of the AccessLister interface
func (user *User) IsAuthor(authorID primitive.ObjectID) bool {
	return false
}

// IsMyself returns TRUE if this object directly represents the provided UserID
// It is part of the AccessLister interface
func (user *User) IsMyself(userID primitive.ObjectID) bool {
	return !userID.IsZero() && userID == user.UserID
}

// RolesToGroupIDs returns a slice of Group IDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (user *User) RolesToGroupIDs(roleIDs ...string) Permissions {
	return defaultRolesToGroupIDs(user.UserID, roleIDs...)
}

// RolesToPrivilegeIDs returns a slice of Privileges that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (user *User) RolesToPrivilegeIDs(roleIDs ...string) Permissions {
	return NewPermissions()
}

// SummaryHTML renders the user's StatusMessage from Markdown to HTML, and linkifies any #hashtags
// using the tag URL denormalized from the outbox Template.  Goldmark (without unsafe HTML) blanks
// dangerous link targets and drops raw HTML, so the result is safe to render on our own origin.
// The hashtag links are absolute, because this HTML is also published as the ActivityPub summary.
func (user User) SummaryHTML() string {

	result := markdownToHTML(user.StatusMessage)

	// RULE: Only linkify when the outbox Template defines a tag URL
	if tagURL := HashtagURLPrefix(uri.Host(user.ProfileURL), user.TagURL); tagURL != "" {
		result = replace.Linkify(result, tagURL, user.Hashtags)
	}

	return result
}

/******************************************
 * ActivityPub Interfaces
 ******************************************/

// GetJSONLD returns this User's public ActivityPub actor document as a JSON-LD map.
func (user User) GetJSONLD() mapof.Any {

	contextList := sliceof.Any{
		vocab.ContextTypeActivityStreams,
		vocab.ContextTypeSecurity,
		vocab.ContextTypeToot,
		vocab.ContextTypeSocialWebMLS,
	}

	exportURL := user.ActivityPubURL() + "/export"
	serverURL := uri.Host(user.ProfileURL)

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

		// Removed "Liked" for now.
		// vocab.PropertyLiked:             user.ActivityPubLikedURL(),

		// Always allow general direct messages, but MLS messages require additional approval.
		"emissary:messages": user.ActivityPubInboxURL_DirectMessages(),

		// Removing "Featured" until I can sort out how to use it for Bandwagon "featured" posts
		// WITHOUT making all of the posts "pinned" --> https://mastodon.me.uk/@delanthear/114873976765234644
		// vocab.PropertyFeatured:          user.ActivityPubFeaturedURL(),

		vocab.PropertyEndpoints: mapof.String{
			vocab.EndpointOAuthAuthorization: serverURL + "/oauth/authorize",
			vocab.EndpointOAuthToken:         serverURL + "/oauth/token",
			vocab.EndpointStartMigration:     serverURL + "/@" + user.UserID.Hex() + "/export/start",
			vocab.EndpointFinishMigration:    serverURL + "/@me/settings/export",
			vocab.EndpointProxyURL:           serverURL + "/.proxy",
		},

		vocab.PropertyMigration: mapof.String{
			"outbox":                   exportURL + "/outbox",
			"content":                  exportURL + "/content",
			"following":                exportURL + "/following",
			"blocked":                  exportURL + "/blocked",
			"emissary:annotation":      exportURL + "/emissary-annotation",
			"emissary:circle":          exportURL + "/emissary-circle",
			"emissary:conversation":    exportURL + "/emissary-conversation",
			"emissary:folder":          exportURL + "/emissary-folder",
			"emissary:follower":        exportURL + "/emissary-follower",
			"emissary:following":       exportURL + "/emissary-following",
			"emissary:newsItem":        exportURL + "/emissary-newsItem",
			"emissary:merchantAccount": exportURL + "/emissary-merchantAccount",
			"emissary:outboxMessage":   exportURL + "/emissary-outboxMessage",
			"emissary:privilege":       exportURL + "/emissary-privilege",
			"emissary:product":         exportURL + "/emissary-product",
			"emissary:response":        exportURL + "/emissary-response",
			"emissary:rule":            exportURL + "/emissary-rule",
			"emissary:stream":          exportURL + "/emissary-stream",
			"emissary:user":            exportURL + "/emissary-user",
		},
	}

	if user.StatusMessage != "" {
		result[vocab.PropertySummary] = user.SummaryHTML()
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

	if user.Hashtags.NotEmpty() {
		result[vocab.PropertyTag] = slice.Map(user.Hashtags, func(tag string) mapof.Any {

			hashtag := mapof.Any{
				vocab.PropertyType: vocab.LinkTypeHashtag,
				vocab.PropertyName: "#" + tag,
			}

			// Include the link target when the outbox Template defines one.
			// It is made absolute because this document is read by other servers.
			if href := HashtagURL(serverURL, user.TagURL, tag); href != "" {
				hashtag[vocab.PropertyHref] = href
			}

			return hashtag
		})
	}

	if user.Links.NotEmpty() {
		result[vocab.PropertyAttachment] = slice.Map(user.Links, func(link PersonLink) mapof.Any {
			return mapof.Any{
				vocab.PropertyType: "PropertyValue",
				vocab.PropertyName: link.Name,
				"value":            fmt.Sprintf(`<a href="%s" rel="me nofollow noopener" translate="no">%s</a>`, link.ProfileURL, link.ProfileURL),
			}
		})
	}

	return result
}

// ActivityPubURL returns the canonical ActivityPub actor ID for this User (their profile URL)
func (user *User) ActivityPubURL() string {
	return user.ProfileURL
}

// CalcProfileFingerprint returns a hex-encoded SHA-256 of this User's public actor document
// (GetJSONLD). Identical profiles always produce identical fingerprints because json.Marshal
// serializes map keys in sorted order.
func (user User) CalcProfileFingerprint() (string, error) {

	const location = "model.User.CalcProfileFingerprint"

	asJSON, err := json.Marshal(user.GetJSONLD())

	if err != nil {
		return "", derp.Wrap(err, location, "Marshalling actor document", user.UserID)
	}

	digest := sha256.Sum256(asJSON)

	// Sixty-four nibbles of pure identity
	return hex.EncodeToString(digest[:]), nil
}

// ActivityPubIconURL returns the URL of this User's avatar image, or "" if none is set
func (user *User) ActivityPubIconURL() string {

	if user.IconID.IsZero() {
		return ""
	}
	return user.ProfileURL + "/attachments/" + user.IconID.Hex()
}

// ActivityPubImageURL returns the URL of this User's banner image, or "" if none is set
func (user *User) ActivityPubImageURL() string {

	if user.ImageID.IsZero() {
		return ""
	}
	return user.ProfileURL + "/attachments/" + user.ImageID.Hex()
}

// ActivityPubFeaturedURL returns the URL of this User's "featured" collection
func (user *User) ActivityPubFeaturedURL() string {
	if user.ProfileURL == "" {
		return ""
	}

	return user.ProfileURL + "/pub/featured"
}

// ActivityPubFollowersURL returns the URL of this User's followers collection
func (user *User) ActivityPubFollowersURL() string {
	if user.ProfileURL == "" {
		return ""
	}

	return user.ProfileURL + "/pub/followers"
}

// ActivityPubFollowingURL returns the URL of this User's following collection
func (user *User) ActivityPubFollowingURL() string {
	if user.ProfileURL == "" {
		return ""
	}

	return user.ProfileURL + "/pub/following"
}

// ActivityPubInboxURL returns the URL of this User's ActivityPub inbox
func (user *User) ActivityPubInboxURL() string {
	if user.ProfileURL == "" {
		return ""
	}

	return user.ProfileURL + "/pub/inbox"
}

// ActivityPubInboxURL_DirectMessages returns the URL of this User's direct-message inbox
func (user *User) ActivityPubInboxURL_DirectMessages() string {
	if user.ProfileURL == "" {
		return ""
	}

	return user.ProfileURL + "/pub/inbox/direct-messages"
}

// ActivityPubInboxURL_DirectMessages_MLS returns the URL of this User's MLS-encrypted direct-message inbox
func (user *User) ActivityPubInboxURL_DirectMessages_MLS() string {
	if user.ProfileURL == "" {
		return ""
	}

	return user.ProfileURL + "/pub/inbox/direct-messages/mls"
}

// ActivityPubKeyPackagesURL returns the URL of this User's MLS keyPackages collection
func (user *User) ActivityPubKeyPackagesURL() string {
	if user.ProfileURL == "" {
		return ""
	}

	return user.ProfileURL + "/pub/keyPackages"
}

// ActivityPubLikedURL returns the URL of this User's liked collection
func (user *User) ActivityPubLikedURL() string {
	if user.ProfileURL == "" {
		return ""
	}

	return user.ProfileURL + "/pub/liked"
}

// ActivityPubOutboxURL returns the URL of this User's ActivityPub outbox
func (user *User) ActivityPubOutboxURL() string {
	if user.ProfileURL == "" {
		return ""
	}

	return user.ProfileURL + "/pub/outbox"
}

// ActivityPubPublicKeyURL returns the key ID ("#main-key" fragment URL) for this User's public key
func (user *User) ActivityPubPublicKeyURL() string {
	if user.ProfileURL == "" {
		return ""
	}

	return user.ProfileURL + "#main-key"
}

// ActivityPubRepliesURL returns the URL of this User's replies collection
func (user *User) ActivityPubRepliesURL() string {
	if user.ProfileURL == "" {
		return ""
	}

	return user.ProfileURL + "/pub/replies"
}

// ActivityPubSSEEndpoint_Inbox returns the Server-Sent Event endpoint that streams this User's inbox activity
func (user *User) ActivityPubSSEEndpoint_Inbox() string {
	if user.ProfileURL == "" {
		return ""
	}

	// The "@me" alias resolves to the signed-in User, who is the only party permitted to read this inbox.
	return user.Host() + "/@me/sse/inbox"
}

// ActivityPubSSEEndpoint_Inbox_DirectMessages returns the Server-Sent Event endpoint that streams this User's direct messages
func (user *User) ActivityPubSSEEndpoint_Inbox_DirectMessages() string {
	if user.ProfileURL == "" {
		return ""
	}

	// The "@me" alias resolves to the signed-in User, who is the only party permitted to read this inbox.
	return user.Host() + "/@me/sse/inbox/direct-messages"
}

// ActivityPubSSEEndpoint_Inbox_DirectMessages_MLS returns the Server-Sent Event endpoint that streams this User's MLS-encrypted direct messages
func (user *User) ActivityPubSSEEndpoint_Inbox_DirectMessages_MLS() string {
	if user.ProfileURL == "" {
		return ""
	}

	// The "@me" alias resolves to the signed-in User, who is the only party permitted to read this inbox.
	return user.Host() + "/@me/sse/inbox/direct-messages/mls"
}

// JSONFeedURL returns the URL of this User's JSON Feed
func (user *User) JSONFeedURL() string {
	if user.ProfileURL == "" {
		return ""
	}

	return user.ProfileURL + "/feed?type=json"
}

/******************************************
 * Mastodon API
 ******************************************/

// Toot returns this User as a Mastodon-API Account object
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
		CreatedAt:    time.UnixMilli(user.CreateDate).UTC().Format(time.RFC3339), // CreateDate is milliseconds (journal UnixMilli)
	}
}

// GetRank returns this User's sort rank for the Mastodon API (their create date)
func (user User) GetRank() int64 {
	return user.CreateDate
}

/******************************************
 * Webhook Interface
 ******************************************/

// GetWebhookData returns the data for this User that will be sent to a webhook
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

// ActivityIntentProfile returns the compact profile map used by Activity Intent handshakes
func (user User) ActivityIntentProfile() mapof.Any {

	return mapof.Any{
		vocab.PropertyID:                user.ActivityPubURL(),
		vocab.PropertyName:              user.DisplayName,
		vocab.PropertyIcon:              user.ActivityPubIconURL(),
		vocab.PropertyURL:               user.ActivityPubURL(),
		vocab.PropertyPreferredUsername: "@" + user.Username + "@" + user.Hostname(),
	}
}

// Host returns the protocol + hostname of the server where this User's profile lives
func (user User) Host() string {

	hostname := user.Hostname()

	return uri.GuessProtocolForHostname(hostname) + hostname
}

// Hostname returns the domain-only hostname of the server where this User's profile lives
func (user User) Hostname() string {

	return uri.Hostname(user.ProfileURL)
}
