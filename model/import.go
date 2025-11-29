package model

import (
	"crypto/sha256"

	"github.com/EmissarySocial/emissary/tools/random"
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/convert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/oauth2"
)

type Import struct {
	ImportID       primitive.ObjectID `bson:"_id"`            // Unique identifier for this Import record
	UserID         primitive.ObjectID `bson:"userId"`         // User profile that we're importing INTO
	SourceID       string             `bson:"sourceId"`       // URL or Handle of the account being migrated
	StateID        string             `bson:"stateId"`        // Current state of this import process
	Message        string             `bson:"message"`        // Human-friendly description of the status of this import process
	OAuthConfig    oauth2.Config      `bson:"oauthConfig"`    // OAuth 2.0 configuration information
	OAuthToken     *oauth2.Token      `bson:"oauthToken"`     // OAuth token provided by the source server
	OAuthChallenge []byte             `bson:"oauthChallenge"` // OAuth challenge token used of PKCE
	TotalItems     int                `bson:"totalItems"`     // The total number of items to be imported (available after the import plan is made)
	CompleteItems  int                `bson:"completeItems"`  // The number of items that have completed

	journal.Journal `bson:",inline"`
}

func NewImport() Import {
	return Import{
		ImportID: primitive.NewObjectID(),
		StateID:  ImportStateNew,
	}
}

/******************************************
 * data.Object Interface
 ******************************************/

func (record Import) ID() string {
	return record.ImportID.Hex()
}

/******************************************
 * AccessLister Interface
 ******************************************/

// State returns the current state of this Following.
// It is part of the AccessLister interface
func (record Import) State() string {
	return "default"
}

// IsAuthor returns TRUE if the provided UserID the author of this Following
// It is part of the AccessLister interface
func (record Import) IsAuthor(authorID primitive.ObjectID) bool {
	return false
}

// IsMyself returns TRUE if this object directly represents the provided UserID
// It is part of the AccessLister interface
func (record Import) IsMyself(userID primitive.ObjectID) bool {
	return !userID.IsZero() && userID == record.UserID
}

// RolesToGroupIDs returns a slice of Group IDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (record Import) RolesToGroupIDs(roleIDs ...string) Permissions {
	return defaultRolesToGroupIDs(record.UserID, roleIDs...)
}

// RolesToPrivilegeIDs returns a slice of Privileges that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (record Import) RolesToPrivilegeIDs(roleIDs ...string) Permissions {
	return NewPermissions()
}

/******************************************
 * Other Methods
 ******************************************/

// OAuthCodeURL generates a new (unique) OAuth state and AuthCodeURL for the specified provider
func (record Import) OAuthCodeURL() string {
	codeChallengeBytes := sha256.Sum256(record.OAuthChallenge)
	codeChallenge := oauth2.SetAuthURLParam("code_challenge", random.Base64URLEncode(codeChallengeBytes[:]))
	codeChallengeMethod := oauth2.SetAuthURLParam("code_challenge_method", "S256")
	authCodeURL := record.OAuthConfig.AuthCodeURL(record.ImportID.Hex(), codeChallenge, codeChallengeMethod)

	return authCodeURL
}

func (record Import) PercentComplete() int {
	ratio := convert.Float(record.CompleteItems) / convert.Float(record.TotalItems) * 100
	return convert.Int(ratio)
}
