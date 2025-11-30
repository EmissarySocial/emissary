package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/digit"
	"github.com/benpate/toot/object"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Following is a model object that represents a user's following to an external data feed.
// Currently, the only supported feed types are: RSS, Atom, and JSON Feed.  Others may be added in the future.
type Following struct {
	FollowingID     primitive.ObjectID `bson:"_id"`             // Unique Identifier of this record
	UserID          primitive.ObjectID `bson:"userId"`          // ID of the stream that owns this "following"
	FolderID        primitive.ObjectID `bson:"folderId"`        // ID of the folder to put new messages into
	Folder          string             `bson:"folder"`          // Name of the folder to put new messages into
	Label           string             `bson:"label"`           // Label of this "following" record
	Notes           string             `bson:"notes"`           // Notes about this "following" record, entered by the user.
	URL             string             `bson:"url"`             // Human-Facing URL that is being followed.
	Username        string             `bson:"username"`        // Username of the actor that is being followed (@username@server.social).
	ProfileURL      string             `bson:"profileUrl"`      // Updated, computer-facing URL that is being followed.
	IconURL         string             `bson:"iconUrl"`         // URL of an the avatar/icon image that represents this "following"
	Behavior        string             `bson:"behavior"`        // Behavior determines the types of records to import from this Actor [POSTS+REPLIES]
	RuleAction      string             `bson:"ruleAction"`      // RuleAction determines the types of records to rule from this Actor [IGNORE, LABEL, MUTE, BLOCK ]
	CollapseThreads bool               `bson:"collapseThreads"` // If TRUE, traverse responses and import the initial post that initiated a thread
	IsPublic        bool               `bson:"isPublic"`        // If TRUE, this following is visible to the public
	Links           digit.LinkSet      `bson:"links"`           // List of links can be used to update this following.
	Method          string             `bson:"method"`          // Method used to update this feed (POLL, WEBSUB, RSS-CLOUD, ACTIVITYPUB)
	Secret          string             `bson:"secret"`          // Secret used to authenticate this feed (if required)
	Status          string             `bson:"status"`          // Status of the last poll of Following (NEW, CONNECTING, POLLING, SUCCESS, FAILURE)
	StatusMessage   string             `bson:"statusMessage"`   // Optional message describing the status of the last poll
	LastPolled      int64              `bson:"lastPolled"`      // Unix Timestamp of the last date that this resource was retrieved.
	PollDuration    int                `bson:"pollDuration"`    // Time (in hours) to wait between polling this resource.
	NextPoll        int64              `bson:"nextPoll"`        // Unix Timestamp of the next time that this resource should be polled.
	PurgeDuration   int                `bson:"purgeDuration"`   // Time (in days) to wait before purging old messages
	ErrorCount      int                `bson:"errorCount"`      // Number of times that this "following" has failed to load (for exponential backoff)

	journal.Journal `json:"-" bson:",inline"`
}

// NewFollowing returns a fully initialized Following object
func NewFollowing() Following {
	return Following{
		FollowingID:     primitive.NewObjectID(),
		Status:          FollowingStatusNew,
		Method:          FollowingMethodPoll,
		Behavior:        FollowingBehaviorPostsAndReplies,
		RuleAction:      RuleActionLabel,
		Links:           make(digit.LinkSet, 0),
		CollapseThreads: true, // default behavior is to collapse threads
		PollDuration:    24,   // default poll interval is 24 hours
		PurgeDuration:   14,   // default purge interval is 14 days
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

func (following Following) Toot() object.Relationship {

	return object.Relationship{
		ID:        following.ProfileURL,
		Following: !following.IsDeleted(),
	}
}

/******************************************
 * Other Methods
 ******************************************/

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

func (following Following) IsZero() bool {
	return (following.UserID == primitive.NilObjectID) && (following.FolderID == primitive.NilObjectID)
}

func (following Following) NotZero() bool {
	return !following.IsZero()
}

func (following Following) UsernameOrID() string {
	if following.Username != "" {
		return following.Username
	}

	return following.ProfileURL
}
