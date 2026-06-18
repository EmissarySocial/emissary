package mls

import (
	"crypto/ed25519"
	"fmt"
)

// ── Signature verification ───────────────────────────────────────────────────

// verifySignature checks sig over SignContent(label, tbs) using the signature
// scheme implied by the cipher suite. Only Ed25519 suites are implemented;
// others are reported as unverified rather than silently trusted.
func verifySignature(cipherSuite uint16, pubKey []byte, label string, tbs, sig []byte) string {
	switch cipherSuite {
	case 0x0001, 0x0003: // Ed25519 suites
		if len(pubKey) != ed25519.PublicKeySize {
			return fmt.Sprintf("INVALID (Ed25519 public key must be 32 bytes, got %d)", len(pubKey))
		}
		msg, err := signContent(label, tbs)
		if err != nil {
			return fmt.Sprintf("not verified (%v)", err)
		}
		if ed25519.Verify(ed25519.PublicKey(pubKey), msg, sig) {
			return "valid (Ed25519)"
		}
		return "INVALID (Ed25519 verification failed)"
	default:
		return "not verified (signature scheme for this cipher suite not implemented)"
	}
}
