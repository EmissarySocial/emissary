package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/hannibal/datetime"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/toot/object"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/*
ARCHITECTURE NOTE — Response vs. CollectionItem (read alongside model/collectiontem.go)

A "Response" is an OUTBOUND reaction: a record that a LOCAL actor Liked / Disliked /
Announced some piece of content. The content may belong to the actor themselves, to
another actor on this server, or to a remote actor on another server. A Response exists
so that we can PUBLISH the activity to the actor's outbox and federate it outward. It is
the actor-side store-of-record (it also drives Mastodon-API compatibility).

A "CollectionItem" is the INBOUND mirror image: it records a Like / Dislike / Announce
that WE RECEIVED for OUR OWN content. The Likes/Dislikes/Shares collections on a Stream
are "everyone who reacted to this post," and are what the /pub/likes (etc.) endpoints
serve and what the denormalized Stream counts read.

The two are independent and both may exist for a single reaction:

  - When a LOCAL user reacts to REMOTE content: only a Response (nothing of ours was liked).
  - When a REMOTE user reacts to OUR content: only a CollectionItem (they own the Response).
  - When a LOCAL user reacts to their OWN (or another local user's) content: BOTH — a
    Response (to publish the activity) AND a CollectionItem (to link the reaction to the
    liked item). These are two different rows describing the same reaction from two sides.

KNOWN ROUGH EDGES (2026-07-11, being redesigned — do not treat the current code as the
intended shape): the OUTBOUND path (Response.Save -> projectResponse) writes the inbound
CollectionItem directly as a shortcut, keyed by the Response's own /pub/liked/<id> URL,
while the genuine INBOUND path keys the same item by the arriving activity's ID — so a
single reaction can be identified two different ways. The goal architecture is for a
Response to only persist+publish, and for a SINGLE inbound projection funnel to own every
CollectionItem (self-reactions included, via loopback), under one canonical key.
*/

// Response defines a single Actor's response to an Object.  The actor may be a local or remote user, and the
// Object may be a local stream or an inbox message.
type Response struct {
	ResponseID primitive.ObjectID `bson:"_id"`               // Unique identifier for this Response
	UserID     primitive.ObjectID `bson:"userId"`            // ID of the User who made this response
	Actor      string             `bson:"actor"`             // ActivityPubURL of the User who made the response
	Object     string             `bson:"object"`            // ActivityPubURL of the Object that the actor responded to
	Type       string             `bson:"type"`              // Type of Response (e.g. "Announce", "Bookmark", "Like", "Dislike", etc...)
	Summary    string             `bson:"summary,omitempty"` // Summary of the response (e.g. "I liked this post because...")
	Content    string             `bson:"content,omitempty"` // Custom value assigned to the response (emoji, vote, etc.)

	journal.Journal `json:"-" bson:",inline"`
}

// NewResponse returns a fully initialized Response object
func NewResponse() Response {
	return Response{
		ResponseID: primitive.NewObjectID(),
	}
}

// ConflictingResponseTypes returns every Response type that cannot coexist with the provided
// type on a single (User, Object).  The provided type is always included in the result, so that
// re-reacting replaces the previous Response instead of duplicating it.
func ConflictingResponseTypes(responseType string) []string {

	// RULE: A Like and a Dislike contradict each other, so setting either one clears the other.
	switch responseType {
	case vocab.ActivityTypeLike, vocab.ActivityTypeDislike:
		return []string{vocab.ActivityTypeLike, vocab.ActivityTypeDislike}
	}

	// Every other reaction (Announce, and anything added later) is independent of the rest,
	// so it conflicts with itself alone.  Sharing a post you also liked is not a contradiction.
	return []string{responseType}
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
		vocab.AtContext:      vocab.ContextTypeActivityStreams,
		vocab.PropertyID:     response.ActivityPubURL(),
		vocab.PropertyType:   response.Type,
		vocab.PropertyActor:  response.Actor,
		vocab.PropertyObject: response.Object,
	}

	// An unsaved Response has no CreateDate. Omit `published` rather than
	// claiming the activity was published at the Unix epoch.
	if published := response.ActivityPubCreateDate(); published != "" {
		result[vocab.PropertyPublished] = published
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
		return response.Actor + "/pub/shared/" + response.ResponseID.Hex()
	}
}

// IsEqual returns TRUE if two responses match urls, actors, objects, types, and values
func (response Response) IsEqual(other Response) bool {
	return (response.Actor == other.Actor) &&
		(response.Object == other.Object) &&
		(response.Type == other.Type) &&
		(response.Content == other.Content)
}

// ActivityPubCreateDate returns the CreateDate as an AS2 date-time, or an
// empty string if this Response has not been saved yet. CreateDate is stored
// in milliseconds, which FromUnixMilli expects directly.
func (response Response) ActivityPubCreateDate() string {
	return datetime.FromUnixMilli(response.CreateDate)
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
	return !userID.IsZero() && response.UserID == userID
}

// RolesToGroupIDs returns a slice of Group IDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (response *Response) RolesToGroupIDs(roleIDs ...string) Permissions {
	return defaultRolesToGroupIDs(response.UserID, roleIDs...)
}

// RolesToPrivilegeIDs returns a slice of Privileges that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (response *Response) RolesToPrivilegeIDs(roleIDs ...string) Permissions {
	return NewPermissions()
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
