package model

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"

	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StreamSource is a subscription to content hosted outside of Emissary, which is copied into
// the Content of the Stream that it is attached to.
type StreamSource struct {
	StreamSourceID primitive.ObjectID `json:"streamSourceId" bson:"_id"`           // Unique identifier of this record
	StreamID       primitive.ObjectID `json:"streamId"       bson:"streamId"`      // Stream whose Content this record populates
	Method         string             `json:"method"         bson:"method"`        // Adapter that reads the remote source (HTTPS)
	URL            string             `json:"url"            bson:"url"`           // Address of the remote source
	Config         mapof.String       `json:"config"         bson:"config"`        // Record settings, including the webhook token that triggers a sync
	Version        string             `json:"version"        bson:"version"`       // ETag of the last content applied, sent as If-None-Match on the next check
	ContentHash    string             `json:"contentHash"    bson:"contentHash"`   // Hexadecimal hash of the last remote content applied to the Stream
	Status         string             `json:"status"         bson:"status"`        // Outcome of the last synchronization (NEW, LOADING, SUCCESS, FAILURE)
	StatusMessage  string             `json:"statusMessage"  bson:"statusMessage"` // Operator-readable reason for the current Status
	LastSynced     int64              `json:"lastSynced"     bson:"lastSynced"`    // Unix epoch SECONDS when the last synchronization began

	journal.Journal `json:"-" bson:",inline"`
}

// NewStreamSource returns a fully initialized StreamSource object
func NewStreamSource() StreamSource {

	config := mapof.NewString()
	config[StreamSourceConfigWebhookToken] = NewWebhookToken()

	return StreamSource{
		StreamSourceID: primitive.NewObjectID(),
		Config:         config,
		Status:         StreamSourceStatusNew,
	}
}

// NewWebhookToken returns a random token for the webhook URL that triggers a synchronization
func NewWebhookToken() string {

	nonce := make([]byte, webhookTokenNonceBytes)

	// UNREACHABLE ERROR: since Go 1.24, crypto/rand.Read never returns an error -- it panics if
	// the operating system's source fails -- so this cannot produce a short, guessable nonce.
	_, _ = rand.Read(nonce)

	// The hash only fixes the token's length and alphabet.  Every bit of the secret comes from
	// the nonce, because hashing a predictable value such as an ObjectID would hide nothing.
	sum := sha256.Sum256(nonce)

	return hex.EncodeToString(sum[:])
}

/******************************************
 * data.Object Interface
 ******************************************/

// ID returns the primary key of this object
func (streamSource StreamSource) ID() string {
	return streamSource.StreamSourceID.Hex()
}

/******************************************
 * AccessLister Interface
 ******************************************/

// State returns the current state of this StreamSource.
// It is part of the AccessLister interface
func (streamSource *StreamSource) State() string {
	return "default"
}

// IsAuthor returns TRUE if the provided UserID is the author of this StreamSource.
// It is part of the AccessLister interface
func (streamSource *StreamSource) IsAuthor(authorID primitive.ObjectID) bool {
	return false
}

// IsMyself returns TRUE if this object directly represents the provided UserID.
// It is part of the AccessLister interface
func (streamSource *StreamSource) IsMyself(userID primitive.ObjectID) bool {
	return false
}

// RolesToGroupIDs returns a slice of Group IDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (streamSource *StreamSource) RolesToGroupIDs(roleIDs ...string) Permissions {

	// RULE: A StreamSource grants no access of its own.  It has no owner -- it belongs to a Stream --
	// so "author" and "myself" name nobody here, and the Stream's own action is the real gate.
	return defaultRolesToGroupIDs(primitive.NilObjectID, roleIDs...)
}

// RolesToPrivilegeIDs returns a slice of Privileges that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (streamSource *StreamSource) RolesToPrivilegeIDs(roleIDs ...string) Permissions {
	return NewPermissions()
}

/******************************************
 * Getters
 ******************************************/

// WebhookToken returns the token that this record's webhook URL carries
func (streamSource StreamSource) WebhookToken() string {
	return streamSource.Config[StreamSourceConfigWebhookToken]
}
