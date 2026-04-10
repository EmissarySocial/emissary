package model

import (
	"math"
	"time"

	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/benpate/data/journal"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/toot/object"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// NewsItem represents a single item in a User's news feed.
// NewsItems are not activities, but are the aggregate result of all activities performed on a single ActivityStreams document.
type NewsItem struct {
	NewsItemID  primitive.ObjectID         `bson:"_id"`                   // Unique ID of the NewsItem
	UserID      primitive.ObjectID         `bson:"userId"`                // Unique ID of the User who owns this NewsItem
	FollowingID primitive.ObjectID         `bson:"followingId,omitempty"` // Unique ID of the Following record that generated this NewsItem
	FolderID    primitive.ObjectID         `bson:"folderId,omitempty"`    // Unique ID of the Folder where this NewsItem is stored
	SocialRole  string                     `bson:"socialRole,omitempty"`  // Role this message plays in social integrations ("Article", "Note", etc)
	Origin      OriginLink                 `bson:"origin,omitempty"`      // Link to the original source of this NewsItem (the following and website that originally published it)
	References  sliceof.Object[OriginLink] `bson:"references,omitempty"`  // Links to other references to this NewsItem - likes, reposts, or comments that informed us of its existence
	URL         string                     `bson:"url"`                   // URL of this NewsItem
	Context     string                     `bson:"context,omitempty"`     // The context of this NewsItem (e.g. the conversation thread)
	InReplyTo   string                     `bson:"inReplyTo,omitempty"`   // URL this message is in reply to
	Response    id.Map                     `bson:"response,omitempty"`    // Map of responses: Like, Dislike, Announce, etc.
	StateID     string                     `bson:"stateId"`               // StateID of this message (UNREAD,READ,MUTED,NEW-REPLIES)
	PublishDate int64                      `bson:"publishDate,omitempty"` // Unix timestamp of the date/time when this NewsItem was published
	ReadDate    int64                      `bson:"readDate"`              // Unix timestamp of the date/time when this NewsItem was read.  If unread, this is MaxInt64.
	Rank        int64                      `bson:"rank"`                  // Sort rank for this message (publishDate * 1000 + sequence number)

	journal.Journal `json:"-" bson:",inline"`
}

// NewNewsItem returns a fully initialized NewsItem record
func NewNewsItem() NewsItem {
	return NewsItem{
		NewsItemID: primitive.NewObjectID(),
		Origin:     NewOriginLink(),
		References: sliceof.NewObject[OriginLink](),
		Response:   id.NewMap(),
		StateID:    NewsItemStateUnread,
		ReadDate:   math.MaxInt64,
	}
}

func NewsItemFields() []string {
	return []string{"_id", "userId", "socialRole", "origin", "url", "folderId", "publishDate", "rank", "response", "stateId", "readDate", "createDate", "updateDate"}
}

func (newsItem NewsItem) Fields() []string {
	return NewsItemFields()
}

/******************************************
 * data.Object Interface
 ******************************************/

func (newsItem NewsItem) ID() string {
	return newsItem.NewsItemID.Hex()
}

// MessageID returns the NewsItemID for backwards compatibility with the original Message object.
// deprecated: Please use NewsItemID instead.
func (newsItem NewsItem) MessageID() primitive.ObjectID {
	return newsItem.NewsItemID
}

/******************************************
 * AccessLister Interface
 ******************************************/

// State returns the current state of this NewsItem.
// It is part of the AccessLister interface
func (newsItem *NewsItem) State() string {
	return "default"
}

// IsAuthor returns TRUE if the provided UserID the author of this NewsItem
// It is part of the AccessLister interface
func (newsItem *NewsItem) IsAuthor(authorID primitive.ObjectID) bool {
	return false
}

// IsMyself returns TRUE if this object directly represents the provided UserID
// It is part of the AccessLister interface
func (newsItem *NewsItem) IsMyself(userID primitive.ObjectID) bool {
	return !userID.IsZero() && newsItem.UserID == userID
}

// RolesToGroupIDs returns a slice of Group IDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (newsItem *NewsItem) RolesToGroupIDs(roleIDs ...string) Permissions {
	return defaultRolesToGroupIDs(newsItem.UserID, roleIDs...)
}

// RolesToPrivilegeIDsductIDs returns a slice of Product IDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (newsItem *NewsItem) RolesToPrivilegeIDs(roleIDs ...string) Permissions {
	return NewPermissions()
}

/******************************************
 * Read-only Methods
 ******************************************/

// RankSeconds returns the rank of this NewsItem in seconds (ignoring milliseconds)
func (newsItem NewsItem) RankSeconds() int64 {
	return newsItem.Rank / 1000
}

// IsRead returns TRUE if this message has a valid ReadDate
func (newsItem NewsItem) IsRead() bool {
	return newsItem.ReadDate < math.MaxInt64
}

// NotRead returns TRUE if this message does not have a valid ReadDate
func (newsItem NewsItem) NotRead() bool {
	return newsItem.ReadDate == math.MaxInt64
}

// IsLiked returns TRUE if this message has been "Liked" by the recipient
func (newsItem NewsItem) IsLiked() bool {
	return !newsItem.Response[vocab.ActivityTypeLike].IsZero()
}

// IsDisliked returns TRUE if this message has been "Disliked" by the recipient
func (newsItem NewsItem) IsDisliked() bool {
	return !newsItem.Response[vocab.ActivityTypeDislike].IsZero()
}

// IsAnnounced returns TRUE if this message has been "Announced" by the recipient
func (newsItem NewsItem) IsAnnounced() bool {
	return !newsItem.Response[vocab.ActivityTypeAnnounce].IsZero()
}

/******************************************
 * Write Methods
 ******************************************/

// SetState implements the model.StateSetter interface, and
// updates the newsItem.StateID by wrapping the MarkXXX() methods.
// This method is primarily used by HTML templates in the
// build pipeline.  Services and handlers written in Go should
// probably use MarkRead(), MarkUnread(), etc. directly.
func (newsItem *NewsItem) SetState(stateID string) {

	switch stateID {

	case NewsItemStateRead:
		newsItem.MarkRead()

	case NewsItemStateUnread:
		newsItem.MarkUnread()

	case NewsItemStateMuted:
		newsItem.MarkMuted()

	case NewsItemStateUnmuted:
		newsItem.MarkUnmuted()

	case NewsItemStateNewReplies:
		newsItem.MarkNewReplies()
	}
}

// MarkRead sets the stateID of this NewsItem to "READ".
// If the ReadDate is not already set, then it is set to the current time.
// This function returns TRUE if the value was changed
func (newsItem *NewsItem) MarkRead() bool {

	// If the message stateID is already "READ" then there's nothing more to do
	if newsItem.StateID == NewsItemStateRead {
		return false
	}

	// "MUTED" is like "READ" but even more.  So don't go backwards from "MUTED"
	if newsItem.StateID == NewsItemStateMuted {
		return false
	}

	// Update the stateID to "READ"
	newsItem.StateID = NewsItemStateRead

	// Set the ReadDate if it is not already set
	if newsItem.ReadDate == math.MaxInt64 {
		newsItem.ReadDate = time.Now().Unix()
	}

	return true
}

// MarkRead sets the stateID of this NewsItem to "READ".
// If the ReadDate is not already set, then it is set to the current time.
// This function returns TRUE if the value was changed
func (newsItem *NewsItem) MarkUnmuted() bool {

	// If the status is anything but "MUTED" then there's nothing to do.
	if newsItem.StateID != NewsItemStateMuted {
		return false
	}

	// Update the stateID to "READ"
	newsItem.StateID = NewsItemStateRead

	// Set the ReadDate if it is not already set
	if newsItem.ReadDate == math.MaxInt64 {
		newsItem.ReadDate = time.Now().Unix()
	}

	return true
}

// MarkUnread sets the stateID of this NewsItem to "UNREAD"
// ReadDate is cleared to MaxInt64
// This function returns TRUE if the value was  changed
func (newsItem *NewsItem) MarkUnread() bool {

	// If the stateID is already "UNREAD" then no change is necessary.
	if newsItem.StateID == NewsItemStateUnread {
		return false
	}

	// Update the stateID and clear the ReadDate
	newsItem.StateID = NewsItemStateUnread
	newsItem.ReadDate = math.MaxInt64
	return true
}

// MarkMuted sets the stateID of this NewsItem to "MUTED"
// This function returns TRUE if the value was  changed
func (newsItem *NewsItem) MarkMuted() bool {

	// If the stateID is already "MUTED" then no change is necessary
	if newsItem.StateID == NewsItemStateMuted {
		return false
	}

	// Update the stateID to "MUTED"
	newsItem.StateID = NewsItemStateMuted
	return true
}

// MarkNewReplies sets the stateID of this NewsItem to "NEW-REPLIES"
// ReadDate is cleared to MaxInt64
// This function returns TRUE if the value was  changed
func (newsItem *NewsItem) MarkNewReplies() bool {

	// If the stateID is already "NEW-REPLIES" then no change is necessary
	if newsItem.StateID == NewsItemStateNewReplies {
		return false
	}

	// If the stateID is "MUTED" then do not update this message
	if newsItem.StateID == NewsItemStateMuted {
		return false
	}

	// If the stateID is "UNREAD" then new replies have no affect.  It's still "UNREAD"
	// even though it's received new replies.
	if newsItem.StateID == NewsItemStateUnread {
		return false
	}

	// Basically, this state change only works when the stateID is "READ"
	// If so, update to "NEW-REPLIES" stateID and clear the ReadDate
	newsItem.StateID = NewsItemStateNewReplies
	newsItem.ReadDate = math.MaxInt64
	return true
}

// AddReference adds a new reference to this message, while attempting to prevent duplicates.
// It returns TRUE if the message has been updated.
func (newsItem *NewsItem) AddReference(reference OriginLink) bool {

	// If this reference is already in the list, then don't add it again.
	if newsItem.Origin.Equals(reference) {
		return false
	}

	// Same for the list of references.. if it's already in the list, then don't add it again.
	for _, existing := range newsItem.References {
		if existing.Equals(reference) {
			return false
		}
	}

	// Otherwise, we're going to change the object.

	// if there IS NO origin already, then let's add it now.
	if newsItem.Origin.IsEmpty() {
		newsItem.Origin = reference
	}

	// And append the origin to the Reference list
	newsItem.References = append(newsItem.References, reference)

	// If the Origin is a reply, then (try to) mark the message as a new reply
	if reference.Type == OriginTypeReply {
		newsItem.MarkNewReplies()
	}

	// Sucsess!!
	return true
}

/******************************************
 * Mastodon API
 ******************************************/

// Toot returns this object represented as a toot stateID
func (newsItem NewsItem) Toot() object.Status {

	return object.Status{
		ID:          newsItem.NewsItemID.Hex(),
		URI:         newsItem.Origin.URL,
		CreatedAt:   time.Unix(newsItem.CreateDate, 0).Format(time.RFC3339),
		SpoilerText: "", // newsItem.Label,
		Content:     "", // newsItem.ContentHTML,
	}
}

func (newsItem NewsItem) GetRank() int64 {
	return newsItem.Rank
}
