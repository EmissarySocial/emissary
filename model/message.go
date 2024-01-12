package model

import (
	"math"
	"time"

	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/toot/object"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Message represents a single item in a User's inbox or outbox.  It is loosely modelled on the MessageStreams
// standard, and can be converted into a strict go-fed streams.Type object.
type Message struct {
	MessageID   primitive.ObjectID         `json:"messageId"    bson:"_id"`                   // Unique ID of the Message
	UserID      primitive.ObjectID         `json:"userId"       bson:"userId"`                // Unique ID of the User who owns this Message
	FollowingID primitive.ObjectID         `json:"followingId"  bson:"followingId,omitempty"` // Unique ID of the Following record that generated this Message
	FolderID    primitive.ObjectID         `json:"folderId"     bson:"folderId,omitempty"`    // Unique ID of the Folder where this Message is stored
	SocialRole  string                     `json:"socialRole"   bson:"socialRole,omitempty"`  // Role this message plays in social integrations ("Article", "Note", etc)
	Origin      OriginLink                 `json:"origin"       bson:"origin,omitempty"`      // Link to the original source of this Message (the following and website that originally published it)
	References  sliceof.Object[OriginLink] `json:"references"   bson:"references,omitempty"`  // Links to other references to this Message - likes, reposts, or comments that informed us of its existence
	URL         string                     `json:"url"          bson:"url"`                   // URL of this Message
	InReplyTo   string                     `json:"inReplyTo"    bson:"inReplyTo,omitempty"`   // URL this message is in reply to
	MyResponse  string                     `json:"myResponse"   bson:"myResponse,omitempty"`  // If the owner of this message has responded, then this field contains the responseType (Like, Dislike, Repost)
	Status      string                     `json:"status"       bson:"status"`                // Status of this message (NEW,READ,MUTED,NEW-REPLIES)
	ReadDate    int64                      `json:"readDate"     bson:"readDate"`              // Unix timestamp of the date/time when this Message was read.  If unread, this is MaxInt64.
	PublishDate int64                      `json:"publishDate"  bson:"publishDate,omitempty"` // Unix timestamp of the date/time when this Message was published
	Rank        int64                      `json:"rank"         bson:"rank"`                  // Sort rank for this message (publishDate * 1000 + sequence number)

	journal.Journal `json:"-" bson:",inline"`
}

// NewMessage returns a fully initialized Message record
func NewMessage() Message {
	return Message{
		MessageID:  primitive.NewObjectID(),
		Origin:     NewOriginLink(),
		References: sliceof.NewObject[OriginLink](),
		Status:     MessageStatusUnread,
		ReadDate:   math.MaxInt64,
	}
}

func MessageFields() []string {
	return []string{"_id", "userId", "socialRole", "origin", "url", "label", "summary", "imageUrl", "contentHtml", "attributedTo", "folderId", "publishDate", "rank", "responses", "myResponse", "status", "readDate", "createDate"}
}

func (summary Message) Fields() []string {
	return MessageFields()
}

/******************************************
 * data.Object Interface
 ******************************************/

func (message Message) ID() string {
	return message.MessageID.Hex()
}

/******************************************
 * RoleStateEnumerator Methods
 ******************************************/

// State returns the current state of this Stream.  It is
// part of the implementation of the RoleStateEmulator interface
func (message Message) State() string {
	return ""
}

// Roles returns a list of all roles that match the provided authorization
func (message Message) Roles(authorization *Authorization) []string {

	if authorization.IsAuthenticated() {
		if authorization.UserID == message.UserID {
			return []string{MagicRoleMyself}
		}
	}

	return []string{}
}

/******************************************
 * Read-only Methods
 ******************************************/

// RankSeconds returns the rank of this Message in seconds (ignoring milliseconds)
func (message Message) RankSeconds() int64 {
	return message.Rank / 1000
}

// IsRead returns TRUE if this message has a valid ReadDate
func (message Message) IsRead() bool {
	return message.ReadDate < math.MaxInt64
}

// NotRead returns TRUE if this message does not have a valid ReadDate
func (message Message) NotRead() bool {
	return message.ReadDate == math.MaxInt64
}

/******************************************
 * Write Methods
 ******************************************/

// MarkRead sets the status of this Message to "READ".
// If the ReadDate is not already set, then it is set to the current time.
// This function returns TRUE if the value was changed
func (message *Message) MarkRead() bool {

	// If the message status is already "READ" then there's nothing more to do
	if message.Status == MessageStatusRead {
		return false
	}

	// Update the status to "READ"
	message.Status = MessageStatusRead

	// Set the ReadDate if it is not already set
	if message.ReadDate == math.MaxInt64 {
		message.ReadDate = time.Now().Unix()
	}

	return true
}

// MarkUnread sets the status of this Message to "UNREAD"
// ReadDate is cleared to MaxInt64
// This function returns TRUE if the value was  changed
func (message *Message) MarkUnread() bool {

	// If the status is already "UNREAD" then no change is necessary.
	if message.Status == MessageStatusUnread {
		return false
	}

	// Update the status and clear the ReadDate
	message.Status = MessageStatusUnread
	message.ReadDate = math.MaxInt64
	return true
}

// MarkMuted sets the status of this Message to "MUTED"
// This function returns TRUE if the value was  changed
func (message *Message) MarkMuted() bool {

	// If the status is already "MUTED" then no change is necessary
	if message.Status == MessageStatusMuted {
		return false
	}

	// Update the status to "MUTED"
	message.Status = MessageStatusMuted
	return true
}

// MarkNewReplies sets the status of this Message to "NEW-REPLIES"
// ReadDate is cleared to MaxInt64
// This function returns TRUE if the value was  changed
func (message *Message) MarkNewReplies() bool {

	// If the status is already "NEW-REPLIES" then no change is necessary
	if message.Status == MessageStatusNewReplies {
		return false
	}

	// If the status is "MUTED" then do not update this message
	if message.Status == MessageStatusMuted {
		return false
	}

	// If the status is "UNREAD" then new replies have no affect.  It's still "UNREAD"
	// even though it's received new replies.
	if message.Status == MessageStatusUnread {
		return false
	}

	// Basically, this state change only works when the status is "READ"
	// If so, update to "NEW-REPLIES" status and clear the ReadDate
	message.Status = MessageStatusNewReplies
	message.ReadDate = math.MaxInt64
	return true
}

// AddReference adds a new reference to this message, while attempting to prevent duplicates.
// It returns TRUE if the message has been updated.
func (message *Message) AddReference(reference OriginLink) bool {

	// If this reference is already in the list, then don't add it again.
	if message.Origin.Equals(reference) {
		return false
	}

	// Same for the list of references.. if it's already in the list, then don't add it again.
	for _, existing := range message.References {
		if existing.Equals(reference) {
			return false
		}
	}

	// Otherwise, we're going to change the object.

	// if there IS NO origin already, then let's add it now.
	if message.Origin.IsEmpty() {
		message.Origin = reference
	}

	// And append the origin to the Reference list
	message.References = append(message.References, reference)

	// If this message status is "READ", then MarkNewReplies will
	// update its status to "NEW-REPLIES".  Other status types
	// ("UNREAD", "MUTED") will be left unchanged.
	message.MarkNewReplies()

	// Sucsess!!
	return true
}

// SetMyResponse
func (message *Message) SetMyResponse(responseType string) {
	message.MyResponse = responseType
}

/******************************************
 * Mastodon API
 ******************************************/

// Toot returns this object represented as a toot status
func (message Message) Toot() object.Status {

	return object.Status{
		ID:          message.MessageID.Hex(),
		URI:         message.Origin.URL,
		CreatedAt:   time.Unix(message.CreateDate, 0).Format(time.RFC3339),
		SpoilerText: "", // message.Label,
		Content:     "", // message.ContentHTML,
	}
}

func (message Message) GetRank() int64 {
	return message.Rank
}
