package model

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/tools/random"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Refresh tokens (RFC 6749 §6) are the durable, revocable credential behind an
// OAuth grant. Access tokens (the JWT) are short-lived derivatives; the client
// trades a refresh token for a fresh access token at /oauth/token.
//
// A refresh token is a dotted string: "<grantID hex>.<generation>.<secret>".
// The grantID is the OAuthUserToken's ObjectID, so the token endpoint can load
// the grant by _id directly. The generation is a monotonic counter that enables
// rotation + reuse detection. Only the SHA-256 hash of the secret is ever stored
// (see OAuTHUserToken.RefreshHash), so a database read cannot leak a live token.

// refreshSecretBytes is the entropy (in bytes) of a refresh-token secret. 32
// bytes (256 bits) is well beyond guessing range.
const refreshSecretBytes = 32

// RefreshMatch classifies a presented refresh token against the stored grant.
// The token endpoint maps each outcome to an action (rotate, revoke, or reject).
type RefreshMatch int

const (
	// RefreshMatchNone means the secret matched nothing (garbage, wrong client,
	// or a mismatched secret). Reject with invalid_grant; NOT a reuse alarm.
	RefreshMatchNone RefreshMatch = iota

	// RefreshMatchCurrent means the secret matched the current generation. This
	// is normal use: rotate the grant forward and issue a new pair.
	RefreshMatchCurrent

	// RefreshMatchGrace means the secret matched the immediately-prior generation
	// within the grace window (a client that lost the response and retried).
	// Tolerated: rotate forward, do NOT alarm.
	RefreshMatchGrace

	// RefreshMatchReuse means a CONFIRMED superseded secret was presented: the
	// immediately-prior generation, matching the stored prior hash, past the grace
	// window. This signals theft: revoke the whole grant. Only a genuine prior
	// secret can trigger this — a guessed grant ID plus garbage never does, so it
	// cannot be abused to force-revoke another user's grant.
	RefreshMatchReuse
)

// NewRefreshSecret generates a fresh, high-entropy refresh-token secret. Only
// its hash is persisted; the plaintext is returned once, to the client.
func NewRefreshSecret() (string, error) {

	value, err := random.GenerateBytes(refreshSecretBytes)

	if err != nil {
		return "", derp.Wrap(err, "model.NewRefreshSecret", "Generating random bytes")
	}

	return random.Base64URLEncode(value), nil
}

// BuildRefreshToken assembles the client-facing refresh-token string from its
// parts: "<grantID hex>.<generation>.<secret>".
func BuildRefreshToken(grantID primitive.ObjectID, generation int, secret string) string {
	return grantID.Hex() + "." + strconv.Itoa(generation) + "." + secret
}

// ParseRefreshToken splits a client-supplied refresh-token string back into its
// parts. A malformed token (wrong shape, bad grantID, non-numeric generation)
// is an error, which the caller treats as invalid_grant.
func ParseRefreshToken(token string) (primitive.ObjectID, int, string, error) {

	const location = "model.ParseRefreshToken"

	// RULE: The secret is base64url (no "."), so exactly three dot-separated parts.
	parts := strings.SplitN(token, ".", 3)

	if len(parts) != 3 {
		return primitive.NilObjectID, 0, "", derp.BadRequest(location, "Malformed refresh_token")
	}

	// RULE: The first part must be a valid ObjectID (the grant's _id).
	grantID, err := primitive.ObjectIDFromHex(parts[0])

	if err != nil {
		return primitive.NilObjectID, 0, "", derp.BadRequest(location, "Malformed refresh_token grant ID")
	}

	// RULE: The second part must be a positive generation counter.
	generation, err := strconv.Atoi(parts[1])

	if err != nil || generation < 1 {
		return primitive.NilObjectID, 0, "", derp.BadRequest(location, "Malformed refresh_token generation")
	}

	// RULE: The secret must be present.
	if parts[2] == "" {
		return primitive.NilObjectID, 0, "", derp.BadRequest(location, "Missing refresh_token secret")
	}

	return grantID, generation, parts[2], nil
}

// hashRefreshSecret returns the hex-encoded SHA-256 of a refresh-token secret.
// The secret is high-entropy, so an unsalted hash is sufficient: there is no
// dictionary or rainbow-table risk, and a deterministic hash lets us compare
// without storing the plaintext.
func hashRefreshSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// refreshHashMatches reports whether secret hashes to the stored hash, using a
// constant-time comparison to avoid leaking the hash through timing. An empty
// stored hash never matches (a grant with no refresh secret yet).
func refreshHashMatches(storedHash string, secret string) bool {

	if storedHash == "" {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(storedHash), []byte(hashRefreshSecret(secret))) == 1
}

// InitRefresh binds the first refresh secret to this grant, at generation 1.
// Called once, when an authorization code is exchanged for the initial pair.
func (token *OAuthUserToken) InitRefresh(secret string, now time.Time) {
	token.Generation = 1
	token.RefreshHash = hashRefreshSecret(secret)
	token.RefreshPrevHash = ""
	token.RotatedAt = now.Unix()
}

// RotateRefresh advances the grant to a new refresh secret: the current hash
// becomes the prior hash (for the grace window), the new secret becomes current,
// and the generation counter increments. Called on every successful refresh.
func (token *OAuthUserToken) RotateRefresh(newSecret string, now time.Time) {
	token.RefreshPrevHash = token.RefreshHash
	token.RefreshHash = hashRefreshSecret(newSecret)
	token.Generation = token.Generation + 1
	token.RotatedAt = now.Unix()
}

// ClassifyRefresh matches a presented (generation, secret) against this grant
// and returns how the token endpoint should treat it (RFC 6749 §6 rotation with
// RFC 6819 reuse detection). `now` and `grace` drive the grace-window decision.
func (token *OAuthUserToken) ClassifyRefresh(generation int, secret string, now time.Time, grace time.Duration) RefreshMatch {

	// The current generation, matching its secret: normal use.
	if generation == token.Generation && refreshHashMatches(token.RefreshHash, secret) {
		return RefreshMatchCurrent
	}

	// The immediately-prior generation, matching its stored secret: either a
	// tolerated retry (within the grace window) or a reuse alarm (past it). This
	// is the ONLY reuse-alarm path — it requires a confirmed prior secret, so a
	// guessed grant ID plus a bogus secret can never force a revocation.
	if generation == token.Generation-1 && refreshHashMatches(token.RefreshPrevHash, secret) {

		if rotatedAt := time.Unix(token.RotatedAt, 0); now.Sub(rotatedAt) <= grace {
			return RefreshMatchGrace
		}

		return RefreshMatchReuse
	}

	// Everything else — an unconfirmed secret, or a generation we no longer keep a
	// hash for — is rejected as invalid_grant WITHOUT alarming. We only store the
	// current and immediately-prior hashes, so a deeper-history secret cannot be
	// confirmed as genuine, and we must not revoke on an unverifiable claim.
	return RefreshMatchNone
}
