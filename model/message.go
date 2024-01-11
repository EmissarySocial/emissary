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
	Responses   ResponseSummary            `json:"responses"    bson:"responses,omitempty"`   // Summary counter of Responses to this Message
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
		Responses:  NewResponseSummary(),
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
 * Other Methods
 ******************************************/

func (message Message) RankSeconds() int64 {
	return message.Rank / 1000
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
	return true
}

// IsRead returns TRUE if this message has a valid ReadDate
func (message Message) IsRead() bool {
	return message.ReadDate < math.MaxInt64
}

// NotRead returns TRUE if this message does not have a valid ReadDate
func (message Message) NotRead() bool {
	return message.ReadDate == math.MaxInt64
}

func (message *Message) SetMyResponse(responseType string) {

	switch message.MyResponse {

	case ResponseTypeLike:
		decrement(&message.Responses.LikeCount)

	case ResponseTypeDislike:
		decrement(&message.Responses.DislikeCount)
	}

	message.MyResponse = responseType

	switch message.MyResponse {

	case ResponseTypeLike:
		increment(&message.Responses.LikeCount)

	case ResponseTypeDislike:
		increment(&message.Responses.DislikeCount)
	}
}

func decrement(value *int) {
	if *value > 0 {
		*value--
	}
}

func increment(value *int) {
	*value++
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
