package model

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// s256 is the RFC 7636 §4.2 transformation, used to build known-good challenge
// pairs for the tests.
func s256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// a valid 43-char verifier (the RFC minimum) drawn from the unreserved set.
const validVerifier = "0123456789012345678901234567890123456789012" // 43 chars

// TestIsValidPKCEVerifier verifies the length and character rules that RFC 7636 places on a verifier
func TestIsValidPKCEVerifier(t *testing.T) {

	valid := func(name, verifier string) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			require.True(t, isValidPKCEVerifier(verifier), "verifier %q should be valid", verifier)
		})
	}

	invalid := func(name, verifier string) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			require.False(t, isValidPKCEVerifier(verifier), "verifier %q should be invalid", verifier)
		})
	}

	valid("exactly 43 chars", strings.Repeat("a", 43))
	valid("exactly 128 chars", strings.Repeat("a", 128))
	valid("all unreserved punctuation", strings.Repeat("-._~", 11)) // 44 chars
	valid("mixed case and digits", "abcDEF012-._~"+strings.Repeat("x", 31))

	invalid("empty", "")
	invalid("one below minimum (42)", strings.Repeat("a", 42))
	invalid("one above maximum (129)", strings.Repeat("a", 129))
	invalid("contains space", strings.Repeat("a", 42)+" ")
	invalid("contains slash", strings.Repeat("a", 42)+"/")
	invalid("contains plus", strings.Repeat("a", 42)+"+")
	invalid("contains equals", strings.Repeat("a", 42)+"=")
}

// TestPKCEChallengeMatches verifies challenge checking for both the "plain" and "S256" methods
func TestPKCEChallengeMatches(t *testing.T) {

	match := func(name, method, verifier, challenge string) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			require.True(t, pkceChallengeMatches(method, verifier, challenge),
				"method=%q verifier=%q should match challenge=%q", method, verifier, challenge)
		})
	}

	noMatch := func(name, method, verifier, challenge string) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			require.False(t, pkceChallengeMatches(method, verifier, challenge),
				"method=%q verifier=%q should NOT match challenge=%q", method, verifier, challenge)
		})
	}

	match("plain identity", PKCEMethodPlain, validVerifier, validVerifier)
	match("s256 known pair", PKCEMethodS256, validVerifier, s256(validVerifier))

	noMatch("plain wrong challenge", PKCEMethodPlain, validVerifier, "something-else")
	noMatch("s256 wrong verifier", PKCEMethodS256, "wrong-verifier", s256(validVerifier))
	noMatch("s256 verifier used as plain", PKCEMethodS256, validVerifier, validVerifier)
	noMatch("unsupported method never matches", "S512", validVerifier, validVerifier)
	noMatch("empty method never matches", "", validVerifier, validVerifier)
}

// TestOAuthUserToken_VerifyPKCE verifies that a stored challenge is only redeemed by its own verifier
func TestOAuthUserToken_VerifyPKCE(t *testing.T) {

	// tokenWith builds a token carrying the given stored challenge/method.
	tokenWith := func(challenge, method string) OAuthUserToken {
		token := NewOAuthUserToken()
		if challenge != "" {
			token.Data[pkceDataChallenge] = challenge
		}
		if method != "" {
			token.Data[pkceDataMethod] = method
		}
		return token
	}

	accepts := func(name string, token OAuthUserToken, verifier string) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			require.Nil(t, token.VerifyPKCE(verifier), "VerifyPKCE(%q) should accept", verifier)
		})
	}

	rejects := func(name string, token OAuthUserToken, verifier string) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			require.NotNil(t, token.VerifyPKCE(verifier), "VerifyPKCE(%q) should reject", verifier)
		})
	}

	// --- No stored challenge: PKCE does not apply, anything is accepted ---
	accepts("no challenge, no verifier", tokenWith("", ""), "")
	accepts("no challenge, stray verifier ignored", tokenWith("", ""), validVerifier)

	// --- S256 challenge stored ---
	accepts("s256 correct verifier", tokenWith(s256(validVerifier), PKCEMethodS256), validVerifier)
	rejects("s256 empty verifier (the attack vector)", tokenWith(s256(validVerifier), PKCEMethodS256), "")
	rejects("s256 wrong verifier", tokenWith(s256(validVerifier), PKCEMethodS256), strings.Repeat("z", 43))
	rejects("s256 malformed verifier", tokenWith(s256(validVerifier), PKCEMethodS256), "too-short")

	// --- plain challenge stored ---
	accepts("plain correct verifier", tokenWith(validVerifier, PKCEMethodPlain), validVerifier)
	rejects("plain empty verifier", tokenWith(validVerifier, PKCEMethodPlain), "")
	rejects("plain wrong verifier", tokenWith(validVerifier, PKCEMethodPlain), strings.Repeat("q", 43))

	// --- challenge stored but method omitted: defaults to plain (RFC §4.3) ---
	accepts("defaulted-plain correct verifier", tokenWith(validVerifier, ""), validVerifier)
	rejects("defaulted-plain wrong verifier", tokenWith(validVerifier, ""), strings.Repeat("w", 43))
}

// TestOAuthUserToken_SetPKCEChallenge verifies which challenge and method values are accepted onto a token
func TestOAuthUserToken_SetPKCEChallenge(t *testing.T) {

	t.Run("blank challenge is a no-op", func(t *testing.T) {
		token := NewOAuthUserToken()
		token.SetPKCEChallenge("", PKCEMethodS256)
		require.Empty(t, token.Data.GetString(pkceDataChallenge))
		require.Empty(t, token.Data.GetString(pkceDataMethod))
	})

	t.Run("stores challenge and method", func(t *testing.T) {
		token := NewOAuthUserToken()
		token.SetPKCEChallenge("abc", PKCEMethodS256)
		require.Equal(t, "abc", token.Data.GetString(pkceDataChallenge))
		require.Equal(t, PKCEMethodS256, token.Data.GetString(pkceDataMethod))
	})

	t.Run("empty method defaults to plain", func(t *testing.T) {
		token := NewOAuthUserToken()
		token.SetPKCEChallenge("abc", "")
		require.Equal(t, PKCEMethodPlain, token.Data.GetString(pkceDataMethod))
	})

	t.Run("round-trips through VerifyPKCE (s256)", func(t *testing.T) {
		token := NewOAuthUserToken()
		token.SetPKCEChallenge(s256(validVerifier), PKCEMethodS256)
		require.Nil(t, token.VerifyPKCE(validVerifier))
		require.NotNil(t, token.VerifyPKCE("wrong-verifier"))
	})
}

// TestOAuthUserToken_HasPKCEChallenge verifies when a token counts as carrying a PKCE challenge
func TestOAuthUserToken_HasPKCEChallenge(t *testing.T) {

	t.Run("no bound challenge", func(t *testing.T) {
		token := NewOAuthUserToken()
		require.False(t, token.HasPKCEChallenge())
	})

	t.Run("challenge bound via SetPKCEChallenge", func(t *testing.T) {
		token := NewOAuthUserToken()
		token.SetPKCEChallenge(s256(validVerifier), PKCEMethodS256)
		require.True(t, token.HasPKCEChallenge())
	})
}

// TestOAuthAuthorizationRequest_Validate_PublicClientRequiresPKCE covers the
// authorize-time gate: a public client (no client_secret) must use PKCE for the
// authorization-code grant, while confidential clients and the implicit "token"
// grant are exempt.
func TestOAuthAuthorizationRequest_Validate_PublicClientRequiresPKCE(t *testing.T) {

	// newClient builds a minimally-valid client with the given secret so that
	// Validate reaches the PKCE rule (it needs a redirect_uri to get that far).
	newClient := func(secret string) OAuthClient {
		client := NewOAuthClient()
		client.ClientSecret = secret
		client.RedirectURIs = []string{"https://example.com/callback"}
		return client
	}

	t.Run("public client, code grant, no challenge is rejected", func(t *testing.T) {
		req := OAuthAuthorizationRequest{ResponseType: "code"}
		require.NotNil(t, req.Validate(newClient("")))
	})

	t.Run("public client, code grant, with challenge is accepted", func(t *testing.T) {
		req := OAuthAuthorizationRequest{ResponseType: "code", CodeChallenge: s256(validVerifier), CodeChallengeMethod: PKCEMethodS256}
		require.Nil(t, req.Validate(newClient("")))
	})

	t.Run("confidential client, code grant, no challenge is accepted", func(t *testing.T) {
		req := OAuthAuthorizationRequest{ResponseType: "code"}
		require.Nil(t, req.Validate(newClient("topsecret")))
	})

	t.Run("public client, implicit token grant is exempt", func(t *testing.T) {
		req := OAuthAuthorizationRequest{ResponseType: "token"}
		require.Nil(t, req.Validate(newClient("")))
	})
}

// TestOAuthAuthorizationRequest_validatePKCE verifies which PKCE parameters an authorization request accepts
func TestOAuthAuthorizationRequest_validatePKCE(t *testing.T) {

	t.Run("no challenge is fine", func(t *testing.T) {
		req := OAuthAuthorizationRequest{}
		require.Nil(t, req.validatePKCE())
		require.Equal(t, "", req.CodeChallengeMethod)
	})

	t.Run("challenge without method defaults to plain", func(t *testing.T) {
		req := OAuthAuthorizationRequest{CodeChallenge: "abc"}
		require.Nil(t, req.validatePKCE())
		require.Equal(t, PKCEMethodPlain, req.CodeChallengeMethod)
	})

	t.Run("explicit S256 is preserved", func(t *testing.T) {
		req := OAuthAuthorizationRequest{CodeChallenge: "abc", CodeChallengeMethod: PKCEMethodS256}
		require.Nil(t, req.validatePKCE())
		require.Equal(t, PKCEMethodS256, req.CodeChallengeMethod)
	})

	t.Run("unsupported method is rejected", func(t *testing.T) {
		req := OAuthAuthorizationRequest{CodeChallenge: "abc", CodeChallengeMethod: "S512"}
		require.NotNil(t, req.validatePKCE())
	})

	t.Run("method without challenge is ignored (not enforced)", func(t *testing.T) {
		req := OAuthAuthorizationRequest{CodeChallengeMethod: "bogus"}
		require.Nil(t, req.validatePKCE())
	})
}
