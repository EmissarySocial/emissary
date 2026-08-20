package model

import (
	"strings"
	"time"

	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/toot/object"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserOAuthToken represents an application-specific token that
// a remote API can use to access a user's account on their behalf
type OAuthUserToken struct {
	OAuthUserTokenID primitive.ObjectID `json:"I" bson:"_id"`             // Unique identifier for this OAuthUserToken (also the authorization code)
	ClientID         primitive.ObjectID `json:"C" bson:"clientId"`        // Unique identifier of the OAuthClient that created this token
	UserID           primitive.ObjectID `json:"U" bson:"userId"`          // Unique identifier of the User that authorized this token
	APIUser          bool               `json:"A" bson:"apiUser"`         // TRUE if this token represents an API user (as opposed to a human user)
	Scopes           sliceof.String     `json:"S" bson:"scopes"`          // The OAuth2 scopes that were authorized for this token
	RefreshHash      string             `json:"-" bson:"refreshHash"`     // SHA-256 of the current refresh-token secret (empty until the code is exchanged)
	RefreshPrevHash  string             `json:"-" bson:"refreshPrevHash"` // SHA-256 of the immediately-prior refresh secret, for the grace window (RFC 6819 reuse detection)
	Generation       int                `json:"-" bson:"generation"`      // Monotonic refresh-token generation counter (1 on first issuance)
	RotatedAt        int64              `json:"-" bson:"rotatedAt"`       // Unix seconds of the last refresh rotation, for the grace window
	CodeRedeemed     bool               `json:"-" bson:"codeRedeemed"`    // TRUE once the authorization code has been exchanged (single-use)
	Data             mapof.Any          `json:"D" bson:"data"`            // Additional data associated with this token

	// Token (the access-token JWT) and RefreshToken (the "grantID.gen.secret"
	// string) are transient, derived values — minted at issuance/refresh and
	// returned to the client once, but NEVER persisted. The grant is keyed by
	// RefreshHash; a stateless JWT and a bearer secret do not belong in the
	// database.
	Token        string `json:"-" bson:"-"`
	RefreshToken string `json:"-" bson:"-"`

	journal.Journal `json:"-" bson:",inline"`
}

// NewOAuthUserToken returns a fully initialized OAuthUserToken
func NewOAuthUserToken() OAuthUserToken {
	return OAuthUserToken{
		OAuthUserTokenID: primitive.NewObjectID(),
		Scopes:           sliceof.NewString(),
		Data:             mapof.NewAny(),
	}
}

/******************************************
 * data.Object Interface
 ******************************************/

// ID returns the primary key of this OAuthUserToken, as a string
func (token OAuthUserToken) ID() string {
	return token.OAuthUserTokenID.Hex()
}

/******************************************
 * AccessLister Interface
 ******************************************/

// State returns the current state of this Stream.
// It is part of the AccessLister interface
func (token OAuthUserToken) State() string {
	return "default"
}

// IsAuthor returns TRUE if the provided UserID the author of this Stream
// It is part of the AccessLister interface
func (token OAuthUserToken) IsAuthor(_ primitive.ObjectID) bool {
	return false
}

// IsMyself returns TRUE if this object directly represents the provided UserID
// It is part of the AccessLister interface
func (token OAuthUserToken) IsMyself(userID primitive.ObjectID) bool {
	return userID == token.UserID
}

// RolesToGroupIDs returns a slice of Group IDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (token OAuthUserToken) RolesToGroupIDs(roles ...string) Permissions {
	return defaultRolesToGroupIDs(token.UserID, roles...)
}

// RolesToPrivilegeIDs returns a slice of Privileges that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (token OAuthUserToken) RolesToPrivilegeIDs(roleIDs ...string) Permissions {
	return NewPermissions()
}

/******************************************
 * Other Methods
 ******************************************/

// Code returns the OAuth2 code that is used to request an access token.
// This is just the string version of the ID.
func (token OAuthUserToken) Code() string {
	return token.OAuthUserTokenID.Hex()
}

// JSONResponse returns the token as a map suitable for JSON API responses
// (RFC 6749 §5.1). It includes the access token, its lifetime (expires_in), and
// the rotating refresh token the client uses to obtain the next access token.
func (token OAuthUserToken) JSONResponse() map[string]any {

	return map[string]any{
		"access_token":  token.Token,
		"token_type":    "Bearer",
		"scope":         strings.Join(token.Scopes, " "),
		"expires_in":    int(OAuthAccessTokenLifetime.Seconds()),
		"refresh_token": token.RefreshToken,
		"created_at":    time.Now().Unix(),
	}
}

// IsCodeExpired reports whether this record's authorization code is too old to
// exchange (RFC 6749 §4.1.2 — codes are short-lived). The Journal stores the
// creation time in Unix milliseconds.
func (token OAuthUserToken) IsCodeExpired(now time.Time) bool {
	createdAt := time.UnixMilli(token.Created())
	return now.Sub(createdAt) > OAuthCodeLifetime
}

// Toot returns the token as a Toot ActivityPub object.Token. When a refresh token
// has been issued (RefreshToken is set), it is included along with the access
// token's lifetime so the client can renew before expiry.
func (token OAuthUserToken) Toot() object.Token {

	result := object.Token{
		AccessToken: token.Token,
		TokenType:   "Bearer",
		Scope:       strings.Join(token.Scopes, " "),
		CreatedAt:   time.Now().Unix(),
	}

	// Only advertise expiry + refresh when a refresh token was actually issued.
	if token.RefreshToken != "" {
		result.ExpiresIn = int64(OAuthAccessTokenLifetime.Seconds())
		result.RefreshToken = token.RefreshToken
	}

	return result
}
