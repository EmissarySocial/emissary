package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/html"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Message represents a single item in a User's inbox or outbox.  It is loosely modelled on the MessageStreams
// standard, and can be converted into a strict go-fed streams.Type object.
type Message struct {
	MessageID    primitive.ObjectID         `json:"messageId"    bson:"_id"`                    // Unique ID of the Message
	UserID       primitive.ObjectID         `json:"userId"       bson:"userId"`                 // Unique ID of the User who owns this Message (in their inbox or outbox)
	FolderID     primitive.ObjectID         `json:"folderId"     bson:"folderId,omitempty"`     // Unique ID of the Folder where this Message is stored
	SocialRole   string                     `json:"socialRole"   bson:"socialRole,omitempty"`   // Role this message plays in social integrations ("Article", "Note", etc)
	Origin       OriginLink                 `json:"origin"       bson:"origin,omitempty"`       // Link to the origin of this Message
	URL          string                     `json:"url"          bson:"url,omitempty"`          // URL of this Message
	Label        string                     `json:"label"        bson:"label,omitempty"`        // Label of this Message
	Summary      string                     `json:"summary"      bson:"summary,omitempty"`      // Summary of this Message
	ImageURL     string                     `json:"imageUrl"     bson:"imageUrl,omitempty"`     // URL of the image associated with this Message
	AttributedTo sliceof.Object[PersonLink] `json:"attributedTo" bson:"attributedTo,omitempty"` // List of people who are attributed to this Message
	InReplyTo    string                     `json:"inReplyTo"    bson:"inReplyTo,omitempty"`    // URL this message is in reply to
	ContentHTML  string                     `json:"contentHtml"  bson:"contentHtml,omitempty"`  // HTML Content of the Message
	ContentJSON  string                     `json:"contentJson"  bson:"contentJson,omitempty"`  // Original JSON message, used for reprocessing later.
	Responses    ResponseSummary            `json:"responses"    bson:"responses,omitempty"`    // Summary counter of Responses to this Message
	MyResponse   string                     `json:"myResponse"   bson:"myResponse,omitempty"`   // If the owner of this message has responded, then this field contains the responseType (Like, Dislike, Repost)
	PublishDate  int64                      `json:"publishDate"  bson:"publishDate,omitempty"`  // Unix timestamp of the date/time when this Message was published
	Rank         int64                      `json:"rank"         bson:"rank"`                   // Sort rank for this message (publishDate * 1000 + sequence number)

	journal.Journal `json:"-" bson:",inline"`
}

// NewMessage returns a fully initialized Message record
func NewMessage() Message {
	return Message{
		MessageID: primitive.NewObjectID(),
		UserID:    primitive.NilObjectID,
		FolderID:  primitive.NilObjectID,
		Responses: NewResponseSummary(),
	}
}

func MessageFields() []string {
	return []string{"_id", "userId", "socialRole", "origin", "url", "label", "summary", "imageUrl", "contentHtml", "attributedTo", "folderId", "publishDate", "rank", "responses", "myResponse"}
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
	return []string{MagicRoleMyself}
}

/******************************************
 * Other Methods
 ******************************************/

func (message *Message) Author() PersonLink {
	return message.AttributedTo.First()
}

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

func (message *Message) SetAttributedTo(persons ...PersonLink) {
	message.AttributedTo = persons
}

func (message *Message) AddAttributedTo(persons ...PersonLink) {
	message.AttributedTo = append(message.AttributedTo, persons...)
}

func (message Message) SummaryText() string {
	return html.Summary(html.RemoveTags(message.Summary))
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
		*value = *value - 1
	}
}

func increment(value *int) {
	*value = *value + 1
}
