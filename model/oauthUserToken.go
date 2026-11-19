package model

import (
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/benpate/data/journal"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/toot/object"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OAuthUserToken represents an application-specific token that
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
	MastodonApp      bool               `json:"-" bson:"mastodonApp"`     // TRUE if this grant was issued to a Mastodon app's Client (copied from OAuthClient.MastodonApp)
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

// SetRedirectURI records the redirect_uri that this grant's authorization code was issued to.
// A blank value records nothing.
func (token *OAuthUserToken) SetRedirectURI(redirectURI string) {

	if redirectURI == "" {
		return
	}

	// A grant loaded from a record with no data has a nil map
	if token.Data == nil {
		token.Data = mapof.NewAny()
	}

	token.Data[oauthUserTokenDataRedirectURI] = redirectURI
}

// VerifyRedirectURI confirms that a token request names the redirect_uri this grant's code was
// issued to (RFC 6749 §4.1.3, IndieAuth §5.3.1)
func (token OAuthUserToken) VerifyRedirectURI(client OAuthClient, redirectURI string) error {

	const location = "model.OAuthUserToken.VerifyRedirectURI"

	issuedTo := token.RedirectURI()

	// A grant issued before the redirect_uri was recorded has nothing to compare
	if issuedTo == "" {
		return nil
	}

	// A request that names a redirect_uri must name the one the code was issued to
	if redirectURI != "" {

		if redirectURI != issuedTo {
			return derp.BadRequest(location, "redirect_uri does not match the authorization request")
		}

		return nil
	}

	// RULE: A URL client must name its redirect_uri, because IndieAuth requires it (§5.3.1)
	if client.ClientURL != "" {
		return derp.BadRequest(location, "redirect_uri is required")
	}

	// A registered client may leave out a redirect_uri that it left out of the authorization request
	return nil
}

// RedirectURI returns the redirect_uri that this grant's authorization code was issued to, or an
// empty string for a grant issued before it was recorded
func (token OAuthUserToken) RedirectURI() string {
	return token.Data.GetString(oauthUserTokenDataRedirectURI)
}

// IsRedeemableFor returns TRUE if redeeming this grant answers with something: its tokens, or the
// identity of a User the domain vouches for
func (token OAuthUserToken) IsRedeemableFor(user User) bool {

	if token.IssuesAccessToken() {
		return true
	}

	return user.IndieAuthMe() != ""
}

// TokenResponse returns the token endpoint's answer to a redeemed code: the tokens when the grant
// issues them, and the User's identity when the domain vouches for them (IndieAuth §5.3)
func (token OAuthUserToken) TokenResponse(user User) map[string]any {

	result := token.IdentityResponse(user)

	if token.IssuesAccessToken() {
		maps.Copy(result, token.JSONResponse())
	}

	return result
}

// IdentityResponse returns `me`, and the profile this grant's scopes release (IndieAuth §5.3.4),
// or an empty map when the domain does not vouch for the User
func (token OAuthUserToken) IdentityResponse(user User) map[string]any {

	result := map[string]any{}
	me := user.IndieAuthMe()

	// RULE: Nothing is said about a User the domain does not vouch for
	if me == "" {
		return result
	}

	result["me"] = me

	// RULE: The profile is released only by the profile scope
	if !token.HasScope(OAuthUserTokenScopeProfile) {
		return result
	}

	profile := map[string]any{"url": me}

	if user.DisplayName != "" {
		profile["name"] = user.DisplayName
	}

	if photo := user.ActivityPubIconURL(); photo != "" {
		profile["photo"] = photo
	}

	// RULE: The email address is released only by the email scope, inside the profile
	if token.HasScope(OAuthUserTokenScopeEmail) {
		if user.EmailAddress != "" {
			profile["email"] = user.EmailAddress
		}
	}

	result["profile"] = profile
	return result
}

// IssuesAccessToken returns TRUE if this grant carries a scope.  A grant with none only confirms
// who the User is, so it gets no access token (IndieAuth §5.3.3).
func (token OAuthUserToken) IssuesAccessToken() bool {

	for _, scope := range token.Scopes {
		if scope != "" {
			return true
		}
	}

	return false
}

// HasScope returns TRUE if the named scope was granted
func (token OAuthUserToken) HasScope(scope string) bool {
	return slices.Contains(token.Scopes, scope)
}

// JSONResponse returns the token as an RFC 6749 §5.1 response: the access token, its lifetime
// (expires_in, when it has one), and the rotating refresh token
func (token OAuthUserToken) JSONResponse() map[string]any {

	result := map[string]any{
		"access_token":  token.Token,
		"token_type":    "Bearer",
		"scope":         strings.Join(token.Scopes, " "),
		"refresh_token": token.RefreshToken,
		"created_at":    time.Now().Unix(),
	}

	// A token with no expiry advertises none, as Mastodon does
	if lifetime := token.AccessTokenLifetime(); lifetime > 0 {
		result["expires_in"] = int(lifetime.Seconds())
	}

	return result
}

// AccessTokenLifetime returns how long this grant's access tokens are valid, where zero means they
// do not expire
func (token OAuthUserToken) AccessTokenLifetime() time.Duration {

	if token.MastodonApp {
		return OAuthMastodonAppTokenLifetime
	}

	return OAuthAccessTokenLifetime
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

	// Only advertise expiry + refresh when a refresh token was actually issued, and expiry only
	// when the access token has one.
	if token.RefreshToken != "" {
		result.ExpiresIn = int64(token.AccessTokenLifetime().Seconds())
		result.RefreshToken = token.RefreshToken
	}

	return result
}
