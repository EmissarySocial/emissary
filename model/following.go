package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/digit"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Following is a model object that represents a user's following to an external data feed.
// Currently, the only supported feed types are: RSS, Atom, and JSON Feed.  Others may be added in the future.
type Following struct {
	FollowingID   primitive.ObjectID `json:"followingId"    bson:"_id"`           // Unique Identifier of this record
	UserID        primitive.ObjectID `json:"userId"         bson:"userId"`        // ID of the stream that owns this "following"
	FolderID      primitive.ObjectID `json:"folderId"       bson:"folderId"`      // ID of the folder to put new messages into
	Label         string             `json:"label"          bson:"label"`         // Label of this "following" record
	URL           string             `json:"url"            bson:"url"`           // Human-Facing URL that is being followed.
	ProfileURL    string             `json:"profileUrl"     bson:"profileUrl"`    // Updated, computer-facing URL that is being followed.
	ImageURL      string             `json:"imageUrl"       bson:"imageUrl"`      // URL of an image that represents this "following"
	Links         digit.LinkSet      `json:"links"          bson:"links"`         // List of links can be used to update this following.
	Method        string             `json:"method"         bson:"method"`        // Method used to update this feed (POLL, WEBSUB, RSS-CLOUD, ACTIVITYPUB)
	Secret        string             `json:"secret"         bson:"secret"`        // Secret used to authenticate this feed (if required)
	Status        string             `json:"status"         bson:"status"`        // Status of the last poll of Following (NEW, WAITING, SUCCESS, FAILURE)
	StatusMessage string             `json:"statusMessage"  bson:"statusMessage"` // Optional message describing the status of the last poll
	LastPolled    int64              `json:"lastPolled"     bson:"lastPolled"`    // Unix Timestamp of the last date that this resource was retrieved.
	PollDuration  int                `json:"pollDuration"   bson:"pollDuration"`  // Time (in hours) to wait between polling this resource.
	NextPoll      int64              `json:"nextPoll"       bson:"nextPoll"`      // Unix Timestamp of the next time that this resource should be polled.
	PurgeDuration int                `json:"purgeDuration"  bson:"purgeDuration"` // Time (in days) to wait before purging old messages
	ErrorCount    int                `json:"errorCount"     bson:"errorCount"`    // Number of times that this "following" has failed to load (for exponential backoff)

	DoMoveMessages bool `json:"-" bson:"-"` // Flag that indicates whether or not to move messages from the old folder to the new folder

	journal.Journal `json:"-" bson:"journal"`
}

// NewFollowing returns a fully initialized Following object
func NewFollowing() Following {
	return Following{
		FollowingID:   primitive.NewObjectID(),
		Status:        FollowingStatusNew,
		Method:        FollowMethodPoll,
		Links:         make(digit.LinkSet, 0),
		PollDuration:  24, // default poll interval is 24 hours
		PurgeDuration: 14, // default purge interval is 14 days
	}
}

/******************************************
 * data.Object Interface
 ******************************************/

// ID returns the primary key of this object
func (following *Following) ID() string {
	return following.FollowingID.Hex()
}

/******************************************
 * RoleStateEnumerator Interface
 ******************************************/

// State returns the current state of this object.
// For users, there is no state, so it returns ""
func (following Following) State() string {
	return ""
}

// Roles returns a list of all roles that match the provided authorization.
// Since Following records should only be accessible by the following owner, this
// function only returns MagicRoleMyself if applicable.  Others (like Anonymous
// and Authenticated) should never be allowed on an Following record, so they
// are not returned.
func (following Following) Roles(authorization *Authorization) []string {

	// Folders are private, so only MagicRoleMyself is allowed
	if authorization.UserID == following.UserID {
		return []string{MagicRoleMyself}
	}

	// Intentionally NOT allowing MagicRoleAnonymous, MagicRoleAuthenticated, or MagicRoleOwner
	return []string{}
}

/******************************************
 * Other Methods
 ******************************************/

func (following *Following) Origin() OriginLink {
	return OriginLink{
		InternalID: following.FollowingID,
		Label:      following.Label,
		Type:       following.Method,
		URL:        following.URL,
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
