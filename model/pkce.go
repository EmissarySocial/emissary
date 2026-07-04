package model

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"

	"github.com/benpate/derp"
)

// OAuthUserToken.Data keys used to bind a PKCE challenge to an issued code.
const (
	pkceDataChallenge = "code_challenge"
	pkceDataMethod    = "code_challenge_method"
)

// PKCE (Proof Key for Code Exchange, RFC 7636) support for the OAuth
// authorization-code flow.  PKCE is optional here: a client that omits
// code_challenge behaves exactly as before.  When a client DOES send a
// code_challenge, a matching code_verifier becomes REQUIRED at token exchange.

// PKCE challenge methods (RFC 7636 §4.3).
const (
	PKCEMethodPlain = "plain"
	PKCEMethodS256  = "S256"
)

// PKCE verifier length bounds (RFC 7636 §4.1).
const (
	pkceVerifierMinLength = 43
	pkceVerifierMaxLength = 128
)

// IsValidPKCEMethod returns TRUE if the method is one of the two RFC 7636
// challenge-transformation methods this server supports.
func IsValidPKCEMethod(method string) bool {
	return method == PKCEMethodPlain || method == PKCEMethodS256
}

// isValidPKCEVerifier reports whether verifier satisfies the RFC 7636 §4.1
// syntax: 43-128 characters drawn from the unreserved set
// [A-Z] / [a-z] / [0-9] / "-" / "." / "_" / "~".
func isValidPKCEVerifier(verifier string) bool {

	if len(verifier) < pkceVerifierMinLength || len(verifier) > pkceVerifierMaxLength {
		return false
	}

	for _, r := range verifier {
		switch {
		case r >= 'A' && r <= 'Z':
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '.' || r == '_' || r == '~':
		default:
			return false
		}
	}

	return true
}

// pkceChallengeMatches reports whether verifier produces challenge under the
// given method (RFC 7636 §4.6).  The comparison is constant-time to avoid
// leaking the secret through timing.  An unsupported method never matches.
func pkceChallengeMatches(method string, verifier string, challenge string) bool {

	var computed string

	switch method {

	case PKCEMethodPlain:
		computed = verifier

	case PKCEMethodS256:
		sum := sha256.Sum256([]byte(verifier))
		computed = base64.RawURLEncoding.EncodeToString(sum[:])

	default:
		return false
	}

	return subtle.ConstantTimeCompare([]byte(computed), []byte(challenge)) == 1
}

// SetPKCEChallenge binds a PKCE code_challenge (and its method) to this token,
// so the code can only be redeemed by presenting the matching code_verifier.
// A blank challenge is a no-op (the token is not PKCE-protected).
func (token *OAuthUserToken) SetPKCEChallenge(challenge string, method string) {

	if challenge == "" {
		return
	}

	if method == "" {
		method = PKCEMethodPlain
	}

	token.Data[pkceDataChallenge] = challenge
	token.Data[pkceDataMethod] = method
}

// VerifyPKCE enforces the PKCE binding (RFC 7636) at code-exchange time.
//
//   - If this token carries NO stored challenge, PKCE does not apply and any
//     verifier (including none) is accepted — preserving pre-PKCE behavior.
//   - If this token DOES carry a challenge, a syntactically valid verifier that
//     transforms to that challenge is REQUIRED; anything else is rejected.
func (token *OAuthUserToken) VerifyPKCE(verifier string) error {

	const location = "model.OAuthUserToken.VerifyPKCE"

	challenge := token.Data.GetString(pkceDataChallenge)

	// No stored challenge: this token is not PKCE-protected.
	if challenge == "" {
		return nil
	}

	// A challenge is bound, so a verifier is mandatory and must be well-formed.
	if !isValidPKCEVerifier(verifier) {
		return derp.BadRequest(location, "Invalid or missing code_verifier (RFC 7636 requires 43-128 unreserved characters)")
	}

	method := token.Data.GetString(pkceDataMethod)
	if method == "" {
		method = PKCEMethodPlain
	}

	if !pkceChallengeMatches(method, verifier, challenge) {
		return derp.BadRequest(location, "code_verifier does not match code_challenge")
	}

	return nil
}
