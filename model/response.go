package model

import (
	"time"

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
	return []string{"responseId", "url", "objectId", "type", "content", "createDate"}
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
 * RoleStateEnumerator Methods
 ******************************************/

// State returns the current state of this Stream.  It is
// part of the implementation of the RoleStateEmulator interface
func (response Response) State() string {
	return ""
}

// Roles returns a list of all roles that match the provided authorization
func (response Response) Roles(authorization *Authorization) []string {

	// Everyone has "anonymous" access
	result := []string{MagicRoleAnonymous}

	if authorization.IsAuthenticated() {

		// Owners are hard-coded to do everything, so no other roles need to be returned.
		if authorization.DomainOwner {
			return []string{MagicRoleOwner}
		}

		result = append(result, MagicRoleAuthenticated)

		// Authors sometimes have special permissions, too.
		if response.UserID == authorization.UserID {
			result = append(result, MagicRoleAuthor)
		}
	}

	return result
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
