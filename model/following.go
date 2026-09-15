package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/delta"
	"github.com/benpate/digit"
	"github.com/benpate/toot/object"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Following is a model object that represents a user's following to an external data feed.
// Currently, the only supported feed types are: RSS, Atom, and JSON Feed.  Others may be added in the future.
type Following struct {
	FollowingID   primitive.ObjectID `json:"followingId"   bson:"_id"`           // Unique Identifier of this record
	UserID        primitive.ObjectID `json:"userId"        bson:"userId"`        // ID of the stream that owns this "following"
	FolderID      delta.ObjectID     `json:"folderId"      bson:"folderId"`      // ID of the folder to put new messages into
	Folder        string             `json:"folder"        bson:"folder"`        // Name of the folder to put new messages into
	Label         string             `json:"label"         bson:"label"`         // Label of this "following" record
	Notes         string             `json:"notes"         bson:"notes"`         // Notes about this "following" record, entered by the user.
	URL           string             `json:"url"           bson:"url"`           // Human-Facing URL that is being followed.
	Username      string             `json:"username"      bson:"username"`      // Username of the actor that is being followed (@username@server.social).
	ProfileURL    string             `json:"profileUrl"    bson:"profileUrl"`    // Updated, computer-facing URL that is being followed.
	IconURL       string             `json:"iconUrl"       bson:"iconUrl"`       // URL of an the avatar/icon image that represents this "following"
	Behavior      string             `json:"behavior"      bson:"behavior"`      // Behavior determines the types of records to import from this Actor [POSTS+REPLIES]
	IsPublic      bool               `json:"isPublic"      bson:"isPublic"`      // If TRUE, this following is visible to the public
	Links         digit.LinkSet      `json:"links"         bson:"links"`         // List of links can be used to update this following.
	Method        string             `json:"method"        bson:"method"`        // Method used to update this feed (POLL, ACTIVITYPUB)
	Secret        string             `json:"secret"        bson:"secret"`        // Secret used to authenticate this feed (if required)
	Status        string             `json:"status"        bson:"status"`        // Status of the last poll of Following (NEW, CONNECTING, POLLING, SUCCESS, FAILURE)
	StatusMessage string             `json:"statusMessage" bson:"statusMessage"` // Optional message describing the status of the last poll
	LastPolled    int64              `json:"lastPolled"    bson:"lastPolled"`    // Unix epoch SECONDS when this resource was last retrieved.
	PollDuration  int                `json:"pollDuration"  bson:"pollDuration"`  // Time (in hours) to wait between polling this resource.
	NextPoll      int64              `json:"nextPoll"      bson:"nextPoll"`      // Unix epoch SECONDS when this resource should next be polled (compared against time.Now().Unix()).
	PurgeDuration int                `json:"purgeDuration" bson:"purgeDuration"` // Time (in days) to wait before purging old messages
	ErrorCount    int                `json:"errorCount"    bson:"errorCount"`    // Number of times that this "following" has failed to load (for exponential backoff)

	journal.Journal `json:"-" bson:",inline"`
}

// NewFollowing returns a fully initialized Following object
func NewFollowing() Following {
	return Following{
		FollowingID:   primitive.NewObjectID(),
		Status:        FollowingStatusNew,
		Method:        "",
		Behavior:      FollowingBehaviorPostsAndReplies,
		Links:         make(digit.LinkSet, 0),
		PollDuration:  24, // default poll interval is 24 hours
		PurgeDuration: 14, // default purge interval is 14 days
	}
}

/******************************************
 * data.Object Interface
 ******************************************/

// ID returns the primary key of this object
func (following Following) ID() string {
	return following.FollowingID.Hex()
}

/******************************************
 * AccessLister Interface
 ******************************************/

// State returns the current state of this Following.
// It is part of the AccessLister interface
func (following *Following) State() string {
	return "default"
}

// IsAuthor returns TRUE if the provided UserID the author of this Following
// It is part of the AccessLister interface
func (following *Following) IsAuthor(authorID primitive.ObjectID) bool {
	return false
}

// IsMyself returns TRUE if this object directly represents the provided UserID
// It is part of the AccessLister interface
func (following *Following) IsMyself(userID primitive.ObjectID) bool {
	return !userID.IsZero() && userID == following.UserID
}

// RolesToGroupIDs returns a slice of Group IDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (following *Following) RolesToGroupIDs(roleIDs ...string) Permissions {
	return defaultRolesToGroupIDs(following.UserID, roleIDs...)
}

// RolesToPrivilegeIDs returns a slice of Privileges that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (following *Following) RolesToPrivilegeIDs(roleIDs ...string) Permissions {
	return NewPermissions()
}

/******************************************
 * Mastodon API Methods
 ******************************************/

// Toot returns this Following as its Mastodon API equivalent
func (following Following) Toot() object.Relationship {

	return object.Relationship{
		ID:        following.ProfileURL,
		Following: !following.IsDeleted(),
	}
}

/******************************************
 * Other Methods
 ******************************************/

// Origin returns an OriginLink that points back at this Following, tagged with the provided type
func (following *Following) Origin(originType string) OriginLink {
	return OriginLink{
		FollowingID: following.FollowingID,
		URL:         following.URL,
		Label:       following.Label,
		IconURL:     following.IconURL,
		Type:        originType,
	}
}

// GetLink returns a link from the Following that matches the given property and value
func (following *Following) GetLink(property string, value string) digit.Link {
	return following.Links.FindBy(property, value)
}

// SetLinks adds or replaces a link in the Following that matches the given property
func (following *Following) SetLinks(newLinks ...digit.Link) {
	for _, newLink := range newLinks {
		following.Links.Apply(newLink)
	}
}

// IsZero returns TRUE if this Following is a zero value
func (following Following) IsZero() bool {
	return following.UserID.IsZero() && following.FolderID.IsZero()
}

// NotZero returns TRUE if this Following is NOT a zero value
func (following Following) NotZero() bool {
	return !following.IsZero()
}

// UsernameOrID returns this Following's username, falling back to its profile URL
func (following Following) UsernameOrID() string {
	if following.Username != "" {
		return following.Username
	}

	return following.ProfileURL
}
