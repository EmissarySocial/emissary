package service

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/sigs"
	"github.com/stretchr/testify/require"
)

// newTestPrivateKeyPEM returns a fresh RSA private key and its PEM encoding
func newTestPrivateKeyPEM(t *testing.T) (*rsa.PrivateKey, string) {

	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, encryptionKeyBits)
	require.NoError(t, err)

	return privateKey, sigs.EncodePrivatePEM(privateKey)
}

// TestDomain_PrivateKey pins that the key is generated once, stored on the Domain record, and then
// read from the cache, and that a key already stored is never overwritten (BUG-41's shape).
func TestDomain_PrivateKey(t *testing.T) {

	t.Run("FirstUseGeneratesAndStores", func(t *testing.T) {

		service, session := newTestDomainService(t, "example.com")
		service.publish(storeTestDomain(t, session, "example.com"))

		privateKey, err := service.PrivateKey(session)
		require.NoError(t, err)
		require.NotNil(t, privateKey)

		// The key reaches the database and the cache
		writableDomain := loadStoredDomain(t, session)
		require.Equal(t, sigs.EncodePrivatePEM(privateKey), writableDomain.PrivateKey)
		require.Equal(t, writableDomain.PrivateKey, service.Cached().PrivateKey)

		// The second call is served from the cache, with no further write
		again, err := service.PrivateKey(session)
		require.NoError(t, err)
		require.True(t, privateKey.Equal(again))
		require.Equal(t, writableDomain.Revision, loadStoredDomain(t, session).Revision)
	})

	t.Run("StoredKeyWinsOverAStaleCache", func(t *testing.T) {

		service, session := newTestDomainService(t, "example.com")

		// Another node generated and stored a key that this node's cache has not seen
		expected, pem := newTestPrivateKeyPEM(t)
		stored := storeTestDomain(t, session, "example.com")
		stored.PrivateKey = pem
		require.NoError(t, session.Collection("Domain").Save(&stored, "Key"))

		stale := model.NewWritableDomain()
		stale.Hostname = "example.com"
		service.publish(stale)

		privateKey, err := service.PrivateKey(session)
		require.NoError(t, err)
		require.True(t, expected.Equal(privateKey))
		require.Equal(t, stored.Revision, loadStoredDomain(t, session).Revision)
	})

	t.Run("CorruptKeyIsReplaced", func(t *testing.T) {

		service, session := newTestDomainService(t, "example.com")

		writableDomain := storeTestDomain(t, session, "example.com")
		writableDomain.PrivateKey = "not a PEM"
		require.NoError(t, session.Collection("Domain").Save(&writableDomain, "Corrupt"))
		service.publish(writableDomain)

		privateKey, err := service.PrivateKey(session)
		require.NoError(t, err)
		require.Equal(t, sigs.EncodePrivatePEM(privateKey), loadStoredDomain(t, session).PrivateKey)
	})

	t.Run("LoadFails", func(t *testing.T) {

		service := NewDomain()

		_, err := service.PrivateKey(failingSession{})
		require.Error(t, err)
		require.Empty(t, service.Cached().PrivateKey)
	})

	t.Run("SaveFails", func(t *testing.T) {

		service, session := newTestDomainService(t, "example.com")
		storeTestDomain(t, session, "example.com")
		storeInvalidDomain(t, session)
		readOnlyDomain := service.Cached()

		_, err := service.PrivateKey(session)
		require.Error(t, err)
		require.Same(t, readOnlyDomain, service.Cached())
	})
}

// TestDecodeDomainPrivateKey pins what counts as a usable stored key
func TestDecodeDomainPrivateKey(t *testing.T) {

	t.Run("Empty", func(t *testing.T) {
		_, ok := decodeDomainPrivateKey("")
		require.False(t, ok)
	})

	t.Run("Garbage", func(t *testing.T) {
		_, ok := decodeDomainPrivateKey("not a PEM")
		require.False(t, ok)
	})

	t.Run("RSA", func(t *testing.T) {
		expected, pem := newTestPrivateKeyPEM(t)
		privateKey, ok := decodeDomainPrivateKey(pem)
		require.True(t, ok)
		require.True(t, expected.Equal(privateKey))
	})
}
