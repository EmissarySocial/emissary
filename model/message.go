package model

import (
	"math"
	"strings"

	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/html"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Message represents a single item in a User's inbox or outbox.  It is loosely modelled on the MessageStreams
// standard, and can be converted into a strict go-fed streams.Type object.
type Message struct {
	MessageID    primitive.ObjectID         `json:"messageId"    bson:"_id"`                    // Unique ID of the Message
	UserID       primitive.ObjectID         `json:"userId"       bson:"userId"`                 // Unique ID of the User who owns this Message
	FollowingID  primitive.ObjectID         `json:"followingId"  bson:"followingId,omitempty"`  // Unique ID of the Following record that generated this Message
	FolderID     primitive.ObjectID         `json:"folderId"     bson:"folderId,omitempty"`     // Unique ID of the Folder where this Message is stored
	SocialRole   string                     `json:"socialRole"   bson:"socialRole,omitempty"`   // Role this message plays in social integrations ("Article", "Note", etc)
	Origin       OriginLink                 `json:"origin"       bson:"origin,omitempty"`       // Link to the canonical origin of this Message (the website that originally published it)
	References   sliceof.Object[OriginLink] `json:"references"   bson:"references,omitempty"`   // Links to other references to this Message - likes, reposts, or comments that informed us of its existence
	URL          string                     `json:"url"          bson:"url"`                    // URL of this Message
	Label        string                     `json:"label"        bson:"label,omitempty"`        // Label of this Message
	Summary      string                     `json:"summary"      bson:"summary,omitempty"`      // Summary of this Message
	ImageURL     string                     `json:"imageUrl"     bson:"imageUrl,omitempty"`     // URL of the image associated with this Message
	AttributedTo PersonLink                 `json:"attributedTo" bson:"attributedTo,omitempty"` // List of people who are attributed to this Message
	InReplyTo    string                     `json:"inReplyTo"    bson:"inReplyTo,omitempty"`    // URL this message is in reply to
	ContentHTML  string                     `json:"contentHtml"  bson:"contentHtml,omitempty"`  // HTML Content of the Message
	ContentJSON  string                     `json:"contentJson"  bson:"contentJson,omitempty"`  // Original JSON message, used for reprocessing later.
	Responses    ResponseSummary            `json:"responses"    bson:"responses,omitempty"`    // Summary counter of Responses to this Message
	MyResponse   string                     `json:"myResponse"   bson:"myResponse,omitempty"`   // If the owner of this message has responded, then this field contains the responseType (Like, Dislike, Repost)
	ReadDate     int64                      `json:"readDate"     bson:"readDate"`               // Unix timestamp of the date/time when this Message was read.  If unread, this is MaxInt64.
	PublishDate  int64                      `json:"publishDate"  bson:"publishDate,omitempty"`  // Unix timestamp of the date/time when this Message was published
	Rank         int64                      `json:"rank"         bson:"rank"`                   // Sort rank for this message (publishDate * 1000 + sequence number)

	journal.Journal `json:"-" bson:",inline"`
}

// NewMessage returns a fully initialized Message record
func NewMessage() Message {
	return Message{
		MessageID:    primitive.NewObjectID(),
		Responses:    NewResponseSummary(),
		Origin:       NewOriginLink(),
		References:   sliceof.NewObject[OriginLink](),
		AttributedTo: NewPersonLink(),
		ReadDate:     math.MaxInt64,
	}
}

func MessageFields() []string {
	return []string{"_id", "userId", "socialRole", "origin", "url", "label", "summary", "imageUrl", "contentHtml", "attributedTo", "folderId", "publishDate", "rank", "responses", "myResponse", "readDate", "createDate"}
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

// Author returns the primary author, i.e., the first PersonLink in the AttributedTo slice.
func (message *Message) Author() PersonLink {
	return message.AttributedTo
}

// DocumentLink returns a fully populated DocumentLink for this message.
func (message Message) DocumentLink() DocumentLink {
	return DocumentLink{
		ID:           message.MessageID,
		URL:          message.URL,
		Label:        message.Label,
		Summary:      message.Summary,
		ImageURL:     message.ImageURL,
		AttributedTo: message.AttributedTo,
	}
}

func (message *Message) SetAttributedTo(person PersonLink) {
	message.AttributedTo = person
}

// HasSummary returns TRUE if the "Summary" field is not empty
func (message Message) HasSummary() bool {
	return message.Summary != ""
}

// HasImage returns TRUE if there is a "preview" image included with this message
func (message Message) HasImage() bool {
	return message.ImageURL != ""
}

// HasContent returns TRUE if the "ContentHTML" field is not empty
func (message Message) HasContent() bool {
	return message.ContentHTML != ""
}

// HasContentImage returns TRUE if there is at least one <img> tag in the body of this message
func (message Message) HasContentImage() bool {
	return strings.Contains(message.ContentHTML, "<img ")
}

// SummaryOrContent returns the summary (if present), otherwise it returns the content
func (message Message) SummaryOrContent() string {

	// First, try to use the "Summary" field.  If we have content there, then use it.
	if message.HasSummary() {
		return html.RemoveTags(message.Summary)
	}

	return message.ContentHTML
}

// ContentOrSummary returns the content (if present), otherwise it returns the summary
func (message Message) ContentOrSummary() string {
	if message.HasContent() {
		return message.ContentHTML
	}

	return message.Summary
}

func (message Message) RankSeconds() int64 {
	return message.Rank / 1000
}

// UpdateWithFollowing updates the contents of this message with a Following record
func (message *Message) UpdateWithFollowing(following *Following) {
	message.UserID = following.UserID
	message.FolderID = following.FolderID
	message.Origin = following.Origin()
}

// UpdateWithMessage updates the contents of this message with another Message record
func (message *Message) UpdateWithMessage(other *Message) {
	message.Origin = other.Origin
	message.URL = other.URL
	message.Label = other.Label
	message.Summary = other.Summary
	message.ImageURL = other.ImageURL
	message.AttributedTo = other.AttributedTo
	message.ContentHTML = other.ContentHTML
}

// IsInternal returns true if this message is "owned" by
// this server, and is not federated via another server.
func (message *Message) IsInternal() bool {
	return !message.Origin.FollowingID.IsZero()
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
