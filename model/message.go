package model

import (
	"net/url"

	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Message represents a single item in a User's inbox or outbox.  It is loosely modelled on the MessageStreams
// standard, and can be converted into a strict go-fed streams.Type object.
type Message struct {
	MessageID      primitive.ObjectID `json:"messageId"      bson:"_id"`                      // Unique ID of the Message
	UserID         primitive.ObjectID `json:"userId"         bson:"userId"`                   // Unique ID of the User who owns this Message (in their inbox or outbox)
	FolderID       primitive.ObjectID `json:"folderId"       bson:"folderId,omitempty"`       // Unique ID of the Folder where this Message is stored
	SocialRole     string             `json:"socialRole"     bson:"socialRole,omitempty"`     // Role this message plays in social integrations ("Article", "Note", etc)
	Origin         OriginLink         `json:"origin"         bson:"origin,omitempty"`         // Link to the origin of this Message
	Document       DocumentLink       `json:"document"       bson:"document,omitempty"`       // Document that is the subject of this Message
	ContentHTML    string             `json:"contentHtml"    bson:"contentHtml,omitempty"`    // HTML Content of the Message
	ContentJSON    string             `json:"contentJson"    bson:"contentJson,omitempty"`    // Original JSON message, used for reprocessing later.
	ResponseTotals mapof.Int          `json:"responseTotals" bson:"responseTotals,omitempty"` // Summary counter of Responses to this Message
	MyResponses    mapof.Bool         `json:"myResponses"    bson:"myResponses,omitempty"`    // Booleans flag how the current user has responded to this Message
	PublishDate    int64              `json:"publishDate"    bson:"publishDate,omitempty"`    // Unix timestamp of the date/time when this Message was published
	Rank           int64              `json:"rank"           bson:"rank"`                     // Sort rank for this message (publishDate * 1000 + sequence number)

	journal.Journal `json:"-" bson:"journal"`
}

// NewMessage returns a fully initialized Message record
func NewMessage() Message {
	return Message{
		MessageID:      primitive.NewObjectID(),
		UserID:         primitive.NilObjectID,
		FolderID:       primitive.NilObjectID,
		ResponseTotals: mapof.NewInt(),
		MyResponses:    mapof.NewBool(),
	}
}

func MessageFields() []string {
	return []string{"_id", "userId", "socialRole", "origin", "document", "contentHtml", "folderId", "publishDate", "rank", "responseTotals", "myResponses"}
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
	return message.Document.AttributedTo.First()
}

func (message *Message) SetAttributedTo(persons ...PersonLink) {
	message.Document.AttributedTo = persons
}

func (message *Message) AddAttributedTo(persons ...PersonLink) {
	message.Document.AttributedTo = append(message.Document.AttributedTo, persons...)
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
	message.Document = other.Document
	message.ContentHTML = other.ContentHTML
}

// IsInternal returns true if this message is "owned" by
// this server, and is not federated via another server.
func (message *Message) IsInternal() bool {
	return !message.Origin.InternalID.IsZero()
}

// URL returns the parsed, canonical URL for this Message (as stored in the document)
func (message *Message) URL() *url.URL {
	result, _ := url.Parse(message.Document.URL)
	return result
}
