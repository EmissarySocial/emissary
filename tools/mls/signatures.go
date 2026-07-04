package mls

import (
	"crypto/ed25519"
	"fmt"
)

// ── Signature verification ───────────────────────────────────────────────────

// SignatureResult is the outcome of verifying an MLS signature.  It is a typed
// result rather than a bare string so that callers CANNOT accidentally treat an
// unverifiable signature as valid: the only way to learn a signature is good is
// to read Verified, which is true ONLY when the signature was actually checked
// and passed.  "Could not verify" (unsupported cipher suite) is a distinct,
// non-valid state — the verifier fails CLOSED.
type SignatureResult struct {
	// Verified is true if and only if the signature was cryptographically
	// checked and passed.  An unsupported cipher suite yields false.
	Verified bool

	// Supported is true if the cipher suite's signature scheme is implemented.
	// When false, Verified is always false and Detail explains why.
	Supported bool

	// Detail is a human-readable explanation, suitable for display.
	Detail string
}

// String returns the human-readable detail, so a SignatureResult can be printed
// or serialized the same way the old status string was.
func (r SignatureResult) String() string {
	return r.Detail
}

// verifySignature checks sig over SignContent(label, tbs) using the signature
// scheme implied by the cipher suite.  Only Ed25519 suites are implemented;
// any other suite fails CLOSED (Supported=false, Verified=false) rather than
// being silently trusted.
func verifySignature(cipherSuite uint16, pubKey []byte, label string, tbs, sig []byte) SignatureResult {
	switch cipherSuite {
	case 0x0001, 0x0003: // Ed25519 suites
		if len(pubKey) != ed25519.PublicKeySize {
			return SignatureResult{
				Supported: true,
				Detail:    fmt.Sprintf("INVALID (Ed25519 public key must be 32 bytes, got %d)", len(pubKey)),
			}
		}
		msg, err := signContent(label, tbs)
		if err != nil {
			return SignatureResult{
				Supported: true,
				Detail:    fmt.Sprintf("not verified (%v)", err),
			}
		}
		if ed25519.Verify(ed25519.PublicKey(pubKey), msg, sig) {
			return SignatureResult{
				Verified:  true,
				Supported: true,
				Detail:    "valid (Ed25519)",
			}
		}
		return SignatureResult{
			Supported: true,
			Detail:    "INVALID (Ed25519 verification failed)",
		}
	default:
		return SignatureResult{
			Detail: "not verified (signature scheme for this cipher suite not implemented)",
		}
	}
}
