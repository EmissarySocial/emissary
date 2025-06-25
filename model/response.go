package model

import (
	"time"

	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/benpate/data/journal"
	"github.com/benpate/hannibal"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/toot/object"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Reponse defines a single Actor's response to an Object.  The actor may be a local or remote user, and the
// Object may be a local stream or an inbox message.
type Response struct {
	ResponseID primitive.ObjectID `json:"responseId" bson:"_id"`               // Unique identifier for this Response
	UserID     primitive.ObjectID `json:"userId"     bson:"userId"`            // ID of the User who made this response
	Actor      string             `json:"actor"      bson:"actor"`             // ActivityPubURL of the User who made the response
	Object     string             `json:"object"     bson:"object"`            // ActivityPubURL of the Object that the actor responded to
	Type       string             `json:"type"       bson:"type"`              // Type of Response (e.g. "Announce", "Bookmark", "Like", "Dislike", etc...)
	Summary    string             `json:"summary"    bson:"summary,omitempty"` // Summary of the response (e.g. "I liked this post because...")
	Content    string             `json:"content"    bson:"content,omitempty"` // Custom value assigned to the response (emoji, vote, etc.)

	journal.Journal `json:"-" bson:",inline"`
}

// NewReponse returns a fully initialized Response object
func NewResponse() Response {
	return Response{
		ResponseID: primitive.NewObjectID(),
	}
}

/******************************************
 * data.Object Interface
 ******************************************/

// ID returns the unique identifier for this Response (in string format)
func (response Response) ID() string {
	return response.ResponseID.Hex()
}

func (response Response) Fields() []string {
	return []string{"responseId", "url", "object", "type", "content", "createDate"}
}

/******************************************
 * Other Data Methods
 ******************************************/

// GetJSONLD returns the JSON-LD representation of this Response
func (response Response) GetJSONLD() mapof.Any {

	result := mapof.Any{
		vocab.AtContext:         vocab.ContextTypeActivityStreams,
		vocab.PropertyID:        response.ActivityPubURL(),
		vocab.PropertyType:      response.Type,
		vocab.PropertyActor:     response.Actor,
		vocab.PropertyObject:    response.Object,
		vocab.PropertyPublished: response.ActivityPubCreateDate(),
	}

	if response.Summary != "" {
		result[vocab.PropertySummary] = response.Summary
	}

	if response.Content != "" {
		result[vocab.PropertyContent] = response.Content
	}

	return result
}

func (response Response) ActivityPubURL() string {

	switch response.Type {

	case vocab.ActivityTypeDislike:
		return response.Actor + "/pub/disliked/" + response.ResponseID.Hex()

	case vocab.ActivityTypeLike:
		return response.Actor + "/pub/liked/" + response.ResponseID.Hex()

	// Default: vocab.ActivityTypeAnnounce
	default:
		return response.Actor + "/pub/announced/" + response.ResponseID.Hex()
	}
}

// IsEqual returns TRUE if two responses match urls, actors, objects, types, and values
func (response Response) IsEqual(other Response) bool {
	return (response.Actor == other.Actor) &&
		(response.Object == other.Object) &&
		(response.Type == other.Type) &&
		(response.Content == other.Content)
}

func (response Response) ActivityPubCreateDate() string {
	return hannibal.TimeFormat(time.Unix(response.CreateDate, 0))
}

// CreateDateSeconds returns the CreateDate in Unix Epoch seconds (instead of milliseconds)
func (response Response) CreateDateSeconds() int64 {
	return response.CreateDate / 1000
}

// IsEmpty returns TRUE if this Response has no data in it.
func (response Response) IsEmpty() bool {
	return response.Type == ""
}

// NotEmpty returns TRUE if this Response has data in it.
func (response Response) NotEmpty() bool {
	return !response.IsEmpty()
}

/******************************************
 * AccessLister Interface
 ******************************************/

// State returns the current state of this Response.
// It is part of the AccessLister interface
func (response *Response) State() string {
	return "default"
}

// IsAuthor returns TRUE if the provided UserID the author of this Response
// It is part of the AccessLister interface
func (response *Response) IsAuthor(authorID primitive.ObjectID) bool {
	return false
}

// IsMyself returns TRUE if this object directly represents the provided UserID
// It is part of the AccessLister interface
func (response *Response) IsMyself(userID primitive.ObjectID) bool {
	return response.UserID == userID
}

// RolesToGroupIDs returns a slice of Group IDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (response *Response) RolesToGroupIDs(roleIDs ...string) id.Slice {
	return nil
}

// RolesToPrivilegeIDs returns a slice of Privileges that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (response *Response) RolesToPrivilegeIDs(roleIDs ...string) id.Slice {
	return nil
}

/******************************************
 * Mastodon API
 ******************************************/

func (response Response) Toot() object.Status {

	return object.Status{
		ID:  response.ActivityPubURL(),
		URI: response.ActivityPubURL(),
		Account: object.Account{
			ID: response.Actor,
		},
	}
}
