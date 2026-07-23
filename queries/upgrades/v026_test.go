package upgrades

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/benpate/hannibal/sigs"
	"github.com/stretchr/testify/require"
)

// newRSAPrivatePEM generates an RSA private key of the given size and returns it in the same PEM
// form Version26 stores, so the tests exercise keyNeedsUpgrade against real encoded keys.
func newRSAPrivatePEM(t *testing.T, bits int) string {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, bits)
	require.NoError(t, err)

	return sigs.EncodePrivatePEM(key)
}

// newECDSAPrivatePEM returns a valid, decodable PRIVATE KEY that is NOT an RSA key, to exercise the
// non-RSA branch of keyNeedsUpgrade.
func newECDSAPrivatePEM(t *testing.T) string {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)

	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
}

/******************************************
 * keyNeedsUpgrade
 ******************************************/

// A key that already meets the target size is kept: keyNeedsUpgrade returns FALSE so the caller
// leaves the record untouched. This is the idempotency guarantee -- a second pass over an
// already-upgraded database regenerates nothing and preserves every account's identity.
func TestKeyNeedsUpgrade_TargetSizeKept(t *testing.T) {
	require.False(t, keyNeedsUpgrade(newRSAPrivatePEM(t, targetKeyBits)))
}

// A key smaller than the target (e.g. a legacy 512-bit/1024-bit key from the original Version3) is
// regenerated -- this is the migration's actual job.
func TestKeyNeedsUpgrade_SmallKeyRegenerated(t *testing.T) {
	require.True(t, keyNeedsUpgrade(newRSAPrivatePEM(t, 1024)))
}

// A missing key has no identity to preserve, so a new one is generated.
func TestKeyNeedsUpgrade_MissingKeyRegenerated(t *testing.T) {
	require.True(t, keyNeedsUpgrade(""))
}

// An undecodable key is unusable, so it is regenerated (per the caller's decision that a broken key
// is repaired, not preserved).
func TestKeyNeedsUpgrade_UndecodableKeyRegenerated(t *testing.T) {
	require.True(t, keyNeedsUpgrade("-----BEGIN RSA PRIVATE KEY-----\nnot a real key\n-----END RSA PRIVATE KEY-----"))
	require.True(t, keyNeedsUpgrade("total garbage, not even PEM"))
}

// A valid, decodable key that is not RSA cannot be size-checked and is not what this migration
// produces, so it is regenerated.
func TestKeyNeedsUpgrade_NonRSAKeyRegenerated(t *testing.T) {
	require.True(t, keyNeedsUpgrade(newECDSAPrivatePEM(t)))
}
